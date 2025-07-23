package slack

import (
	"fmt"
	"strings"

	"github.com/slack-go/slack"
	"github.com/troyxmccall/janet"
	"github.com/troyxmccall/janet/database"
)

// Service handles Slack integration operations
type Service struct {
	bot *janet.Bot
}

// NewService creates a new Slack service
func NewService(bot *janet.Bot) *Service {
	return &Service{
		bot: bot,
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