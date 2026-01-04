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
	popularBackfillOrderMu sync.Mutex
	popularBackfillOrder   int64
	popularBackfillProcessed int64
	popularBackfillPositions map[string]int64
	popularDropMu        sync.Mutex
	popularDropCount     int
	popularDropLastLog   time.Time
	popularFailMu        sync.Mutex
	popularFailCounts    map[string]int
	popularFailLastLog   time.Time
	popularQueueLogMu    sync.Mutex
	popularQueueLastLog  time.Time
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
		popularBackfillPositions: make(map[string]int64),
		popularDropLastLog:   time.Now(),
		popularFailCounts:    make(map[string]int),
		popularFailLastLog:   time.Now(),
		popularQueueLastLog:  time.Now(),
	}
	h.startPopularMessageBackfillWorker()
	h.startDailyPopularRefresh(7 * 24 * time.Hour)
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

func (h *Handler) popularBackfillQueuePosition(messageID string) int {
	h.popularBackfillOrderMu.Lock()
	order := h.popularBackfillPositions[messageID]
	processed := h.popularBackfillProcessed
	h.popularBackfillOrderMu.Unlock()
	if order == 0 {
		return 0
	}
	pos := int(order - processed)
	if pos < 1 {
		return 0
	}
	return pos
}

func (h *Handler) startDailyPopularRefresh(window time.Duration) {
	if h.slack == nil {
		return
	}

	go func() {
		for {
			now := time.Now()
			nextRun := time.Date(now.Year(), now.Month(), now.Day(), 23, 55, 0, 0, now.Location())
			if !nextRun.After(now) {
				nextRun = nextRun.Add(24 * time.Hour)
			}
			sleepDuration := time.Until(nextRun)
			h.logger.KV("window", window.String()).KV("next_run", nextRun).Info("popular_queue daily_refresh_scheduled")
			time.Sleep(sleepDuration)
			h.refreshPopularMessagesSince(time.Now().Add(-window))
		}
	}()
}

func (h *Handler) refreshPopularMessagesSince(since time.Time) {
	if h.slack == nil {
		return
	}

	messages, err := h.db.GetPopularMessagesSince(since)
	if err != nil {
		h.logger.Err(err).KV("since", since).Error("popular_queue daily_refresh_failed")
		return
	}
	h.logger.KV("since", since).KV("count", len(messages)).Info("popular_queue daily_refresh_start")
	for _, msg := range messages {
		channelID := ""
		if msg.ChannelID != nil {
			channelID = *msg.ChannelID
		}
		h.enqueuePopularMessageRefresh(msg.MessageID, channelID)
	}
	h.logger.KV("since", since).KV("count", len(messages)).Info("popular_queue daily_refresh_enqueued")
}
