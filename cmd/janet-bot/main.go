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
	"github.com/troyxmccall/janet"
	"github.com/troyxmccall/janet/database"
	janetruntime "github.com/troyxmccall/janet/internal/runtime"
)

// BotService represents the dedicated bot service
type BotService struct {
	config  *janetruntime.BotOptions
	db      *database.V2DB
	bot     *janet.Bot
	logger  *log.Log
	runtime *janetruntime.SlackBotRuntime
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
func NewBotService(config *janetruntime.BotOptions, logger *log.Log) (*BotService, error) {
	// Initialize database
	v2db, err := database.NewV2(&database.Config{
		Driver: janetruntime.GetEnv("JANET_DATABASE_DRIVER", "postgres"),
		URL:    janetruntime.GetEnv("JANET_DATABASE_URL", "postgres://janet:password@localhost:5432/janet?sslmode=disable"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	logger.Info("connected to database")

	service := &BotService{
		config: config,
		db:     v2db,
		logger: logger,
	}

	// Initialize bot
	if err := service.initializeBot(); err != nil {
		return nil, fmt.Errorf("failed to initialize bot: %w", err)
	}

	return service, nil
}

// initializeBot sets up the Slack bot
func (s *BotService) initializeBot() error {
	runtime, err := janetruntime.NewSlackBotRuntime(*s.config, s.db, s.logger)
	if err != nil {
		return err
	}
	s.runtime = runtime
	s.bot = runtime.Bot
	s.logger.Info("slack bot initialized")

	return nil
}

// Start starts the bot service
func (s *BotService) Start() error {
	s.logger.Info("starting slack bot service")

	// Start bot in goroutine
	go func() {
		s.logger.Info("bot listening for slack events via socket mode")
		s.bot.ListenWithSocketMode(s.runtime.SocketClient)
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
	if s.db != nil {
		s.logger.Info("closing database connection")
		_ = s.db.Close()
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
func loadConfigFromEnv() *janetruntime.BotOptions {
	return &janetruntime.BotOptions{
		SlackToken:          janetruntime.GetEnv("JANET_SLACK_TOKEN", ""),
		SlackSocketToken:    janetruntime.GetEnv("JANET_SLACK_SOCKET_TOKEN", ""),
		GoodPlaceJudgeBotID: janetruntime.GetEnv("JANET_GOOD_PLACE_JUDGE_BOT_ID", ""),
		MaxPoints:           janetruntime.ParseInt(os.Getenv("JANET_MAX_POINTS"), 5),
		LeaderboardLimit:    janetruntime.ParseInt(os.Getenv("JANET_LEADERBOARD_LIMIT"), 10),
		ReplyType:           janetruntime.GetEnv("JANET_REPLY_TYPE", "thread"),
		Debug:               janetruntime.ParseBool(os.Getenv("JANET_DEBUG"), false),
		SelfKarma:           janetruntime.ParseBool(os.Getenv("JANET_SELF_KARMA"), false),
		Motivate:            janetruntime.ParseBool(os.Getenv("JANET_MOTIVATE"), true),
		ReactjiEnabled:      janetruntime.ParseBool(os.Getenv("JANET_REACTJI_ENABLED"), true),
		GoodJanetUsername:   janetruntime.GetEnv("JANET_GOOD_USERNAME", "Good Janet"),
		GoodJanetIconURL:    janetruntime.GetEnv("JANET_GOOD_ICON_URL", ""),
		BadJanetUsername:    janetruntime.GetEnv("JANET_BAD_USERNAME", "Bad Janet"),
		BadJanetIconURL:     janetruntime.GetEnv("JANET_BAD_ICON_URL", ""),
		UserBlacklist:       janetruntime.ParseCSV(os.Getenv("JANET_USER_BLACKLIST")),
		UserAliases:         janetruntime.ParseStringMap(os.Getenv("JANET_USER_ALIASES")),
	}
}
