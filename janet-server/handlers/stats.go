package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
	"sort"

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

// HandleAPIPopularMessages handles requests for popular messages with reactions
func (h *Handler) HandleAPIPopularMessages(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	year := 0
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if parsed, err := strconv.Atoi(yearStr); err == nil {
			year = parsed
		}
	}

	filterUser := r.URL.Query().Get("user")
	minReactions := 0
	if mr := r.URL.Query().Get("min_reactions"); mr != "" {
		if parsed, err := strconv.Atoi(mr); err == nil {
			minReactions = parsed
		}
	}
	mediaOnly := r.URL.Query().Get("has_media") == "1"
	includeMeta := r.URL.Query().Get("include_meta") == "1"

	// Fetch more messages than requested since we apply filters and backfill state
	// Use a higher cap to make totals more accurate for pagination.
	fetchLimit := (offset + limit) * 10
	minFetch := 500
	if filterUser != "" {
		minFetch = 2000
	}
	if fetchLimit < minFetch {
		fetchLimit = minFetch
	}
	if fetchLimit > 5000 {
		fetchLimit = 5000
	}

	var messages []*database.PopularMessage
	var err error
	if filterUser != "" {
		messages, err = h.db.GetPopularMessagesByUser(fetchLimit, year, filterUser)
	} else {
		messages, err = h.db.GetPopularMessages(fetchLimit, year)
	}
	if err != nil {
		h.logger.Err(err).Error("failed to get popular messages")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// For each message, get the Slack permalink and message text
	type popularEntry struct {
		data          map[string]interface{}
		reactionCount int
		totalPoints   int
	}
	entries := make([]popularEntry, 0, len(messages))
	skippedMissing := 0
	enqueuedBackfill := 0
	skippedReplies := 0
	skippedTestChannels := 0
	skippedIgnored := 0
	for _, msg := range messages {
		msgData := map[string]interface{}{
			"channel_id":     msg.ChannelID,
			"message_id":     msg.MessageID,
			"reaction_count": msg.ReactionCount,
			"total_points":   msg.TotalPoints,
		}

		// Get channel_id if missing (fallback to search)
		channelID := ""
		if msg.ChannelID != nil && *msg.ChannelID != "" {
			channelID = *msg.ChannelID
		}

		// Check cached message details first to avoid Slack API calls
		hasText := false
		hasPermalink := false
		hasAuthor := false
		imageKnown := false
		attachmentKnown := false
		reactionKnown := false
		detailsFetched := false
		detailsFetchedKnown := false
		if cached, err := h.db.GetPopularMessageDetails(msg.MessageID); err == nil && cached != nil {
			if cached.ChannelID != nil && channelID == "" {
				channelID = *cached.ChannelID
				msgData["channel_id"] = cached.ChannelID
			}
			if cached.Text != nil && *cached.Text != "" {
				msgData["text"] = *cached.Text
				hasText = true
			}
			if cached.Permalink != nil && *cached.Permalink != "" {
				msgData["permalink"] = *cached.Permalink
				hasPermalink = true
			}
			if cached.AuthorName != nil && *cached.AuthorName != "" {
				msgData["author_name"] = *cached.AuthorName
				hasAuthor = true
			}
			if cached.AuthorAvatar != nil && *cached.AuthorAvatar != "" {
				msgData["author_avatar"] = *cached.AuthorAvatar
				hasAuthor = true
			}
			if cached.ImageURL != nil {
				imageKnown = true
				if *cached.ImageURL != "" {
					msgData["image_url"] = *cached.ImageURL
				}
			}
			if cached.AttachmentURL != nil && *cached.AttachmentURL != "" {
				msgData["attachment_url"] = *cached.AttachmentURL
			}
			if cached.AttachmentURL != nil {
				attachmentKnown = true
				if cached.AttachmentMime != nil && *cached.AttachmentMime != "" {
					msgData["attachment_mime"] = *cached.AttachmentMime
				}
			}
			if cached.ImageURL == nil && cached.AttachmentURL == nil && cached.AttachmentMime == nil {
				imageKnown = true
				attachmentKnown = true
			}
			if cached.ReactionCount != nil {
				reactionKnown = true
				msgData["reaction_count"] = *cached.ReactionCount
			}
			if cached.IsReply != nil && *cached.IsReply {
				skippedReplies++
				continue
			}
			if cached.IsIgnored != nil && *cached.IsIgnored {
				skippedIgnored++
				continue
			}
			if cached.DetailsFetched != nil {
				detailsFetchedKnown = true
				detailsFetched = *cached.DetailsFetched
			}
		} else if err != nil {
			h.logger.Err(err).KV("message_id", msg.MessageID).Error("failed to read popular message cache")
		}

		if channelID == "" {
			if storedChannelID, err := h.db.GetChannelIDForMessage(msg.MessageID); err == nil && storedChannelID != nil {
				channelID = *storedChannelID
				msgData["channel_id"] = storedChannelID
			} else if err != nil {
				h.logger.Err(err).KV("message_id", msg.MessageID).Error("failed to resolve channel_id from transactions")
			}
		}

		if strings.HasPrefix(channelID, "TEST") {
			skippedTestChannels++
			continue
		}

		if !hasAuthor {
			if derivedAuthor, err := h.db.GetMessageAuthorByMessageID(msg.MessageID); err == nil && derivedAuthor != nil {
				msgData["author_name"] = *derivedAuthor
				hasAuthor = true
				if err := h.db.UpsertPopularMessageDetails(msg.MessageID, nil, nil, nil, nil, derivedAuthor, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
					h.logger.Err(err).KV("message_id", msg.MessageID).Error("failed to cache derived author")
				}
			} else if err != nil {
				h.logger.Err(err).KV("message_id", msg.MessageID).Error("failed to derive author from transactions")
			}
		}

		if channelID != "" {
			cachedChannelID := channelID
			if err := h.db.UpsertPopularMessageDetails(msg.MessageID, &cachedChannelID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
				h.logger.Err(err).KV("message_id", msg.MessageID).Error("failed to update popular message cache")
			}
		}

		completeDetails := hasText && hasPermalink && hasAuthor && imageKnown && attachmentKnown && reactionKnown
		if !detailsFetchedKnown && completeDetails {
			detailsFetched = true
			detailsFetchedKnown = true
			if err := h.db.UpsertPopularMessageDetails(msg.MessageID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &detailsFetched); err != nil {
				h.logger.Err(err).KV("message_id", msg.MessageID).Error("failed to mark popular message details fetched")
			}
		}
		missingDetails := !detailsFetchedKnown || !detailsFetched
		if missingDetails {
			h.enqueuePopularMessageBackfill(msg.MessageID, channelID)
			enqueuedBackfill++
			skippedMissing++
			if position := h.popularBackfillQueuePosition(msg.MessageID); position > 0 {
				msgData["queue_position"] = position
			}
		}
		msgData["pending_details"] = missingDetails
		reactionCount := msg.ReactionCount
		if cachedCount, ok := msgData["reaction_count"].(int); ok {
			reactionCount = cachedCount
		}
		if filterUser != "" {
			author, _ := msgData["author_name"].(string)
			if author != filterUser {
				continue
			}
		}
		if minReactions > 0 && reactionCount < minReactions {
			continue
		}
		if mediaOnly {
			_, hasImage := msgData["image_url"]
			_, hasAttachment := msgData["attachment_url"]
			if !hasImage && !hasAttachment {
				continue
			}
		}

		entries = append(entries, popularEntry{
			data:          msgData,
			reactionCount: reactionCount,
			totalPoints:   msg.TotalPoints,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].reactionCount == entries[j].reactionCount {
			return entries[i].totalPoints > entries[j].totalPoints
		}
		return entries[i].reactionCount > entries[j].reactionCount
	})
	total := len(entries)
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	sliced := entries[start:end]

	response := make([]map[string]interface{}, 0, len(sliced))
	for _, entry := range sliced {
		response = append(response, entry.data)
	}

	queuePosition := 0
	for _, entry := range sliced {
		pending, _ := entry.data["pending_details"].(bool)
		if !pending {
			continue
		}
		position, _ := entry.data["queue_position"].(int)
		if position == 0 {
			messageID, _ := entry.data["message_id"].(string)
			if messageID == "" {
				continue
			}
			position = h.popularBackfillQueuePosition(messageID)
		}
		if position == 0 {
			continue
		}
		if queuePosition == 0 || position < queuePosition {
			queuePosition = position
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if includeMeta {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items":   response,
			"total":   total,
			"limit":   limit,
			"offset":  offset,
			"pending": skippedMissing,
			"queue_size": h.popularBackfillQueueSize(),
			"queue_position": queuePosition,
		})
	} else {
		json.NewEncoder(w).Encode(response)
	}
	logEvent := h.logger.KV("returned", len(response)).KV("requested", limit).KV("total", total).KV("offset", offset).KV("skipped_missing", skippedMissing).KV("skipped_replies", skippedReplies).KV("skipped_ignored", skippedIgnored).KV("skipped_test_channels", skippedTestChannels).KV("backfill_enqueued", enqueuedBackfill)
	if failures := h.popPopularBackfillFailures(); failures != nil {
		logEvent = logEvent.KV("backfill_failures", failures)
	}
	logEvent.Info("popular messages response")
}
