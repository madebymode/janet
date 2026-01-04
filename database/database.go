package database

import (
	"time"
)

// Ensure V2DB implements all repository interfaces
var _ Database = (*V2DB)(nil)

// Service composition - V2DB embeds all service functionality
func (db *V2DB) transactionService() *TransactionService {
	return NewTransactionService(db)
}

func (db *V2DB) userService() *UserService {
	return NewUserService(db)
}

func (db *V2DB) emojiService() *EmojiService {
	return NewEmojiService(db)
}

func (db *V2DB) statsService() *StatsService {
	return NewStatsService(db)
}

func (db *V2DB) summaryService() *SummaryService {
	return NewSummaryService(db)
}

func (db *V2DB) backfillService() *BackfillService {
	return NewBackfillService(db)
}

func (db *V2DB) legacyService() *LegacyService {
	return NewLegacyService(db)
}

// TransactionRepository implementation
func (db *V2DB) InsertTransaction(tx *Transaction) error {
	return db.transactionService().InsertTransaction(tx)
}

// Recent activity methods
func (db *V2DB) GetRecentActivity(limit int) ([]*Transaction, error) {
	return db.transactionService().GetRecentActivity(limit)
}

func (db *V2DB) GetRecentActivityByCurrentYear(limit int) ([]*Transaction, error) {
	return db.transactionService().GetRecentActivityByCurrentYear(limit)
}

func (db *V2DB) GetRecentActivityByYear(limit int, year int) ([]*Transaction, error) {
	return db.transactionService().GetRecentActivityByYear(limit, year)
}

func (db *V2DB) GetRecentActivityCumulative(limit int) ([]*Transaction, error) {
	return db.transactionService().GetRecentActivityCumulative(limit)
}

// User transactions methods
func (db *V2DB) GetTransactionsByUser(username string) ([]*Transaction, error) {
	return db.transactionService().GetTransactionsByUserByCurrentYear(username)
}

func (db *V2DB) GetTransactionsByUserByCurrentYear(username string) ([]*Transaction, error) {
	return db.transactionService().GetTransactionsByUserByCurrentYear(username)
}

func (db *V2DB) GetTransactionsByUserByYear(username string, year int) ([]*Transaction, error) {
	return db.transactionService().GetTransactionsByUserByYear(username, year)
}

func (db *V2DB) GetTransactionsByUserCumulative(username string) ([]*Transaction, error) {
	return db.transactionService().GetTransactionsByUserCumulative(username)
}

// Transaction count methods
func (db *V2DB) GetTotalTransactions() (int, error) {
	return db.transactionService().GetTotalTransactionsByCurrentYear()
}

func (db *V2DB) GetTotalTransactionsByCurrentYear() (int, error) {
	return db.transactionService().GetTotalTransactionsByCurrentYear()
}

func (db *V2DB) GetTotalTransactionsByYear(year int) (int, error) {
	return db.transactionService().GetTotalTransactionsByYear(year)
}

func (db *V2DB) GetTotalTransactionsCumulative() (int, error) {
	return db.transactionService().GetTotalTransactionsCumulative()
}

func (db *V2DB) GetPositiveTransactions() (int, error) {
	return db.transactionService().GetPositiveTransactionsByCurrentYear()
}

func (db *V2DB) GetPositiveTransactionsByCurrentYear() (int, error) {
	return db.transactionService().GetPositiveTransactionsByCurrentYear()
}

func (db *V2DB) GetPositiveTransactionsByYear(year int) (int, error) {
	return db.transactionService().GetPositiveTransactionsByYear(year)
}

func (db *V2DB) GetPositiveTransactionsCumulative() (int, error) {
	return db.transactionService().GetPositiveTransactionsCumulative()
}

func (db *V2DB) GetNegativeTransactions() (int, error) {
	return db.transactionService().GetNegativeTransactionsByCurrentYear()
}

func (db *V2DB) GetNegativeTransactionsByCurrentYear() (int, error) {
	return db.transactionService().GetNegativeTransactionsByCurrentYear()
}

