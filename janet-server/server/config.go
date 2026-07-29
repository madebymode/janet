package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	janetruntime "github.com/troyxmccall/janet/internal/runtime"
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
	WebListenAddr        string  `json:"webListenAddr"`
	WebPublicURL         string  `json:"webPublicURL"`
	RateLimitRPS         float64 `json:"rateLimitRps"`
	RateLimitBurst       int     `json:"rateLimitBurst"`
	BotEnabled           bool    `json:"botEnabled"`
	AttachmentsDir       string  `json:"attachmentsDir"`
	RunChannelIDBackfill bool    `json:"runChannelIDBackfill"`

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
		DatabaseDriver:       "postgres",
		DatabaseURL:          "postgres://janet:janet@localhost:5432/janet?sslmode=disable",
		MaxPoints:            5,
		LeaderboardLimit:     10,
		ReplyType:            "thread",
		Debug:                false,
		SelfKarma:            false,
		Motivate:             true,
		ReactjiEnabled:       true,
		WebListenAddr:        ":8080",
		WebPublicURL:         "http://localhost:8080",
		RateLimitRPS:         20,
		RateLimitBurst:       60,
		BotEnabled:           true,
		AttachmentsDir:       "attachments",
		RunChannelIDBackfill: false,
		GoodJanetUsername:    "Good Janet",
		BadJanetUsername:     "Bad Janet",
		UserBlacklist:        []string{},
		UserAliases:          make(map[string]string),
	}
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	config := defaultConfig()

	// Try to load from file if it exists
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %q: %w", configPath, err)
		}
		if err := json.Unmarshal(data, config); err != nil {
			return nil, err
		}
	}

	// Override with environment variables
	config.DatabaseDriver = janetruntime.GetEnv("JANET_DATABASE_DRIVER", config.DatabaseDriver)
	config.DatabaseURL = janetruntime.GetEnv("JANET_DATABASE_URL", config.DatabaseURL)
	config.SlackToken = janetruntime.GetEnv("JANET_SLACK_TOKEN", config.SlackToken)
	config.SlackSocketToken = janetruntime.GetEnv("JANET_SLACK_SOCKET_TOKEN", config.SlackSocketToken)
	config.GoodPlaceJudgeBotID = janetruntime.GetEnv("JANET_GOOD_PLACE_JUDGE_BOT_ID", config.GoodPlaceJudgeBotID)
	config.MaxPoints = janetruntime.ParseInt(os.Getenv("JANET_MAX_POINTS"), config.MaxPoints)
	config.LeaderboardLimit = janetruntime.ParseInt(os.Getenv("JANET_LEADERBOARD_LIMIT"), config.LeaderboardLimit)
	config.ReplyType = janetruntime.GetEnv("JANET_REPLY_TYPE", config.ReplyType)
	config.Debug = janetruntime.ParseBool(os.Getenv("JANET_DEBUG"), config.Debug)
	config.SelfKarma = janetruntime.ParseBool(os.Getenv("JANET_SELF_KARMA"), config.SelfKarma)
	config.Motivate = janetruntime.ParseBool(os.Getenv("JANET_MOTIVATE"), config.Motivate)
	config.ReactjiEnabled = janetruntime.ParseBool(os.Getenv("JANET_REACTJI_ENABLED"), config.ReactjiEnabled)
	config.WebListenAddr = janetruntime.GetEnv("JANET_WEB_LISTEN_ADDR", config.WebListenAddr)
	config.WebPublicURL = janetruntime.GetEnv("JANET_WEB_PUBLIC_URL", config.WebPublicURL)
	config.RateLimitRPS = janetruntime.ParseFloat(os.Getenv("JANET_WEB_RATE_LIMIT_RPS"), config.RateLimitRPS)
	config.RateLimitBurst = janetruntime.ParseInt(os.Getenv("JANET_WEB_RATE_LIMIT_BURST"), config.RateLimitBurst)
	config.BotEnabled = janetruntime.ParseBool(os.Getenv("JANET_BOT_ENABLED"), config.BotEnabled)
	config.AttachmentsDir = janetruntime.GetEnv("JANET_ATTACHMENTS_DIR", config.AttachmentsDir)
	config.RunChannelIDBackfill = janetruntime.ParseBool(os.Getenv("JANET_RUN_CHANNELID_BACKFILL"), config.RunChannelIDBackfill)
	config.GoodJanetUsername = janetruntime.GetEnv("JANET_GOOD_USERNAME", config.GoodJanetUsername)
	config.GoodJanetIconURL = janetruntime.GetEnv("JANET_GOOD_ICON_URL", config.GoodJanetIconURL)
	config.BadJanetUsername = janetruntime.GetEnv("JANET_BAD_USERNAME", config.BadJanetUsername)
	config.BadJanetIconURL = janetruntime.GetEnv("JANET_BAD_ICON_URL", config.BadJanetIconURL)
	if blacklist := janetruntime.ParseCSV(os.Getenv("JANET_USER_BLACKLIST")); len(blacklist) > 0 {
		config.UserBlacklist = blacklist
	}
	if aliases := janetruntime.ParseStringMap(os.Getenv("JANET_USER_ALIASES")); len(aliases) > 0 {
		config.UserAliases = aliases
	}

	return config, config.normalize()
}

