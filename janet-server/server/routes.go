package server

import (
	"mime"
	"net/http"
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

	// Static files with proper MIME types
	s.router.PathPrefix("/static/").Handler(s.staticFileHandler())

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
		path := strings.TrimPrefix(r.URL.Path, "/static/")
		path = "web/static/" + path

		// Read the file from embedded filesystem
		content, err := s.webFS.ReadFile(path)
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

// handleReactApp serves the React application for SPA routing
func (s *Server) handleReactApp(w http.ResponseWriter, r *http.Request) {
	// Serve the React app for all non-API routes
	if err := s.templates["app"].Execute(w, nil); err != nil {
		s.logger.Err(err).Error("failed to execute React app template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}