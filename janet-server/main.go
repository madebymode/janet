package main

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	stdlog "log"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
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

//go:embed web/static/* web/templates/*
var webFS embed.FS

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
	WebPassword   string `json:"webPassword"`
	WebListenAddr string `json:"webListenAddr"`
	WebPublicURL  string `json:"webPublicURL"`
	BotEnabled    bool   `json:"botEnabled"`

	// Bot personalities
	GoodJanetUsername string `json:"goodJanetUsername"`
	GoodJanetIconURL  string `json:"goodJanetIconURL"`
	BadJanetUsername  string `json:"badJanetUsername"`
	BadJanetIconURL   string `json:"badJanetIconURL"`

	// User management
	UserBlacklist []string          `json:"userBlacklist"`
	UserAliases   map[string]string `json:"userAliases"`
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
	sessions  map[string]time.Time
	mutex     sync.RWMutex
}

// NewServer creates a new janet server
func NewServer(configPath string) (*Server, error) {
	// Load environment variables
	godotenv.Load()

	logger := log.KV("component", "server")

	// Load configuration
	config, err := loadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
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

	// Create server
	server := &Server{
		config:    config,
		db:        v2db,
		logger:    logger,
		startTime: time.Now(),
		sessions:  make(map[string]time.Time),
	}

	// Setup templates
	if err := server.setupTemplates(); err != nil {
		return nil, fmt.Errorf("failed to setup templates: %w", err)
	}

	// Setup routes
	server.setupRoutes()

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

func (s *Server) setupTemplates() error {
	s.templates = make(map[string]*template.Template)

	// Load templates from embedded filesystem
	templateFiles := []string{
		"web/templates/app.html",
		"web/templates/index.html",
		"web/templates/leaderboard.html",
		"web/templates/stats.html",
		"web/templates/admin/login.html",
		"web/templates/admin/dashboard.html",
		"web/templates/admin/config.html",
		"web/templates/admin/backfill.html",
		"web/templates/admin/users.html",
	}

	for _, file := range templateFiles {
		content, err := webFS.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", file, err)
		}

		name := filepath.Base(file)
		name = strings.TrimSuffix(name, filepath.Ext(name))

		var tmpl *template.Template
		if strings.Contains(file, "/admin/") && name != "login" {
			// Admin templates use base template
			baseTmpl, err := webFS.ReadFile("web/templates/base.html")
			if err != nil {
				return fmt.Errorf("failed to read base template: %w", err)
			}
			tmpl, err = template.New("base").Parse(string(baseTmpl))
			if err != nil {
				return fmt.Errorf("failed to parse base template: %w", err)
			}
			tmpl, err = tmpl.Parse(string(content))
			if err != nil {
				return fmt.Errorf("failed to parse template %s: %w", file, err)
			}
		} else if name == "login" {
			// Login template is standalone
			tmpl, err = template.New(name).Parse(string(content))
			if err != nil {
				return fmt.Errorf("failed to parse template %s: %w", file, err)
			}
		} else {
			// Public templates use base template
			baseTmpl, err := webFS.ReadFile("web/templates/base.html")
			if err != nil {
				return fmt.Errorf("failed to read base template: %w", err)
			}
			tmpl, err = template.New("base").Parse(string(baseTmpl))
			if err != nil {
				return fmt.Errorf("failed to parse base template: %w", err)
			}
			tmpl, err = tmpl.Parse(string(content))
			if err != nil {
				return fmt.Errorf("failed to parse template %s: %w", file, err)
			}
		}

		s.templates[name] = tmpl
	}

	return nil
}

