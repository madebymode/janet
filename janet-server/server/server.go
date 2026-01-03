package server

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aybabtme/log"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
	"github.com/troyxmccall/janet"
	"github.com/troyxmccall/janet/database"
)

// HandlerService interface for API handlers
type HandlerService interface {
	HandleAPILeaderboard(w http.ResponseWriter, r *http.Request)
	HandleAPICurrentLeaderboard(w http.ResponseWriter, r *http.Request)
	HandleAPIYearlyLeaderboard(w http.ResponseWriter, r *http.Request)
	HandleAPIStatsV2(w http.ResponseWriter, r *http.Request)
	HandleAPIStatsDetailed(w http.ResponseWriter, r *http.Request)
	HandleAPITopGivers(w http.ResponseWriter, r *http.Request)
	HandleAPITopGiversByYear(w http.ResponseWriter, r *http.Request)
	HandleAPIRecentActivity(w http.ResponseWriter, r *http.Request)
	HandleAPIAvailableYears(w http.ResponseWriter, r *http.Request)
	HandleAPITopEmojis(w http.ResponseWriter, r *http.Request)
	HandleAPIKarmaDistribution(w http.ResponseWriter, r *http.Request)
	HandleAPIActivityTimeline(w http.ResponseWriter, r *http.Request)
	HandleAPIPointsOverTime(w http.ResponseWriter, r *http.Request)
	HandleAPIPopularMessages(w http.ResponseWriter, r *http.Request)
	HandleAPIStatus(w http.ResponseWriter, r *http.Request)
	HandleAPIUser(w http.ResponseWriter, r *http.Request)
	HandleAPIUserAllTimePointsOverTime(w http.ResponseWriter, r *http.Request)
	HandleAPIUserByYear(w http.ResponseWriter, r *http.Request)
	HandleAPIUserPointsOverTime(w http.ResponseWriter, r *http.Request)
}

// Server represents the janet server
type Server struct {
	config    *Config
	db        *database.V2DB
	bot       *janet.Bot
	logger    *log.Log
	startTime time.Time
	router    *mux.Router
	templates map[string]*template.Template
	webFS     embed.FS
	handlers  HandlerService
}

// NewServer creates a new janet server instance
func NewServer(configPath string, webFS embed.FS) (*Server, error) {
	// Load environment variables
	godotenv.Load()

	logger := log.KV("component", "server")

	// Load configuration
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if config.AttachmentsDir != "" {
		if err := os.MkdirAll(config.AttachmentsDir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create attachments dir: %w", err)
		}
	}

	// Initialize database
	v2db, err := database.NewV2(&database.Config{
		Driver: config.DatabaseDriver,
		URL:    config.DatabaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	logger.Info("using postgres database")

	// Rebuild summary tables on startup for optimal performance
	logger.Info("rebuilding summary tables for optimal API performance")
	if err := v2db.RebuildSummaryTables(); err != nil {
		logger.Err(err).Error("failed to rebuild summary tables, APIs may be slower")
	} else {
		logger.Info("summary tables rebuilt successfully")
	}

	// Reset sequences on startup to prevent duplicate key errors after data imports
	logger.Info("resetting database sequences")
	if err := resetSequences(v2db); err != nil {
		logger.Err(err).Error("failed to reset sequences, may cause duplicate key errors")
	} else {
		logger.Info("database sequences reset successfully")
	}

	if config.RunChannelIDBackfill {
		logger.Info("backfilling missing channel_id values from transactions")
		updated, err := v2db.BackfillChannelIDsFromTransactions()
		if err != nil {
			logger.Err(err).Error("failed to backfill channel_id values")
		} else {
			logger.KV("updated_rows", updated).Info("channel_id backfill complete")
		}
	}

	// Create server
	server := &Server{
		config:    config,
		db:        v2db,
		logger:    logger,
		startTime: time.Now(),
		webFS:     webFS,
	}

	// Setup templates
	if err := server.setupTemplates(); err != nil {
		return nil, fmt.Errorf("failed to setup templates: %w", err)
	}

	// Note: Routes will be setup after handlers are registered

	// Initialize bot if enabled and slack tokens are provided
	if config.BotEnabled && config.SlackToken != "" && config.SlackSocketToken != "" {
		if err := server.initializeBot(); err != nil {
			return nil, fmt.Errorf("failed to initialize bot: %w", err)
		}
		server.logger.Info("slack bot initialized")
	} else {
		server.logger.Info("slack bot disabled or tokens not provided - running in web-only mode")
	}

	return server, nil
}

// Start starts the HTTP server and handles graceful shutdown
func (s *Server) Start() error {
	s.logger.KV("addr", s.config.WebListenAddr).Info("starting web server")

	server := &http.Server{
		Addr:    s.config.WebListenAddr,
		Handler: s.router,
	}

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Err(err).Fatal("web server failed")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("shutting down server")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return server.Shutdown(ctx)
}

// initializeBot initializes the Slack bot if enabled
func (s *Server) initializeBot() error {
	slackClient := slack.New(s.config.SlackToken, slack.OptionDebug(s.config.Debug), slack.OptionAppLevelToken(s.config.SlackSocketToken))
	socketClient := socketmode.New(slackClient, socketmode.OptionDebug(s.config.Debug))

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
		Enabled:      s.config.ReactjiEnabled,
		UpVote:       janet.StringList{"thumbsup": struct{}{}, "+1": struct{}{}},
		DownVote:     janet.StringList{"thumbsdown": struct{}{}, "-1": struct{}{}},
		RepeatPoints: janet.StringList{"bangbang": struct{}{}},
	}

	botConfig := &janet.Config{
		Slack:               &janet.SlackChatService{slackClient},
		SlackWebClient:      slackClient,
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

	// Start bot in goroutine
	go func() {
		s.logger.Info("starting slack bot")
		s.bot.ListenWithSocketMode(socketClient)
	}()

	return nil
}

// resetSequences resets all database sequences to prevent duplicate key errors after imports
func resetSequences(db *database.V2DB) error {
	// Use a transaction to ensure atomicity
	tx, err := db.SQL.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Lock the table briefly to prevent concurrent inserts during sequence reset
	_, err = tx.Exec(`LOCK TABLE karma_transactions IN ACCESS EXCLUSIVE MODE NOWAIT`)
	if err != nil {
		// If we can't get the lock immediately, that means there are active operations
		// In this case, just skip resetting sequences to avoid blocking
		return nil
	}

	// Reset karma_transactions_id_seq to max(id) + 1
	_, err = tx.Exec(`
		SELECT setval('karma_transactions_id_seq', COALESCE((SELECT MAX(id) FROM karma_transactions), 0) + 1, false);
	`)
	if err != nil {
		return fmt.Errorf("failed to reset karma_transactions_id_seq: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit sequence reset: %w", err)
	}

	return nil
}

// GetDB returns the database instance
func (s *Server) GetDB() *database.V2DB {
	return s.db
}

// GetBot returns the bot instance
func (s *Server) GetBot() *janet.Bot {
	return s.bot
}

// GetLogger returns the logger instance
func (s *Server) GetLogger() *log.Log {
	return s.logger
}

// RegisterHandlers registers the handler service with the server and sets up routes
func (s *Server) RegisterHandlers(handlers HandlerService) {
	s.handlers = handlers
	s.setupRoutes()
}
