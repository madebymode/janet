package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// HandleAPIRecentActivity handles requests for recent karma activity
func (h *Handler) HandleAPIRecentActivity(w http.ResponseWriter, r *http.Request) {
	// Parse and validate query parameters
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
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
		if parsed, err := strconv.Atoi(yearStr); err == nil && parsed >= 2020 && parsed <= 2030 {
			year = parsed
		}
	}

	// Input validation and sanitization for user search terms
	fromUser := strings.TrimSpace(r.URL.Query().Get("from"))
	toUser := strings.TrimSpace(r.URL.Query().Get("to"))
	
	// Limit search term length and validate characters
	if len(fromUser) > 50 {
		fromUser = fromUser[:50]
	}
	if len(toUser) > 50 {
		toUser = toUser[:50]
	}
	
	// Basic validation - only allow alphanumeric, underscore, hyphen, and dot
	validUserRegex := `^[a-zA-Z0-9._-]*$`
	if fromUser != "" {
		if match, _ := regexp.MatchString(validUserRegex, fromUser); !match {
			http.Error(w, "Invalid from user parameter", http.StatusBadRequest)
			return
		}
	}
	if toUser != "" {
		if match, _ := regexp.MatchString(validUserRegex, toUser); !match {
			http.Error(w, "Invalid to user parameter", http.StatusBadRequest)
			return
		}
	}

	// Build query using proper parameterized queries
	var queryParts []string
	var args []interface{}
	
	baseQuery := `
		SELECT from_user, to_user, points, reason, transaction_type, emoji_name, 
		       channel_id, message_id, timestamp
		FROM karma_transactions
	`
	
	queryParts = append(queryParts, baseQuery)
	
	var conditions []string
	argIndex := 0
	
	if year > 0 {
		argIndex++
		conditions = append(conditions, "EXTRACT(YEAR FROM timestamp) = $"+strconv.Itoa(argIndex))
		args = append(args, year)
	}
	
	if fromUser != "" {
		argIndex++
		conditions = append(conditions, "from_user ILIKE $"+strconv.Itoa(argIndex))
		args = append(args, "%"+fromUser+"%")
	}
	
	if toUser != "" {
		argIndex++
		conditions = append(conditions, "to_user ILIKE $"+strconv.Itoa(argIndex))
		args = append(args, "%"+toUser+"%")
	}
	
	if len(conditions) > 0 {
		queryParts = append(queryParts, " WHERE ", strings.Join(conditions, " AND "))
	}
	
	// Add ordering and pagination with parameterized queries
	argIndex++
	queryParts = append(queryParts, " ORDER BY timestamp DESC LIMIT $"+strconv.Itoa(argIndex))
	args = append(args, limit)
	
	argIndex++
	queryParts = append(queryParts, " OFFSET $"+strconv.Itoa(argIndex))
	args = append(args, offset)

	query := strings.Join(queryParts, "")

	// Execute query
	rows, err := h.db.SQL.Query(query, args...)
	if err != nil {
		h.logger.Err(err).KV("query", query).Error("failed to get recent activity")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var activities []map[string]interface{}
	for rows.Next() {
		var fromUserResult, toUserResult, reason, transactionType, emojiName, channelID, messageID string
		var points int
		var timestamp time.Time

		err := rows.Scan(&fromUserResult, &toUserResult, &points, &reason, &transactionType, 
			&emojiName, &channelID, &messageID, &timestamp)
		if err != nil {
			h.logger.Err(err).Error("failed to scan activity row")
			continue
		}

		activities = append(activities, map[string]interface{}{
			"from":            fromUserResult,
			"to":              toUserResult,
			"points":          points,
			"reason":          reason,
			"transactionType": transactionType,
			"emojiName":       emojiName,
			"channelId":       channelID,
			"messageId":       messageID,
			"date":            timestamp.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	// Get total count for pagination using parameterized query
	var countQueryParts []string
	var countArgs []interface{}
	
	countQueryParts = append(countQueryParts, "SELECT COUNT(*) FROM karma_transactions")
	
	countArgIndex := 0
	var countConditions []string
	
	if year > 0 {
		countArgIndex++
		countConditions = append(countConditions, "EXTRACT(YEAR FROM timestamp) = $"+strconv.Itoa(countArgIndex))
		countArgs = append(countArgs, year)
	}
	
	if fromUser != "" {
		countArgIndex++
		countConditions = append(countConditions, "from_user ILIKE $"+strconv.Itoa(countArgIndex))
		countArgs = append(countArgs, "%"+fromUser+"%")
	}
	
	if toUser != "" {
		countArgIndex++
		countConditions = append(countConditions, "to_user ILIKE $"+strconv.Itoa(countArgIndex))
		countArgs = append(countArgs, "%"+toUser+"%")
	}
	
	if len(countConditions) > 0 {
		countQueryParts = append(countQueryParts, " WHERE ", strings.Join(countConditions, " AND "))
	}

	countQuery := strings.Join(countQueryParts, "")

	var totalCount int
	err = h.db.SQL.QueryRow(countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		h.logger.Err(err).KV("countQuery", countQuery).Error("failed to get activity count")
		totalCount = 0
	}

	// Prepare response with pagination info
	response := map[string]interface{}{
		"activities": activities,
		"pagination": map[string]interface{}{
			"limit":       limit,
			"offset":      offset,
			"total":       totalCount,
			"hasMore":     offset+limit < totalCount,
			"currentPage": (offset / limit) + 1,
			"totalPages":  (totalCount + limit - 1) / limit,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}