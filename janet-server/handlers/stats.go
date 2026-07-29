package handlers

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// HandleAPIStatsV2 handles requests for basic statistics (V2 API)
func (h *Handler) HandleAPIStatsV2(w http.ResponseWriter, r *http.Request) {
	year, err := parseOptionalYear(r)
	if err != nil {
		http.Error(w, "Invalid year", http.StatusBadRequest)
		return
	}

	var totalPoints int
	if year == 0 {
		totalPoints, err = h.db.GetTotalPointsCumulative()
	} else {
		totalPoints, err = h.db.GetTotalPointsByYear(year)
	}
	if err != nil {
		h.logger.Err(err).KV("year", year).Error("failed to get total points")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var totalUsers int
	if year == 0 {
		totalUsers, err = h.db.GetTotalUsersCumulative()
	} else {
		totalUsers, err = h.db.GetTotalUsersByYear(year)
	}
	if err != nil {
		h.logger.Err(err).KV("year", year).Error("failed to get total users")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var totalTransactions int
	if year == 0 {
		totalTransactions, err = h.db.GetTotalTransactionsCumulative()
	} else {
		totalTransactions, err = h.db.GetTotalTransactionsByYear(year)
	}
	if err != nil {
		h.logger.Err(err).KV("year", year).Error("failed to get total transactions")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"totalUsers":        totalUsers,
		"totalPoints":       totalPoints,
		"totalTransactions": totalTransactions,
	}

	writeJSON(w, http.StatusOK, response)
}

// HandleAPIStatsDetailed handles requests for detailed statistics
func (h *Handler) HandleAPIStatsDetailed(w http.ResponseWriter, r *http.Request) {
	year, err := parseOptionalYear(r)
	if err != nil {
		http.Error(w, "Invalid year", http.StatusBadRequest)
		return
	}

	var totalUsers int
	if year == 0 {
		totalUsers, err = h.db.GetTotalUsersCumulative()
	} else {
		totalUsers, err = h.db.GetTotalUsersByYear(year)
	}
	if err != nil {
		h.logger.Err(err).KV("year", year).Error("failed to get total users")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var totalPoints int
	if year == 0 {
		totalPoints, err = h.db.GetTotalPointsCumulative()
	} else {
		totalPoints, err = h.db.GetTotalPointsByYear(year)
	}
	if err != nil {
		h.logger.Err(err).KV("year", year).Error("failed to get total points")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var totalTransactions int
	if year == 0 {
		totalTransactions, err = h.db.GetTotalTransactionsCumulative()
	} else {
		totalTransactions, err = h.db.GetTotalTransactionsByYear(year)
	}
	if err != nil {
		h.logger.Err(err).KV("year", year).Error("failed to get total transactions")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var positiveTransactions int
	if year == 0 {
		positiveTransactions, err = h.db.GetPositiveTransactionsCumulative()
	} else {
		positiveTransactions, err = h.db.GetPositiveTransactionsByYear(year)
	}
	if err != nil {
		h.logger.Err(err).KV("year", year).Error("failed to get positive transactions")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var negativeTransactions int
	if year == 0 {
		negativeTransactions, err = h.db.GetNegativeTransactionsCumulative()
	} else {
		negativeTransactions, err = h.db.GetNegativeTransactionsByYear(year)
	}
	if err != nil {
		h.logger.Err(err).KV("year", year).Error("failed to get negative transactions")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
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

	writeJSON(w, http.StatusOK, response)
}

// HandleAPITopGivers handles requests for top karma givers
func (h *Handler) HandleAPITopGivers(w http.ResponseWriter, r *http.Request) {
	limit := parseIntQuery(r, "limit", 10, 1, 100)

	topGivers, err := h.db.GetTopGiversCumulative(limit)
	if err != nil {
		h.logger.Err(err).Error("failed to get top givers")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Enrich user data with Slack information
	enrichedTopGivers := h.slack.EnrichUsersWithSlackInfo(topGivers)

	// Filter out bots and deleted users by default (unless show_all=true)
	if !parseBoolQuery(r, "show_all") {
		enrichedTopGivers = filterActiveUsers(enrichedTopGivers)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"users": enrichedTopGivers})
}

// HandleAPITopGiversByYear handles requests for top karma givers by year
func (h *Handler) HandleAPITopGiversByYear(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	year, err := parseRequiredYear(vars["year"])
	if err != nil {
		http.Error(w, "Invalid year", http.StatusBadRequest)
		return
	}

	limit := parseIntQuery(r, "limit", 10, 1, 100)

	topGivers, err := h.db.GetTopGiversByYear(limit, year)
	if err != nil {
		h.logger.Err(err).Error("failed to get top givers by year")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Enrich user data with Slack information
	enrichedTopGivers := h.slack.EnrichUsersWithSlackInfo(topGivers)

	// Filter out bots and deleted users by default (unless show_all=true)
	if !parseBoolQuery(r, "show_all") {
		enrichedTopGivers = filterActiveUsers(enrichedTopGivers)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"users": enrichedTopGivers})
}

// HandleAPIAvailableYears handles requests for available data years
func (h *Handler) HandleAPIAvailableYears(w http.ResponseWriter, r *http.Request) {
	years, err := h.db.GetAvailableYears()
	if err != nil {
		h.logger.Err(err).Error("failed to get available years")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, years)
}

// HandleAPITopEmojis handles requests for top emoji usage
func (h *Handler) HandleAPITopEmojis(w http.ResponseWriter, r *http.Request) {
	limit := parseIntQuery(r, "limit", 10, 1, 100)
	year, err := parseOptionalYear(r)
	if err != nil {
		http.Error(w, "Invalid year", http.StatusBadRequest)
		return
	}
	if year == 0 {
		year = time.Now().Year()
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

	writeJSON(w, http.StatusOK, response)
}

// HandleAPIKarmaDistribution handles requests for karma point distribution
func (h *Handler) HandleAPIKarmaDistribution(w http.ResponseWriter, r *http.Request) {
	year, err := parseOptionalYear(r)
	if err != nil {
		http.Error(w, "Invalid year", http.StatusBadRequest)
		return
	}

	response, err := h.db.GetKarmaDistributionByYear(year)
	if err != nil {
		h.logger.Err(err).Error("failed to get karma distribution")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// HandleAPIActivityTimeline handles requests for activity timeline data
func (h *Handler) HandleAPIActivityTimeline(w http.ResponseWriter, r *http.Request) {
	year, err := parseOptionalYear(r)
	if err != nil {
		http.Error(w, "Invalid year", http.StatusBadRequest)
		return
	}

	response, err := h.db.GetActivityTimelineByYear(year)
	if err != nil {
		h.logger.Err(err).Error("failed to get activity timeline")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// HandleAPIPointsOverTime handles requests for points over time data
func (h *Handler) HandleAPIPointsOverTime(w http.ResponseWriter, r *http.Request) {
	year, err := parseOptionalYear(r)
	if err != nil {
		http.Error(w, "Invalid year", http.StatusBadRequest)
		return
	}

	response, err := h.db.GetPointsOverTimeMonthlyByYear(year)
	if err != nil {
		h.logger.Err(err).Error("failed to get points over time")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// HandleAPIPopularMessages handles requests for popular messages with reactions
func (h *Handler) HandleAPIPopularMessages(w http.ResponseWriter, r *http.Request) {
	query, err := parsePopularMessagesQuery(r)
	if err != nil {
		if err == errInvalidYear {
			http.Error(w, "Invalid year", http.StatusBadRequest)
			return
		}
		http.Error(w, "Invalid user parameter", http.StatusBadRequest)
		return
	}

	result, err := h.buildPopularMessagesResult(*query)
	if err != nil {
		h.logger.Err(err).Error("failed to get popular messages")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if query.IncludeMeta {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"items":          result.Items,
			"total":          result.Total,
			"limit":          query.Limit,
			"offset":         query.Offset,
			"pending":        result.Pending,
			"queue_size":     h.popularBackfillQueueSize(),
			"queue_position": result.QueuePosition,
			"funny_bias":     query.FunnyBias,
		})
	} else {
		writeJSON(w, http.StatusOK, result.Items)
	}
	logEvent := h.logger.KV("returned", len(result.Items)).KV("requested", query.Limit).KV("total", result.Total).KV("offset", query.Offset).KV("skipped_missing", result.Pending).KV("skipped_replies", result.SkippedReplies).KV("skipped_ignored", result.SkippedIgnored).KV("skipped_test_channels", result.SkippedTestChannels).KV("backfill_enqueued", result.BackfillEnqueued)
	if failures := h.popPopularBackfillFailures(); failures != nil {
		logEvent = logEvent.KV("backfill_failures", failures)
	}
	logEvent.Info("popular messages response")
}