func (db *V2DB) GetNegativeTransactionsByYear(year int) (int, error) {
	return db.transactionService().GetNegativeTransactionsByYear(year)
}

func (db *V2DB) GetNegativeTransactionsCumulative() (int, error) {
	return db.transactionService().GetNegativeTransactionsCumulative()
}

// Popular messages methods
func (db *V2DB) GetPopularMessages(limit int, year int) ([]*PopularMessage, error) {
	return db.transactionService().GetPopularMessages(limit, year)
}

func (db *V2DB) GetPopularMessagesByUser(limit int, year int, username string) ([]*PopularMessage, error) {
	return db.transactionService().GetPopularMessagesByUser(limit, year, username)
}

func (db *V2DB) GetPopularMessagesSince(since time.Time) ([]*PopularMessage, error) {
	return db.transactionService().GetPopularMessagesSince(since)
}

func (db *V2DB) GetPopularMessageCount(year int, username string, minReactions int) (int, error) {
	return db.transactionService().GetPopularMessageCount(year, username, minReactions)
}

func (db *V2DB) UpdateChannelIDForMessage(messageID, channelID string) error {
	return db.transactionService().UpdateChannelIDForMessage(messageID, channelID)
}

func (db *V2DB) BackfillChannelIDsFromTransactions() (int64, error) {
	return db.transactionService().BackfillChannelIDsFromTransactions()
}

func (db *V2DB) GetMessageAuthorByMessageID(messageID string) (*string, error) {
	return db.transactionService().GetMessageAuthorByMessageID(messageID)
}

func (db *V2DB) GetPopularMessageIDsNeedingBackfill(limit int) ([]string, error) {
	return db.transactionService().GetPopularMessageIDsNeedingBackfill(limit)
}

func (db *V2DB) GetChannelIDForMessage(messageID string) (*string, error) {
	return db.transactionService().GetChannelIDForMessage(messageID)
}

func (db *V2DB) GetPopularMessageDetails(messageID string) (*PopularMessageDetails, error) {
	return db.transactionService().GetPopularMessageDetails(messageID)
}

func (db *V2DB) UpsertPopularMessageDetails(messageID string, channelID, text, permalink, authorID, authorName, authorAvatar, imageURL, attachmentURL, attachmentMime *string, reactionCount *int, isReply, isIgnored, detailsFetched *bool) error {
	return db.transactionService().UpsertPopularMessageDetails(messageID, channelID, text, permalink, authorID, authorName, authorAvatar, imageURL, attachmentURL, attachmentMime, reactionCount, isReply, isIgnored, detailsFetched)
}

// UserRepository implementation

// User data methods
func (db *V2DB) GetUser(username string) (*UserSummary, error) {
	return db.userService().GetUser(username)
}

func (db *V2DB) GetUserByCurrentYear(username string) (*UserSummary, error) {
	return db.userService().GetUserByCurrentYear(username)
}

func (db *V2DB) GetUserByYear(username string, year int) (*UserSummary, error) {
	return db.userService().GetUserByYear(username, year)
}

func (db *V2DB) GetUserCumulative(username string) (*UserSummary, error) {
	return db.userService().GetUserCumulative(username)
}

// Leaderboard methods
func (db *V2DB) GetLeaderboard(limit int) ([]*UserSummary, error) {
	return db.userService().GetLeaderboard(limit)
}

func (db *V2DB) GetLeaderboardByCurrentYear(limit int) ([]*UserSummary, error) {
	return db.userService().GetLeaderboardByCurrentYear(limit)
}

func (db *V2DB) GetLeaderboardByYear(year, limit int) ([]*UserSummary, error) {
	return db.userService().GetYearlyLeaderboard(year, limit)
}

func (db *V2DB) GetLeaderboardCumulative(limit int) ([]*UserSummary, error) {
	return db.userService().GetLeaderboardCumulative(limit)
}

// Legacy methods for backward compatibility
func (db *V2DB) GetCurrentLeaderboard(limit int) ([]*UserSummary, error) {
	return db.userService().GetCurrentLeaderboard(limit)
}

