package database

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrInvalidUsername is returned when a username contains invalid characters
	ErrInvalidUsername = errors.New("username contains invalid characters")
	validUsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

// TransactionService handles transaction-related database operations
type TransactionService struct {
	db *V2DB
}

// NewTransactionService creates a new transaction service
func NewTransactionService(db *V2DB) *TransactionService {
	return &TransactionService{db: db}
}

// isValidUsername validates that a username only contains allowed characters
// Returns true if username is valid (alphanumeric, dots, underscores, hyphens only)
// Rejects Slack special syntax like <!subteam^>, <@U...>, :emoji: patterns, etc.
func isValidUsername(username string) bool {
	if username == "" || len(username) > 50 {
		return false
	}
	return validUsernameRegex.MatchString(username)
}

func buildTransactionDedupeKey(tx *Transaction) string {
	emojiName := ""
	if tx.EmojiName != nil {
		emojiName = *tx.EmojiName
	}

	channelID := ""
	if tx.ChannelID != nil {
		channelID = *tx.ChannelID
	}

	messageID := ""
	if tx.MessageID != nil {
		messageID = *tx.MessageID
	}

	channelName := ""
	if tx.ChannelName != nil {
		channelName = *tx.ChannelName
	}

	timestamp := tx.Timestamp.UTC().Format(time.RFC3339Nano)
	raw := strings.Join([]string{
		tx.FromUser,
		tx.ToUser,
		strconv.Itoa(tx.Points),
		tx.Reason,
		tx.TransactionType,
		emojiName,
		channelID,
		channelName,
		messageID,
		timestamp,
	}, "\x1f")

	sum := sha1.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// InsertTransaction inserts a new karma transaction
func (ts *TransactionService) InsertTransaction(tx *Transaction) error {
	// Validate usernames to prevent special characters from being stored
	if !isValidUsername(tx.FromUser) || !isValidUsername(tx.ToUser) {
		return ErrInvalidUsername
	}

	// Extract emoji name from reason if not provided
	if tx.EmojiName == nil && tx.Reason != "" {
		emojiRegex := regexp.MustCompile(`added a :([^:]+): emoji`)
		if matches := emojiRegex.FindStringSubmatch(tx.Reason); len(matches) > 1 {
			tx.EmojiName = &matches[1]
			tx.TransactionType = "reactji"
		}
	}

	// Set year from timestamp
	tx.Year = tx.Timestamp.Year()
	dedupeKey := buildTransactionDedupeKey(tx)

	// Use a transaction to prevent race conditions during mass imports
	dbTx, err := ts.db.SQL.Begin()
	if err != nil {
		return err
	}
	defer dbTx.Rollback()

	query := `
		INSERT INTO karma_transactions
		(from_user, to_user, points, reason, transaction_type, emoji_name, channel_id, channel_name, message_id, dedupe_key, timestamp, year)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT DO NOTHING
	`

	result, err := dbTx.Exec(query,
		tx.FromUser, tx.ToUser, tx.Points, tx.Reason,
		tx.TransactionType, tx.EmojiName, tx.ChannelID, tx.ChannelName, tx.MessageID,
		dedupeKey, tx.Timestamp, tx.Year,
	)
	if err != nil {
		return err
	}

	// Commit the transaction first
	if err := dbTx.Commit(); err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return nil
	}

	// Update summary tables after successful insert
	summaryService := NewSummaryService(ts.db)
	return summaryService.UpdateSummaryTables(tx.ToUser, tx.FromUser, tx.Year)
}

