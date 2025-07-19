package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aybabtme/log"
	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
	"github.com/troyxmccall/janet"
	"github.com/troyxmccall/janet/database"
)

// Config holds the bot configuration
type BotConfig struct {
	// Database
	DatabaseDriver string
	DatabaseURL    string

	// Slack
	SlackToken          string
	SlackSocketToken    string
	GoodPlaceJudgeBotID string

	// Bot behavior
	MaxPoints        int
	LeaderboardLimit int
	ReplyType        string
	Debug            bool
	SelfKarma        bool
	Motivate         bool
	ReactjiEnabled   bool

	// Bot personalities
	GoodJanetUsername string
	GoodJanetIconURL  string
	BadJanetUsername  string
	BadJanetIconURL   string

	// User management
	UserBlacklist []string
	UserAliases   map[string]string
}

// BotService represents the dedicated bot service
type BotService struct {
	config       *BotConfig
	db           *database.V2DB
	bot          *janet.Bot
	logger       *log.Log
	slackClient  *slack.Client
	socketClient *socketmode.Client
}

func main() {
	// Load environment variables
	godotenv.Load()

	// Parse command line flags
	var (
		debug = flag.Bool("debug", false, "Enable debug logging")
	)
	flag.Parse()

	logger := log.KV("service", "janet-bot")
	if *debug {
		logger = logger.KV("debug", true)
	}

	// Load configuration from environment
	config := loadConfigFromEnv()
	if *debug {
		config.Debug = true
	}

	// Create bot service
	service, err := NewBotService(config, logger)
	if err != nil {
		logger.Err(err).Fatal("failed to create bot service")
	}

	// Start the bot
	if err := service.Start(); err != nil {
		logger.Err(err).Fatal("failed to start bot service")
	}
}