func (db *V2DB) GetYearlyLeaderboard(year, limit int) ([]*UserSummary, error) {
	return db.userService().GetYearlyLeaderboard(year, limit)
}

// Top givers methods
func (db *V2DB) GetTopGivers(limit int) ([]*UserSummary, error) {
	return db.userService().GetTopGiversByCurrentYear(limit)
}

func (db *V2DB) GetTopGiversByCurrentYear(limit int) ([]*UserSummary, error) {
	return db.userService().GetTopGiversByCurrentYear(limit)
}

func (db *V2DB) GetTopGiversByYear(limit int, year int) ([]*UserSummary, error) {
	return db.userService().GetTopGiversByYear(limit, year)
}

func (db *V2DB) GetTopGiversCumulative(limit int) ([]*UserSummary, error) {
	return db.userService().GetTopGiversCumulative(limit)
}

// User count methods
func (db *V2DB) GetTotalUsers() (int, error) {
	return db.userService().GetTotalUsersByCurrentYear()
}

func (db *V2DB) GetTotalUsersByCurrentYear() (int, error) {
	return db.userService().GetTotalUsersByCurrentYear()
}

func (db *V2DB) GetTotalUsersByYear(year int) (int, error) {
	return db.userService().GetTotalUsersByYear(year)
}

func (db *V2DB) GetTotalUsersCumulative() (int, error) {
	return db.userService().GetTotalUsersCumulative()
}

// EmojiRepository implementation

// Top emojis methods
func (db *V2DB) GetTopEmojis(limit int) ([]*EmojiStats, error) {
	return db.emojiService().GetTopEmojis(limit)
}

func (db *V2DB) GetTopEmojisByCurrentYear(limit int) ([]*EmojiStats, error) {
	return db.emojiService().GetTopEmojisByCurrentYear(limit)
}

func (db *V2DB) GetTopEmojisByYear(year, limit int) ([]*EmojiStats, error) {
	return db.emojiService().GetTopEmojisByYear(year, limit)
}

func (db *V2DB) GetTopEmojisCumulative(limit int) ([]*EmojiStats, error) {
	return db.emojiService().GetTopEmojisCumulative(limit)
}

// Emoji usage methods
func (db *V2DB) GetTotalEmojiUsage() (int, error) {
	return db.emojiService().GetTotalEmojiUsage()
}

func (db *V2DB) GetTotalEmojiUsageByCurrentYear() (int, error) {
	return db.emojiService().GetTotalEmojiUsageByCurrentYear()
}

func (db *V2DB) GetTotalEmojiUsageByYear(year int) (int, error) {
	return db.emojiService().GetTotalEmojiUsageByYear(year)
}

func (db *V2DB) GetTotalEmojiUsageCumulative() (int, error) {
	return db.emojiService().GetTotalEmojiUsageCumulative()
}

// Emoji leaderboard methods
func (db *V2DB) GetEmojiLeaderboard(emojiName string) ([]*EmojiLeaderboard, error) {
	return db.emojiService().GetEmojiLeaderboardByCurrentYear(emojiName)
}

func (db *V2DB) GetEmojiLeaderboardByCurrentYear(emojiName string) ([]*EmojiLeaderboard, error) {
	return db.emojiService().GetEmojiLeaderboardByCurrentYear(emojiName)
}

func (db *V2DB) GetEmojiLeaderboardByYear(emojiName string, year int) ([]*EmojiLeaderboard, error) {
	return db.emojiService().GetEmojiLeaderboardByYear(emojiName, year)
}

func (db *V2DB) GetEmojiLeaderboardCumulative(emojiName string) ([]*EmojiLeaderboard, error) {
	return db.emojiService().GetEmojiLeaderboardCumulative(emojiName)
}

// StatsRepository implementation

// Total points methods
func (db *V2DB) GetTotalPoints() (int, error) {
	return db.statsService().GetTotalPointsByCurrentYear()
}