func (c *Config) normalize() error {
	c.DatabaseDriver = strings.TrimSpace(c.DatabaseDriver)
	c.DatabaseURL = strings.TrimSpace(c.DatabaseURL)
	c.SlackToken = strings.TrimSpace(c.SlackToken)
	c.SlackSocketToken = strings.TrimSpace(c.SlackSocketToken)
	c.GoodPlaceJudgeBotID = strings.TrimSpace(c.GoodPlaceJudgeBotID)
	c.WebListenAddr = strings.TrimSpace(c.WebListenAddr)
	c.WebPublicURL = strings.TrimSpace(c.WebPublicURL)
	c.GoodJanetUsername = strings.TrimSpace(c.GoodJanetUsername)
	c.GoodJanetIconURL = strings.TrimSpace(c.GoodJanetIconURL)
	c.BadJanetUsername = strings.TrimSpace(c.BadJanetUsername)
	c.BadJanetIconURL = strings.TrimSpace(c.BadJanetIconURL)

	if c.DatabaseDriver == "" {
		c.DatabaseDriver = "postgres"
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("database URL must not be empty")
	}
	if c.WebListenAddr == "" {
		c.WebListenAddr = ":8080"
	}
	if c.WebPublicURL == "" {
		c.WebPublicURL = "http://localhost:8080"
	}
	if c.MaxPoints <= 0 || c.MaxPoints > 100 {
		c.MaxPoints = 5
	}
	if c.LeaderboardLimit <= 0 || c.LeaderboardLimit > 100 {
		c.LeaderboardLimit = 10
	}
	switch strings.ToLower(c.ReplyType) {
	case "", "thread":
		c.ReplyType = "thread"
	case "message":
		c.ReplyType = "message"
	default:
		return fmt.Errorf("invalid reply type %q", c.ReplyType)
	}
	if c.RateLimitRPS <= 0 || c.RateLimitRPS > 1000 {
		c.RateLimitRPS = 20
	}
	if c.RateLimitBurst <= 0 || c.RateLimitBurst > 5000 {
		c.RateLimitBurst = 60
	}
	if c.AttachmentsDir != "" {
		c.AttachmentsDir = filepath.Clean(c.AttachmentsDir)
	}
	c.UserBlacklist = normalizeStringList(c.UserBlacklist)
	c.UserAliases = normalizeAliasMap(c.UserAliases)
	if c.GoodJanetUsername == "" {
		c.GoodJanetUsername = "Good Janet"
	}
	if c.BadJanetUsername == "" {
		c.BadJanetUsername = "Bad Janet"
	}

	return nil
}

func normalizeStringList(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func normalizeAliasMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return make(map[string]string)
	}

	result := make(map[string]string, len(values))
	for alias, main := range values {
		alias = strings.TrimSpace(alias)
		main = strings.TrimSpace(main)
		if alias == "" || main == "" {
			continue
		}
		result[alias] = main
	}
	return result
}
