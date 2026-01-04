package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/troyxmccall/janet/database"
)

// HandleAPILeaderboard handles requests for the general leaderboard
func (h *Handler) HandleAPILeaderboard(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	var leaderboard []*database.UserSummary
	var err error

	// Check for year query parameter
	yearParam := r.URL.Query().Get("year")
	if yearParam != "" {
		// Year specified, get year-specific leaderboard
		yearInt, parseErr := strconv.Atoi(yearParam)
		if parseErr != nil || yearInt < 2020 || yearInt > 2030 {
			http.Error(w, "Invalid year", http.StatusBadRequest)
			return
		}
		leaderboard, err = h.db.GetYearlyLeaderboard(yearInt, limit)
		if err != nil {
			h.logger.Err(err).Error("failed to get yearly leaderboard")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	} else {
		// No year specified, get all-time leaderboard
		leaderboard, err = h.db.GetCurrentLeaderboard(limit)
		if err != nil {
			h.logger.Err(err).Error("failed to get leaderboard")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	// Only keep users who have given points (sanity check for legit usernames).
	filteredLeaderboard := make([]*database.UserSummary, 0, len(leaderboard))
	for _, user := range leaderboard {
		if user.TransactionsGiven > 0 {
			filteredLeaderboard = append(filteredLeaderboard, user)
		}
	}

	// Enrich user data with Slack information
	enrichedLeaderboard := h.slack.EnrichUsersWithSlackInfo(filteredLeaderboard)

	// Filter out bots and deleted users by default (unless show_all=true)
	if r.URL.Query().Get("show_all") != "true" {
		enrichedLeaderboard = filterActiveUsers(enrichedLeaderboard)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"users": enrichedLeaderboard})
}

// HandleAPIYearlyLeaderboard handles requests for yearly leaderboards
func (h *Handler) HandleAPIYearlyLeaderboard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	year := vars["year"]

	yearInt, err := strconv.Atoi(year)
	if err != nil || yearInt < 2020 || yearInt > 2030 {
		http.Error(w, "Invalid year", http.StatusBadRequest)
		return
	}

	limit := 25
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	leaderboard, err := h.db.GetYearlyLeaderboard(yearInt, limit)
	if err != nil {
		h.logger.Err(err).Error("failed to get yearly leaderboard")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Enrich user data with Slack information
	enrichedLeaderboard := h.slack.EnrichUsersWithSlackInfo(leaderboard)

	// Filter out bots and deleted users by default (unless show_all=true)
	if r.URL.Query().Get("show_all") != "true" {
		enrichedLeaderboard = filterActiveUsers(enrichedLeaderboard)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"users": enrichedLeaderboard})
}

// HandleAPICurrentLeaderboard handles requests for current year vs all-time leaderboards
func (h *Handler) HandleAPICurrentLeaderboard(w http.ResponseWriter, r *http.Request) {
	limit := 25
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	currentYear := time.Now().Year()

	// Get current year leaderboard
	currentLeaderboard, err := h.db.GetYearlyLeaderboard(currentYear, limit)
	if err != nil {
		h.logger.Err(err).Error("failed to get current year leaderboard")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get all-time leaderboard
	allTimeLeaderboard, err := h.db.GetCurrentLeaderboard(limit)
	if err != nil {
		h.logger.Err(err).Error("failed to get all-time leaderboard")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Enrich both leaderboards with Slack information
	enrichedCurrentLeaderboard := h.slack.EnrichUsersWithSlackInfo(currentLeaderboard)
	enrichedAllTimeLeaderboard := h.slack.EnrichUsersWithSlackInfo(allTimeLeaderboard)

	// Filter out bots and deleted users by default (unless show_all=true)
	if r.URL.Query().Get("show_all") != "true" {
		enrichedCurrentLeaderboard = filterActiveUsers(enrichedCurrentLeaderboard)
		enrichedAllTimeLeaderboard = filterActiveUsers(enrichedAllTimeLeaderboard)
	}

	response := map[string]interface{}{
		"current": enrichedCurrentLeaderboard,
		"allTime": enrichedAllTimeLeaderboard,
		"year":    currentYear,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
