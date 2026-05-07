package slack

import (
	"fmt"
	"time"

	goslack "github.com/slack-go/slack"
)

// MessageDetails captures Slack message metadata used in the UI.
type MessageDetails struct {
	Text           string
	Permalink      string
	AuthorID       string
	AuthorName     string
	AuthorAvatar   string
	ImageURL       string
	AttachmentURL  string
	AttachmentMime string
	HasImage       bool
	HasAttachment  bool
	IsReply        bool
	IsIgnored      bool
	ReactionCount  int
}

// GetMessagePermalink gets a permalink to a Slack message.
func (s *Service) GetMessagePermalink(channelID, messageID string) (string, error) {
	client := s.client()
	if client == nil {
		return "", fmt.Errorf("no slack client available")
	}

	params := &goslack.PermalinkParameters{
		Channel: channelID,
		Ts:      messageID,
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		permalink, err := client.GetPermalink(params)
		if err == nil {
			return permalink, nil
		}
		if rateLimitErr, ok := err.(*goslack.RateLimitedError); ok {
			waitDuration := rateLimitErr.RetryAfter + (1 * time.Second)
			if s.logger != nil {
				s.logger.KV("operation", "get_permalink").KV("channel_id", channelID).KV("message_id", messageID).KV("retry_after", waitDuration.String()).Info("popular_queue waiting_slack")
			}
			time.Sleep(waitDuration)
			lastErr = err
			continue
		}
		return "", err
	}

	return "", lastErr
}

// GetMessageText fetches the text content of a Slack message.
func (s *Service) GetMessageText(channelID, messageID string) (string, error) {
	message, err := s.getMessage(channelID, messageID)
	if err != nil {
		return "", err
	}
	return message.Text, nil
}

// GetMessageDetails fetches message text, permalink, author, and media URLs.
func (s *Service) GetMessageDetails(channelID, messageID string) (*MessageDetails, error) {
	if details, ok := s.getCachedMessageDetails(channelID, messageID); ok {
		return details, nil
	}

	permalink, err := s.GetMessagePermalink(channelID, messageID)
	if err != nil {
		return nil, err
	}

	message, err := s.getMessage(channelID, messageID)
	if err != nil {
		return nil, err
	}

	details := &MessageDetails{
		Text:      message.Text,
		Permalink: permalink,
	}
	if message.ThreadTimestamp != "" && message.ThreadTimestamp != message.Timestamp {
		details.IsReply = true
	}
	if ignoredPlusRegex.MatchString(message.Text) {
		details.IsIgnored = true
	}

	imageURL, hasImage := s.getMessageImageURL(message)
	details.HasImage = hasImage
	if imageURL != "" {
		details.ImageURL = imageURL
	}

	attachmentURL, attachmentMime, hasAttachment := s.getMessageAttachment(message)
	details.HasAttachment = hasAttachment
	if attachmentURL != "" {
		details.AttachmentURL = attachmentURL
		details.AttachmentMime = attachmentMime
	}

	details.ReactionCount = countMessageReactions(message)

	if message.User != "" {
		details.AuthorID = message.User
		if user, err := s.getUserInfoWithRetry(message.User); err == nil {
			details.AuthorName = user.Name
			details.AuthorAvatar = user.Profile.Image72
		}
	} else if message.Username != "" {
		details.AuthorName = message.Username
	}

	s.setCachedMessageDetails(channelID, messageID, details)
	return details, nil
}

func (s *Service) getMessage(channelID, messageID string) (goslack.Message, error) {
	client := s.client()
	if client == nil {
		return goslack.Message{}, fmt.Errorf("no slack client available")
	}

	var history *goslack.GetConversationHistoryResponse
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		history, err = client.GetConversationHistory(&goslack.GetConversationHistoryParameters{
			ChannelID: channelID,
			Latest:    messageID,
			Inclusive: true,
			Limit:     1,
		})
		if err == nil {
			break
		}
		if rateLimitErr, ok := err.(*goslack.RateLimitedError); ok {
			waitDuration := rateLimitErr.RetryAfter + (1 * time.Second)
			if s.logger != nil {
				s.logger.KV("operation", "get_message").KV("channel_id", channelID).KV("message_id", messageID).KV("retry_after", waitDuration.String()).Info("popular_queue waiting_slack")
			}
			time.Sleep(waitDuration)
			continue
		}
		return goslack.Message{}, fmt.Errorf("failed to get conversation history: %w", err)
	}
	if err != nil {
		return goslack.Message{}, fmt.Errorf("failed to get conversation history: %w", err)
	}
	if len(history.Messages) == 0 {
		return goslack.Message{}, fmt.Errorf("message not found")
	}

	return history.Messages[0], nil
}

func countMessageReactions(message goslack.Message) int {
	total := 0
	for _, reaction := range message.Reactions {
		total += reaction.Count
	}
	return total
}
