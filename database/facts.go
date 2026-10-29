package database

import "database/sql"

// FactsService handles fun-facts and colluder queries
type FactsService struct {
	db *V2DB
}

// NewFactsService creates a new facts service
func NewFactsService(db *V2DB) *FactsService {
	return &FactsService{db: db}
}

// fetchColluderTopChannels returns the top channels (by transaction count) where user1 and user2 exchanged karma.
func (s *FactsService) fetchColluderTopChannels(user1, user2 string, year, month int) ([]*ChannelExchange, error) {
	query := `
		SELECT channel_name, COUNT(*) AS cnt
		FROM karma_transactions
		WHERE points > 0
		  AND ((from_user = $1 AND to_user = $2) OR (from_user = $2 AND to_user = $1))
		  AND channel_name IS NOT NULL AND channel_name != ''
		  AND ($3 = 0 OR year = $3)
		  AND ($4 = 0 OR EXTRACT(MONTH FROM timestamp) = $4)
		GROUP BY channel_name
		ORDER BY cnt DESC
		LIMIT 3
	`
	rows, err := s.db.SQL.Query(query, user1, user2, year, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*ChannelExchange
	for rows.Next() {
		c := &ChannelExchange{}
		if err := rows.Scan(&c.ChannelName, &c.Count); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

// GetKarmaColluders returns the top mutual karma-exchange pairs for a given year (0 = all time).
func (s *FactsService) GetKarmaColluders(limit, year int) ([]*KarmaColluder, error) {
	query := `
		WITH exchanges AS (
			SELECT from_user, to_user, SUM(points) AS points_given
			FROM karma_transactions
			WHERE points > 0
			  AND ($1 = 0 OR year = $1)
			GROUP BY from_user, to_user
		)
		SELECT
			a.from_user   AS user1,
			a.to_user     AS user2,
			a.points_given AS user1_to_user2,
			b.points_given AS user2_to_user1,
			(a.points_given + b.points_given) AS total_exchange
		FROM exchanges a
		JOIN exchanges b
		  ON a.from_user = b.to_user
		 AND a.to_user   = b.from_user
		WHERE a.from_user < a.to_user
		ORDER BY total_exchange DESC
		LIMIT $2
	`
	rows, err := s.db.SQL.Query(query, year, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*KarmaColluder
	for rows.Next() {
		c := &KarmaColluder{}
		if err := rows.Scan(&c.User1, &c.User2, &c.User1ToUser2, &c.User2ToUser1, &c.TotalExchange); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, c := range result {
		c.TopChannels, _ = s.fetchColluderTopChannels(c.User1, c.User2, year, 0)
	}
	return result, nil
}

// GetFunFacts returns computed interesting stats for a given year (0 = all time).
func (s *FactsService) GetFunFacts(year int) (*FunFacts, error) {
	f := &FunFacts{}
	yearCond := "$1 = 0 OR year = $1"

	// Most active hour of day
	hourRow := s.db.SQL.QueryRow(`
		SELECT EXTRACT(HOUR FROM timestamp)::int, COUNT(*) AS cnt
		FROM karma_transactions
		WHERE `+yearCond+`
		GROUP BY 1 ORDER BY cnt DESC LIMIT 1
	`, year)
	if err := hourRow.Scan(&f.MostActiveHour, &f.MostActiveHourCount); err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Most active day of week (0=Sun … 6=Sat)
	dowRow := s.db.SQL.QueryRow(`
		SELECT EXTRACT(DOW FROM timestamp)::int, COUNT(*) AS cnt
		FROM karma_transactions
		WHERE `+yearCond+`
		GROUP BY 1 ORDER BY cnt DESC LIMIT 1
	`, year)
	if err := dowRow.Scan(&f.MostActiveDOW, &f.MostActiveDOWCount); err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Biggest karma event — sum all transactions on the same message_id so bangbang
	// reactions stack correctly; fall back to individual transactions with no message_id.
	var fromUser, toUser string
	var points int
	var reason sql.NullString
	eventRow := s.db.SQL.QueryRow(`
		SELECT from_user, to_user, total_points, reason FROM (
			SELECT MIN(from_user) AS from_user, to_user, SUM(points) AS total_points, MIN(reason) AS reason
			FROM karma_transactions
			WHERE (`+yearCond+`) AND points > 0 AND message_id IS NOT NULL
			GROUP BY message_id, to_user
			UNION ALL
			SELECT from_user, to_user, points AS total_points, reason
			FROM karma_transactions
			WHERE (`+yearCond+`) AND points > 0 AND message_id IS NULL
		) events
		ORDER BY total_points DESC LIMIT 1
	`, year)
	if err := eventRow.Scan(&fromUser, &toUser, &points, &reason); err == nil {
		ev := &BiggestKarmaEvent{FromUser: fromUser, ToUser: toUser, Points: points}
		if reason.Valid && reason.String != "" {
			ev.Reason = &reason.String
		}
		f.BiggestKarmaEvent = ev
	} else if err != sql.ErrNoRows {
		return nil, err
	}

	// Most active channel
	var channelName sql.NullString
	var channelCount int
	chanRow := s.db.SQL.QueryRow(`
		SELECT channel_name, COUNT(*) AS cnt
		FROM karma_transactions
		WHERE (`+yearCond+`) AND channel_name IS NOT NULL AND channel_name != ''
		GROUP BY channel_name ORDER BY cnt DESC LIMIT 1
	`, year)
	if err := chanRow.Scan(&channelName, &channelCount); err == nil && channelName.Valid {
		f.MostActiveChannel = &channelName.String
		f.MostActiveChannelCount = channelCount
	} else if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Total unique channels
	uniqueChanRow := s.db.SQL.QueryRow(`
		SELECT COUNT(DISTINCT channel_name)
		FROM karma_transactions
		WHERE (`+yearCond+`) AND channel_name IS NOT NULL AND channel_name != ''
	`, year)
	if err := uniqueChanRow.Scan(&f.TotalUniqueChannels); err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return f, nil
}

// GetKarmaColludersByYearMonth returns top mutual karma-exchange pairs for a given year+month.
func (s *FactsService) GetKarmaColludersByYearMonth(limit, year, month int) ([]*KarmaColluder, error) {
	query := `
		WITH exchanges AS (
			SELECT from_user, to_user, SUM(points) AS points_given
			FROM karma_transactions
			WHERE points > 0 AND year = $1 AND EXTRACT(MONTH FROM timestamp) = $2
			GROUP BY from_user, to_user
		)
		SELECT
			a.from_user   AS user1,
			a.to_user     AS user2,
			a.points_given AS user1_to_user2,
			b.points_given AS user2_to_user1,
			(a.points_given + b.points_given) AS total_exchange
		FROM exchanges a
		JOIN exchanges b
		  ON a.from_user = b.to_user
		 AND a.to_user   = b.from_user
		WHERE a.from_user < a.to_user
		ORDER BY total_exchange DESC
		LIMIT $3
	`
	rows, err := s.db.SQL.Query(query, year, month, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*KarmaColluder
	for rows.Next() {
		c := &KarmaColluder{}
		if err := rows.Scan(&c.User1, &c.User2, &c.User1ToUser2, &c.User2ToUser1, &c.TotalExchange); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, c := range result {
		c.TopChannels, _ = s.fetchColluderTopChannels(c.User1, c.User2, year, month)
	}
	return result, nil
}

// GetFunFactsByYearMonth returns computed interesting stats for a given year+month.
func (s *FactsService) GetFunFactsByYearMonth(year, month int) (*FunFacts, error) {
	f := &FunFacts{}
	cond := "year = $1 AND EXTRACT(MONTH FROM timestamp) = $2"

	hourRow := s.db.SQL.QueryRow(`
		SELECT EXTRACT(HOUR FROM timestamp)::int, COUNT(*) AS cnt
		FROM karma_transactions
		WHERE `+cond+`
		GROUP BY 1 ORDER BY cnt DESC LIMIT 1
	`, year, month)
	if err := hourRow.Scan(&f.MostActiveHour, &f.MostActiveHourCount); err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	dowRow := s.db.SQL.QueryRow(`
		SELECT EXTRACT(DOW FROM timestamp)::int, COUNT(*) AS cnt
		FROM karma_transactions
		WHERE `+cond+`
		GROUP BY 1 ORDER BY cnt DESC LIMIT 1
	`, year, month)
	if err := dowRow.Scan(&f.MostActiveDOW, &f.MostActiveDOWCount); err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	var fromUser, toUser string
	var points int
	var reason sql.NullString
	eventRow := s.db.SQL.QueryRow(`
		SELECT from_user, to_user, total_points, reason FROM (
			SELECT MIN(from_user) AS from_user, to_user, SUM(points) AS total_points, MIN(reason) AS reason
			FROM karma_transactions
			WHERE (`+cond+`) AND points > 0 AND message_id IS NOT NULL
			GROUP BY message_id, to_user
			UNION ALL
			SELECT from_user, to_user, points AS total_points, reason
			FROM karma_transactions
			WHERE (`+cond+`) AND points > 0 AND message_id IS NULL
		) events
		ORDER BY total_points DESC LIMIT 1
	`, year, month)
	if err := eventRow.Scan(&fromUser, &toUser, &points, &reason); err == nil {
		ev := &BiggestKarmaEvent{FromUser: fromUser, ToUser: toUser, Points: points}
		if reason.Valid && reason.String != "" {
			ev.Reason = &reason.String
		}
		f.BiggestKarmaEvent = ev
	} else if err != sql.ErrNoRows {
		return nil, err
	}

	var channelName sql.NullString
	var channelCount int
	chanRow := s.db.SQL.QueryRow(`
		SELECT channel_name, COUNT(*) AS cnt
		FROM karma_transactions
		WHERE (`+cond+`) AND channel_name IS NOT NULL AND channel_name != ''
		GROUP BY channel_name ORDER BY cnt DESC LIMIT 1
	`, year, month)
	if err := chanRow.Scan(&channelName, &channelCount); err == nil && channelName.Valid {
		f.MostActiveChannel = &channelName.String
		f.MostActiveChannelCount = channelCount
	} else if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	uniqueChanRow := s.db.SQL.QueryRow(`
		SELECT COUNT(DISTINCT channel_name)
		FROM karma_transactions
		WHERE (`+cond+`) AND channel_name IS NOT NULL AND channel_name != ''
	`, year, month)
	if err := uniqueChanRow.Scan(&f.TotalUniqueChannels); err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return f, nil
}

// GetUserTopChannels returns the most active channels for a user (year=0 means all-time, month=0 means all months).
func (s *FactsService) GetUserTopChannels(username string, year, month, limit int) ([]*ChannelStats, error) {
	query := `
		SELECT channel_name, COUNT(*) AS transaction_count, COALESCE(SUM(points), 0) AS total_points
		FROM karma_transactions
		WHERE (from_user = $1 OR to_user = $1)
		  AND channel_name IS NOT NULL AND channel_name != ''
		  AND ($2 = 0 OR year = $2)
		  AND ($3 = 0 OR EXTRACT(MONTH FROM timestamp) = $3)
		GROUP BY channel_name
		ORDER BY transaction_count DESC
		LIMIT $4
	`
	rows, err := s.db.SQL.Query(query, username, year, month, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*ChannelStats
	for rows.Next() {
		c := &ChannelStats{}
		if err := rows.Scan(&c.ChannelName, &c.TransactionCount, &c.TotalPoints); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

// GetUserColluders returns colluder pairs involving a specific user.
func (s *FactsService) GetUserColluders(username string, limit, year, month int) ([]*KarmaColluder, error) {
	var query string
	var args []interface{}
	if month > 0 {
		query = `
			WITH exchanges AS (
				SELECT from_user, to_user, SUM(points) AS points_given
				FROM karma_transactions
				WHERE points > 0 AND year = $2 AND EXTRACT(MONTH FROM timestamp) = $3
				GROUP BY from_user, to_user
			)
			SELECT
				a.from_user AS user1,
				a.to_user   AS user2,
				a.points_given AS user1_to_user2,
				b.points_given AS user2_to_user1,
				(a.points_given + b.points_given) AS total_exchange
			FROM exchanges a
			JOIN exchanges b ON a.from_user = b.to_user AND a.to_user = b.from_user
			WHERE a.from_user < a.to_user
			  AND (a.from_user = $1 OR a.to_user = $1)
			ORDER BY total_exchange DESC
			LIMIT $4
		`
		args = []interface{}{username, year, month, limit}
	} else {
		query = `
			WITH exchanges AS (
				SELECT from_user, to_user, SUM(points) AS points_given
				FROM karma_transactions
				WHERE points > 0 AND ($2 = 0 OR year = $2)
				GROUP BY from_user, to_user
			)
			SELECT
				a.from_user AS user1,
				a.to_user   AS user2,
				a.points_given AS user1_to_user2,
				b.points_given AS user2_to_user1,
				(a.points_given + b.points_given) AS total_exchange
			FROM exchanges a
			JOIN exchanges b ON a.from_user = b.to_user AND a.to_user = b.from_user
			WHERE a.from_user < a.to_user
			  AND (a.from_user = $1 OR a.to_user = $1)
			ORDER BY total_exchange DESC
			LIMIT $3
		`
		args = []interface{}{username, year, limit}
	}

	rows, err := s.db.SQL.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*KarmaColluder
	for rows.Next() {
		c := &KarmaColluder{}
		if err := rows.Scan(&c.User1, &c.User2, &c.User1ToUser2, &c.User2ToUser1, &c.TotalExchange); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, c := range result {
		c.TopChannels, _ = s.fetchColluderTopChannels(c.User1, c.User2, year, month)
	}
	return result, nil
}
