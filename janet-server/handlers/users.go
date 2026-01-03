package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/troyxmccall/janet/database"
)

// HandleAPIUser handles requests for individual user information
func (h *Handler) HandleAPIUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]

	// Validate username input
	if !isValidUsername(username) {
		http.Error(w, "Invalid username", http.StatusBadRequest)
		return
	}

	summary, err := h.db.GetUser(username)
	if err != nil {
		if err == database.ErrNoSuchUser {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			h.logger.Err(err).Error("failed to get user")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// HandleAPIUserByYear handles requests for user information for a specific year
func (h *Handler) HandleAPIUserByYear(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]
	year := vars["year"]

	// Validate username input
	if !isValidUsername(username) {
		http.Error(w, "Invalid username", http.StatusBadRequest)
		return
	}

	yearInt, err := strconv.Atoi(year)
	if err != nil || yearInt < 2020 || yearInt > 2030 {
		http.Error(w, "Invalid year", http.StatusBadRequest)
		return
	}

	user, err := h.db.GetUserByYear(username, yearInt)
	if err != nil {
		if err == database.ErrNoSuchUser {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			h.logger.Err(err).Error("failed to get user by year")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// HandleAPIUserPointsOverTime handles requests for user points over time for a specific year
func (h *Handler) HandleAPIUserPointsOverTime(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]
	year := vars["year"]

	// Validate username input
	if !isValidUsername(username) {
		http.Error(w, "Invalid username", http.StatusBadRequest)
		return
	}

	yearInt, err := strconv.Atoi(year)
	if err != nil || yearInt < 2020 || yearInt > 2030 {
		http.Error(w, "Invalid year", http.StatusBadRequest)
		return
	}

	// Query user_summary_monthly table for user-specific data
	query := `
		SELECT month, total_points
		FROM user_summary_monthly
		WHERE username = $1 AND year = $2
		ORDER BY month
	`

	rows, err := h.db.SQL.Query(query, username, yearInt)
	if err != nil {
		h.logger.Err(err).Error("failed to query user points over time")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type MonthlyPoints struct {
		Month       int `json:"month"`
		TotalPoints int `json:"totalPoints"`
	}

	var results []MonthlyPoints
	for rows.Next() {
		var mp MonthlyPoints
		if err := rows.Scan(&mp.Month, &mp.TotalPoints); err != nil {
			h.logger.Err(err).Error("failed to scan monthly points")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		results = append(results, mp)
	}

	if err := rows.Err(); err != nil {
		h.logger.Err(err).Error("row iteration error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// HandleAPIUserAllTimePointsOverTime handles requests for user points over time across all years
func (h *Handler) HandleAPIUserAllTimePointsOverTime(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]

	// Validate username input
	if !isValidUsername(username) {
		http.Error(w, "Invalid username", http.StatusBadRequest)
		return
	}

	// Query user-specific yearly data for all-time view
	query := `
		SELECT 
			EXTRACT(YEAR FROM timestamp) as year,
			SUM(points) as totalPoints
		FROM karma_transactions
		WHERE to_user = $1
		GROUP BY EXTRACT(YEAR FROM timestamp)
		ORDER BY year
	`

	rows, err := h.db.SQL.Query(query, username)
	if err != nil {
		h.logger.Err(err).Error("failed to get user all-time points over time")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type YearlyPoints struct {
		Year        int `json:"year"`
		TotalPoints int `json:"totalPoints"`
	}

	var results []YearlyPoints
	for rows.Next() {
		var yp YearlyPoints
		err := rows.Scan(&yp.Year, &yp.TotalPoints)
		if err != nil {
			h.logger.Err(err).Error("failed to scan yearly points")
			continue
		}
		results = append(results, yp)
	}

	if err := rows.Err(); err != nil {
		h.logger.Err(err).Error("row iteration error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}