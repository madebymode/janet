package database

import "time"

// EmojiService handles emoji-related database operations
type EmojiService struct {
	db *V2DB
}

// NewEmojiService creates a new emoji service
func NewEmojiService(db *V2DB) *EmojiService {
	return &EmojiService{db: db}
}

// GetTopEmojis returns most used emojis for current year
func (es *EmojiService) GetTopEmojis(limit int) ([]*EmojiStats, error) {
	return es.GetTopEmojisByCurrentYear(limit)
}

// GetTopEmojisByYear returns most used emojis for a specific year
func (es *EmojiService) GetTopEmojisByYear(year, limit int) ([]*EmojiStats, error) {
	query := `
		SELECT
		    emoji_name,
		    year,
		    COUNT(*) as usage_count,
		    SUM(ABS(points)) as points_awarded,
		    COUNT(DISTINCT from_user) as unique_users,
		    RANK() OVER (ORDER BY COUNT(*) DESC) as rank
		FROM karma_transactions
		WHERE emoji_name IS NOT NULL AND year = $1
		GROUP BY emoji_name, year
		ORDER BY usage_count DESC
		LIMIT $2
	`

	rows, err := es.db.SQL.Query(query, year, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emojis []*EmojiStats
	for rows.Next() {
		emoji := &EmojiStats{}
		err := rows.Scan(&emoji.EmojiName, &emoji.Year, &emoji.UsageCount,
			&emoji.PointsAwarded, &emoji.UniqueUsers, &emoji.Rank)
		if err != nil {
			return nil, err
		}
		emojis = append(emojis, emoji)
	}

	return emojis, nil
}

// GetTotalEmojiUsageByYear returns the total count of all emoji transactions for a specific year
func (es *EmojiService) GetTotalEmojiUsageByYear(year int) (int, error) {
	var total int
	query := `
		SELECT COALESCE(COUNT(*), 0)
		FROM karma_transactions
		WHERE emoji_name IS NOT NULL AND year = $1
	`
	err := es.db.SQL.QueryRow(query, year).Scan(&total)
	return total, err
}

// GetEmojiLeaderboardByYear returns top users for a specific emoji
func (es *EmojiService) GetEmojiLeaderboardByYear(emojiName string, year int) ([]*EmojiLeaderboard, error) {
	query := `
		SELECT
			emoji_name,
			year,
			username,
			times_given,
			times_received,
			points_from_emoji
		FROM (
			SELECT
				$1 as emoji_name,
				$2 as year,
				username,
				SUM(times_given) as times_given,
				SUM(times_received) as times_received,
				SUM(points_from_emoji) as points_from_emoji,
				RANK() OVER (ORDER BY SUM(points_from_emoji) DESC) as rank
			FROM (
				SELECT
					from_user as username,
					COUNT(*) as times_given,
					0 as times_received,
					SUM(ABS(points)) as points_from_emoji
				FROM karma_transactions
				WHERE emoji_name = $1 AND year = $2
				GROUP BY from_user
				UNION ALL
				SELECT
					to_user as username,
					0 as times_given,
					COUNT(*) as times_received,
					SUM(ABS(points)) as points_from_emoji
				FROM karma_transactions
				WHERE emoji_name = $1 AND year = $2
				GROUP BY to_user
			) as emoji_activity
			GROUP BY username
			ORDER BY points_from_emoji DESC
			LIMIT 20
		) as leaderboard
	`

	rows, err := es.db.SQL.Query(query, emojiName, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leaderboard []*EmojiLeaderboard
	for rows.Next() {
		entry := &EmojiLeaderboard{}
		err := rows.Scan(&entry.EmojiName, &entry.Year, &entry.Username,
			&entry.TimesGiven, &entry.TimesReceived, &entry.PointsFromEmoji)
		if err != nil {
			return nil, err
		}
		leaderboard = append(leaderboard, entry)
	}

	return leaderboard, nil
}

// Helper function to get current year
func (es *EmojiService) getCurrentYear() int {
	return time.Now().Year()
}

// New methods implementing the byCurrentYear, byYear, and Cumulative pattern

// Top emojis methods
func (es *EmojiService) GetTopEmojisByCurrentYear(limit int) ([]*EmojiStats, error) {
	return es.GetTopEmojisByYear(es.getCurrentYear(), limit)
}

func (es *EmojiService) GetTopEmojisCumulative(limit int) ([]*EmojiStats, error) {
	query := `
		SELECT
		    emoji_name,
		    0 as year,
		    COUNT(*) as usage_count,
		    SUM(ABS(points)) as points_awarded,
		    COUNT(DISTINCT from_user) as unique_users,
		    RANK() OVER (ORDER BY COUNT(*) DESC) as rank
		FROM karma_transactions
		WHERE emoji_name IS NOT NULL
		GROUP BY emoji_name
		ORDER BY usage_count DESC
		LIMIT $1
	`

	rows, err := es.db.SQL.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emojis []*EmojiStats
	for rows.Next() {
		emoji := &EmojiStats{}
		err := rows.Scan(&emoji.EmojiName, &emoji.Year, &emoji.UsageCount,
			&emoji.PointsAwarded, &emoji.UniqueUsers, &emoji.Rank)
		if err != nil {
			return nil, err
		}
		emojis = append(emojis, emoji)
	}

	return emojis, nil
}

// Emoji usage methods
func (es *EmojiService) GetTotalEmojiUsage() (int, error) {
	return es.GetTotalEmojiUsageByCurrentYear()
}

func (es *EmojiService) GetTotalEmojiUsageByCurrentYear() (int, error) {
	return es.GetTotalEmojiUsageByYear(es.getCurrentYear())
}

func (es *EmojiService) GetTotalEmojiUsageCumulative() (int, error) {
	var total int
	query := `
		SELECT COALESCE(COUNT(*), 0)
		FROM karma_transactions
		WHERE emoji_name IS NOT NULL
	`
	err := es.db.SQL.QueryRow(query).Scan(&total)
	return total, err
}

// Emoji leaderboard methods

func (es *EmojiService) GetEmojiLeaderboardByCurrentYear(emojiName string) ([]*EmojiLeaderboard, error) {
	return es.GetEmojiLeaderboardByYear(emojiName, es.getCurrentYear())
}

func (es *EmojiService) GetEmojiLeaderboardCumulative(emojiName string) ([]*EmojiLeaderboard, error) {
	query := `
		SELECT
			$1 as emoji_name,
			0 as year,
			username,
			times_given,
			times_received,
			points_from_emoji
		FROM (
			SELECT
				username,
				SUM(times_given) as times_given,
				SUM(times_received) as times_received,
				SUM(points_from_emoji) as points_from_emoji,
				RANK() OVER (ORDER BY SUM(points_from_emoji) DESC) as rank
			FROM (
				SELECT
					from_user as username,
					COUNT(*) as times_given,
					0 as times_received,
					SUM(ABS(points)) as points_from_emoji
				FROM karma_transactions
				WHERE emoji_name = $1
				GROUP BY from_user
				UNION ALL
				SELECT
					to_user as username,
					0 as times_given,
					COUNT(*) as times_received,
					SUM(ABS(points)) as points_from_emoji
				FROM karma_transactions
				WHERE emoji_name = $1
				GROUP BY to_user
			) as emoji_activity
			GROUP BY username
			ORDER BY points_from_emoji DESC
			LIMIT 20
		) as leaderboard
	`

	rows, err := es.db.SQL.Query(query, emojiName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leaderboard []*EmojiLeaderboard
	for rows.Next() {
		entry := &EmojiLeaderboard{}
		err := rows.Scan(&entry.EmojiName, &entry.Year, &entry.Username,
			&entry.TimesGiven, &entry.TimesReceived, &entry.PointsFromEmoji)
		if err != nil {
			return nil, err
		}
		leaderboard = append(leaderboard, entry)
	}

	return leaderboard, nil
}
