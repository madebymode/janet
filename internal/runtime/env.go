package runtime

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

func GetEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func ParseBool(value string, fallback bool) bool {
	if value == "" {
		return fallback
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func ParseInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func ParseFloat(value string, fallback float64) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func ParseCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	results := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		results = append(results, trimmed)
	}
	return results
}

func ParseStringMap(value string) map[string]string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parsed := make(map[string]string)
	if err := json.Unmarshal([]byte(value), &parsed); err == nil {
		return parsed
	}

	results := make(map[string]string)
	for _, part := range strings.Split(value, ",") {
		pieces := strings.SplitN(part, "=", 2)
		if len(pieces) != 2 {
			continue
		}
		key := strings.TrimSpace(pieces[0])
		val := strings.TrimSpace(pieces[1])
		if key == "" || val == "" {
			continue
		}
		results[key] = val
	}
	return results
}
