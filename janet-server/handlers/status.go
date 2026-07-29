package handlers

import "net/http"

// HandleAPIStatus handles requests for system status
func (h *Handler) HandleAPIStatus(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"botOnline": h.slack.CheckBotHealth(),
	}

	writeJSON(w, http.StatusOK, response)
}
