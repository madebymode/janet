package database

import "time"

// TransactionRepository defines methods for transaction operations
type TransactionRepository interface {
	InsertTransaction(tx *Transaction) error

	// Recent activity methods
	GetRecentActivity(limit int) ([]*Transaction, error) // Default: current year
	GetRecentActivityByCurrentYear(limit int) ([]*Transaction, error)
	GetRecentActivityByYear(limit int, year int) ([]*Transaction, error)
	GetRecentActivityCumulative(limit int) ([]*Transaction, error)

	// User transactions methods
	GetTransactionsByUser(username string) ([]*Transaction, error) // Default: current year
	GetTransactionsByUserByCurrentYear(username string) ([]*Transaction, error)
	GetTransactionsByUserByYear(username string, year int) ([]*Transaction, error)
	GetTransactionsByUserCumulative(username string) ([]*Transaction, error)

	// Transaction count methods
	GetTotalTransactions() (int, error) // Default: current year
	GetTotalTransactionsByCurrentYear() (int, error)
	GetTotalTransactionsByYear(year int) (int, error)
	GetTotalTransactionsCumulative() (int, error)

	GetPositiveTransactions() (int, error) // Default: current year
	GetPositiveTransactionsByCurrentYear() (int, error)
	GetPositiveTransactionsByYear(year int) (int, error)
	GetPositiveTransactionsCumulative() (int, error)

	GetNegativeTransactions() (int, error) // Default: current year
	GetNegativeTransactionsByCurrentYear() (int, error)
	GetNegativeTransactionsByYear(year int) (int, error)
	GetNegativeTransactionsCumulative() (int, error)
}

// UserRepository defines methods for user operations
type UserRepository interface {
	// User data methods
	GetUser(username string) (*UserSummary, error)
	GetUserByCurrentYear(username string) (*UserSummary, error)
	GetUserByYear(username string, year int) (*UserSummary, error)
	GetUserCumulative(username string) (*UserSummary, error)

	// Leaderboard methods
	GetLeaderboard(limit int) ([]*UserSummary, error) // Default: current year
	GetLeaderboardByCurrentYear(limit int) ([]*UserSummary, error)
	GetLeaderboardByYear(year, limit int) ([]*UserSummary, error)
	GetLeaderboardCumulative(limit int) ([]*UserSummary, error)

	// Top givers methods
	GetTopGivers(limit int) ([]*UserSummary, error) // Default: current year
	GetTopGiversByCurrentYear(limit int) ([]*UserSummary, error)
	GetTopGiversByYear(limit int, year int) ([]*UserSummary, error)
	GetTopGiversCumulative(limit int) ([]*UserSummary, error)

	// User count methods
	GetTotalUsers() (int, error) // Default: current year
	GetTotalUsersByCurrentYear() (int, error)
	GetTotalUsersByYear(year int) (int, error)
	GetTotalUsersCumulative() (int, error)
}

// EmojiRepository defines methods for emoji operations
type EmojiRepository interface {
	// Top emojis methods
	GetTopEmojis(limit int) ([]*EmojiStats, error) // Default: current year
	GetTopEmojisByCurrentYear(limit int) ([]*EmojiStats, error)
	GetTopEmojisByYear(year, limit int) ([]*EmojiStats, error)
	GetTopEmojisCumulative(limit int) ([]*EmojiStats, error)

	// Emoji usage methods
	GetTotalEmojiUsage() (int, error) // Default: current year
	GetTotalEmojiUsageByCurrentYear() (int, error)
	GetTotalEmojiUsageByYear(year int) (int, error)
	GetTotalEmojiUsageCumulative() (int, error)

	// Emoji leaderboard methods
	GetEmojiLeaderboard(emojiName string) ([]*EmojiLeaderboard, error) // Default: current year
	GetEmojiLeaderboardByCurrentYear(emojiName string) ([]*EmojiLeaderboard, error)
	GetEmojiLeaderboardByYear(emojiName string, year int) ([]*EmojiLeaderboard, error)
	GetEmojiLeaderboardCumulative(emojiName string) ([]*EmojiLeaderboard, error)
}

// StatsRepository defines methods for statistics operations
type StatsRepository interface {
	// Total points methods
	GetTotalPoints() (int, error) // Default: current year
	GetTotalPointsByCurrentYear() (int, error)
	GetTotalPointsByYear(year int) (int, error)
	GetTotalPointsCumulative() (int, error)

	// Monthly stats methods
	GetMonthlyStats() ([]*MonthlyStats, error) // Default: current year
	GetMonthlyStatsByCurrentYear() ([]*MonthlyStats, error)
	GetMonthlyStatsByYear(year int) ([]*MonthlyStats, error)

	// Points over time methods
	GetPointsOverTimeMonthly() ([]map[string]interface{}, error) // Default: current year
	GetPointsOverTimeMonthlyByCurrentYear() ([]map[string]interface{}, error)
	GetPointsOverTimeMonthlyByYear(year int) ([]map[string]interface{}, error)

	// Activity timeline methods
	GetActivityTimeline() ([]map[string]interface{}, error) // Default: current year
	GetActivityTimelineByCurrentYear() ([]map[string]interface{}, error)
	GetActivityTimelineByYear(year int) ([]map[string]interface{}, error)

	// Karma distribution methods
	GetKarmaDistribution() ([]map[string]interface{}, error) // Default: current year
	GetKarmaDistributionByCurrentYear() ([]map[string]interface{}, error)
	GetKarmaDistributionByYear(year int) ([]map[string]interface{}, error)
	GetKarmaDistributionCumulative() ([]map[string]interface{}, error)

	// Utility methods
	GetAvailableYears() ([]int, error)
}

// SummaryRepository defines methods for summary table operations
type SummaryRepository interface {
	RebuildSummaryTables() error
	UpdateSummaryTables(toUser, fromUser string, year int) error
}

// BackfillRepository defines methods for backfill operations
type BackfillRepository interface {
	InsertBackfillRecord(record *BackfillRecord) error
	GetBackfillStats(startTime, endTime time.Time) (*BackfillStats, error)
}

// LegacyRepository defines methods for backward compatibility
type LegacyRepository interface {
	InsertPoints(points *Points) error
	GetLegacyLeaderboard(limit int) (Leaderboard, error)
	GetThrowback(user string) (*Throwback, error)
}

// Database defines the complete database interface
type Database interface {
	TransactionRepository
	UserRepository
	EmojiRepository
	StatsRepository
	SummaryRepository
	BackfillRepository
	LegacyRepository

	// Connection management
	Init() error
	Close() error
}
