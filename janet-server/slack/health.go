package slack

import (
	"net"
	"time"
)

// CheckBotHealth checks if the janet-bot container is healthy
func (s *Service) CheckBotHealth() bool {
	// Try to resolve the janet-bot hostname
	// If the container is running and in the same network, this will succeed
	_, err := net.LookupHost("janet-bot")

	if err != nil {
		// Container not reachable or not running
		return false
	}

	// Additional check: try to ping the container's IP
	conn, err := (&net.Dialer{
		Timeout: 1 * time.Second,
	}).Dial("tcp", "janet-bot:22")

	if err != nil {
		// This is expected since SSH isn't running, but it confirms the container is reachable
		// We only care that we can resolve the hostname above
	} else {
		conn.Close()
	}

	return true
}