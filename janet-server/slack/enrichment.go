package slack

import (
	"fmt"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/troyxmccall/janet"
	"github.com/troyxmccall/janet/database"
)

// Service handles Slack integration operations
type Service struct {
	bot           *janet.Bot
	slackClient   *slack.Client
	channelCache  []slack.Channel
	cacheFetchedAt time.Time
}

// NewService creates a new Slack service
func NewService(bot *janet.Bot) *Service {
	return &Service{
		bot:         bot,
		slackClient: nil,
	}
}

// NewWebService creates a new Slack service with a standalone client for web-only mode
func NewWebService(slackClient *slack.Client) *Service {
	return &Service{
		bot:         nil,
		slackClient: slackClient,
	}
}

// EnrichUsersWithSlackInfo enriches user summaries with Slack profile information
func (s *Service) EnrichUsersWithSlackInfo(users []*database.UserSummary) []*database.UserSummary {
	if s.bot == nil {
		return users // Return unchanged if no bot available
	}

	for _, user := range users {
		// Try to get Slack user info
		if slackUser, err := s.getSlackUserByUsername(user.Username); err == nil {
			// Enrich with Slack profile data
			user.DisplayName = &slackUser.Profile.DisplayName
			user.RealName = &slackUser.RealName
			if slackUser.Profile.Image192 != "" {
				user.AvatarURL = &slackUser.Profile.Image192
			}
			user.IsBot = slackUser.IsBot
			user.IsDeleted = slackUser.Deleted
		} else {
			// If we can't find the user in Slack, mark them as deleted/invalid
			// This handles corrupted usernames, deleted users, etc.
			user.IsDeleted = true
			user.IsBot = false // Assume not a bot if we can't verify
		}
	}

	return users
}

// getSlackUserByUsername attempts to find a Slack user by their username
func (s *Service) getSlackUserByUsername(username string) (*slack.User, error) {
	if s.bot == nil {
		return nil, fmt.Errorf("bot not available")
	}

	// Skip obviously corrupted usernames
	if strings.Contains(username, "@") || strings.Contains(username, "+++++") ||
	   strings.HasPrefix(username, "*") || strings.HasPrefix(username, "<") ||
	   strings.HasPrefix(username, ":") || strings.Contains(username, "http") {
		return nil, fmt.Errorf("corrupted username: %s", username)
	}

	// Try to get user info directly by username
	return s.bot.GetSlackUserInfo(username)
}

// GetMessagePermalink gets a permalink to a Slack message
func (s *Service) GetMessagePermalink(channelID, messageID string) (string, error) {
	var client *slack.Client

	// Use standalone client if available, otherwise use bot's client
	if s.slackClient != nil {
		client = s.slackClient
	} else if s.bot != nil && s.bot.Config.SlackWebClient != nil {
		client = s.bot.Config.SlackWebClient
	} else {
		return "", fmt.Errorf("no slack client available")
	}

	params := &slack.PermalinkParameters{
		Channel: channelID,
		Ts:      messageID,
	}

	return client.GetPermalink(params)
}

// GetMessageText fetches the text content of a Slack message
func (s *Service) GetMessageText(channelID, messageID string) (string, error) {
	var client *slack.Client

	// Use standalone client if available, otherwise use bot's client
	if s.slackClient != nil {
		client = s.slackClient
	} else if s.bot != nil && s.bot.Config.SlackWebClient != nil {
		client = s.bot.Config.SlackWebClient
	} else {
		return "", fmt.Errorf("no slack client available")
	}

	// Fetch conversation history for this specific message
	history, err := client.GetConversationHistory(&slack.GetConversationHistoryParameters{
		ChannelID: channelID,
		Latest:    messageID,
		Inclusive: true,
		Limit:     1,
	})

	if err != nil {
		return "", fmt.Errorf("failed to get conversation history: %w", err)
	}

	if len(history.Messages) == 0 {
		return "", fmt.Errorf("message not found")
	}

	return history.Messages[0].Text, nil
}

// FindChannelByMessageID finds the channel_id for a message by checking all accessible channels
// This is a fallback when channel_id is not stored in the database
func (s *Service) FindChannelByMessageID(messageID string) (string, error) {
	var client *slack.Client

	// Use standalone client if available, otherwise use bot's client
	if s.slackClient != nil {
		client = s.slackClient
	} else if s.bot != nil && s.bot.Config.SlackWebClient != nil {
		client = s.bot.Config.SlackWebClient
	} else {
		return "", fmt.Errorf("no slack client available")
	}

	// Use cached channel list if available and recent (within 5 minutes)
	// This prevents hitting rate limits by re-fetching the channel list for every message
	var allChannels []slack.Channel
	cacheAge := time.Since(s.cacheFetchedAt)

	if len(s.channelCache) > 0 && cacheAge < 5*time.Minute {
		// Use cached channels
		allChannels = s.channelCache
	} else {
		// Fetch fresh channel list with pagination
		cursor := ""

		for {
			params := &slack.GetConversationsParameters{
				Types:           []string{"public_channel", "private_channel"},
				ExcludeArchived: false,
				Limit:           200,
				Cursor:          cursor,
			}

			channels, nextCursor, err := client.GetConversations(params)
			if err != nil {
				// Handle rate limit errors
				if rateLimitErr, ok := err.(*slack.RateLimitedError); ok {
					// Wait for the retry after duration plus buffer
					waitDuration := rateLimitErr.RetryAfter + (1 * time.Second)
					time.Sleep(waitDuration)
					// Retry this page (don't advance cursor)
					continue
				}
				return "", fmt.Errorf("failed to list channels: %w", err)
			}

			allChannels = append(allChannels, channels...)

			if nextCursor == "" {
				break
			}
			cursor = nextCursor

			// Add small delay between pagination requests to avoid rate limits
			time.Sleep(100 * time.Millisecond)
		}

		// Cache the channel list
		s.channelCache = allChannels
		s.cacheFetchedAt = time.Now()
	}

	// Try to fetch the message from each channel using the exact timestamp
	// This mirrors how the bot fetches messages: GetConversationHistory with Latest=timestamp
	for i := 0; i < len(allChannels); i++ {
		channel := allChannels[i]

		// Rate limiting: Slack allows ~50 requests/min for Tier 3 methods (conversations.history)
		// Add a small delay between requests to avoid hitting limits
		// 100ms delay = max 600 requests/min, well under the limit
		if i > 0 {
			time.Sleep(100 * time.Millisecond)
		}

		history, err := client.GetConversationHistory(&slack.GetConversationHistoryParameters{
			ChannelID: channel.ID,
			Latest:    messageID,  // Use message timestamp as Latest
			Inclusive: true,       // Include the message at Latest timestamp
			Limit:     1,
		})

		if err != nil {
			// Check if it's a rate limit error
			if rateLimitErr, ok := err.(*slack.RateLimitedError); ok {
				// Wait for the retry after duration plus a small buffer
				time.Sleep(rateLimitErr.RetryAfter + (1 * time.Second))
				// Retry this channel
				i--
				continue
			}
			// Channel might be inaccessible or other error, skip it
			continue
		}

		// Check if we found the exact message by comparing timestamps
		if len(history.Messages) > 0 && history.Messages[0].Timestamp == messageID {
			// Found it! Return the channel ID
			return channel.ID, nil
		}
	}

	return "", fmt.Errorf("message not found in any accessible channel")
}