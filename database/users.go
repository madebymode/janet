package database

import (
	"database/sql"
	"time"
)

// UserService handles user-related database operations
type UserService struct {
	db *V2DB
}

// NewUserService creates a new user service
func NewUserService(db *V2DB) *UserService {
	return &UserService{db: db}
}

// GetCurrentLeaderboard returns the all-time leaderboard using optimized summary table
func (us *UserService) GetCurrentLeaderboard(limit int) ([]*UserSummary, error) {
	query := `
		SELECT
			username,
			total_points,
			points_given,
			points_received,
			transactions_given,
			transactions_received,
			emoji_reactions_given,
			emoji_reactions_received,
			last_activity,
			RANK() OVER (ORDER BY total_points DESC) as rank
		FROM user_summary_current
		ORDER BY total_points DESC
		LIMIT $1
	`

	rows, err := us.db.SQL.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leaderboard []*UserSummary
	for rows.Next() {
		user := &UserSummary{}
		var lastActivity sql.NullTime
		err := rows.Scan(
			&user.Username, &user.TotalPoints, &user.PointsGiven, &user.PointsReceived,
			&user.TransactionsGiven, &user.TransactionsReceived,
			&user.EmojiReactionsGiven, &user.EmojiReactionsReceived, &lastActivity, &user.Rank,
		)
		if err != nil {
			return nil, err
		}
		if lastActivity.Valid {
			user.LastActivity = &lastActivity.Time
		}
		leaderboard = append(leaderboard, user)
	}

	return leaderboard, nil
}

