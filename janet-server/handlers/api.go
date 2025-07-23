package handlers

import (
	"regexp"

	"github.com/troyxmccall/janet/database"
)

// isValidUsername validates username input to prevent injection attacks
func isValidUsername(username string) bool {
	if username == "" || len(username) > 50 {
		return false
	}
	// Allow alphanumeric characters, underscores, hyphens, and dots
	validUsernameRegex := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	return validUsernameRegex.MatchString(username)
}

// filterActiveUsers filters out bots, deleted users, and users with less than 20 points
func filterActiveUsers(users []*database.UserSummary) []*database.UserSummary {
	filtered := make([]*database.UserSummary, 0, len(users))
	for _, user := range users {
		if !user.IsBot && !user.IsDeleted && user.TotalPoints >= 20 {
			filtered = append(filtered, user)
		}
	}
	return filtered
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}