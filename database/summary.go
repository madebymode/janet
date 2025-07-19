package database

import (
	"fmt"
)

// SummaryService handles summary table operations
type SummaryService struct {
	db *V2DB
}

// NewSummaryService creates a new summary service
func NewSummaryService(db *V2DB) *SummaryService {
	return &SummaryService{db: db}
}

// UpdateSummaryTables updates summary tables after transaction
func (ss *SummaryService) UpdateSummaryTables(toUser, fromUser string, year int) error {
	// Update current summary for toUser
	_, err := ss.db.SQL.Exec(`
		INSERT INTO user_summary_current (username, total_points, points_given, points_received, transactions_given, transactions_received, emoji_reactions_given, emoji_reactions_received, last_activity)
		VALUES ($1,
			(SELECT COALESCE(SUM(points), 0) FROM karma_transactions WHERE to_user = $1),
			(SELECT COALESCE(SUM(CASE WHEN points > 0 THEN points ELSE 0 END), 0) FROM karma_transactions WHERE from_user = $1),
			(SELECT COALESCE(SUM(points), 0) FROM karma_transactions WHERE to_user = $1),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE from_user = $1),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE to_user = $1),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE from_user = $1 AND transaction_type = 'reactji'),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE to_user = $1 AND transaction_type = 'reactji'),
			(SELECT MAX(timestamp) FROM karma_transactions WHERE to_user = $1 OR from_user = $1)
		)
		ON CONFLICT (username) DO UPDATE SET
			total_points = EXCLUDED.total_points,
			points_given = EXCLUDED.points_given,
			points_received = EXCLUDED.points_received,
			transactions_given = EXCLUDED.transactions_given,
			transactions_received = EXCLUDED.transactions_received,
			emoji_reactions_given = EXCLUDED.emoji_reactions_given,
			emoji_reactions_received = EXCLUDED.emoji_reactions_received,
			last_activity = EXCLUDED.last_activity;
	`, toUser)
	if err != nil {
		return fmt.Errorf("failed to update current summary for toUser %s: %w", toUser, err)
	}

	// Update current summary for fromUser
	_, err = ss.db.SQL.Exec(`
		INSERT INTO user_summary_current (username, total_points, points_given, points_received, transactions_given, transactions_received, emoji_reactions_given, emoji_reactions_received, last_activity)
		VALUES ($1,
			(SELECT COALESCE(SUM(points), 0) FROM karma_transactions WHERE to_user = $1),
			(SELECT COALESCE(SUM(CASE WHEN points > 0 THEN points ELSE 0 END), 0) FROM karma_transactions WHERE from_user = $1),
			(SELECT COALESCE(SUM(points), 0) FROM karma_transactions WHERE to_user = $1),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE from_user = $1),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE to_user = $1),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE from_user = $1 AND transaction_type = 'reactji'),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE to_user = $1 AND transaction_type = 'reactji'),
			(SELECT MAX(timestamp) FROM karma_transactions WHERE to_user = $1 OR from_user = $1)
		)
		ON CONFLICT (username) DO UPDATE SET
			total_points = EXCLUDED.total_points,
			points_given = EXCLUDED.points_given,
			points_received = EXCLUDED.points_received,
			transactions_given = EXCLUDED.transactions_given,
			transactions_received = EXCLUDED.transactions_received,
			emoji_reactions_given = EXCLUDED.emoji_reactions_given,
			emoji_reactions_received = EXCLUDED.emoji_reactions_received,
			last_activity = EXCLUDED.last_activity;
	`, fromUser)
	if err != nil {
		return fmt.Errorf("failed to update current summary for fromUser %s: %w", fromUser, err)
	}

	// Update yearly summary for toUser
	_, err = ss.db.SQL.Exec(`
		INSERT INTO user_summary_yearly (username, year, total_points, points_given, points_received, transactions_given, transactions_received, emoji_reactions_given, emoji_reactions_received)
		VALUES ($1, $2,
			(SELECT COALESCE(SUM(points), 0) FROM karma_transactions WHERE to_user = $1 AND year = $2),
			(SELECT COALESCE(SUM(CASE WHEN points > 0 THEN points ELSE 0 END), 0) FROM karma_transactions WHERE from_user = $1 AND year = $2),
			(SELECT COALESCE(SUM(points), 0) FROM karma_transactions WHERE to_user = $1 AND year = $2),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE from_user = $1 AND year = $2),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE to_user = $1 AND year = $2),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE from_user = $1 AND year = $2 AND transaction_type = 'reactji'),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE to_user = $1 AND year = $2 AND transaction_type = 'reactji')
		)
		ON CONFLICT (username, year) DO UPDATE SET
			total_points = EXCLUDED.total_points,
			points_given = EXCLUDED.points_given,
			points_received = EXCLUDED.points_received,
			transactions_given = EXCLUDED.transactions_given,
			transactions_received = EXCLUDED.transactions_received,
			emoji_reactions_given = EXCLUDED.emoji_reactions_given,
			emoji_reactions_received = EXCLUDED.emoji_reactions_received;
	`, toUser, year)
	if err != nil {
		return fmt.Errorf("failed to update yearly summary for toUser %s, year %d: %w", toUser, year, err)
	}

	// Update yearly summary for fromUser
	_, err = ss.db.SQL.Exec(`
		INSERT INTO user_summary_yearly (username, year, total_points, points_given, points_received, transactions_given, transactions_received, emoji_reactions_given, emoji_reactions_received)
		VALUES ($1, $2,
			(SELECT COALESCE(SUM(points), 0) FROM karma_transactions WHERE to_user = $1 AND year = $2),
			(SELECT COALESCE(SUM(CASE WHEN points > 0 THEN points ELSE 0 END), 0) FROM karma_transactions WHERE from_user = $1 AND year = $2),
			(SELECT COALESCE(SUM(points), 0) FROM karma_transactions WHERE to_user = $1 AND year = $2),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE from_user = $1 AND year = $2),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE to_user = $1 AND year = $2),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE from_user = $1 AND year = $2 AND transaction_type = 'reactji'),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE to_user = $1 AND year = $2 AND transaction_type = 'reactji')
		)
		ON CONFLICT (username, year) DO UPDATE SET
			total_points = EXCLUDED.total_points,
			points_given = EXCLUDED.points_given,
			points_received = EXCLUDED.points_received,
			transactions_given = EXCLUDED.transactions_given,
			transactions_received = EXCLUDED.transactions_received,
			emoji_reactions_given = EXCLUDED.emoji_reactions_given,
			emoji_reactions_received = EXCLUDED.emoji_reactions_received;
	`, fromUser, year)
	if err != nil {
		return fmt.Errorf("failed to update yearly summary for fromUser %s, year %d: %w", fromUser, year, err)
	}

	// Update monthly summary for toUser
	_, err = ss.db.SQL.Exec(`
		INSERT INTO user_summary_monthly (username, year, month, total_points, points_given, points_received, transactions_given, transactions_received, emoji_reactions_given, emoji_reactions_received)
		SELECT $1, $2, EXTRACT(MONTH FROM MAX(timestamp)),
			(SELECT COALESCE(SUM(points), 0) FROM karma_transactions WHERE to_user = $1 AND year = $2 AND EXTRACT(MONTH FROM timestamp) = EXTRACT(MONTH FROM MAX(kt.timestamp))),
			(SELECT COALESCE(SUM(CASE WHEN points > 0 THEN points ELSE 0 END), 0) FROM karma_transactions WHERE from_user = $1 AND year = $2 AND EXTRACT(MONTH FROM timestamp) = EXTRACT(MONTH FROM MAX(kt.timestamp))),
			(SELECT COALESCE(SUM(points), 0) FROM karma_transactions WHERE to_user = $1 AND year = $2 AND EXTRACT(MONTH FROM timestamp) = EXTRACT(MONTH FROM MAX(kt.timestamp))),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE from_user = $1 AND year = $2 AND EXTRACT(MONTH FROM timestamp) = EXTRACT(MONTH FROM MAX(kt.timestamp))),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE to_user = $1 AND year = $2 AND EXTRACT(MONTH FROM timestamp) = EXTRACT(MONTH FROM MAX(kt.timestamp))),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE from_user = $1 AND year = $2 AND EXTRACT(MONTH FROM timestamp) = EXTRACT(MONTH FROM MAX(kt.timestamp)) AND transaction_type = 'reactji'),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE to_user = $1 AND year = $2 AND EXTRACT(MONTH FROM timestamp) = EXTRACT(MONTH FROM MAX(kt.timestamp)) AND transaction_type = 'reactji')
		FROM karma_transactions kt WHERE (to_user = $1 OR from_user = $1) AND year = $2
		ON CONFLICT (username, year, month) DO UPDATE SET
			total_points = EXCLUDED.total_points,
			points_given = EXCLUDED.points_given,
			points_received = EXCLUDED.points_received,
			transactions_given = EXCLUDED.transactions_given,
			transactions_received = EXCLUDED.transactions_received,
			emoji_reactions_given = EXCLUDED.emoji_reactions_given,
			emoji_reactions_received = EXCLUDED.emoji_reactions_received;
	`, toUser, year)
	if err != nil {
		return fmt.Errorf("failed to update monthly summary for toUser %s, year %d: %w", toUser, year, err)
	}

	// Update monthly summary for fromUser
	_, err = ss.db.SQL.Exec(`
		INSERT INTO user_summary_monthly (username, year, month, total_points, points_given, points_received, transactions_given, transactions_received, emoji_reactions_given, emoji_reactions_received)
		SELECT $1, $2, EXTRACT(MONTH FROM MAX(timestamp)),
			(SELECT COALESCE(SUM(points), 0) FROM karma_transactions WHERE to_user = $1 AND year = $2 AND EXTRACT(MONTH FROM timestamp) = EXTRACT(MONTH FROM MAX(kt.timestamp))),
			(SELECT COALESCE(SUM(CASE WHEN points > 0 THEN points ELSE 0 END), 0) FROM karma_transactions WHERE from_user = $1 AND year = $2 AND EXTRACT(MONTH FROM timestamp) = EXTRACT(MONTH FROM MAX(kt.timestamp))),
			(SELECT COALESCE(SUM(points), 0) FROM karma_transactions WHERE to_user = $1 AND year = $2 AND EXTRACT(MONTH FROM timestamp) = EXTRACT(MONTH FROM MAX(kt.timestamp))),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE from_user = $1 AND year = $2 AND EXTRACT(MONTH FROM timestamp) = EXTRACT(MONTH FROM MAX(kt.timestamp))),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE to_user = $1 AND year = $2 AND EXTRACT(MONTH FROM timestamp) = EXTRACT(MONTH FROM MAX(kt.timestamp))),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE from_user = $1 AND year = $2 AND EXTRACT(MONTH FROM timestamp) = EXTRACT(MONTH FROM MAX(kt.timestamp)) AND transaction_type = 'reactji'),
			(SELECT COALESCE(COUNT(*), 0) FROM karma_transactions WHERE to_user = $1 AND year = $2 AND EXTRACT(MONTH FROM timestamp) = EXTRACT(MONTH FROM MAX(kt.timestamp)) AND transaction_type = 'reactji')
		FROM karma_transactions kt WHERE (to_user = $1 OR from_user = $1) AND year = $2
		ON CONFLICT (username, year, month) DO UPDATE SET
			total_points = EXCLUDED.total_points,
			points_given = EXCLUDED.points_given,
			points_received = EXCLUDED.points_received,
			transactions_given = EXCLUDED.transactions_given,
			transactions_received = EXCLUDED.transactions_received,
			emoji_reactions_given = EXCLUDED.emoji_reactions_given,
			emoji_reactions_received = EXCLUDED.emoji_reactions_received;
	`, fromUser, year)
	if err != nil {
		return fmt.Errorf("failed to update monthly summary for fromUser %s, year %d: %w", fromUser, year, err)
	}

	return nil
}