// NewBotService creates a new bot service
func NewBotService(config *BotConfig, logger *log.Log) (*BotService, error) {
	// Initialize database
	v2db, err := database.NewV2(&database.Config{
		Driver: config.DatabaseDriver,
		URL:    config.DatabaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	logger.Info("connected to database")

	// Initialize Slack clients
	slackClient := slack.New(config.SlackToken, slack.OptionDebug(config.Debug), slack.OptionAppLevelToken(config.SlackSocketToken))
	socketClient := socketmode.New(slackClient, socketmode.OptionDebug(config.Debug))

	service := &BotService{
		config:       config,
		db:           v2db,
		logger:       logger,
		slackClient:  slackClient,
		socketClient: socketClient,
	}

	// Initialize bot
	if err := service.initializeBot(); err != nil {
		return nil, fmt.Errorf("failed to initialize bot: %w", err)
	}

	return service, nil
}

// initializeBot sets up the Slack bot
func (s *BotService) initializeBot() error {
	// Convert config to janet format
	blacklistMap := make(janet.StringList)
	for _, user := range s.config.UserBlacklist {
		blacklistMap[user] = struct{}{}
	}

	userAliases := make(janet.UserAliases)
	for alias, main := range s.config.UserAliases {
		userAliases[alias] = main
	}

	reactjiConfig := &janet.ReactjiConfig{
		Enabled: s.config.ReactjiEnabled,
		UpVote: janet.StringList{
			// All emojis that award +3 points (matching backfill service)
			"thumbsup":      struct{}{},
			"+1":            struct{}{},
			"thumbsup_all":  struct{}{},
			"joy":           struct{}{},
			"100":           struct{}{},
			"heart":         struct{}{},
			"clap":          struct{}{},
			"coffin":        struct{}{},
			"fire":          struct{}{},
			"heart_on_fire": struct{}{},
			"lol":           struct{}{},
			"nail_care":     struct{}{},
			"rainbow":       struct{}{},
			"rip":           struct{}{},
			"skull":         struct{}{},
			"sparkles":      struct{}{},
			"star-struck":   struct{}{},
			"star":          struct{}{},
			"star2":         struct{}{},
			"unicorn_face":  struct{}{},
			"yellow_heart":  struct{}{},
			"zach-cowboy":   struct{}{},
		},
		DownVote:     janet.StringList{"thumbsdown": struct{}{}, "-1": struct{}{}},
		RepeatPoints: janet.StringList{"bangbang": struct{}{}, "exclamation": struct{}{}, "!!!": struct{}{}},
	}

	botConfig := &janet.Config{
		Slack:               &janet.SlackChatService{s.slackClient},
		SlackWebClient:      s.slackClient,
		Debug:               s.config.Debug,
		MaxPoints:           s.config.MaxPoints,
		LeaderboardLimit:    s.config.LeaderboardLimit,
		Log:                 s.logger.KV("component", "bot"),
		DB:                  s.db,
		UserBlacklist:       blacklistMap,
		Reactji:             reactjiConfig,
		Motivate:            s.config.Motivate,
		Aliases:             userAliases,
		SelfPoints:          s.config.SelfKarma,
		ReplyType:           s.config.ReplyType,
		GoodPlaceJudgeBotID: s.config.GoodPlaceJudgeBotID,
		GoodPersonality: janet.BotPersonality{
			Username: s.config.GoodJanetUsername,
			IconURL:  s.config.GoodJanetIconURL,
			IsGood:   true,
		},
		BadPersonality: janet.BotPersonality{
			Username: s.config.BadJanetUsername,
			IconURL:  s.config.BadJanetIconURL,
			IsGood:   false,
		},
	}

	s.bot = janet.New(botConfig)
	s.logger.Info("slack bot initialized")

	return nil
}

// Start starts the bot service
func (s *BotService) Start() error {
	s.logger.Info("starting slack bot service")

	// Start bot in goroutine
	go func() {
		s.logger.Info("bot listening for slack events via socket mode")
		s.bot.ListenWithSocketMode(s.socketClient)
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("shutting down bot service")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Close database connection
	if s.db != nil && s.db.SQL != nil {
		s.logger.Info("closing database connection")
		s.db.SQL.Close()
	}

	select {
	case <-ctx.Done():
		s.logger.KV("error", "shutdown timed out").Error("shutdown timed out")
		return ctx.Err()
	default:
		s.logger.Info("bot service shutdown complete")
		return nil
	}
}

// loadConfigFromEnv loads configuration from environment variables
func loadConfigFromEnv() *BotConfig {
	return &BotConfig{
		// Database
		DatabaseDriver: getEnv("JANET_DATABASE_DRIVER", "postgres"),
		DatabaseURL:    getEnv("JANET_DATABASE_URL", "postgres://janet:password@localhost:5432/janet?sslmode=disable"),

		// Slack
		SlackToken:          getEnv("JANET_SLACK_TOKEN", ""),
		SlackSocketToken:    getEnv("JANET_SLACK_SOCKET_TOKEN", ""),
		GoodPlaceJudgeBotID: getEnv("JANET_GOOD_PLACE_JUDGE_BOT_ID", ""),

		// Bot behavior
		MaxPoints:        parseInt(getEnv("JANET_MAX_POINTS", "5")),
		LeaderboardLimit: parseInt(getEnv("JANET_LEADERBOARD_LIMIT", "10")),
		ReplyType:        getEnv("JANET_REPLY_TYPE", "message"),
		Debug:            parseBool(getEnv("JANET_DEBUG", "false")),
		SelfKarma:        parseBool(getEnv("JANET_SELF_KARMA", "false")),
		Motivate:         parseBool(getEnv("JANET_MOTIVATE", "true")),
		ReactjiEnabled:   parseBool(getEnv("JANET_REACTJI_ENABLED", "true")),

		// Bot personalities
		GoodJanetUsername: getEnv("JANET_GOOD_USERNAME", "Good Janet"),
		GoodJanetIconURL:  getEnv("JANET_GOOD_ICON_URL", ""),
		BadJanetUsername:  getEnv("JANET_BAD_USERNAME", "Bad Janet"),
		BadJanetIconURL:   getEnv("JANET_BAD_ICON_URL", ""),

		// User management (can be extended to parse from env)
		UserBlacklist: []string{},
		UserAliases:   make(map[string]string),
	}
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseInt(s string) int {
	if s == "" {
		return 0
	}
	// Simple int parsing, can be improved with proper error handling
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

func parseBool(s string) bool {
	return s == "true" || s == "1" || s == "yes"
}
