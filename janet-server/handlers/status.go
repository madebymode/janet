package handlers

import (
	"encoding/json"
	"net/http"
)

// HandleAPIStatus handles requests for system status
func (h *Handler) HandleAPIStatus(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"botOnline": h.slack.CheckBotHealth(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