// InsertTransactionsBulk inserts multiple karma transactions in a single transaction
// This is more efficient and safer for mass imports
func (ts *TransactionService) InsertTransactionsBulk(transactions []*Transaction) error {
	if len(transactions) == 0 {
		return nil
	}

	// Use a single transaction for all inserts
	dbTx, err := ts.db.SQL.Begin()
	if err != nil {
		return err
	}
	defer dbTx.Rollback()

	// Prepare the insert statement once
	query := `
		INSERT INTO karma_transactions
		(from_user, to_user, points, reason, transaction_type, emoji_name, channel_id, channel_name, message_id, dedupe_key, timestamp, year)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT DO NOTHING
	`
	stmt, err := dbTx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Process each transaction
	insertedUsersByYear := make(map[string]map[int]struct{})
	for _, tx := range transactions {
		// Validate usernames to prevent special characters from being stored
		if !isValidUsername(tx.FromUser) || !isValidUsername(tx.ToUser) {
			return ErrInvalidUsername
		}

		// Extract emoji name from reason if not provided
		if tx.EmojiName == nil && tx.Reason != "" {
			emojiRegex := regexp.MustCompile(`added a :([^:]+): emoji`)
			if matches := emojiRegex.FindStringSubmatch(tx.Reason); len(matches) > 1 {
				tx.EmojiName = &matches[1]
				tx.TransactionType = "reactji"
			}
		}

		// Set year from timestamp
		tx.Year = tx.Timestamp.Year()
		dedupeKey := buildTransactionDedupeKey(tx)

		// Execute the insert
		result, err := stmt.Exec(
			tx.FromUser, tx.ToUser, tx.Points, tx.Reason,
			tx.TransactionType, tx.EmojiName, tx.ChannelID, tx.ChannelName, tx.MessageID,
			dedupeKey, tx.Timestamp, tx.Year,
		)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			continue
		}

		if insertedUsersByYear[tx.ToUser] == nil {
			insertedUsersByYear[tx.ToUser] = make(map[int]struct{})
		}
		if insertedUsersByYear[tx.FromUser] == nil {
			insertedUsersByYear[tx.FromUser] = make(map[int]struct{})
		}
		insertedUsersByYear[tx.ToUser][tx.Year] = struct{}{}
		insertedUsersByYear[tx.FromUser][tx.Year] = struct{}{}
	}

	// Commit all inserts at once
	if err := dbTx.Commit(); err != nil {
		return err
	}

	// Update summary tables for all affected users
	summaryService := NewSummaryService(ts.db)
	processedPairs := make(map[string]struct{})
	for user, years := range insertedUsersByYear {
		for year := range years {
			key := fmt.Sprintf("%s:%d", user, year)
			if _, seen := processedPairs[key]; seen {
				continue
			}
			if err := summaryService.UpdateSummaryTables(user, user, year); err != nil {
				return err
			}
			processedPairs[key] = struct{}{}
		}
	}

	return nil
}

// GetRecentActivity returns recent karma transactions
func (ts *TransactionService) GetRecentActivity(limit int) ([]*Transaction, error) {
	query := `
		SELECT from_user, to_user, points, reason, transaction_type, emoji_name, timestamp, year
		FROM karma_transactions
		ORDER BY timestamp DESC
		LIMIT $1
	`
	rows, err := ts.db.SQL.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []*Transaction
	for rows.Next() {
		tx := &Transaction{}
		err := rows.Scan(&tx.FromUser, &tx.ToUser, &tx.Points, &tx.Reason,
			&tx.TransactionType, &tx.EmojiName, &tx.Timestamp, &tx.Year)
		if err != nil {
			return nil, err
		}
		activities = append(activities, tx)
	}

	return activities, nil
}