func (s *Server) setupRoutes() {
	s.router = mux.NewRouter()

	// Static files with proper MIME types
	s.router.PathPrefix("/static/").Handler(s.staticFileHandler())

	// Public API routes
	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/leaderboard", s.handleAPILeaderboard).Methods("GET")
	api.HandleFunc("/leaderboard/current", s.handleAPICurrentLeaderboard).Methods("GET")
	api.HandleFunc("/leaderboard/{year:[0-9]+}", s.handleAPIYearlyLeaderboard).Methods("GET")
	api.HandleFunc("/stats", s.handleAPIStatsV2).Methods("GET")
	api.HandleFunc("/stats/detailed", s.handleAPIStatsDetailed).Methods("GET")
	api.HandleFunc("/stats/top-givers", s.handleAPITopGivers).Methods("GET")
	api.HandleFunc("/stats/top-givers/{year}", s.handleAPITopGiversByYear).Methods("GET")
	api.HandleFunc("/stats/recent-activity", s.handleAPIRecentActivity).Methods("GET")
	api.HandleFunc("/stats/years", s.handleAPIAvailableYears).Methods("GET")
	api.HandleFunc("/stats/emojis", s.handleAPITopEmojis).Methods("GET")
	api.HandleFunc("/stats/karma-distribution", s.handleAPIKarmaDistribution).Methods("GET")
	api.HandleFunc("/stats/activity-timeline", s.handleAPIActivityTimeline).Methods("GET")
	api.HandleFunc("/stats/points-over-time", s.handleAPIPointsOverTime).Methods("GET")
	api.HandleFunc("/status", s.handleAPIStatus).Methods("GET")
	api.HandleFunc("/user/{username}", s.handleAPIUser).Methods("GET")
	api.HandleFunc("/user/{username}/{year}", s.handleAPIUserByYear).Methods("GET")
	api.HandleFunc("/user/{username}/{year}/points-over-time", s.handleAPIUserPointsOverTime).Methods("GET")

	// Admin API routes (authentication handled by middleware)
	adminAPI := s.router.PathPrefix("/admin/api").Subrouter()
	adminAPI.Use(s.requireAuthMiddleware)
	adminAPI.HandleFunc("/config", s.handleAdminAPIConfig).Methods("GET", "POST")
	adminAPI.HandleFunc("/config/reset", s.handleAdminAPIConfigReset).Methods("POST")
	adminAPI.HandleFunc("/status", s.handleAdminAPIStatus).Methods("GET")
	adminAPI.HandleFunc("/restart", s.handleAdminAPIRestart).Methods("POST")
	adminAPI.HandleFunc("/export", s.handleAdminAPIExport).Methods("GET")
	adminAPI.HandleFunc("/data", s.handleAdminAPIDeleteData).Methods("DELETE")
	adminAPI.HandleFunc("/backfill", s.handleAdminAPIBackfill).Methods("POST")
	adminAPI.HandleFunc("/backfill/bulk", s.handleAdminAPIBackfillBulk).Methods("POST")
	adminAPI.HandleFunc("/backfill/history", s.handleAdminAPIBackfillHistory).Methods("GET")
	adminAPI.HandleFunc("/backfill/check", s.handleAdminAPIBackfillCheck).Methods("POST")
	adminAPI.HandleFunc("/channels", s.handleAdminAPIChannels).Methods("GET")
	adminAPI.HandleFunc("/users", s.handleAdminAPIUsers).Methods("GET")
	adminAPI.HandleFunc("/users/{username}/details", s.handleAdminAPIUserDetails).Methods("GET")
	adminAPI.HandleFunc("/users/adjust-points", s.handleAdminAPIAdjustPoints).Methods("POST")
	adminAPI.HandleFunc("/users/merge", s.handleAdminAPIMergeUsers).Methods("POST")
	adminAPI.HandleFunc("/users/{username}", s.handleAdminAPIDeleteUser).Methods("DELETE")
	adminAPI.HandleFunc("/blacklist", s.handleAdminAPIBlacklist).Methods("GET", "POST")
	adminAPI.HandleFunc("/blacklist/{username}", s.handleAdminAPIBlacklistDelete).Methods("DELETE")
	adminAPI.HandleFunc("/aliases", s.handleAdminAPIAliases).Methods("GET", "POST")
	adminAPI.HandleFunc("/aliases/{alias}", s.handleAdminAPIAliasDelete).Methods("DELETE")
	adminAPI.HandleFunc("/rebuild-summary", s.handleAdminAPIRebuildSummary).Methods("POST")

	// Admin login endpoint (special handling for POST)
	s.router.HandleFunc("/admin/login", s.handleAdminLogin).Methods("GET", "POST")
	s.router.HandleFunc("/admin/logout", s.handleAdminLogout).Methods("POST")

	// Catch-all route for React app (must be last)
	s.router.PathPrefix("/").HandlerFunc(s.handleReactApp).Methods("GET")
}

