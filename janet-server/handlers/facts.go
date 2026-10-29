package handlers

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/troyxmccall/janet/database"
)

func timeNowYear() int { return time.Now().Year() }

// HandleAPIKarmaColluders returns the top mutual karma-exchange pairs.
func (h *Handler) HandleAPIKarmaColluders(w http.ResponseWriter, r *http.Request) {
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
		year = timeNowYear()
	}
	limit := parseIntQuery(r, "limit", 10, 1, 50)

	var colluders []*database.KarmaColluder
	if month > 0 {
		colluders, err = h.db.GetKarmaColludersByYearMonth(limit, year, month)
	} else {
		colluders, err = h.db.GetKarmaColluders(limit, year)
	}
	if err != nil {
		h.logger.Err(err).Error("failed to get karma colluders")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Enrich both users in each pair with Slack profile info
	enriched := h.enrichColluders(colluders)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"colluders": enriched,
	})
}

// HandleAPIFunFacts returns computed interesting stats.
func (h *Handler) HandleAPIFunFacts(w http.ResponseWriter, r *http.Request) {
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
		year = timeNowYear()
	}

	var facts *database.FunFacts
	if month > 0 {
		facts, err = h.db.GetFunFactsByYearMonth(year, month)
	} else {
		facts, err = h.db.GetFunFacts(year)
	}
	if err != nil {
		h.logger.Err(err).Error("failed to get fun facts")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, facts)
}

// HandleAPIUserColluders returns colluder pairs involving a specific user.
func (h *Handler) HandleAPIUserColluders(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	if !isValidUsername(username) {
		http.Error(w, "Invalid username", http.StatusBadRequest)
		return
	}
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
		year = timeNowYear()
	}
	limit := parseIntQuery(r, "limit", 10, 1, 50)

	colluders, err := h.db.GetUserColluders(username, limit, year, month)
	if err != nil {
		h.logger.Err(err).Error("failed to get user colluders")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"colluders": h.enrichColluders(colluders),
	})
}

// HandleAPIUserTopChannels returns the most active channels for a specific user.
func (h *Handler) HandleAPIUserTopChannels(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	if !isValidUsername(username) {
		http.Error(w, "Invalid username", http.StatusBadRequest)
		return
	}
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
		year = timeNowYear()
	}
	limit := parseIntQuery(r, "limit", 8, 1, 20)

	channels, err := h.db.GetUserTopChannels(username, year, month, limit)
	if err != nil {
		h.logger.Err(err).Error("failed to get user top channels")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"channels": channels,
	})
}

// enrichColluders populates display names and avatars using the Slack service.
func (h *Handler) enrichColluders(colluders []*database.KarmaColluder) []*database.KarmaColluder {
	if h.slack == nil || len(colluders) == 0 {
		return colluders
	}

	// Collect unique usernames into UserSummary stubs for batch enrichment
	seen := map[string]*database.UserSummary{}
	for _, c := range colluders {
		if _, ok := seen[c.User1]; !ok {
			seen[c.User1] = &database.UserSummary{Username: c.User1}
		}
		if _, ok := seen[c.User2]; !ok {
			seen[c.User2] = &database.UserSummary{Username: c.User2}
		}
	}
	stubs := make([]*database.UserSummary, 0, len(seen))
	for _, s := range seen {
		stubs = append(stubs, s)
	}
	enriched := h.slack.EnrichUsersWithSlackInfo(stubs)
	enrichedMap := make(map[string]*database.UserSummary, len(enriched))
	for _, u := range enriched {
		enrichedMap[u.Username] = u
	}

	for _, c := range colluders {
		if u, ok := enrichedMap[c.User1]; ok {
			c.User1DisplayName = u.DisplayName
			c.User1Avatar = u.AvatarURL
		}
		if u, ok := enrichedMap[c.User2]; ok {
			c.User2DisplayName = u.DisplayName
			c.User2Avatar = u.AvatarURL
		}
	}
	return colluders
}
