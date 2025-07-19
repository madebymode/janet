package janet

import "strings"

// Munge formats a username for display
func Munge(username string) string {
	return strings.ToLower(username)
}