func (s *Server) staticFileHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Remove /static/ prefix and prepend web/ for embedded filesystem
		path := strings.TrimPrefix(r.URL.Path, "/static/")
		path = "web/static/" + path

		// Read the file from embedded filesystem
		content, err := webFS.ReadFile(path)
		if err != nil {
			s.logger.Err(err).KV("path", path).Error("failed to read static file")
			http.NotFound(w, r)
			return
		}

		// Set the correct MIME type based on file extension
		ext := filepath.Ext(path)
		contentType := mime.TypeByExtension(ext)
		if contentType == "" {
			// Fallback for common web file types
			switch ext {
			case ".css":
				contentType = "text/css"
			case ".js":
				contentType = "application/javascript"
			case ".png":
				contentType = "image/png"
			case ".jpg", ".jpeg":
				contentType = "image/jpeg"
			case ".svg":
				contentType = "image/svg+xml"
			case ".ico":
				contentType = "image/x-icon"
			default:
				contentType = "application/octet-stream"
			}
		}

		w.Header().Set("Content-Type", contentType)
		w.Write(content)
	})
}

func (s *Server) handleReactApp(w http.ResponseWriter, r *http.Request) {
	// Serve the React app for all non-API routes
	if err := s.templates["app"].Execute(w, nil); err != nil {
		s.logger.Err(err).Error("failed to execute React app template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

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

// Template data structures
type TemplateData struct {
	Title     string
	IsAdmin   bool
	BotOnline bool
	Version   string
	ExtraJS   []string
	Data      interface{}
	Error     string
}

// Auth helpers
func (s *Server) generateSessionToken() string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d-%s", time.Now().UnixNano(), s.config.WebPassword)))
	return hex.EncodeToString(hash[:])
}

func (s *Server) isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("janet_session")
	if err != nil {
		return false
	}

	s.mutex.RLock()
	expiry, exists := s.sessions[cookie.Value]
	s.mutex.RUnlock()

	if !exists || time.Now().After(expiry) {
		return false
	}

	return true
}

func (s *Server) isBasicAuthenticated(r *http.Request) bool {
	username, password, ok := r.BasicAuth()
	if !ok {
		return false
	}

	// Check against configured admin credentials
	// For now, we'll use "admin" as username and the web password from config
	return username == "admin" && password == s.config.WebPassword
}

func (s *Server) requireAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.isAuthenticated(r) {
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}
		handler(w, r)
	}
}

func (s *Server) requireAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a request from the React dashboard (has cookies)
		// If so, use cookie auth, otherwise use basic auth
		if s.isAuthenticated(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Fallback to basic auth for direct API access
		if !s.isBasicAuthenticated(r) {
			w.Header().Set("WWW-Authenticate", `Basic realm="Admin Area"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Default configuration
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
		GoodJanetUsername: "Good Janet",
		BadJanetUsername:  "Bad Janet",
		UserBlacklist:     []string{},
		UserAliases:       make(map[string]string),
	}
}

func loadConfig(configPath string) (*Config, error) {
	config := defaultConfig()

	// Try to load from file if it exists
	if configPath != "" {
		if data, err := os.ReadFile(configPath); err == nil {
			if err := json.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("invalid config file: %w", err)
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
	if val := os.Getenv("JANET_WEB_PASSWORD"); val != "" {
		config.WebPassword = val
	}
	if val := os.Getenv("JANET_WEB_LISTEN_ADDR"); val != "" {
		config.WebListenAddr = val
	}
	if val := os.Getenv("JANET_BOT_ENABLED"); val != "" {
		config.BotEnabled = strings.ToLower(val) == "true" || val == "1" || strings.ToLower(val) == "yes"
	}

	// Generate a default password if none provided
	if config.WebPassword == "" {
		config.WebPassword = "admin"
		stdlog.Println("WARNING: Using default password 'admin'. Set JANET_WEB_PASSWORD environment variable for security.")
	}

	return config, nil
}

func main() {
	configPath := ""
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	server, err := NewServer(configPath)
	if err != nil {
		stdlog.Fatal("Failed to create server:", err)
	}

	if err := server.Start(); err != nil {
		stdlog.Fatal("Server error:", err)
	}
}

// Helper to get V2DB instance
func (s *Server) getV2DB() (*database.V2DB, bool) {
	return s.db, true
}
