package server

import (
	"encoding/json"
	"os"
	"strings"
)

// Config holds the application configuration
type Config struct {
	// Database
	DatabaseDriver string `json:"databaseDriver"`
	DatabaseURL    string `json:"databaseURL"`

	// Slack
	SlackToken          string `json:"slackToken"`
	SlackSocketToken    string `json:"slackSocketToken"`
	GoodPlaceJudgeBotID string `json:"goodPlaceJudgeBotID"`

	// Bot behavior
	MaxPoints        int    `json:"maxPoints"`
	LeaderboardLimit int    `json:"leaderboardLimit"`
	ReplyType        string `json:"replyType"`
	Debug            bool   `json:"debug"`
	SelfKarma        bool   `json:"selfKarma"`
	Motivate         bool   `json:"motivate"`
	ReactjiEnabled   bool   `json:"reactjiEnabled"`

	// Web server
	WebListenAddr string `json:"webListenAddr"`
	WebPublicURL  string `json:"webPublicURL"`
	BotEnabled    bool   `json:"botEnabled"`
	AttachmentsDir string `json:"attachmentsDir"`
	RunChannelIDBackfill bool `json:"runChannelIDBackfill"`

	// Bot personalities
	GoodJanetUsername string `json:"goodJanetUsername"`
	GoodJanetIconURL  string `json:"goodJanetIconURL"`
	BadJanetUsername  string `json:"badJanetUsername"`
	BadJanetIconURL   string `json:"badJanetIconURL"`

	// User management
	UserBlacklist []string          `json:"userBlacklist"`
	UserAliases   map[string]string `json:"userAliases"`
}

// defaultConfig returns default configuration values
func defaultConfig() *Config {
	return &Config{
		DatabaseDriver:    "postgres",
		DatabaseURL:       "postgres://janet:janet@localhost:5432/janet?sslmode=disable",
		MaxPoints:         5,
		LeaderboardLimit:  10,
		ReplyType:         "thread",
		Debug:             false,
		SelfKarma:         false,
		Motivate:          true,
		ReactjiEnabled:    true,
		WebListenAddr:     ":8080",
		WebPublicURL:      "http://localhost:8080",
		BotEnabled:        true,
		AttachmentsDir:    "attachments",
		RunChannelIDBackfill: false,
		GoodJanetUsername: "Good Janet",
		BadJanetUsername:  "Bad Janet",
		UserBlacklist:     []string{},
		UserAliases:       make(map[string]string),
	}
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	config := defaultConfig()

	// Try to load from file if it exists
	if configPath != "" {
		if data, err := os.ReadFile(configPath); err == nil {
			if err := json.Unmarshal(data, config); err != nil {
				return nil, err
			}
		}
	}

	// Override with environment variables
	if val := os.Getenv("JANET_DATABASE_DRIVER"); val != "" {
		config.DatabaseDriver = val
	}
	if val := os.Getenv("JANET_DATABASE_URL"); val != "" {
		config.DatabaseURL = val
	}
	if val := os.Getenv("JANET_SLACK_TOKEN"); val != "" {
		config.SlackToken = val
	}
	if val := os.Getenv("JANET_SLACK_SOCKET_TOKEN"); val != "" {
		config.SlackSocketToken = val
	}
	if val := os.Getenv("JANET_WEB_LISTEN_ADDR"); val != "" {
		config.WebListenAddr = val
	}
	if val := os.Getenv("JANET_WEB_PUBLIC_URL"); val != "" {
		config.WebPublicURL = val
	}
	if val := os.Getenv("JANET_ATTACHMENTS_DIR"); val != "" {
		config.AttachmentsDir = val
	}
	if val := os.Getenv("JANET_RUN_CHANNELID_BACKFILL"); val != "" {
		config.RunChannelIDBackfill = strings.ToLower(val) == "true" || val == "1" || strings.ToLower(val) == "yes"
	}
	if val := os.Getenv("JANET_BOT_ENABLED"); val != "" {
		config.BotEnabled = strings.ToLower(val) == "true" || val == "1" || strings.ToLower(val) == "yes"
	}

	return config, nil
}
