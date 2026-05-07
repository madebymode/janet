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

type DataStore interface {
	GetPopularMessages(limit int, year int, funnyBias bool) ([]*database.PopularMessage, error)
	GetPopularMessagesByUser(limit int, year int, username string, funnyBias bool) ([]*database.PopularMessage, error)
	GetPopularMessagesSince(since time.Time) ([]*database.PopularMessage, error)
	GetPopularMessageCount(year int, username string, minReactions int, funnyBias bool) (int, error)
	GetPopularMessageCountWithMedia(year int, username string, minReactions int, funnyBias bool) (int, error)
	GetPopularMessageDetails(messageID string) (*database.PopularMessageDetails, error)
	GetChannelIDForMessage(messageID string) (*string, error)
	GetMessageAuthorByMessageID(messageID string) (*string, error)
	UpdateChannelIDForMessage(messageID, channelID string) error
	UpsertPopularMessageDetails(messageID string, channelID, text, permalink, authorID, authorName, authorAvatar, imageURL, attachmentURL, attachmentMime *string, reactionCount *int, isReply, isIgnored, detailsFetched *bool) error

	GetUser(username string) (*database.UserSummary, error)
	GetUserByYear(username string, year int) (*database.UserSummary, error)
	GetUserMonthlyPointsByYear(username string, year int) ([]*database.MonthlyPoints, error)
	GetUserYearlyPoints(username string) ([]*database.YearlyPoints, error)

	GetTotalPointsByYear(year int) (int, error)
	GetTotalPointsCumulative() (int, error)
	GetTotalUsersByYear(year int) (int, error)
	GetTotalUsersCumulative() (int, error)
	GetTotalTransactionsByYear(year int) (int, error)
	GetTotalTransactionsCumulative() (int, error)
	GetPositiveTransactionsByYear(year int) (int, error)
	GetPositiveTransactionsCumulative() (int, error)
	GetNegativeTransactionsByYear(year int) (int, error)
	GetNegativeTransactionsCumulative() (int, error)
	GetTopGiversByYear(limit int, year int) ([]*database.UserSummary, error)
	GetTopGiversCumulative(limit int) ([]*database.UserSummary, error)
	GetAvailableYears() ([]int, error)
	GetTopEmojisByYear(year, limit int) ([]*database.EmojiStats, error)
	GetTotalEmojiUsageByYear(year int) (int, error)
	GetKarmaDistributionByYear(year int) ([]map[string]interface{}, error)
	GetActivityTimelineByYear(year int) ([]map[string]interface{}, error)
	GetPointsOverTimeMonthlyByYear(year int) ([]map[string]interface{}, error)
	GetCurrentLeaderboard(limit int) ([]*database.UserSummary, error)
	GetYearlyLeaderboard(year, limit int) ([]*database.UserSummary, error)
	GetRecentActivityPage(filter database.RecentActivityFilter) (*database.RecentActivityPage, error)
}

// Handler contains dependencies for API handlers
type Handler struct {
	db     DataStore
	bot    *janet.Bot
	logger *log.Log
	slack  SlackService

	popularBackfillQueue     chan popularBackfillJob
	popularBackfillSeen      map[string]struct{}
	popularBackfillMu        sync.Mutex
	popularBackfillOrderMu   sync.Mutex
	popularBackfillOrder     int64
	popularBackfillProcessed int64
	popularBackfillPositions map[string]int64
	popularDropMu            sync.Mutex
	popularDropCount         int
	popularDropLastLog       time.Time
	popularFailMu            sync.Mutex
	popularFailCounts        map[string]int
	popularFailLastLog       time.Time
	popularQueueLogMu        sync.Mutex
	popularQueueLastLog      time.Time
}

// NewHandler creates a new handler instance
func NewHandler(db DataStore, bot *janet.Bot, logger *log.Log, slack SlackService) *Handler {
	h := &Handler{
		db:                       db,
		bot:                      bot,
		logger:                   logger,
		slack:                    slack,
		popularBackfillQueue:     make(chan popularBackfillJob, 2000),
		popularBackfillSeen:      make(map[string]struct{}),
		popularBackfillPositions: make(map[string]int64),
		popularDropLastLog:       time.Now(),
		popularFailCounts:        make(map[string]int),
		popularFailLastLog:       time.Now(),
		popularQueueLastLog:      time.Now(),
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