// GetYearlyLeaderboard returns leaderboard for a specific year
func (us *UserService) GetYearlyLeaderboard(year, limit int) ([]*UserSummary, error) {
	query := `
		SELECT
			username,
			total_points,
			points_given,
			points_received,
			transactions_given,
			transactions_received,
			emoji_reactions_given,
			emoji_reactions_received,
			RANK() OVER (ORDER BY total_points DESC) as rank
		FROM user_summary_yearly
		WHERE year = $1
		ORDER BY total_points DESC
		LIMIT $2
	`

	rows, err := us.db.SQL.Query(query, year, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leaderboard []*UserSummary
	for rows.Next() {
		user := &UserSummary{Year: &year}
		err := rows.Scan(
			&user.Username, &user.TotalPoints, &user.PointsGiven, &user.PointsReceived,
			&user.TransactionsGiven, &user.TransactionsReceived,
			&user.EmojiReactionsGiven, &user.EmojiReactionsReceived, &user.Rank,
		)
		if err != nil {
			return nil, err
		}
		leaderboard = append(leaderboard, user)
	}

	return leaderboard, nil
}

// GetUser returns comprehensive user information for current year
func (us *UserService) GetUser(username string) (*UserSummary, error) {
	query := `
		SELECT
			username,
			total_points,
			points_given,
			points_received,
			transactions_given,
			transactions_received,
			emoji_reactions_given,
			emoji_reactions_received,
			last_activity,
			rank
		FROM (
			SELECT
				username,
				total_points,
				points_given,
				points_received,
				transactions_given,
				transactions_received,
				emoji_reactions_given,
				emoji_reactions_received,
				last_activity,
				RANK() OVER (ORDER BY total_points DESC) as rank
			FROM user_summary_current
		) ranked_users
		WHERE username = $1
	`

	user := &UserSummary{}
	var lastActivity sql.NullTime

	err := us.db.SQL.QueryRow(query, username).Scan(
		&user.Username, &user.TotalPoints, &user.PointsGiven, &user.PointsReceived,
		&user.TransactionsGiven, &user.TransactionsReceived,
		&user.EmojiReactionsGiven, &user.EmojiReactionsReceived,
		&lastActivity, &user.Rank,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNoSuchUser
	}
	if err != nil {
		return nil, err
	}

	if lastActivity.Valid {
		user.LastActivity = &lastActivity.Time
	}

	return user, nil
}

// GetUserByYear returns user information for a specific year
func (us *UserService) GetUserByYear(username string, year int) (*UserSummary, error) {
	query := `
		SELECT
			username,
			total_points,
			points_given,
			points_received,
			transactions_given,
			transactions_received,
			emoji_reactions_given,
			emoji_reactions_received,
			rank
		FROM (
			SELECT
				username,
				total_points,
				points_given,
				points_received,
				transactions_given,
				transactions_received,
				emoji_reactions_given,
				emoji_reactions_received,
				RANK() OVER (ORDER BY total_points DESC) as rank
			FROM user_summary_yearly
			WHERE year = $1
		) ranked_users
		WHERE username = $2
	`

	user := &UserSummary{Year: &year}

	err := us.db.SQL.QueryRow(query, year, username).Scan(
		&user.Username, &user.TotalPoints, &user.PointsGiven, &user.PointsReceived,
		&user.TransactionsGiven, &user.TransactionsReceived,
		&user.EmojiReactionsGiven, &user.EmojiReactionsReceived, &user.Rank,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNoSuchUser
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetTopGivers returns users who have given the most points using optimized summary table
func (us *UserService) GetTopGivers(limit int, year int) ([]*UserSummary, error) {
	var query string
	var args []interface{}

	if year > 0 {
		query = `
			SELECT username,
				total_points,
				points_given,
				points_received,
				transactions_given,
				transactions_received,
				emoji_reactions_given,
				emoji_reactions_received,
				ROW_NUMBER() OVER (ORDER BY points_given DESC) as rank
			FROM user_summary_yearly
			WHERE year = $1 AND points_given > 0
			ORDER BY points_given DESC
			LIMIT $2
		`
		args = append(args, year, limit)
	} else {
		query = `
			SELECT username,
				total_points,
				points_given,
				points_received,
				transactions_given,
				transactions_received,
				emoji_reactions_given,
				emoji_reactions_received,
				ROW_NUMBER() OVER (ORDER BY points_given DESC) as rank
			FROM user_summary_current
			WHERE points_given > 0
			ORDER BY points_given DESC
			LIMIT $1
		`
		args = append(args, limit)
	}

	rows, err := us.db.SQL.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topGivers []*UserSummary
	for rows.Next() {
		user := &UserSummary{}
		err := rows.Scan(
			&user.Username, &user.TotalPoints, &user.PointsGiven, &user.PointsReceived,
			&user.TransactionsGiven, &user.TransactionsReceived,
			&user.EmojiReactionsGiven, &user.EmojiReactionsReceived, &user.Rank,
		)
		if err != nil {
			return nil, err
		}
		topGivers = append(topGivers, user)
	}

	return topGivers, nil
}

// GetTotalUsers returns total unique users across all years
func (us *UserService) GetTotalUsers(year int) (int, error) {
	if year > 0 {
		var total int
		err := us.db.SQL.QueryRow(`SELECT COUNT(*) FROM user_summary_yearly WHERE year = $1`, year).Scan(&total)
		return total, err
	}

	var total int
	err := us.db.SQL.QueryRow(`SELECT COUNT(*) FROM user_summary_current`).Scan(&total)
	return total, err
}

// Helper function to get current year
func (us *UserService) getCurrentYear() int {
	return time.Now().Year()
}

// New methods implementing the byCurrentYear, byYear, and Cumulative pattern

// User data methods
func (us *UserService) GetUserByCurrentYear(username string) (*UserSummary, error) {
	return us.GetUserByYear(username, us.getCurrentYear())
}

func (us *UserService) GetUserCumulative(username string) (*UserSummary, error) {
	return us.GetUser(username) // GetUser already returns cumulative data
}

// Leaderboard methods
func (us *UserService) GetLeaderboard(limit int) ([]*UserSummary, error) {
	return us.GetLeaderboardByCurrentYear(limit)
}

func (us *UserService) GetLeaderboardByCurrentYear(limit int) ([]*UserSummary, error) {
	return us.GetYearlyLeaderboard(us.getCurrentYear(), limit)
}

func (us *UserService) GetLeaderboardCumulative(limit int) ([]*UserSummary, error) {
	return us.GetCurrentLeaderboard(limit) // GetCurrentLeaderboard already returns cumulative data
}

// Top givers methods
func (us *UserService) GetTopGiversDefault(limit int) ([]*UserSummary, error) {
	return us.GetTopGiversByCurrentYear(limit)
}

func (us *UserService) GetTopGiversByCurrentYear(limit int) ([]*UserSummary, error) {
	return us.GetTopGiversByYear(limit, us.getCurrentYear())
}

func (us *UserService) GetTopGiversByYear(limit int, year int) ([]*UserSummary, error) {
	return us.GetTopGivers(limit, year)
}

func (us *UserService) GetTopGiversCumulative(limit int) ([]*UserSummary, error) {
	return us.GetTopGiversByYear(limit, 0) // 0 means all-time/cumulative
}

// User count methods
func (us *UserService) GetTotalUsersDefault() (int, error) {
	return us.GetTotalUsersByCurrentYear()
}

func (us *UserService) GetTotalUsersByCurrentYear() (int, error) {
	return us.GetTotalUsersByYear(us.getCurrentYear())
}

func (us *UserService) GetTotalUsersByYear(year int) (int, error) {
	return us.GetTotalUsers(year)
}

func (us *UserService) GetTotalUsersCumulative() (int, error) {
	return us.GetTotalUsersByYear(0) // 0 means all-time/cumulative
}
