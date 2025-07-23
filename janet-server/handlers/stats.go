package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/troyxmccall/janet/database"
)

// HandleAPIStatsV2 handles requests for basic statistics (V2 API)
func (h *Handler) HandleAPIStatsV2(w http.ResponseWriter, r *http.Request) {
	year := 0
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if parsed, err := strconv.Atoi(yearStr); err == nil {
			year = parsed
		}
	}

	var totalPoints int
	if year == 0 {
		totalPoints, _ = h.db.GetTotalPointsCumulative()
	} else {
		totalPoints, _ = h.db.GetTotalPointsByYear(year)
	}

	var totalUsers int
	if year == 0 {
		totalUsers, _ = h.db.GetTotalUsersCumulative()
	} else {
		totalUsers, _ = h.db.GetTotalUsersByYear(year)
	}

	var totalTransactions int
	if year == 0 {
		totalTransactions, _ = h.db.GetTotalTransactionsCumulative()
	} else {
		totalTransactions, _ = h.db.GetTotalTransactionsByYear(year)
	}

	response := map[string]interface{}{
		"totalUsers":        totalUsers,
		"totalPoints":       totalPoints,
		"totalTransactions": totalTransactions,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleAPIStatsDetailed handles requests for detailed statistics
func (h *Handler) HandleAPIStatsDetailed(w http.ResponseWriter, r *http.Request) {
	year := 0
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if parsed, err := strconv.Atoi(yearStr); err == nil {
			year = parsed
		}
	}

	var err error
	var totalUsers int
	if year == 0 {
		totalUsers, err = h.db.GetTotalUsersCumulative()
	} else {
		totalUsers, err = h.db.GetTotalUsersByYear(year)
	}
	if err != nil {
		h.logger.Err(err).Error("failed to get total users")
		totalUsers = 0
	}

	var totalPoints int
	if year == 0 {
		totalPoints, err = h.db.GetTotalPointsCumulative()
	} else {
		totalPoints, err = h.db.GetTotalPointsByYear(year)
	}
	if err != nil {
		h.logger.Err(err).Error("failed to get total points")
		totalPoints = 0
	}

	var totalTransactions int
	if year == 0 {
		totalTransactions, err = h.db.GetTotalTransactionsCumulative()
	} else {
		totalTransactions, err = h.db.GetTotalTransactionsByYear(year)
	}
	if err != nil {
		h.logger.Err(err).Error("failed to get total transactions")
		totalTransactions = 0
	}

	var positiveTransactions int
	if year == 0 {
		positiveTransactions, err = h.db.GetPositiveTransactionsCumulative()
	} else {
		positiveTransactions, err = h.db.GetPositiveTransactionsByYear(year)
	}
	if err != nil {
		h.logger.Err(err).Error("failed to get positive transactions")
		positiveTransactions = 0
	}

	var negativeTransactions int
	if year == 0 {
		negativeTransactions, err = h.db.GetNegativeTransactionsCumulative()
	} else {
		negativeTransactions, err = h.db.GetNegativeTransactionsByYear(year)
	}
	if err != nil {
		h.logger.Err(err).Error("failed to get negative transactions")
		negativeTransactions = 0
	}

	var avgPointsPerUser float64
	if totalUsers > 0 {
		avgPointsPerUser = float64(totalPoints) / float64(totalUsers)
	}

	response := map[string]interface{}{
		"totalUsers":           totalUsers,
		"totalPoints":          totalPoints,
		"totalTransactions":    totalTransactions,
		"positiveTransactions": positiveTransactions,
		"negativeTransactions": negativeTransactions,
		"avgPointsPerUser":     avgPointsPerUser,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleAPITopGivers handles requests for top karma givers
func (h *Handler) HandleAPITopGivers(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	topGivers, err := h.db.GetTopGiversCumulative(limit)
	if err != nil {
		h.logger.Err(err).Error("failed to get top givers")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Enrich user data with Slack information
	enrichedTopGivers := h.slack.EnrichUsersWithSlackInfo(topGivers)

	// Filter out bots and deleted users by default (unless show_all=true)
	if r.URL.Query().Get("show_all") != "true" {
		enrichedTopGivers = filterActiveUsers(enrichedTopGivers)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"users": enrichedTopGivers})
}

// HandleAPITopGiversByYear handles requests for top karma givers by year
func (h *Handler) HandleAPITopGiversByYear(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	year, err := strconv.Atoi(vars["year"])
	if err != nil {
		http.Error(w, "Invalid year", http.StatusBadRequest)
		return
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	topGivers, err := h.db.GetTopGiversByYear(limit, year)
	if err != nil {
		h.logger.Err(err).Error("failed to get top givers by year")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Enrich user data with Slack information
	enrichedTopGivers := h.slack.EnrichUsersWithSlackInfo(topGivers)

	// Filter out bots and deleted users by default (unless show_all=true)
	if r.URL.Query().Get("show_all") != "true" {
		enrichedTopGivers = filterActiveUsers(enrichedTopGivers)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"users": enrichedTopGivers})
}

// HandleAPIAvailableYears handles requests for available data years
func (h *Handler) HandleAPIAvailableYears(w http.ResponseWriter, r *http.Request) {
	years, err := h.db.GetAvailableYears()
	if err != nil {
		h.logger.Err(err).Error("failed to get available years")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(years)
}

// HandleAPITopEmojis handles requests for top emoji usage
func (h *Handler) HandleAPITopEmojis(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	year := time.Now().Year()
	if y := r.URL.Query().Get("year"); y != "" {
		if parsed, err := strconv.Atoi(y); err == nil {
			year = parsed
		}
	}

	emojis, err := h.db.GetTopEmojisByYear(year, limit)
	if err != nil {
		h.logger.Err(err).Error("failed to get top emojis")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	totalEmojiUsage, err := h.db.GetTotalEmojiUsageByYear(year)
	if err != nil {
		h.logger.Err(err).Error("failed to get total emoji usage")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	response := make([]map[string]interface{}, len(emojis))
	for i, emoji := range emojis {
		response[i] = map[string]interface{}{
			"emoji_name":        emoji.EmojiName,
			"year":              emoji.Year,
			"usage_count":       emoji.UsageCount,
			"points_awarded":    emoji.PointsAwarded,
			"unique_users":      emoji.UniqueUsers,
			"rank":              emoji.Rank,
			"total_usage_count": totalEmojiUsage,
		}
	}

	json.NewEncoder(w).Encode(response)
}

// HandleAPIKarmaDistribution handles requests for karma point distribution
func (h *Handler) HandleAPIKarmaDistribution(w http.ResponseWriter, r *http.Request) {
	year := 0
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if parsed, err := strconv.Atoi(yearStr); err == nil {
			year = parsed
		}
	}

	// Get leaderboard data
	var leaderboard []*database.UserSummary
	var err error

	if year > 0 {
		leaderboard, err = h.db.GetYearlyLeaderboard(year, 1000) // Get more users for distribution
	} else {
		leaderboard, err = h.db.GetCurrentLeaderboard(1000)
	}

	if err != nil {
		h.logger.Err(err).Error("failed to get leaderboard for karma distribution")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Group users into karma ranges
	distribution := map[string]int{
		"0 to 10":     0,
		"11 to 50":    0,
		"51 to 100":   0,
		"101 to 200":  0,
		"201 to 500":  0,
		"501 to 1000": 0,
		"1000+":       0,
	}

	for _, user := range leaderboard {
		points := user.TotalPoints
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

	// Convert to chart-friendly format
	response := make([]map[string]interface{}, 0, len(distribution))
	for rangeStr, count := range distribution {
		if count > 0 { // Only include ranges with users
			response = append(response, map[string]interface{}{
				"range": rangeStr,
				"count": count,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleAPIActivityTimeline handles requests for activity timeline data
func (h *Handler) HandleAPIActivityTimeline(w http.ResponseWriter, r *http.Request) {
	year := 0
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if parsed, err := strconv.Atoi(yearStr); err == nil {
			year = parsed
		}
	}

	var query string
	var args []interface{}

	query = `
		SELECT
			DATE_TRUNC('month', timestamp) as date,
			SUM(CASE WHEN points > 0 THEN points ELSE 0 END) as positive,
			SUM(CASE WHEN points < 0 THEN ABS(points) ELSE 0 END) as negative,
			COUNT(*) as total
		FROM karma_transactions
		WHERE year = $1
		GROUP BY DATE_TRUNC('month', timestamp) ORDER BY DATE_TRUNC('month', timestamp)
	`
	args = append(args, year)

	rows, err := h.db.SQL.Query(query, args...)
	if err != nil {
		h.logger.Err(err).Error("failed to get activity timeline")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var response []map[string]interface{}
	for rows.Next() {
		var date time.Time
		var positive, negative, total int

		err := rows.Scan(&date, &positive, &negative, &total)
		if err != nil {
			h.logger.Err(err).Error("failed to scan activity timeline row")
			continue
		}

		response = append(response, map[string]interface{}{
			"date":     date.Format("2006-01-02"),
			"positive": positive,
			"negative": negative,
			"total":    total,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleAPIPointsOverTime handles requests for points over time data
func (h *Handler) HandleAPIPointsOverTime(w http.ResponseWriter, r *http.Request) {
	year := 0
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if parsed, err := strconv.Atoi(yearStr); err == nil {
			year = parsed
		}
	}

	var query string
	var args []interface{}

	if year > 0 {
		query = `
			SELECT
				EXTRACT(MONTH FROM timestamp) as month,
				SUM(points) as totalPoints
			FROM karma_transactions
			WHERE EXTRACT(YEAR FROM timestamp) = $1
			GROUP BY EXTRACT(MONTH FROM timestamp)
			ORDER BY month
		`
		args = append(args, year)
	} else {
		query = `
			SELECT
				EXTRACT(MONTH FROM timestamp) as month,
				SUM(points) as totalPoints
			FROM karma_transactions
			GROUP BY EXTRACT(MONTH FROM timestamp)
			ORDER BY month
		`
	}

	rows, err := h.db.SQL.Query(query, args...)
	if err != nil {
		h.logger.Err(err).Error("failed to get points over time")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var response []map[string]interface{}
	for rows.Next() {
		var month int
		var totalPoints int

		err := rows.Scan(&month, &totalPoints)
		if err != nil {
			h.logger.Err(err).Error("failed to scan points over time row")
			continue
		}

		response = append(response, map[string]interface{}{
			"month":       month,
			"totalPoints": totalPoints,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}