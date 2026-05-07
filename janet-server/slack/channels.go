package slack

import (
	"fmt"
	"strconv"
	"time"

	goslack "github.com/slack-go/slack"
)

// FindChannelByMessageID finds the channel_id for a message by checking all accessible channels.
func (s *Service) FindChannelByMessageID(messageID string) (string, error) {
	client := s.client()
	if client == nil {
		return "", fmt.Errorf("no slack client available")
	}

	allChannels, err := s.getAccessibleChannels(client)
	if err != nil {
		return "", err
	}

	for i := 0; i < len(allChannels); i++ {
		channel := allChannels[i]
		if i > 0 {
			time.Sleep(100 * time.Millisecond)
		}

		history, err := client.GetConversationHistory(&goslack.GetConversationHistoryParameters{
			ChannelID: channel.ID,
			Latest:    messageID,
			Inclusive: true,
			Limit:     1,
		})
		if err != nil {
			if rateLimitErr, ok := err.(*goslack.RateLimitedError); ok {
				waitDuration := rateLimitErr.RetryAfter + (1 * time.Second)
				if s.logger != nil {
					s.logger.KV("operation", "get_conversation_history").KV("channel_id", channel.ID).KV("message_id", messageID).KV("retry_after", waitDuration.String()).Info("popular_queue waiting_slack")
				}
				time.Sleep(waitDuration)
				i--
				continue
			}
			continue
		}

		if len(history.Messages) > 0 && history.Messages[0].Timestamp == messageID {
			return channel.ID, nil
		}
	}

	return "", fmt.Errorf("message not found in any accessible channel")
}

// FindChannelByMessageAuthorAndTimestamp searches Slack for a message by author and timestamp.
func (s *Service) FindChannelByMessageAuthorAndTimestamp(authorUsername, messageID string) (string, error) {
	if authorUsername == "" {
		return "", fmt.Errorf("author username required")
	}

	client := s.client()
	if client == nil {
		return "", fmt.Errorf("no slack client available")
	}

	ts, err := strconv.ParseFloat(messageID, 64)
	if err != nil {
		return "", fmt.Errorf("invalid message timestamp: %w", err)
	}

	msgTime := time.Unix(int64(ts), 0)
	after := msgTime.Add(-24 * time.Hour).Format("2006-01-02")
	before := msgTime.Add(24 * time.Hour).Format("2006-01-02")
	query := fmt.Sprintf("from:%s after:%s before:%s", authorUsername, after, before)

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		result, err := client.SearchMessages(query, goslack.SearchParameters{
			Count:     50,
			Page:      1,
			Highlight: false,
		})
		if err != nil {
			if rateLimitErr, ok := err.(*goslack.RateLimitedError); ok {
				waitDuration := rateLimitErr.RetryAfter + (1 * time.Second)
				if s.logger != nil {
					s.logger.KV("operation", "search_messages").KV("author", authorUsername).KV("retry_after", waitDuration.String()).Info("popular_queue waiting_slack")
				}
				time.Sleep(waitDuration)
				lastErr = err
				continue
			}
			return "", err
		}

		for _, match := range result.Matches {
			if match.Timestamp == messageID && match.Channel.ID != "" {
				msg, err := s.getMessage(match.Channel.ID, messageID)
				if err != nil {
					continue
				}
				if len(msg.Reactions) > 0 {
					return match.Channel.ID, nil
				}
			}
		}

		return "", fmt.Errorf("message not found in search results")
	}

	return "", lastErr
}

func (s *Service) getAccessibleChannels(client slackClient) ([]goslack.Channel, error) {
	cacheAge := time.Since(s.cacheFetchedAt)
	if len(s.channelCache) > 0 && cacheAge < 5*time.Minute {
		return s.channelCache, nil
	}

	var allChannels []goslack.Channel
	cursor := ""
	for {
		params := &goslack.GetConversationsParameters{
			Types:           []string{"public_channel", "private_channel"},
			ExcludeArchived: false,
			Limit:           200,
			Cursor:          cursor,
		}

		channels, nextCursor, err := client.GetConversations(params)
		if err != nil {
			if rateLimitErr, ok := err.(*goslack.RateLimitedError); ok {
				waitDuration := rateLimitErr.RetryAfter + (1 * time.Second)
				if s.logger != nil {
					s.logger.KV("operation", "list_channels").KV("retry_after", waitDuration.String()).Info("popular_queue waiting_slack")
				}
				time.Sleep(waitDuration)
				continue
			}
			return nil, fmt.Errorf("failed to list channels: %w", err)
		}

		allChannels = append(allChannels, channels...)
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
		time.Sleep(100 * time.Millisecond)
	}

	s.channelCache = allChannels
	s.cacheFetchedAt = time.Now()
	return allChannels, nil
}
