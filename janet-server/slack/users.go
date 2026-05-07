package slack

import (
	"fmt"
	"strings"
	"time"

	goslack "github.com/slack-go/slack"
	"github.com/troyxmccall/janet/database"
)

// EnrichUsersWithSlackInfo enriches user summaries with Slack profile information.
func (s *Service) EnrichUsersWithSlackInfo(users []*database.UserSummary) []*database.UserSummary {
	for _, user := range users {
		if slackUser, err := s.getSlackUserByUsername(user.Username); err == nil {
			user.DisplayName = &slackUser.Profile.DisplayName
			user.RealName = &slackUser.RealName
			if slackUser.Profile.Image192 != "" {
				user.AvatarURL = &slackUser.Profile.Image192
			}
			user.IsBot = slackUser.IsBot
			user.IsDeleted = slackUser.Deleted
		} else {
			user.IsDeleted = true
			user.IsBot = false
		}
	}

	return users
}

func (s *Service) getSlackUserByUsername(username string) (*goslack.User, error) {
	if strings.Contains(username, "@") || strings.Contains(username, "+++++") ||
		strings.HasPrefix(username, "*") || strings.HasPrefix(username, "<") ||
		strings.HasPrefix(username, ":") || strings.Contains(username, "http") {
		return nil, fmt.Errorf("corrupted username: %s", username)
	}

	s.userCacheMu.RLock()
	if cached, ok := s.userCache[username]; ok {
		s.userCacheMu.RUnlock()
		return &cached, nil
	}
	s.userCacheMu.RUnlock()

	client := s.client()
	if client == nil {
		return nil, fmt.Errorf("no slack client available")
	}

	workspaceUsers, err := client.GetUsers()
	if err != nil {
		return nil, err
	}

	normalized := strings.ToLower(username)
	s.userCacheMu.Lock()
	defer s.userCacheMu.Unlock()
	for _, workspaceUser := range workspaceUsers {
		s.userCache[workspaceUser.ID] = workspaceUser
		s.userCache[workspaceUser.Name] = workspaceUser
		if workspaceUser.Profile.DisplayName != "" {
			s.userCache[strings.ToLower(workspaceUser.Profile.DisplayName)] = workspaceUser
		}
		s.userCache[strings.ToLower(workspaceUser.Name)] = workspaceUser
	}

	if cached, ok := s.userCache[normalized]; ok {
		return &cached, nil
	}
	if cached, ok := s.userCache[username]; ok {
		return &cached, nil
	}

	return nil, fmt.Errorf("user not found: %s", username)
}

func (s *Service) getUserInfoWithRetry(userID string) (*goslack.User, error) {
	client := s.client()
	if client == nil {
		return nil, fmt.Errorf("no slack client available")
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		user, err := client.GetUserInfo(userID)
		if err == nil {
			return user, nil
		}
		if rateLimitErr, ok := err.(*goslack.RateLimitedError); ok {
			waitDuration := rateLimitErr.RetryAfter + (1 * time.Second)
			if s.logger != nil {
				s.logger.KV("operation", "get_user_info").KV("user_id", userID).KV("retry_after", waitDuration.String()).Info("popular_queue waiting_slack")
			}
			time.Sleep(waitDuration)
			lastErr = err
			continue
		}
		return nil, err
	}

	return nil, lastErr
}
