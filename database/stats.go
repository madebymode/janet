package database

import (
	"fmt"
	"time"
)

// StatsService handles statistics-related database operations
type StatsService struct {
	db *V2DB
}

// NewStatsService creates a new stats service
func NewStatsService(db *V2DB) *StatsService {
	return &StatsService{db: db}
}

// GetTotalPoints returns total points across all years
func (ss *StatsService) GetTotalPoints(year int) (int, error) {
	query := `SELECT COALESCE(SUM(ABS(points)), 0) FROM karma_transactions`
	args := []interface{}{}

	if year > 0 {
		query += " WHERE year = $1"
		args = append(args, year)
	}

	var totalPoints int
	err := ss.db.SQL.QueryRow(query, args...).Scan(&totalPoints)
	return totalPoints, err
}

// GetAvailableYears returns all years with data, always including the current year
func (ss *StatsService) GetAvailableYears() ([]int, error) {
	query := `
		SELECT DISTINCT year
		FROM karma_transactions
		ORDER BY year DESC;
	`

	rows, err := ss.db.SQL.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var years []int
	currentYear := time.Now().Year()
	hasCurrentYear := false

	for rows.Next() {
		var year int
		if err := rows.Scan(&year); err != nil {
			return nil, err
		}
		if year == currentYear {
			hasCurrentYear = true
		}
		years = append(years, year)
	}

	// Always include current year even if no data exists yet
	if !hasCurrentYear {
		// Insert at the beginning since years are sorted DESC
		years = append([]int{currentYear}, years...)
	}

	return years, nil
}