// GetTransactionsByUser returns transactions for a specific user
func (ts *TransactionService) GetTransactionsByUser(username string, year int) ([]*Transaction, error) {
	query := `
		SELECT id, from_user, to_user, points, reason, transaction_type, emoji_name, channel_id, message_id, timestamp, year
		FROM karma_transactions
		WHERE (from_user = $1 OR to_user = $1)
	`

	args := []interface{}{username}

	if year > 0 {
		query += " AND year = $2"
		args = append(args, year)
	}

	query += " ORDER BY timestamp DESC"

	rows, err := ts.db.SQL.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*Transaction
	for rows.Next() {
		tx := &Transaction{}
		err := rows.Scan(&tx.ID, &tx.FromUser, &tx.ToUser, &tx.Points, &tx.Reason,
			&tx.TransactionType, &tx.EmojiName, &tx.ChannelID, &tx.MessageID, &tx.Timestamp, &tx.Year)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

// GetTotalTransactions returns total transactions across all years
func (ts *TransactionService) GetTotalTransactions(year int) (int, error) {
	query := `SELECT COALESCE(COUNT(*), 0) FROM karma_transactions`
	args := []interface{}{}

	if year > 0 {
		query += " WHERE year = $1"
		args = append(args, year)
	}

	var totalTransactions int
	err := ts.db.SQL.QueryRow(query, args...).Scan(&totalTransactions)
	return totalTransactions, err
}

// GetPositiveTransactions returns count of positive transactions
func (ts *TransactionService) GetPositiveTransactions(year int) (int, error) {
	query := `SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE points > 0`
	args := []interface{}{}

	if year > 0 {
		query += " AND year = $1"
		args = append(args, year)
	}

	var total int
	err := ts.db.SQL.QueryRow(query, args...).Scan(&total)
	return total, err
}

// GetNegativeTransactions returns count of negative transactions
func (ts *TransactionService) GetNegativeTransactions(year int) (int, error) {
	query := `SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE points < 0`
	args := []interface{}{}

	if year > 0 {
		query += " AND year = $1"
		args = append(args, year)
	}

	var total int
	err := ts.db.SQL.QueryRow(query, args...).Scan(&total)
	return total, err
}

// Helper function to get current year
func (ts *TransactionService) getCurrentYear() int {
	return time.Now().Year()
}

// New methods implementing the byCurrentYear, byYear, and Cumulative pattern

// Recent activity methods
func (ts *TransactionService) GetRecentActivityByCurrentYear(limit int) ([]*Transaction, error) {
	return ts.GetRecentActivityByYear(limit, ts.getCurrentYear())
}

func (ts *TransactionService) GetRecentActivityByYear(limit int, year int) ([]*Transaction, error) {
	query := `
		SELECT from_user, to_user, points, reason, transaction_type, emoji_name, timestamp, year
		FROM karma_transactions
		WHERE year = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`
	rows, err := ts.db.SQL.Query(query, year, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []*Transaction
	for rows.Next() {
		tx := &Transaction{}
		err := rows.Scan(&tx.FromUser, &tx.ToUser, &tx.Points, &tx.Reason,
			&tx.TransactionType, &tx.EmojiName, &tx.Timestamp, &tx.Year)
		if err != nil {
			return nil, err
		}
		activities = append(activities, tx)
	}

	return activities, nil
}

func (ts *TransactionService) GetRecentActivityCumulative(limit int) ([]*Transaction, error) {
	return ts.GetRecentActivity(limit) // GetRecentActivity already returns cumulative data
}

// User transactions methods
func (ts *TransactionService) GetTransactionsByUserByCurrentYear(username string) ([]*Transaction, error) {
	return ts.GetTransactionsByUserByYear(username, ts.getCurrentYear())
}

func (ts *TransactionService) GetTransactionsByUserByYear(username string, year int) ([]*Transaction, error) {
	return ts.GetTransactionsByUser(username, year)
}

func (ts *TransactionService) GetTransactionsByUserCumulative(username string) ([]*Transaction, error) {
	return ts.GetTransactionsByUser(username, 0) // 0 means all-time/cumulative
}

func (ts *TransactionService) GetUserMonthlyPointsByYear(username string, year int) ([]*MonthlyPoints, error) {
	rows, err := ts.db.SQL.Query(`
		SELECT month, total_points
		FROM user_summary_monthly
		WHERE username = $1 AND year = $2
		ORDER BY month
	`, username, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*MonthlyPoints
	for rows.Next() {
		point := &MonthlyPoints{}
		if err := rows.Scan(&point.Month, &point.TotalPoints); err != nil {
			return nil, err
		}
		results = append(results, point)
	}

	return results, rows.Err()
}

func (ts *TransactionService) GetUserYearlyPoints(username string) ([]*YearlyPoints, error) {
	rows, err := ts.db.SQL.Query(`
		SELECT EXTRACT(YEAR FROM timestamp) as year, SUM(points) as total_points
		FROM karma_transactions
		WHERE to_user = $1
		GROUP BY EXTRACT(YEAR FROM timestamp)
		ORDER BY year
	`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*YearlyPoints
	for rows.Next() {
		point := &YearlyPoints{}
		if err := rows.Scan(&point.Year, &point.TotalPoints); err != nil {
			return nil, err
		}
		results = append(results, point)
	}

	return results, rows.Err()
}

func (ts *TransactionService) GetRecentActivityPage(filter RecentActivityFilter) (*RecentActivityPage, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	baseArgs := make([]interface{}, 0, 4)
	conditions := make([]string, 0, 3)

	if filter.Year > 0 {
		baseArgs = append(baseArgs, filter.Year)
		conditions = append(conditions, "year = $"+strconv.Itoa(len(baseArgs)))
	}
	if filter.FromUser != "" {
		baseArgs = append(baseArgs, "%"+filter.FromUser+"%")
		conditions = append(conditions, "from_user ILIKE $"+strconv.Itoa(len(baseArgs)))
	}
	if filter.ToUser != "" {
		baseArgs = append(baseArgs, "%"+filter.ToUser+"%")
		conditions = append(conditions, "to_user ILIKE $"+strconv.Itoa(len(baseArgs)))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	queryArgs := append([]interface{}{}, baseArgs...)
	queryArgs = append(queryArgs, filter.Limit, filter.Offset)
	query := `
		SELECT from_user, to_user, points, reason, transaction_type, emoji_name, channel_id, message_id, timestamp, year
		FROM karma_transactions` + whereClause + `
		ORDER BY timestamp DESC
		LIMIT $` + strconv.Itoa(len(baseArgs)+1) + ` OFFSET $` + strconv.Itoa(len(baseArgs)+2)

	rows, err := ts.db.SQL.Query(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []*Transaction
	for rows.Next() {
		tx := &Transaction{}
		if err := rows.Scan(
			&tx.FromUser,
			&tx.ToUser,
			&tx.Points,
			&tx.Reason,
			&tx.TransactionType,
			&tx.EmojiName,
			&tx.ChannelID,
			&tx.MessageID,
			&tx.Timestamp,
			&tx.Year,
		); err != nil {
			return nil, err
		}
		activities = append(activities, tx)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	countQuery := `SELECT COUNT(*) FROM karma_transactions` + whereClause
	var totalCount int
	if err := ts.db.SQL.QueryRow(countQuery, baseArgs...).Scan(&totalCount); err != nil {
		return nil, err
	}

	return &RecentActivityPage{
		Activities: activities,
		TotalCount: totalCount,
	}, nil
}

// Transaction count methods
func (ts *TransactionService) GetTotalTransactionsByCurrentYear() (int, error) {
	return ts.GetTotalTransactionsByYear(ts.getCurrentYear())
}

func (ts *TransactionService) GetTotalTransactionsByYear(year int) (int, error) {
	return ts.GetTotalTransactions(year)
}

func (ts *TransactionService) GetTotalTransactionsCumulative() (int, error) {
	return ts.GetTotalTransactionsByYear(0) // 0 means all-time/cumulative
}

func (ts *TransactionService) GetPositiveTransactionsByCurrentYear() (int, error) {
	return ts.GetPositiveTransactionsByYear(ts.getCurrentYear())
}

func (ts *TransactionService) GetPositiveTransactionsByYear(year int) (int, error) {
	return ts.GetPositiveTransactions(year)
}

func (ts *TransactionService) GetPositiveTransactionsCumulative() (int, error) {
	return ts.GetPositiveTransactionsByYear(0) // 0 means all-time/cumulative
}

func (ts *TransactionService) GetNegativeTransactionsByCurrentYear() (int, error) {
	return ts.GetNegativeTransactionsByYear(ts.getCurrentYear())
}

func (ts *TransactionService) GetNegativeTransactionsByYear(year int) (int, error) {
	return ts.GetNegativeTransactions(year)
}

func (ts *TransactionService) GetNegativeTransactionsCumulative() (int, error) {
	return ts.GetNegativeTransactionsByYear(0) // 0 means all-time/cumulative
}

// GetPopularMessages returns messages with the most reactions
// Excludes 'bangbang' reactions and prioritizes messages with 'joy' emoji reactions (funny posts)
func (ts *TransactionService) GetPopularMessages(limit int, year int, funnyBias bool) ([]*PopularMessage, error) {
	return ts.getPopularMessages(limit, year, "", funnyBias)
}

func (ts *TransactionService) GetPopularMessagesByUser(limit int, year int, username string, funnyBias bool) ([]*PopularMessage, error) {
	return ts.getPopularMessages(limit, year, username, funnyBias)
}

func (ts *TransactionService) getPopularMessages(limit int, year int, username string, funnyBias bool) ([]*PopularMessage, error) {
	query := `
		WITH message_reactions AS (
			SELECT
			message_id,
			-- Get any non-null channel_id for this message (all reactions to same message should have same channel)
			MAX(channel_id) FILTER (WHERE channel_id IS NOT NULL AND channel_id != '') as channel_id,
			COUNT(*) as total_reactions,
			SUM(CASE WHEN emoji_name = 'joy' THEN 1 ELSE 0 END) as joy_count,
			SUM(CASE WHEN emoji_name = 'lol' THEN 1 ELSE 0 END) as lol_count,
			SUM(points) as total_points
		FROM karma_transactions
			WHERE transaction_type = 'reactji'
				AND message_id IS NOT NULL
				AND emoji_name != 'bangbang'
	`

	args := []interface{}{}
	if year > 0 {
		query += " AND year = $1"
		args = append(args, year)
	}
	if username != "" {
		query += " AND to_user = $" + strconv.Itoa(len(args)+1)
		args = append(args, username)
	}

	query += `
			GROUP BY message_id
		)
		SELECT
			channel_id,
			message_id,
			total_reactions as reaction_count,
			total_points
		FROM message_reactions
	`

	if funnyBias {
		query += `
		WHERE (joy_count + lol_count) > 0
		ORDER BY
			total_reactions DESC
		`
	} else {
		query += `
		ORDER BY
			total_reactions DESC
		`
	}

	query += `
		LIMIT $` + strconv.Itoa(len(args)+1)

	args = append(args, limit)

	rows, err := ts.db.SQL.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*PopularMessage
	for rows.Next() {
		msg := &PopularMessage{}
		err := rows.Scan(&msg.ChannelID, &msg.MessageID, &msg.ReactionCount, &msg.TotalPoints)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (ts *TransactionService) GetPopularMessagesSince(since time.Time) ([]*PopularMessage, error) {
	query := `
		WITH message_reactions AS (
			SELECT
				message_id,
				MAX(channel_id) FILTER (WHERE channel_id IS NOT NULL AND channel_id != '') as channel_id,
				COUNT(*) as total_reactions,
				SUM(points) as total_points
			FROM karma_transactions
			WHERE transaction_type = 'reactji'
				AND message_id IS NOT NULL
				AND emoji_name != 'bangbang'
				AND timestamp >= $1
	`

	args := []interface{}{since}

	query += `
			GROUP BY message_id
		)
		SELECT
			channel_id,
			message_id,
			total_reactions as reaction_count,
			total_points
		FROM message_reactions
		ORDER BY
			total_reactions DESC
	`

	rows, err := ts.db.SQL.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*PopularMessage
	for rows.Next() {
		msg := &PopularMessage{}
		err := rows.Scan(&msg.ChannelID, &msg.MessageID, &msg.ReactionCount, &msg.TotalPoints)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (ts *TransactionService) GetPopularMessageCount(year int, username string, minReactions int, funnyBias bool) (int, error) {
	query := `
		WITH message_reactions AS (
			SELECT
				message_id,
				MAX(channel_id) FILTER (WHERE channel_id IS NOT NULL AND channel_id != '') as channel_id,
				COUNT(*) as total_reactions,
				SUM(CASE WHEN emoji_name = 'joy' THEN 1 ELSE 0 END) as joy_count,
				SUM(CASE WHEN emoji_name = 'lol' THEN 1 ELSE 0 END) as lol_count
			FROM karma_transactions
			WHERE transaction_type = 'reactji'
				AND message_id IS NOT NULL
				AND emoji_name != 'bangbang'
	`

	args := []interface{}{}
	if year > 0 {
		query += " AND year = $1"
		args = append(args, year)
	}
	if username != "" {
		query += " AND to_user = $" + strconv.Itoa(len(args)+1)
		args = append(args, username)
	}

	query += `
			GROUP BY message_id
		)
		SELECT COUNT(*)
		FROM message_reactions mr
		LEFT JOIN popular_message_cache pmc ON pmc.message_id = mr.message_id
		WHERE
			(pmc.is_reply IS NULL OR pmc.is_reply = false)
			AND (pmc.is_ignored IS NULL OR pmc.is_ignored = false)
			AND (COALESCE(pmc.channel_id, mr.channel_id) IS NULL OR COALESCE(pmc.channel_id, mr.channel_id) NOT LIKE 'TEST%')
	`

	if minReactions > 0 {
		query += " AND mr.total_reactions >= $" + strconv.Itoa(len(args)+1)
		args = append(args, minReactions)
	}
	if funnyBias {
		query += " AND (mr.joy_count + mr.lol_count) > 0"
	}

	var count int
	if err := ts.db.SQL.QueryRow(query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (ts *TransactionService) GetPopularMessageCountWithMedia(year int, username string, minReactions int, funnyBias bool) (int, error) {
	query := `
		WITH message_reactions AS (
			SELECT
				message_id,
				MAX(channel_id) FILTER (WHERE channel_id IS NOT NULL AND channel_id != '') as channel_id,
				COUNT(*) as total_reactions,
				SUM(CASE WHEN emoji_name = 'joy' THEN 1 ELSE 0 END) as joy_count,
				SUM(CASE WHEN emoji_name = 'lol' THEN 1 ELSE 0 END) as lol_count
			FROM karma_transactions
			WHERE transaction_type = 'reactji'
				AND message_id IS NOT NULL
				AND emoji_name != 'bangbang'
	`

	args := []interface{}{}
	if year > 0 {
		query += " AND year = $1"
		args = append(args, year)
	}
	if username != "" {
		query += " AND to_user = $" + strconv.Itoa(len(args)+1)
		args = append(args, username)
	}

	query += `
			GROUP BY message_id
		)
		SELECT COUNT(*)
		FROM message_reactions mr
		JOIN popular_message_cache pmc ON pmc.message_id = mr.message_id
		WHERE
			((pmc.image_url IS NOT NULL AND pmc.image_url != '') OR (pmc.attachment_url IS NOT NULL AND pmc.attachment_url != ''))
			AND (pmc.is_reply IS NULL OR pmc.is_reply = false)
			AND (pmc.is_ignored IS NULL OR pmc.is_ignored = false)
			AND (COALESCE(pmc.channel_id, mr.channel_id) IS NULL OR COALESCE(pmc.channel_id, mr.channel_id) NOT LIKE 'TEST%')
	`

	if minReactions > 0 {
		query += " AND mr.total_reactions >= $" + strconv.Itoa(len(args)+1)
		args = append(args, minReactions)
	}
	if funnyBias {
		query += " AND (mr.joy_count + mr.lol_count) > 0"
	}

	var count int
	if err := ts.db.SQL.QueryRow(query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// UpdateChannelIDForMessage updates the channel_id for all transactions with a given message_id
// This is useful for caching channel lookups from the Slack API
func (ts *TransactionService) UpdateChannelIDForMessage(messageID, channelID string) error {
	query := `
		UPDATE karma_transactions
		SET channel_id = $1
		WHERE message_id = $2
			AND (channel_id IS NULL OR channel_id = '')
	`

	result, err := ts.db.SQL.Exec(query, channelID, messageID)
	if err != nil {
		return err
	}

	if _, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("failed to get channel_id update row count: %w", err)
	}

	return nil
}

// BackfillChannelIDsFromTransactions fills missing channel_id values from other rows
// with the same message_id when there is exactly one distinct channel_id.
func (ts *TransactionService) BackfillChannelIDsFromTransactions() (int64, error) {
	query := `
		UPDATE karma_transactions t
		SET channel_id = s.channel_id
		FROM (
			SELECT message_id, MAX(channel_id) AS channel_id
			FROM karma_transactions
			WHERE channel_id IS NOT NULL AND channel_id != ''
			GROUP BY message_id
			HAVING COUNT(DISTINCT channel_id) = 1
		) s
		WHERE t.message_id = s.message_id
		  AND (t.channel_id IS NULL OR t.channel_id = '')
	`

	result, err := ts.db.SQL.Exec(query)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

// GetMessageAuthorByMessageID returns the to_user associated with a message_id.
// This is derived from reactji transactions where to_user is the message author.
func (ts *TransactionService) GetMessageAuthorByMessageID(messageID string) (*string, error) {
	query := `
		SELECT to_user
		FROM karma_transactions
		WHERE message_id = $1
		  AND transaction_type = 'reactji'
		GROUP BY to_user
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`

	var author sql.NullString
	if err := ts.db.SQL.QueryRow(query, messageID).Scan(&author); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if !author.Valid || author.String == "" {
		return nil, nil
	}
	return &author.String, nil
}

// GetPopularMessageIDsNeedingBackfill returns message_ids with incomplete cached details.
func (ts *TransactionService) GetPopularMessageIDsNeedingBackfill(limit int) ([]string, error) {
	query := `
		SELECT message_id
		FROM popular_message_cache
		WHERE (is_reply IS NULL OR is_reply = false)
		  AND (is_ignored IS NULL OR is_ignored = false)
		  AND (details_fetched IS NULL OR details_fetched = false)
		  AND (
			channel_id IS NULL OR channel_id = '' OR
			message_text IS NULL OR
			permalink IS NULL OR
			author_name IS NULL OR
			image_url IS NULL OR
			attachment_url IS NULL OR
			slack_reaction_count IS NULL
		  )
		ORDER BY updated_at ASC
		LIMIT $1
	`

	rows, err := ts.db.SQL.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// GetChannelIDForMessage returns any stored channel_id for a message_id.
func (ts *TransactionService) GetChannelIDForMessage(messageID string) (*string, error) {
	query := `
		SELECT MAX(channel_id) FILTER (WHERE channel_id IS NOT NULL AND channel_id != '')
		FROM karma_transactions
		WHERE message_id = $1
	`

	var channel sql.NullString
	if err := ts.db.SQL.QueryRow(query, messageID).Scan(&channel); err != nil {
		return nil, err
	}
	if !channel.Valid || channel.String == "" {
		return nil, nil
	}
	return &channel.String, nil
}

// GetPopularMessageDetails returns cached Slack metadata for a message_id.
func (ts *TransactionService) GetPopularMessageDetails(messageID string) (*PopularMessageDetails, error) {
	query := `
		SELECT channel_id, message_text, permalink, author_id, author_name, author_avatar, image_url, attachment_url, attachment_mime, slack_reaction_count, is_reply, is_ignored, details_fetched
		FROM popular_message_cache
		WHERE message_id = $1
	`

	details := &PopularMessageDetails{MessageID: messageID}
	err := ts.db.SQL.QueryRow(query, messageID).Scan(
		&details.ChannelID,
		&details.Text,
		&details.Permalink,
		&details.AuthorID,
		&details.AuthorName,
		&details.AuthorAvatar,
		&details.ImageURL,
		&details.AttachmentURL,
		&details.AttachmentMime,
		&details.ReactionCount,
		&details.IsReply,
		&details.IsIgnored,
		&details.DetailsFetched,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return details, nil
}

// UpsertPopularMessageDetails stores cached Slack metadata for a message_id.
func (ts *TransactionService) UpsertPopularMessageDetails(messageID string, channelID, text, permalink, authorID, authorName, authorAvatar, imageURL, attachmentURL, attachmentMime *string, reactionCount *int, isReply, isIgnored, detailsFetched *bool) error {
	query := `
		INSERT INTO popular_message_cache
			(message_id, channel_id, message_text, permalink, author_id, author_name, author_avatar, image_url, attachment_url, attachment_mime, slack_reaction_count, is_reply, is_ignored, details_fetched, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
		ON CONFLICT (message_id) DO UPDATE SET
			channel_id = COALESCE(EXCLUDED.channel_id, popular_message_cache.channel_id),
			message_text = COALESCE(EXCLUDED.message_text, popular_message_cache.message_text),
			permalink = COALESCE(EXCLUDED.permalink, popular_message_cache.permalink),
			author_id = COALESCE(EXCLUDED.author_id, popular_message_cache.author_id),
			author_name = COALESCE(EXCLUDED.author_name, popular_message_cache.author_name),
			author_avatar = COALESCE(EXCLUDED.author_avatar, popular_message_cache.author_avatar),
			image_url = COALESCE(EXCLUDED.image_url, popular_message_cache.image_url),
			attachment_url = COALESCE(EXCLUDED.attachment_url, popular_message_cache.attachment_url),
			attachment_mime = COALESCE(EXCLUDED.attachment_mime, popular_message_cache.attachment_mime),
			slack_reaction_count = COALESCE(EXCLUDED.slack_reaction_count, popular_message_cache.slack_reaction_count),
			is_reply = COALESCE(EXCLUDED.is_reply, popular_message_cache.is_reply),
			is_ignored = COALESCE(EXCLUDED.is_ignored, popular_message_cache.is_ignored),
			details_fetched = COALESCE(EXCLUDED.details_fetched, popular_message_cache.details_fetched),
			updated_at = NOW()
	`

	_, err := ts.db.SQL.Exec(query, messageID, channelID, text, permalink, authorID, authorName, authorAvatar, imageURL, attachmentURL, attachmentMime, reactionCount, isReply, isIgnored, detailsFetched)
	return err
}
