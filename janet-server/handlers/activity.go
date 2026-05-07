package handlers

import (
	"net/http"
	"time"

	"github.com/troyxmccall/janet/database"
)

// HandleAPIRecentActivity handles requests for recent karma activity
func (h *Handler) HandleAPIRecentActivity(w http.ResponseWriter, r *http.Request) {
	limit := parseIntQuery(r, "limit", 20, 1, 100)
	offset := parseIntQuery(r, "offset", 0, 0, 100000)
	year, err := parseOptionalYear(r)
	if err != nil {
		http.Error(w, "Invalid year", http.StatusBadRequest)
		return
	}
	fromUser, err := sanitizeUsernameFilter(r.URL.Query().Get("from"))
	if err != nil {
		http.Error(w, "Invalid from user parameter", http.StatusBadRequest)
		return
	}
	toUser, err := sanitizeUsernameFilter(r.URL.Query().Get("to"))
	if err != nil {
		http.Error(w, "Invalid to user parameter", http.StatusBadRequest)
		return
	}

	page, err := h.db.GetRecentActivityPage(database.RecentActivityFilter{
		Limit:    limit,
		Offset:   offset,
		Year:     year,
		FromUser: fromUser,
		ToUser:   toUser,
	})
	if err != nil {
		h.logger.Err(err).Error("failed to get recent activity")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	activities := make([]map[string]interface{}, 0, len(page.Activities))
	for _, tx := range page.Activities {
		activities = append(activities, map[string]interface{}{
			"from":            tx.FromUser,
			"to":              tx.ToUser,
			"points":          tx.Points,
			"reason":          nullableString(tx.Reason),
			"transactionType": nullableString(tx.TransactionType),
			"emojiName":       tx.EmojiName,
			"channelId":       tx.ChannelID,
			"messageId":       tx.MessageID,
			"date":            tx.Timestamp.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"activities": activities,
		"pagination": map[string]interface{}{
			"limit":       limit,
			"offset":      offset,
			"total":       page.TotalCount,
			"hasMore":     offset+limit < page.TotalCount,
			"currentPage": (offset / limit) + 1,
			"totalPages":  (page.TotalCount + limit - 1) / limit,
		},
	})
}

func nullableString(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}