// RebuildSummaryTables rebuilds all summary tables from transaction data
func (ss *SummaryService) RebuildSummaryTables() error {
	// Drop existing tables if they exist
	_, err := ss.db.SQL.Exec(`
		DROP TABLE IF EXISTS user_summary_current;
		DROP TABLE IF EXISTS user_summary_yearly;
		DROP TABLE IF EXISTS user_summary_monthly;
	`)
	if err != nil {
		return fmt.Errorf("failed to drop existing summary tables: %w", err)
	}

	// Create user_summary_current table
	_, err = ss.db.SQL.Exec(`
		CREATE TABLE user_summary_current (
			username TEXT PRIMARY KEY,
			total_points INTEGER,
			points_given INTEGER,
			points_received INTEGER,
			transactions_given INTEGER,
			transactions_received INTEGER,
			emoji_reactions_given INTEGER,
			emoji_reactions_received INTEGER,
			last_activity TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create user_summary_current table: %w", err)
	}

	// Create user_summary_yearly table
	_, err = ss.db.SQL.Exec(`
		CREATE TABLE user_summary_yearly (
			username TEXT,
			year INTEGER,
			total_points INTEGER,
			points_given INTEGER,
			points_received INTEGER,
			transactions_given INTEGER,
			transactions_received INTEGER,
			emoji_reactions_given INTEGER,
			emoji_reactions_received INTEGER,
			PRIMARY KEY (username, year)
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create user_summary_yearly table: %w", err)
	}

	// Create user_summary_monthly table
	_, err = ss.db.SQL.Exec(`
		CREATE TABLE user_summary_monthly (
			username TEXT,
			year INTEGER,
			month INTEGER,
			total_points INTEGER,
			points_given INTEGER,
			points_received INTEGER,
			transactions_given INTEGER,
			transactions_received INTEGER,
			emoji_reactions_given INTEGER,
			emoji_reactions_received INTEGER,
			PRIMARY KEY (username, year, month)
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create user_summary_monthly table: %w", err)
	}

	// Populate user_summary_current
	_, err = ss.db.SQL.Exec(`
		INSERT INTO user_summary_current (username, total_points, points_given, points_received, transactions_given, transactions_received, emoji_reactions_given, emoji_reactions_received, last_activity)
		SELECT
			username,
			SUM(points_received) as total_points,
			SUM(points_given) as points_given,
			SUM(points_received) as points_received,
			SUM(transactions_given) as transactions_given,
			SUM(transactions_received) as transactions_received,
			SUM(emoji_reactions_given) as emoji_reactions_given,
			SUM(emoji_reactions_received) as emoji_reactions_received,
			MAX(last_activity) as last_activity
		FROM (
			SELECT
				to_user as username,
				points as points_received,
				0 as points_given,
				1 as transactions_received,
				0 as transactions_given,
				CASE WHEN transaction_type = 'reactji' THEN 1 ELSE 0 END as emoji_reactions_received,
				0 as emoji_reactions_given,
				timestamp as last_activity
			FROM karma_transactions
			UNION ALL
			SELECT
				from_user as username,
				0 as points_received,
				points as points_given,
				0 as transactions_received,
				1 as transactions_given,
				0 as emoji_reactions_received,
				CASE WHEN transaction_type = 'reactji' THEN 1 ELSE 0 END as emoji_reactions_given,
				timestamp as last_activity
			FROM karma_transactions
		) as combined_transactions
		GROUP BY username;
	`)
	if err != nil {
		return fmt.Errorf("failed to populate user_summary_current: %w", err)
	}

	// Populate user_summary_yearly
	_, err = ss.db.SQL.Exec(`
		INSERT INTO user_summary_yearly (username, year, total_points, points_given, points_received, transactions_given, transactions_received, emoji_reactions_given, emoji_reactions_received)
		SELECT
			username,
			year,
			SUM(points_received) as total_points,
			SUM(points_given) as points_given,
			SUM(points_received) as points_received,
			SUM(transactions_given) as transactions_given,
			SUM(transactions_received) as transactions_received,
			SUM(emoji_reactions_given) as emoji_reactions_given,
			SUM(emoji_reactions_received) as emoji_reactions_received
		FROM (
			SELECT
				to_user as username,
				year,
				points as points_received,
				0 as points_given,
				1 as transactions_received,
				0 as transactions_given,
				CASE WHEN transaction_type = 'reactji' THEN 1 ELSE 0 END as emoji_reactions_received,
				0 as emoji_reactions_given
			FROM karma_transactions
			UNION ALL
			SELECT
				from_user as username,
				year,
				0 as points_received,
				points as points_given,
				0 as transactions_received,
				1 as transactions_given,
				0 as emoji_reactions_received,
				CASE WHEN transaction_type = 'reactji' THEN 1 ELSE 0 END as emoji_reactions_given
			FROM karma_transactions
		) as combined_transactions
		GROUP BY username, year;
	`)
	if err != nil {
		return fmt.Errorf("failed to populate user_summary_yearly: %w", err)
	}

	// Populate user_summary_monthly
	_, err = ss.db.SQL.Exec(`
		INSERT INTO user_summary_monthly (username, year, month, total_points, points_given, points_received, transactions_given, transactions_received, emoji_reactions_given, emoji_reactions_received)
		SELECT
			username,
			year,
			month,
			SUM(points_received) as total_points,
			SUM(points_given) as points_given,
			SUM(points_received) as points_received,
			SUM(transactions_given) as transactions_given,
			SUM(transactions_received) as transactions_received,
			SUM(emoji_reactions_given) as emoji_reactions_given,
			SUM(emoji_reactions_received) as emoji_reactions_received
		FROM (
			SELECT
				to_user as username,
				year,
				EXTRACT(MONTH FROM timestamp) as month,
				points as points_received,
				0 as points_given,
				1 as transactions_received,
				0 as transactions_given,
				CASE WHEN transaction_type = 'reactji' THEN 1 ELSE 0 END as emoji_reactions_received,
				0 as emoji_reactions_given
			FROM karma_transactions
			UNION ALL
			SELECT
				from_user as username,
				year,
				EXTRACT(MONTH FROM timestamp) as month,
				0 as points_received,
				points as points_given,
				0 as transactions_received,
				1 as transactions_given,
				0 as emoji_reactions_received,
				CASE WHEN transaction_type = 'reactji' THEN 1 ELSE 0 END as emoji_reactions_given
			FROM karma_transactions
		) as combined_transactions
		GROUP BY username, year, month;
	`)
	if err != nil {
		return fmt.Errorf("failed to populate user_summary_monthly: %w", err)
	}

	return nil
}
