package server

import (
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

// setupRoutes configures all HTTP routes and middleware
func (s *Server) setupRoutes() {
	s.router = mux.NewRouter()

	// Apply middleware to all routes (order matters!)
	s.router.Use(s.requestLoggingMiddleware)
	s.router.Use(s.securityHeadersMiddleware)
	s.router.Use(s.rateLimitMiddleware)

	// Static files with proper MIME types
	s.router.PathPrefix("/static/").Handler(s.staticFileHandler())
	if s.config.AttachmentsDir != "" {
		s.router.PathPrefix("/attachments/").Handler(s.attachmentsFileHandler())
	}

	// Public API routes
	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/leaderboard", s.handlers.HandleAPILeaderboard).Methods("GET")
	api.HandleFunc("/leaderboard/current", s.handlers.HandleAPICurrentLeaderboard).Methods("GET")
	api.HandleFunc("/leaderboard/{year:[0-9]+}", s.handlers.HandleAPIYearlyLeaderboard).Methods("GET")
	api.HandleFunc("/stats", s.handlers.HandleAPIStatsV2).Methods("GET")
	api.HandleFunc("/stats/detailed", s.handlers.HandleAPIStatsDetailed).Methods("GET")
	api.HandleFunc("/stats/top-givers", s.handlers.HandleAPITopGivers).Methods("GET")
	api.HandleFunc("/stats/top-givers/{year}", s.handlers.HandleAPITopGiversByYear).Methods("GET")
	api.HandleFunc("/stats/recent-activity", s.handlers.HandleAPIRecentActivity).Methods("GET")
	api.HandleFunc("/stats/years", s.handlers.HandleAPIAvailableYears).Methods("GET")
	api.HandleFunc("/stats/emojis", s.handlers.HandleAPITopEmojis).Methods("GET")
	api.HandleFunc("/stats/karma-distribution", s.handlers.HandleAPIKarmaDistribution).Methods("GET")
	api.HandleFunc("/stats/activity-timeline", s.handlers.HandleAPIActivityTimeline).Methods("GET")
	api.HandleFunc("/stats/points-over-time", s.handlers.HandleAPIPointsOverTime).Methods("GET")
	api.HandleFunc("/stats/popular-messages", s.handlers.HandleAPIPopularMessages).Methods("GET")
	api.HandleFunc("/status", s.handlers.HandleAPIStatus).Methods("GET")
	api.HandleFunc("/user/{username}", s.handlers.HandleAPIUser).Methods("GET")
	api.HandleFunc("/user/{username}/points-over-time/all", s.handlers.HandleAPIUserAllTimePointsOverTime).Methods("GET")
	api.HandleFunc("/user/{username}/{year}", s.handlers.HandleAPIUserByYear).Methods("GET")
	api.HandleFunc("/user/{username}/{year}/points-over-time", s.handlers.HandleAPIUserPointsOverTime).Methods("GET")

	// Catch-all route for React app (must be last)
	s.router.PathPrefix("/").HandlerFunc(s.handleReactApp).Methods("GET")
}

// staticFileHandler serves static files from embedded filesystem
func (s *Server) staticFileHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Remove /static/ prefix and prepend web/ for embedded filesystem
		relPath := path.Clean(strings.TrimPrefix(r.URL.Path, "/static/"))
		if relPath == "." || strings.HasPrefix(relPath, "..") {
			http.NotFound(w, r)
			return
		}
		filePath := path.Join("web/static", relPath)

		// Read the file from embedded filesystem
		content, err := s.webFS.ReadFile(filePath)
		if err != nil {
			s.logger.Err(err).KV("path", filePath).Error("failed to read static file")
			http.NotFound(w, r)
			return
		}

		// Set the correct MIME type based on file extension
		ext := filepath.Ext(filePath)
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
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_, _ = w.Write(content)
	})
}

// attachmentsFileHandler serves cached Slack attachments from disk
func (s *Server) attachmentsFileHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		relPath := strings.TrimPrefix(r.URL.Path, "/attachments/")
		if relPath == "" || relPath == "/" {
			http.NotFound(w, r)
			return
		}
		cleanPath := filepath.Clean(string(filepath.Separator) + relPath)
		cleanPath = strings.TrimPrefix(cleanPath, string(filepath.Separator))
		if cleanPath == "" || cleanPath == "." {
			http.NotFound(w, r)
			return
		}

		fullPath := filepath.Join(s.config.AttachmentsDir, cleanPath)
		baseDir, err := filepath.Abs(s.config.AttachmentsDir)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		fullPath, err = filepath.Abs(fullPath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if fullPath != baseDir && !strings.HasPrefix(fullPath, baseDir+string(filepath.Separator)) {
			http.NotFound(w, r)
			return
		}

		if info, err := os.Lstat(fullPath); err != nil || info.Mode()&os.ModeSymlink != 0 {
			http.NotFound(w, r)
			return
		}

		file, err := os.Open(fullPath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()

		info, err := file.Stat()
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	})
}

// handleReactApp serves the React application for SPA routing
func (s *Server) handleReactApp(w http.ResponseWriter, r *http.Request) {
	// Serve the React app for all non-API routes
	if err := s.templates["app"].Execute(w, nil); err != nil {
		s.logger.Err(err).Error("failed to execute React app template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
