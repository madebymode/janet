package handlers

import (
	"github.com/aybabtme/log"
	"github.com/troyxmccall/janet"
	"github.com/troyxmccall/janet/database"
)

// SlackService interface for Slack operations
type SlackService interface {
	EnrichUsersWithSlackInfo(users []*database.UserSummary) []*database.UserSummary
	CheckBotHealth() bool
	GetMessageText(channelID, messageID string) (string, error)
	GetMessagePermalink(channelID, messageID string) (string, error)
	FindChannelByMessageID(messageID string) (string, error)
}

// Handler contains dependencies for API handlers
type Handler struct {
	db     *database.V2DB
	bot    *janet.Bot
	logger *log.Log
	slack  SlackService
}

// NewHandler creates a new handler instance
func NewHandler(db *database.V2DB, bot *janet.Bot, logger *log.Log, slack SlackService) *Handler {
	return &Handler{
		db:     db,
		bot:    bot,
		logger: logger,
		slack:  slack,
	}
}