func (db *V2DB) GetTotalPointsByCurrentYear() (int, error) {
	return db.statsService().GetTotalPointsByCurrentYear()
}

func (db *V2DB) GetTotalPointsByYear(year int) (int, error) {
	return db.statsService().GetTotalPointsByYear(year)
}

func (db *V2DB) GetTotalPointsCumulative() (int, error) {
	return db.statsService().GetTotalPointsCumulative()
}

// Monthly stats methods
func (db *V2DB) GetMonthlyStats() ([]*MonthlyStats, error) {
	return db.statsService().GetMonthlyStatsByCurrentYear()
}

func (db *V2DB) GetMonthlyStatsByCurrentYear() ([]*MonthlyStats, error) {
	return db.statsService().GetMonthlyStatsByCurrentYear()
}

func (db *V2DB) GetMonthlyStatsByYear(year int) ([]*MonthlyStats, error) {
	return db.statsService().GetMonthlyStatsByYear(year)
}

// Points over time methods
func (db *V2DB) GetPointsOverTimeMonthly() ([]map[string]interface{}, error) {
	return db.statsService().GetPointsOverTimeMonthlyByCurrentYear()
}

func (db *V2DB) GetPointsOverTimeMonthlyByCurrentYear() ([]map[string]interface{}, error) {
	return db.statsService().GetPointsOverTimeMonthlyByCurrentYear()
}

func (db *V2DB) GetPointsOverTimeMonthlyByYear(year int) ([]map[string]interface{}, error) {
	return db.statsService().GetPointsOverTimeMonthlyByYear(year)
}

// Activity timeline methods
func (db *V2DB) GetActivityTimeline() ([]map[string]interface{}, error) {
	return db.statsService().GetActivityTimelineByCurrentYear()
}

func (db *V2DB) GetActivityTimelineByCurrentYear() ([]map[string]interface{}, error) {
	return db.statsService().GetActivityTimelineByCurrentYear()
}

func (db *V2DB) GetActivityTimelineByYear(year int) ([]map[string]interface{}, error) {
	return db.statsService().GetActivityTimelineByYear(year)
}

// Karma distribution methods
func (db *V2DB) GetKarmaDistribution() ([]map[string]interface{}, error) {
	return db.statsService().GetKarmaDistributionByCurrentYear()
}

func (db *V2DB) GetKarmaDistributionByCurrentYear() ([]map[string]interface{}, error) {
	return db.statsService().GetKarmaDistributionByCurrentYear()
}

func (db *V2DB) GetKarmaDistributionByYear(year int) ([]map[string]interface{}, error) {
	return db.statsService().GetKarmaDistributionByYear(year)
}

func (db *V2DB) GetKarmaDistributionCumulative() ([]map[string]interface{}, error) {
	return db.statsService().GetKarmaDistributionCumulative()
}

// Utility methods
func (db *V2DB) GetAvailableYears() ([]int, error) {
	return db.statsService().GetAvailableYears()
}

// SummaryRepository implementation
func (db *V2DB) RebuildSummaryTables() error {
	return db.summaryService().RebuildSummaryTables()
}

func (db *V2DB) UpdateSummaryTables(toUser, fromUser string, year int) error {
	return db.summaryService().UpdateSummaryTables(toUser, fromUser, year)
}

// BackfillRepository implementation

func (db *V2DB) InsertBackfillRecord(record *BackfillRecord) error {
	return db.backfillService().InsertBackfillRecord(record)
}

func (db *V2DB) GetBackfillStats(startTime, endTime time.Time) (*BackfillStats, error) {
	return db.backfillService().GetBackfillStats(startTime, endTime)
}

// LegacyRepository implementation
func (db *V2DB) InsertPoints(points *Points) error {
	return db.legacyService().InsertPoints(points)
}

func (db *V2DB) GetLegacyLeaderboard(limit int) (Leaderboard, error) {
	return db.legacyService().GetLeaderboard(limit)
}

func (db *V2DB) GetThrowback(user string) (*Throwback, error) {
	return db.legacyService().GetThrowback(user)
}
