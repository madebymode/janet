package handlers

import (
	"net/http"

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

	writeJSON(w, http.StatusOK, summary)
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

	yearInt, err := parseRequiredYear(year)
	if err != nil {
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

	writeJSON(w, http.StatusOK, user)
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

	yearInt, err := parseRequiredYear(year)
	if err != nil {
		http.Error(w, "Invalid year", http.StatusBadRequest)
		return
	}

	results, err := h.db.GetUserMonthlyPointsByYear(username, yearInt)
	if err != nil {
		h.logger.Err(err).Error("failed to get user points over time")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, results)
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

	results, err := h.db.GetUserYearlyPoints(username)
	if err != nil {
		h.logger.Err(err).Error("failed to get user all-time points over time")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, results)
}
