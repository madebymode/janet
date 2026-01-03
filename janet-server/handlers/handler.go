package handlers

import (
	"sync"
	"time"

	"github.com/aybabtme/log"
	"github.com/troyxmccall/janet"
	"github.com/troyxmccall/janet/database"
	slacksvc "github.com/troyxmccall/janet/janet-server/slack"
)

// SlackService interface for Slack operations
type SlackService interface {
	EnrichUsersWithSlackInfo(users []*database.UserSummary) []*database.UserSummary
	CheckBotHealth() bool
	GetMessageText(channelID, messageID string) (string, error)
	GetMessagePermalink(channelID, messageID string) (string, error)
	FindChannelByMessageID(messageID string) (string, error)
	FindChannelByMessageAuthorAndTimestamp(authorUsername, messageID string) (string, error)
	GetMessageDetails(channelID, messageID string) (*slacksvc.MessageDetails, error)
}

// Handler contains dependencies for API handlers
type Handler struct {
	db     *database.V2DB
	bot    *janet.Bot
	logger *log.Log
	slack  SlackService

	popularBackfillQueue chan popularBackfillJob
	popularBackfillSeen  map[string]struct{}
	popularBackfillMu    sync.Mutex
	popularDropMu        sync.Mutex
	popularDropCount     int
	popularDropLastLog   time.Time
	popularFailMu        sync.Mutex
	popularFailCounts    map[string]int
	popularFailLastLog   time.Time
}

// NewHandler creates a new handler instance
func NewHandler(db *database.V2DB, bot *janet.Bot, logger *log.Log, slack SlackService) *Handler {
	h := &Handler{
		db:     db,
		bot:    bot,
		logger: logger,
		slack:  slack,
		popularBackfillQueue: make(chan popularBackfillJob, 2000),
		popularBackfillSeen:  make(map[string]struct{}),
		popularDropLastLog:   time.Now(),
		popularFailCounts:    make(map[string]int),
		popularFailLastLog:   time.Now(),
	}
	h.startPopularMessageBackfillWorker()
	return h
}

func (h *Handler) popPopularBackfillFailures() map[string]int {
	h.popularFailMu.Lock()
	defer h.popularFailMu.Unlock()
	if len(h.popularFailCounts) == 0 {
		return nil
	}
	counts := h.popularFailCounts
	h.popularFailCounts = make(map[string]int)
	h.popularFailLastLog = time.Now()
	return counts
}

func (h *Handler) popularBackfillQueueSize() int {
	if h.popularBackfillQueue == nil {
		return 0
	}
	return len(h.popularBackfillQueue)
}
