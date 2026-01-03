package database

import (
	"errors"
	"time"
)

// Errors
var (
	ErrNoSuchUser = errors.New("user is not in the database")
)

// Config for database connections
type Config struct {
	Driver string
	URL    string
}

// Transaction represents a karma transaction with enhanced metadata
type Transaction struct {
	ID              int       `json:"id"`
	FromUser        string    `json:"from_user"`
	ToUser          string    `json:"to_user"`
	Points          int       `json:"points"`
	Reason          string    `json:"reason"`
	TransactionType string    `json:"transaction_type"` // manual, emoji, reactji
	EmojiName       *string   `json:"emoji_name,omitempty"`
	ChannelID       *string   `json:"channel_id,omitempty"`
	ChannelName     *string   `json:"channel_name,omitempty"`
	MessageID       *string   `json:"message_id,omitempty"`
	Year            int       `json:"year"`
	Timestamp       time.Time `json:"timestamp"`
}

// UserSummary represents comprehensive user statistics
type UserSummary struct {
	Username               string     `json:"username"`
	DisplayName            *string    `json:"display_name,omitempty"`
	RealName               *string    `json:"real_name,omitempty"`
	AvatarURL              *string    `json:"avatar_url,omitempty"`
	IsBot                  bool       `json:"is_bot"`
	IsDeleted              bool       `json:"is_deleted"`
	Year                   *int       `json:"year,omitempty"`
	TotalPoints            int        `json:"total_points"`
	PointsGiven            int        `json:"points_given"`
	PointsReceived         int        `json:"points_received"`
	TransactionsGiven      int        `json:"transactions_given"`
	TransactionsReceived   int        `json:"transactions_received"`
	EmojiReactionsGiven    int        `json:"emoji_reactions_given"`
	EmojiReactionsReceived int        `json:"emoji_reactions_received"`
	Rank                   int        `json:"rank"`
	LastActivity           *time.Time `json:"last_activity,omitempty"`
}

// EmojiStats represents emoji usage statistics
type EmojiStats struct {
	EmojiName     string `json:"emoji_name"`
	Year          int    `json:"year"`
	UsageCount    int    `json:"usage_count"`
	PointsAwarded int    `json:"points_awarded"`
	UniqueUsers   int    `json:"unique_users"`
	Rank          int    `json:"rank"`
}

// EmojiLeaderboard represents top users for specific emoji
type EmojiLeaderboard struct {
	EmojiName       string `json:"emoji_name"`
	Year            int    `json:"year"`
	Username        string `json:"username"`
	TimesGiven      int    `json:"times_given"`
	TimesReceived   int    `json:"times_received"`
	PointsFromEmoji int    `json:"points_from_emoji"`
}

// MonthlyStats represents monthly aggregated statistics
type MonthlyStats struct {
	Year               int     `json:"year"`
	Month              int     `json:"month"`
	TotalTransactions  int     `json:"total_transactions"`
	TotalPointsAwarded int     `json:"total_points_awarded"`
	UniqueUsers        int     `json:"unique_users"`
	TopEmoji           *string `json:"top_emoji,omitempty"`
	MostActiveUser     *string `json:"most_active_user,omitempty"`
}

// BackfillRecord represents a potential record to backfill
type BackfillRecord struct {
	FromUser        string
	ToUser          string
	Points          int
	Reason          string
	TransactionType string
	EmojiName       *string
	ChannelID       string
	MessageID       string
	Timestamp       time.Time
}

// BackfillStats represents statistics from a backfill operation
type BackfillStats struct {
	MessagesProcessed int `json:"messages_processed"`
	KarmaFound        int `json:"karma_found"`
	RecordsAdded      int `json:"records_added"`
	DuplicatesSkipped int `json:"duplicates_skipped"`
	ErrorsEncountered int `json:"errors_encountered"`
	DurationMs        int `json:"duration_ms"`
}

// PopularMessage represents a message with reaction counts
type PopularMessage struct {
	ChannelID     *string `json:"channel_id,omitempty"`
	MessageID     string  `json:"message_id"`
	ReactionCount int     `json:"reaction_count"`
	TotalPoints   int     `json:"total_points"`
}

// PopularMessageDetails represents cached Slack metadata for a message
type PopularMessageDetails struct {
	MessageID string
	ChannelID *string
	Text      *string
	Permalink *string
	AuthorID  *string
	AuthorName *string
	AuthorAvatar *string
	ImageURL  *string
	AttachmentURL *string
	AttachmentMime *string
	ReactionCount *int
	IsReply   *bool
	IsIgnored *bool
}

// Legacy types for backward compatibility
type Points struct {
	From, To, Reason string
	Points           int
}

type User struct {
	Name   string
	Points int
}

type Leaderboard []*User

type Throwback struct {
	Points
	Timestamp time.Time
}