// GetMonthlyStats returns monthly statistics for a year
func (ss *StatsService) GetMonthlyStats(year int) ([]*MonthlyStats, error) {
	query := `
		SELECT
		    %d as year,
		    EXTRACT(MONTH FROM kt.timestamp) as month,
		    COUNT(*) as total_transactions,
		    SUM(ABS(kt.points)) as total_points_awarded,
		    COUNT(DISTINCT kt.from_user) + COUNT(DISTINCT kt.to_user) as unique_users,
		    (SELECT emoji_name FROM karma_transactions WHERE year = %d AND EXTRACT(MONTH FROM timestamp) = EXTRACT(MONTH FROM kt.timestamp) AND emoji_name IS NOT NULL GROUP BY emoji_name ORDER BY COUNT(*) DESC LIMIT 1) as top_emoji,
		    (SELECT username FROM (
		        SELECT from_user as username, COUNT(*) as activity_count FROM karma_transactions WHERE year = %d AND EXTRACT(MONTH FROM timestamp) = EXTRACT(MONTH FROM kt.timestamp) GROUP BY from_user
		        UNION ALL
		        SELECT to_user as username, COUNT(*) as activity_count FROM karma_transactions WHERE year = %d AND EXTRACT(MONTH FROM timestamp) = EXTRACT(MONTH FROM kt.timestamp) GROUP BY to_user
		    ) AS monthly_activity GROUP BY username ORDER BY SUM(activity_count) DESC LIMIT 1) as most_active_user
		FROM karma_transactions kt
		WHERE kt.year = %d
		GROUP BY EXTRACT(MONTH FROM kt.timestamp)
		ORDER BY month
	`

	rows, err := ss.db.SQL.Query(fmt.Sprintf(query, year, year, year, year, year))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*MonthlyStats
	for rows.Next() {
		stat := &MonthlyStats{}
		err := rows.Scan(&stat.Year, &stat.Month, &stat.TotalTransactions,
			&stat.TotalPointsAwarded, &stat.UniqueUsers,
			&stat.TopEmoji, &stat.MostActiveUser)
		if err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// GetPointsOverTimeMonthly returns monthly aggregated points data for a specific year
func (ss *StatsService) GetPointsOverTimeMonthly(year int) ([]map[string]interface{}, error) {
	query := `
		SELECT
		    EXTRACT(MONTH FROM timestamp) as month,
		    SUM(points) as total_points
		FROM karma_transactions
		WHERE year = $1
		GROUP BY EXTRACT(MONTH FROM timestamp)
		ORDER BY month
	`

	rows, err := ss.db.SQL.Query(query, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []map[string]interface{}
	for rows.Next() {
		var month int
		var totalPoints int
		err := rows.Scan(&month, &totalPoints)
		if err != nil {
			return nil, err
		}
		data = append(data, map[string]interface{}{
			"month":       month,
			"totalPoints": totalPoints,
		})
	}

	return data, nil
}

// GetKarmaDistribution returns distribution of karma points across users
func (ss *StatsService) GetKarmaDistribution(year int) ([]map[string]interface{}, error) {
	// First, get all users and their total points
	leaderboardQuery := `
		SELECT SUM(points_received) as total_points
		FROM (
		    SELECT
		        to_user as username,
		        points as points_received
		    FROM karma_transactions
		    WHERE ($1 = 0 OR year = $1)
		    UNION ALL
		    SELECT
		        from_user as username,
		        0 as points_received
		    FROM karma_transactions
		    WHERE ($1 = 0 OR year = $1)
		) as combined_transactions
		GROUP BY username
		HAVING SUM(points_received) > 0
		ORDER BY total_points DESC
	`

	rows, err := ss.db.SQL.Query(leaderboardQuery, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Count users in different ranges
	distribution := map[string]int{
		"0 to 10":     0,
		"11 to 50":    0,
		"51 to 100":   0,
		"101 to 200":  0,
		"201 to 500":  0,
		"501 to 1000": 0,
		"1000+":       0,
	}

	for rows.Next() {
		var points int
		if err := rows.Scan(&points); err != nil {
			return nil, err
		}

		switch {
		case points >= 1000:
			distribution["1000+"]++
		case points >= 501:
			distribution["501 to 1000"]++
		case points >= 201:
			distribution["201 to 500"]++
		case points >= 101:
			distribution["101 to 200"]++
		case points >= 51:
			distribution["51 to 100"]++
		case points >= 11:
			distribution["11 to 50"]++
		default:
			distribution["0 to 10"]++
		}
	}

	// Convert to response format
	var response []map[string]interface{}
	for rangeStr, count := range distribution {
		if count > 0 { // Only include ranges with users
			response = append(response, map[string]interface{}{
				"range": rangeStr,
				"count": count,
			})
		}
	}

	return response, nil
}

// GetActivityTimeline returns activity timeline data
func (ss *StatsService) GetActivityTimeline(year int) ([]map[string]interface{}, error) {
	query := `
		SELECT
			DATE_TRUNC('month', timestamp) as date,
			SUM(CASE WHEN points > 0 THEN points ELSE 0 END) as positive,
			SUM(CASE WHEN points < 0 THEN ABS(points) ELSE 0 END) as negative,
			COUNT(*) as total
		FROM karma_transactions
		WHERE year = $1
		GROUP BY DATE_TRUNC('month', timestamp)
		ORDER BY DATE_TRUNC('month', timestamp)
	`

	rows, err := ss.db.SQL.Query(query, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var response []map[string]interface{}
	for rows.Next() {
		var date time.Time
		var positive, negative, total int

		err := rows.Scan(&date, &positive, &negative, &total)
		if err != nil {
			return nil, err
		}

		response = append(response, map[string]interface{}{
			"date":     date.Format("2006-01-02"),
			"positive": positive,
			"negative": negative,
			"total":    total,
		})
	}

	return response, nil
}

// Helper function to get current year
func (ss *StatsService) getCurrentYear() int {
	return time.Now().Year()
}

// New methods implementing the byCurrentYear, byYear, and Cumulative pattern

// Total points methods
func (ss *StatsService) GetTotalPointsByCurrentYear() (int, error) {
	return ss.GetTotalPointsByYear(ss.getCurrentYear())
}

func (ss *StatsService) GetTotalPointsCumulative() (int, error) {
	return ss.GetTotalPointsByYear(0) // 0 means all-time/cumulative
}

// Monthly stats methods
func (ss *StatsService) GetMonthlyStatsByCurrentYear() ([]*MonthlyStats, error) {
	return ss.GetMonthlyStatsByYear(ss.getCurrentYear())
}

// Points over time methods
func (ss *StatsService) GetPointsOverTimeMonthlyByCurrentYear() ([]map[string]interface{}, error) {
	return ss.GetPointsOverTimeMonthlyByYear(ss.getCurrentYear())
}

// Activity timeline methods
func (ss *StatsService) GetActivityTimelineByCurrentYear() ([]map[string]interface{}, error) {
	return ss.GetActivityTimelineByYear(ss.getCurrentYear())
}

// Karma distribution methods
func (ss *StatsService) GetKarmaDistributionByCurrentYear() ([]map[string]interface{}, error) {
	return ss.GetKarmaDistributionByYear(ss.getCurrentYear())
}

func (ss *StatsService) GetKarmaDistributionCumulative() ([]map[string]interface{}, error) {
	return ss.GetKarmaDistributionByYear(0) // 0 means all-time/cumulative
}

// Renamed existing methods to match new pattern
func (ss *StatsService) GetTotalPointsByYear(year int) (int, error) {
	return ss.GetTotalPoints(year)
}

func (ss *StatsService) GetMonthlyStatsByYear(year int) ([]*MonthlyStats, error) {
	return ss.GetMonthlyStats(year)
}

func (ss *StatsService) GetPointsOverTimeMonthlyByYear(year int) ([]map[string]interface{}, error) {
	return ss.GetPointsOverTimeMonthly(year)
}

func (ss *StatsService) GetActivityTimelineByYear(year int) ([]map[string]interface{}, error) {
	return ss.GetActivityTimeline(year)
}

func (ss *StatsService) GetKarmaDistributionByYear(year int) ([]map[string]interface{}, error) {
	return ss.GetKarmaDistribution(year)
}
