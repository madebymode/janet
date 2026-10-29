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
	month, err := parseOptionalMonth(r)
	if err != nil {
		http.Error(w, "Invalid month", http.StatusBadRequest)
		return
	}
	if month > 0 && year == 0 {
		year = time.Now().Year()
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
		Month:    month,
		FromUser: fromUser,
		ToUser:   toUser,
	})
	if err != nil {
		h.logger.Err(err).Error("failed to get recent activity")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Collect unique usernames for batch Slack enrichment
	seen := map[string]*database.UserSummary{}
	for _, tx := range page.Activities {
		if _, ok := seen[tx.FromUser]; !ok {
			seen[tx.FromUser] = &database.UserSummary{Username: tx.FromUser}
		}
		if _, ok := seen[tx.ToUser]; !ok {
			seen[tx.ToUser] = &database.UserSummary{Username: tx.ToUser}
		}
	}
	profileMap := map[string]*database.UserSummary{}
	if h.slack != nil && len(seen) > 0 {
		stubs := make([]*database.UserSummary, 0, len(seen))
		for _, s := range seen {
			stubs = append(stubs, s)
		}
		for _, u := range h.slack.EnrichUsersWithSlackInfo(stubs) {
			profileMap[u.Username] = u
		}
	}

	activities := make([]map[string]interface{}, 0, len(page.Activities))
	for _, tx := range page.Activities {
		item := map[string]interface{}{
			"from":            tx.FromUser,
			"to":              tx.ToUser,
			"points":          tx.Points,
			"reason":          nullableString(tx.Reason),
			"transactionType": nullableString(tx.TransactionType),
			"emojiName":       tx.EmojiName,
			"channelId":       tx.ChannelID,
			"messageId":       tx.MessageID,
			"date":            tx.Timestamp.Format(time.RFC3339),
		}
		if u, ok := profileMap[tx.FromUser]; ok {
			item["fromDisplayName"] = u.DisplayName
			item["fromAvatar"] = u.AvatarURL
		}
		if u, ok := profileMap[tx.ToUser]; ok {
			item["toDisplayName"] = u.DisplayName
			item["toAvatar"] = u.AvatarURL
		}
		activities = append(activities, item)
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
