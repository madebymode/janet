package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	minYear = 2020
	maxYear = 2100
)

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func parseOptionalYear(r *http.Request) (int, error) {
	return parseOptionalYearValue(r.URL.Query().Get("year"))
}

func parseOptionalYearValue(value string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	year, err := strconv.Atoi(value)
	if err != nil || year < minYear || year > max(maxYear, time.Now().Year()+1) {
		return 0, errInvalidYear
	}
	return year, nil
}

func parseRequiredYear(value string) (int, error) {
	year, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || year < minYear || year > max(maxYear, time.Now().Year()+1) {
		return 0, errInvalidYear
	}
	return year, nil
}

func parseIntQuery(r *http.Request, key string, fallback, minValue, maxValue int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if parsed < minValue {
		return minValue
	}
	if parsed > maxValue {
		return maxValue
	}
	return parsed
}

func parseBoolQuery(r *http.Request, key string) bool {
	switch strings.ToLower(strings.TrimSpace(r.URL.Query().Get(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func sanitizeUsernameFilter(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if len(value) > 50 {
		value = value[:50]
	}
	if !isValidUsername(value) {
		return "", errInvalidUsername
	}
	return value, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
