package database

import (
	"database/sql"
	"time"
)

// LegacyService handles backward compatibility operations
type LegacyService struct {
	db *V2DB
}

// NewLegacyService creates a new legacy service
func NewLegacyService(db *V2DB) *LegacyService {
	return &LegacyService{db: db}
}

// InsertPoints converts old Points struct to new Transaction
func (ls *LegacyService) InsertPoints(points *Points) error {
	tx := &Transaction{
		FromUser:        points.From,
		ToUser:          points.To,
		Points:          points.Points,
		Reason:          points.Reason,
		TransactionType: "manual",
		Timestamp:       time.Now(),
	}

	transactionService := NewTransactionService(ls.db)
	return transactionService.InsertTransaction(tx)
}

// GetLeaderboard returns current leaderboard in old format
func (ls *LegacyService) GetLeaderboard(limit int) (Leaderboard, error) {
	userService := NewUserService(ls.db)
	summaries, err := userService.GetCurrentLeaderboard(limit)
	if err != nil {
		return nil, err
	}

	var leaderboard Leaderboard
	for _, summary := range summaries {
		user := &User{
			Name:   summary.Username,
			Points: summary.TotalPoints,
		}
		leaderboard = append(leaderboard, user)
	}

	return leaderboard, nil
}

// GetThrowback returns a random transaction (simplified for backward compatibility)
func (ls *LegacyService) GetThrowback(user string) (*Throwback, error) {
	var record Throwback

	query := `
		SELECT from_user, to_user, reason, points, timestamp
		FROM karma_transactions
		WHERE to_user = $1
		ORDER BY random()
		LIMIT 1
	`
	err := ls.db.SQL.QueryRow(query, user).Scan(&record.From, &record.To, &record.Reason, &record.Points.Points, &record.Timestamp)

	if err == sql.ErrNoRows {
		return nil, ErrNoSuchUser
	}
	if err != nil {
		return nil, err
	}

	return &record, nil
}
