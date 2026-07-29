package slack

import (
	"errors"
	"fmt"
	"strings"
	"time"

	goslack "github.com/slack-go/slack"
	"github.com/troyxmccall/janet/database"
)

const slackUserCacheTTL = 5 * time.Minute

var (
	errUnsafeUsernameLookup = errors.New("unsafe username")
	errSlackUserNotFound    = errors.New("slack user not found")
)

// EnrichUsersWithSlackInfo enriches user summaries with Slack profile information.
func (s *Service) EnrichUsersWithSlackInfo(users []*database.UserSummary) []*database.UserSummary {
	var slackLookupErr error
	for _, user := range users {
		if slackLookupErr != nil {
			continue
		}

		if slackUser, err := s.getSlackUserByUsername(user.Username); err == nil {
			user.DisplayName = &slackUser.Profile.DisplayName
			user.RealName = &slackUser.RealName
			if slackUser.Profile.Image192 != "" {
				user.AvatarURL = &slackUser.Profile.Image192
			}
			user.IsBot = slackUser.IsBot
			user.IsDeleted = slackUser.Deleted
		} else if isUserLookupMiss(err) {
			user.IsDeleted = true
			user.IsBot = false
		} else {
			slackLookupErr = err
			if s.logger != nil {
				s.logger.Err(err).KV("username", user.Username).Error("failed to enrich users with Slack info")
			}
		}
	}

	return users
}

func (s *Service) getSlackUserByUsername(username string) (*goslack.User, error) {
	if !isSafeUsernameLookup(username) {
		return nil, fmt.Errorf("%w: %s", errUnsafeUsernameLookup, username)
	}

	if cached, ok, fresh := s.getCachedSlackUserByUsername(username); ok && fresh {
		return cached, nil
	}

	if err := s.refreshUserCache(); err != nil {
		if cached, ok, _ := s.getCachedSlackUserByUsername(username); ok {
			return cached, nil
		}
		return nil, err
	}

	if cached, ok, _ := s.getCachedSlackUserByUsername(username); ok {
		return cached, nil
	}

	return nil, fmt.Errorf("%w: %s", errSlackUserNotFound, username)
}

func isUserLookupMiss(err error) bool {
	return errors.Is(err, errUnsafeUsernameLookup) || errors.Is(err, errSlackUserNotFound)
}

func isSafeUsernameLookup(username string) bool {
	return !strings.Contains(username, "@") && !strings.Contains(username, "+++++") &&
		!strings.HasPrefix(username, "*") && !strings.HasPrefix(username, "<") &&
		!strings.HasPrefix(username, ":") && !strings.Contains(username, "http")
}

func (s *Service) getCachedSlackUserByUsername(username string) (*goslack.User, bool, bool) {
	normalized := strings.ToLower(username)

	s.userCacheMu.RLock()
	defer s.userCacheMu.RUnlock()

	fresh := !s.userCacheFetchedAt.IsZero() && time.Since(s.userCacheFetchedAt) < slackUserCacheTTL
	if cached, ok := s.userCache[username]; ok {
		return &cached, true, fresh
	}
	if cached, ok := s.userCache[normalized]; ok {
		return &cached, true, fresh
	}

	return nil, false, fresh
}

func (s *Service) refreshUserCache() error {
	s.userCacheRefreshMu.Lock()
	defer s.userCacheRefreshMu.Unlock()

	s.userCacheMu.RLock()
	fresh := !s.userCacheFetchedAt.IsZero() && time.Since(s.userCacheFetchedAt) < slackUserCacheTTL
	s.userCacheMu.RUnlock()
	if fresh {
		return nil
	}

	client := s.client()
	if client == nil {
		return fmt.Errorf("no slack client available")
	}

	workspaceUsers, err := client.GetUsers()
	if err != nil {
		return err
	}

	nextCache := make(map[string]goslack.User, len(workspaceUsers)*4)
	for _, workspaceUser := range workspaceUsers {
		cacheSlackUser(nextCache, workspaceUser)
	}

	s.userCacheMu.Lock()
	s.userCache = nextCache
	s.userCacheFetchedAt = time.Now()
	s.userCacheMu.Unlock()

	return nil
}

func cacheSlackUser(cache map[string]goslack.User, user goslack.User) {
	if user.ID != "" {
		cache[user.ID] = user
	}
	if user.Name != "" {
		cache[user.Name] = user
		cache[strings.ToLower(user.Name)] = user
	}
	if user.Profile.DisplayName != "" {
		cache[user.Profile.DisplayName] = user
		cache[strings.ToLower(user.Profile.DisplayName)] = user
	}
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
