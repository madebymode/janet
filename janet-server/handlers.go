package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/slack-go/slack"
	"github.com/troyxmccall/janet"
	"github.com/troyxmccall/janet/database"
	"golang.org/x/time/rate"
)

// Global rate limiter for Slack API calls
var (
	slackRateLimiter = rate.NewLimiter(rate.Every(2*time.Second), 1) // 1 call every 2 seconds (30 calls per minute)
	rateLimiterMu    sync.RWMutex
)

// Caching structures
var (
	userCache      = make(map[string]string) // userID -> username
	userCacheMu    sync.RWMutex
	channelCache   = make(map[string]string) // channelID -> channelName
	channelCacheMu sync.RWMutex
	cacheTimeout   = 1 * time.Hour // Cache entries expire after 1 hour

	// Full profile cache for Slack user data
	profileCache       = make(map[string]*slack.User) // userID -> full slack.User
	profileCacheMu     sync.RWMutex
	profileCacheTime   = make(map[string]time.Time) // Track when profile entries were cached
	profileCacheExpiry = 2 * time.Hour              // Cache profiles longer since they change less frequently
	userCacheTime      = make(map[string]time.Time) // Track when entries were cached
	channelCacheTime   = make(map[string]time.Time)

	// Target channels for backfill operations and caching - these are the high-activity
	// channels we want to focus on for karma processing and data collection
	targetChannels = map[string]bool{
		"general":       true,
		"in-the-office": true,
		"funstuff":      true,
		"tvtalk":        true,
		"digital":       true,
		"creative":      true,
	}
)

// Initialize karma regex instance for backfill processing
var karmaReg = &janet.KarmaRegexps{}

// Public route handlers

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data := TemplateData{
		Title:     "Home",
		BotOnline: s.bot != nil,
		Version:   janet.Version,
	}

	if err := s.templates["index"].Execute(w, data); err != nil {
		s.logger.Err(err).Error("failed to execute template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (s *Server) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	data := TemplateData{
		Title:     "Leaderboard",
		BotOnline: s.bot != nil,
	}

	if err := s.templates["leaderboard"].Execute(w, data); err != nil {
		s.logger.Err(err).Error("failed to execute template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	data := TemplateData{
		Title:     "Statistics",
		BotOnline: s.bot != nil,
	}

	if err := s.templates["stats"].Execute(w, data); err != nil {
		s.logger.Err(err).Error("failed to execute template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Public API handlers

func (s *Server) handleAPILeaderboard(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	var leaderboard []*database.UserSummary
	var err error

	leaderboard, err = s.db.GetCurrentLeaderboard(limit)

	if err != nil {
		s.logger.Err(err).Error("failed to get leaderboard")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Enrich user data with Slack information
	enrichedLeaderboard := s.enrichUsersWithSlackInfo(leaderboard)

	// Filter out bots and deleted users by default (unless show_all=true)
	if r.URL.Query().Get("show_all") != "true" {
		enrichedLeaderboard = s.filterActiveUsers(enrichedLeaderboard)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"users": enrichedLeaderboard})
}

func (s *Server) handleAPIStats(w http.ResponseWriter, r *http.Request) {
	year := 0
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if parsed, err := strconv.Atoi(yearStr); err == nil {
			year = parsed
		}
	}

	var totalPoints int
	var err error
	if year == 0 {
		totalPoints, err = s.db.GetTotalPointsCumulative()
	} else {
		totalPoints, err = s.db.GetTotalPointsByYear(year)
	}
	if err != nil {
		s.logger.Err(err).Error("failed to get total points")
		totalPoints = 0
	}

	var totalUsers int
	if year == 0 {
		totalUsers, err = s.db.GetTotalUsersCumulative()
	} else {
		totalUsers, err = s.db.GetTotalUsersByYear(year)
	}
	if err != nil {
		s.logger.Err(err).Error("failed to get total users")
		totalUsers = 0
	}

	var totalTransactions int
	if year == 0 {
		totalTransactions, err = s.db.GetTotalTransactionsCumulative()
	} else {
		totalTransactions, err = s.db.GetTotalTransactionsByYear(year)
	}
	if err != nil {
		s.logger.Err(err).Error("failed to get total transactions")
		totalTransactions = 0
	}

	response := map[string]interface{}{
		"totalUsers":        totalUsers,
		"totalPoints":       totalPoints,
		"totalTransactions": totalTransactions,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleAPIStatsDetailed(w http.ResponseWriter, r *http.Request) {
	year := 0
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if parsed, err := strconv.Atoi(yearStr); err == nil {
			year = parsed
		}
	}

	var err error
	var totalUsers int
	if year == 0 {
		totalUsers, err = s.db.GetTotalUsersCumulative()
	} else {
		totalUsers, err = s.db.GetTotalUsersByYear(year)
	}
	if err != nil {
		s.logger.Err(err).Error("failed to get total users")
		totalUsers = 0
	}

	var totalPoints int
	if year == 0 {
		totalPoints, err = s.db.GetTotalPointsCumulative()
	} else {
		totalPoints, err = s.db.GetTotalPointsByYear(year)
	}
	if err != nil {
		s.logger.Err(err).Error("failed to get total points")
		totalPoints = 0
	}

	var totalTransactions int
	if year == 0 {
		totalTransactions, err = s.db.GetTotalTransactionsCumulative()
	} else {
		totalTransactions, err = s.db.GetTotalTransactionsByYear(year)
	}
	if err != nil {
		s.logger.Err(err).Error("failed to get total transactions")
		totalTransactions = 0
	}

	var positiveTransactions int
	if year == 0 {
		positiveTransactions, err = s.db.GetPositiveTransactionsCumulative()
	} else {
		positiveTransactions, err = s.db.GetPositiveTransactionsByYear(year)
	}
	if err != nil {
		s.logger.Err(err).Error("failed to get positive transactions")
		positiveTransactions = 0
	}

	var negativeTransactions int
	if year == 0 {
		negativeTransactions, err = s.db.GetNegativeTransactionsCumulative()
	} else {
		negativeTransactions, err = s.db.GetNegativeTransactionsByYear(year)
	}
	if err != nil {
		s.logger.Err(err).Error("failed to get negative transactions")
		negativeTransactions = 0
	}

	var avgPointsPerUser float64
	if totalUsers > 0 {
		avgPointsPerUser = float64(totalPoints) / float64(totalUsers)
	}

	response := map[string]interface{}{
		"totalUsers":           totalUsers,
		"totalPoints":          totalPoints,
		"totalTransactions":    totalTransactions,
		"positiveTransactions": positiveTransactions,
		"negativeTransactions": negativeTransactions,
		"avgPointsPerUser":     avgPointsPerUser,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleAPITopGivers(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	topGivers, err := s.db.GetTopGiversCumulative(limit)
	if err != nil {
		s.logger.Err(err).Error("failed to get top givers")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Enrich user data with Slack information
	enrichedTopGivers := s.enrichUsersWithSlackInfo(topGivers)

	// Filter out bots and deleted users by default (unless show_all=true)
	if r.URL.Query().Get("show_all") != "true" {
		enrichedTopGivers = s.filterActiveUsers(enrichedTopGivers)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"users": enrichedTopGivers})
}

func (s *Server) handleAPITopGiversByYear(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	year, err := strconv.Atoi(vars["year"])
	if err != nil {
		http.Error(w, "Invalid year", http.StatusBadRequest)
		return
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	topGivers, err := s.db.GetTopGiversByYear(limit, year)
	if err != nil {
		s.logger.Err(err).Error("failed to get top givers by year")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Enrich user data with Slack information
	enrichedTopGivers := s.enrichUsersWithSlackInfo(topGivers)

	// Filter out bots and deleted users by default (unless show_all=true)
	if r.URL.Query().Get("show_all") != "true" {
		enrichedTopGivers = s.filterActiveUsers(enrichedTopGivers)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"users": enrichedTopGivers})
}

func (s *Server) handleAPIRecentActivity(w http.ResponseWriter, r *http.Request) {
	activities, err := s.db.GetRecentActivity(20)
	if err != nil {
		s.logger.Err(err).Error("failed to get recent activity")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	// Transform activities to change 'timestamp' to 'date'
	transformedActivities := make([]map[string]interface{}, len(activities))
	for i, activity := range activities {
		transformedActivities[i] = map[string]interface{}{
			"from":            activity.FromUser,
			"to":              activity.ToUser,
			"points":          activity.Points,
			"reason":          activity.Reason,
			"transactionType": activity.TransactionType,
			"emojiName":       activity.EmojiName,
			"channelId":       activity.ChannelID,
			"messageId":       activity.MessageID,
			"date":            activity.Timestamp.Format("2006-01-02T15:04:05Z07:00"), // Format as ISO 8601 string
		}
	}
	json.NewEncoder(w).Encode(transformedActivities)
}

func (s *Server) handleAPIUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]

	summary, err := s.db.GetUser(username)
	if err != nil {
		if err == database.ErrNoSuchUser {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			s.logger.Err(err).Error("failed to get user")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	response := map[string]interface{}{
		"name":   summary.Username,
		"points": summary.TotalPoints,
		"rank":   summary.Rank,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// New V2 API handlers for enhanced features

func (s *Server) handleAPIYearlyLeaderboard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	year := vars["year"]

	yearInt, err := strconv.Atoi(year)
	if err != nil {
		http.Error(w, "Invalid year", http.StatusBadRequest)
		return
	}

	limit := 25
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	leaderboard, err := s.db.GetYearlyLeaderboard(yearInt, limit)
	if err != nil {
		s.logger.Err(err).Error("failed to get yearly leaderboard")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Enrich user data with Slack information
	enrichedLeaderboard := s.enrichUsersWithSlackInfo(leaderboard)

	// Filter out bots and deleted users by default (unless show_all=true)
	if r.URL.Query().Get("show_all") != "true" {
		enrichedLeaderboard = s.filterActiveUsers(enrichedLeaderboard)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"users": enrichedLeaderboard})
}

func (s *Server) handleAPICurrentLeaderboard(w http.ResponseWriter, r *http.Request) {
	limit := 25
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	currentYear := time.Now().Year()

	// Get current year leaderboard
	currentLeaderboard, err := s.db.GetYearlyLeaderboard(currentYear, limit)
	if err != nil {
		s.logger.Err(err).Error("failed to get current year leaderboard")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get all-time leaderboard
	allTimeLeaderboard, err := s.db.GetCurrentLeaderboard(limit)
	if err != nil {
		s.logger.Err(err).Error("failed to get all-time leaderboard")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Enrich both leaderboards with Slack information
	enrichedCurrentLeaderboard := s.enrichUsersWithSlackInfo(currentLeaderboard)
	enrichedAllTimeLeaderboard := s.enrichUsersWithSlackInfo(allTimeLeaderboard)

	// Filter out bots and deleted users by default (unless show_all=true)
	if r.URL.Query().Get("show_all") != "true" {
		enrichedCurrentLeaderboard = s.filterActiveUsers(enrichedCurrentLeaderboard)
		enrichedAllTimeLeaderboard = s.filterActiveUsers(enrichedAllTimeLeaderboard)
	}

	response := map[string]interface{}{
		"current": enrichedCurrentLeaderboard,
		"allTime": enrichedAllTimeLeaderboard,
		"year":    currentYear,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleAPIAvailableYears(w http.ResponseWriter, r *http.Request) {
	years, err := s.db.GetAvailableYears()
	if err != nil {
		s.logger.Err(err).Error("failed to get available years")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(years)
}

func (s *Server) handleAPITopEmojis(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	year := time.Now().Year()
	if y := r.URL.Query().Get("year"); y != "" {
		if parsed, err := strconv.Atoi(y); err == nil {
			year = parsed
		}
	}

	emojis, err := s.db.GetTopEmojisByYear(year, limit)
	if err != nil {
		s.logger.Err(err).Error("failed to get top emojis")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	totalEmojiUsage, err := s.db.GetTotalEmojiUsageByYear(year)
	if err != nil {
		s.logger.Err(err).Error("failed to get total emoji usage")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	response := make([]map[string]interface{}, len(emojis))
	for i, emoji := range emojis {
		response[i] = map[string]interface{}{
			"emoji_name":        emoji.EmojiName,
			"year":              emoji.Year,
			"usage_count":       emoji.UsageCount,
			"points_awarded":    emoji.PointsAwarded,
			"unique_users":      emoji.UniqueUsers,
			"rank":              emoji.Rank,
			"total_usage_count": totalEmojiUsage,
		}
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleAPIUserByYear(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]
	year := vars["year"]

	yearInt, err := strconv.Atoi(year)
	if err != nil {
		http.Error(w, "Invalid year", http.StatusBadRequest)
		return
	}

	user, err := s.db.GetUserByYear(username, yearInt)
	if err != nil {
		if err == database.ErrNoSuchUser {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			s.logger.Err(err).Error("failed to get user by year")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (s *Server) handleAPIUserPointsOverTime(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]
	year := vars["year"]

	yearInt, err := strconv.Atoi(year)
	if err != nil {
		http.Error(w, "Invalid year", http.StatusBadRequest)
		return
	}

	// Query user_summary_monthly table for user-specific data
	query := `
		SELECT month, total_points
		FROM user_summary_monthly
		WHERE username = $1 AND year = $2
		ORDER BY month
	`

	rows, err := s.db.SQL.Query(query, username, yearInt)
	if err != nil {
		s.logger.Err(err).Error("failed to query user points over time")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type MonthlyPoints struct {
		Month       int `json:"month"`
		TotalPoints int `json:"totalPoints"`
	}

	var results []MonthlyPoints
	for rows.Next() {
		var mp MonthlyPoints
		if err := rows.Scan(&mp.Month, &mp.TotalPoints); err != nil {
			s.logger.Err(err).Error("failed to scan monthly points")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		results = append(results, mp)
	}

	if err := rows.Err(); err != nil {
		s.logger.Err(err).Error("row iteration error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (s *Server) handleAPIStatsV2(w http.ResponseWriter, r *http.Request) {
	year := 0
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if parsed, err := strconv.Atoi(yearStr); err == nil {
			year = parsed
		}
	}

	var totalPoints int
	if year == 0 {
		totalPoints, _ = s.db.GetTotalPointsCumulative()
	} else {
		totalPoints, _ = s.db.GetTotalPointsByYear(year)
	}

	var totalUsers int
	if year == 0 {
		totalUsers, _ = s.db.GetTotalUsersCumulative()
	} else {
		totalUsers, _ = s.db.GetTotalUsersByYear(year)
	}

	var totalTransactions int
	if year == 0 {
		totalTransactions, _ = s.db.GetTotalTransactionsCumulative()
	} else {
		totalTransactions, _ = s.db.GetTotalTransactionsByYear(year)
	}

	response := map[string]interface{}{
		"totalUsers":        totalUsers,
		"totalPoints":       totalPoints,
		"totalTransactions": totalTransactions,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Admin route handlers

// Admin API handlers

func (s *Server) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"botOnline": s.checkBotHealth(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getActiveChannels filters channels by recent activity using Slack API
func (s *Server) getActiveChannels(client *slack.Client, allChannels []slack.Channel, since string) []slack.Channel {
	type channelActivity struct {
		Channel      slack.Channel
		MessageCount int
		LastActivity time.Time
	}

	var activeChannels []channelActivity

	// Parse since date for activity filtering
	var sinceTime time.Time
	if since != "" {
		var err error
		sinceTime, err = time.Parse("2006-01-02", since)
		if err != nil {
			s.logger.Err(err).Error("failed to parse since date for activity filtering")
			sinceTime = time.Now().AddDate(0, -1, 0) // Default to 1 month ago
		}
	} else {
		sinceTime = time.Now().AddDate(0, -1, 0) // Default to 1 month ago
	}

	s.logger.KV("sinceTime", sinceTime).Info("filtering channels by activity")

	// Only check activity for channels we actually care about

	// Check activity for each channel (but only our target channels)
	for _, channel := range allChannels {
		// Skip archived channels
		if channel.IsArchived {
			continue
		}

		// Skip channels we don't care about
		if !targetChannels[channel.Name] {
			continue
		}

		// Get recent messages to check activity
		params := &slack.GetConversationHistoryParameters{
			ChannelID: channel.ID,
			Limit:     10, // Just need a few messages to check activity
			Oldest:    fmt.Sprintf("%.0f", float64(sinceTime.Unix())),
		}

		// Rate limit before checking channel activity
		ctx := context.Background()
		if err := slackRateLimiter.Wait(ctx); err != nil {
			s.logger.Err(err).Error("rate limiter error in activity check")
			continue
		}

		history, err := client.GetConversationHistory(params)
		if err != nil {
			// Skip channels we can't access
			s.logger.KV("channel", channel.Name).KV("error", err.Error()).Error("skipping inaccessible channel")
			continue
		}

		if len(history.Messages) > 0 {
			// Get timestamp of most recent message
			mostRecent := history.Messages[0] // Messages are in reverse chronological order
			timestamp, _ := strconv.ParseFloat(mostRecent.Timestamp, 64)
			lastActivity := time.Unix(int64(timestamp), 0)

			activeChannels = append(activeChannels, channelActivity{
				Channel:      channel,
				MessageCount: len(history.Messages),
				LastActivity: lastActivity,
			})
		}
	}

	// Sort by activity (message count + recency)
	sort.Slice(activeChannels, func(i, j int) bool {
		// Primary sort: more recent activity
		if !activeChannels[i].LastActivity.Equal(activeChannels[j].LastActivity) {
			return activeChannels[i].LastActivity.After(activeChannels[j].LastActivity)
		}
		// Secondary sort: more messages
		return activeChannels[i].MessageCount > activeChannels[j].MessageCount
	})

	// Return top 20 most active channels (or all if fewer than 20)
	maxChannels := 20
	if len(activeChannels) < maxChannels {
		maxChannels = len(activeChannels)
	}

	result := make([]slack.Channel, maxChannels)
	for i := 0; i < maxChannels; i++ {
		result[i] = activeChannels[i].Channel
	}

	s.logger.KV("totalChannels", len(allChannels)).
		KV("activeChannels", len(activeChannels)).
		KV("selectedChannels", len(result)).
		Info("channel activity filtering completed")

	return result
}

// calculateKarmaFromMessage calculates karma points from a message's text content
func (s *Server) calculateKarmaFromMessage(msg slack.Message, toUser string) int {
	textToParse := msg.Text

	// Handle motivates like the main bot does
	if match := karmaReg.MatchMotivate().FindStringSubmatch(textToParse); len(match) > 0 {
		textToParse = match[1] + "++ for doing good work"
	}

	totalPoints := 0

	// Check for give karma patterns (both @user++ and username++)
	if matches := karmaReg.MatchGive().FindAllStringSubmatch(textToParse, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) >= 4 {
				targetUser := strings.TrimPrefix(match[1], "@")

				// Note: User resolution is handled elsewhere in the backfill process

				// Check if this karma is for our target user
				if targetUser == toUser {
					// Parse karma points
					karmaText := match[3]
					points := len(karmaText)
					if strings.Contains(karmaText, "-") {
						points = -points
					}
					totalPoints += points
				}
			}
		}
	}

	return totalPoints
}

// getUserNameByID resolves a user ID to username using Slack API with rate limiting
func (s *Server) getUserNameByID(userID string, slackClient *slack.Client) (string, error) {
	// Try bot first if available (it may have its own caching)
	if s.bot != nil {
		if username, err := s.bot.GetUserNameByID(userID); err == nil {
			return username, nil
		}
	}

	// Fallback to direct Slack API call with rate limiting
	if slackClient != nil {
		// Wait for rate limiter
		ctx := context.Background()
		err := slackRateLimiter.Wait(ctx)
		if err != nil {
			return userID, err
		}

		user, err := slackClient.GetUserInfo(userID)
		if err != nil {
			// Check if it's a rate limit error and log it
			if rateLimitErr, ok := err.(*slack.RateLimitedError); ok {
				s.logger.KV("retryAfter", rateLimitErr.RetryAfter).Error("slack rate limit exceeded in user resolution")
				time.Sleep(rateLimitErr.RetryAfter)
				// Retry once after waiting
				user, err = slackClient.GetUserInfo(userID)
				if err != nil {
					return userID, err
				}
			} else {
				return userID, err // Return user ID if resolution fails
			}
		}
		return user.Name, nil
	}

	return userID, nil // Return user ID if no client available
}

// enrichUsersWithSlackInfo enriches user summaries with Slack profile information
func (s *Server) enrichUsersWithSlackInfo(users []*database.UserSummary) []*database.UserSummary {
	if s.bot == nil {
		return users // Return unchanged if no bot available
	}

	for _, user := range users {
		// Try to get cached Slack user info
		if slackUser, err := s.getCachedSlackUserByUsername(user.Username); err == nil {
			// Enrich with Slack profile data
			user.DisplayName = &slackUser.Profile.DisplayName
			user.RealName = &slackUser.RealName
			if slackUser.Profile.Image192 != "" {
				user.AvatarURL = &slackUser.Profile.Image192
			}
			user.IsBot = slackUser.IsBot
			user.IsDeleted = slackUser.Deleted
		} else {
			// If we can't find the user in Slack, mark them as deleted/invalid
			// This handles corrupted usernames, deleted users, etc.
			user.IsDeleted = true
			user.IsBot = false // Assume not a bot if we can't verify
		}
	}

	return users
}

// getCachedSlackUserByUsername gets a full Slack user from cache or fetches and caches it
func (s *Server) getCachedSlackUserByUsername(username string) (*slack.User, error) {
	if s.bot == nil {
		return nil, fmt.Errorf("bot not available")
	}

	// Skip obviously corrupted usernames
	if strings.Contains(username, "@") || strings.Contains(username, "+++++") || strings.HasPrefix(username, "*") || strings.HasPrefix(username, "<") || strings.HasPrefix(username, ":") || strings.Contains(username, "http") {
		return nil, fmt.Errorf("corrupted username: %s", username)
	}

	// Try to find the user ID from the existing username cache first
	var userID string
	userCacheMu.RLock()
	for id, cachedUsername := range userCache {
		if cachedUsername == username {
			userID = id
			break
		}
	}
	userCacheMu.RUnlock()

	// If we found a user ID, try to get the full profile from cache
	if userID != "" {
		profileCacheMu.RLock()
		if cachedUser, exists := profileCache[userID]; exists {
			if cacheTime, timeExists := profileCacheTime[userID]; timeExists {
				if time.Since(cacheTime) < profileCacheExpiry {
					profileCacheMu.RUnlock()
					return cachedUser, nil
				}
			}
		}
		profileCacheMu.RUnlock()

		// Cache expired or doesn't exist, fetch fresh data
		return s.fetchAndCacheSlackUser(userID)
	}

	// Fallback: try to get user info directly by username (this may not work with all Slack APIs)
	return s.bot.GetSlackUserInfo(username)
}

// fetchAndCacheSlackUser fetches a Slack user by ID and caches the result
func (s *Server) fetchAndCacheSlackUser(userID string) (*slack.User, error) {
	if s.bot == nil {
		return nil, fmt.Errorf("bot not available")
	}

	// Rate limit the API call
	ctx := context.Background()
	if err := slackRateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	// Fetch user info from Slack API
	slackUser, err := s.bot.GetSlackUserInfo(userID)
	if err != nil {
		return nil, err
	}

	// Cache the full profile
	profileCacheMu.Lock()
	profileCache[userID] = slackUser
	profileCacheTime[userID] = time.Now()
	profileCacheMu.Unlock()

	// Also update the username cache
	userCacheMu.Lock()
	userCache[userID] = slackUser.Name
	userCacheTime[userID] = time.Now()
	userCacheMu.Unlock()

	return slackUser, nil
}

// getSlackUserByUsername attempts to find a Slack user by their username
func (s *Server) getSlackUserByUsername(username string) (*slack.User, error) {
	if s.bot == nil {
		return nil, fmt.Errorf("bot not available")
	}

	// Try to get user info directly by username (this may not work with all Slack APIs)
	return s.bot.GetSlackUserInfo(username)
}

// filterActiveUsers filters out bots, deleted users, and users with less than 20 points
func (s *Server) filterActiveUsers(users []*database.UserSummary) []*database.UserSummary {
	filtered := make([]*database.UserSummary, 0, len(users))
	for _, user := range users {
		if !user.IsBot && !user.IsDeleted && user.TotalPoints >= 20 {
			filtered = append(filtered, user)
		}
	}
	return filtered
}

// getCachedUsername gets a username from cache or fetches and caches it
func (s *Server) getCachedUsername(userID string, slackClient *slack.Client) (string, error) {
	// Check cache first
	userCacheMu.RLock()
	if username, exists := userCache[userID]; exists {
		cacheTime, timeExists := userCacheTime[userID]
		if timeExists && time.Since(cacheTime) < cacheTimeout {
			userCacheMu.RUnlock()
			return username, nil
		}
	}
	userCacheMu.RUnlock()

	// Not in cache or expired, fetch from API
	username, err := s.getUserNameByID(userID, slackClient)
	if err != nil {
		return userID, err // Return user ID on error
	}

	// Cache the result
	userCacheMu.Lock()
	userCache[userID] = username
	userCacheTime[userID] = time.Now()
	userCacheMu.Unlock()

	return username, nil
}

// preloadUserCache bulk loads users to reduce API calls
// preloadAllWorkspaceUsers fetches all users from the workspace and caches them
func (s *Server) preloadAllWorkspaceUsers(slackClient *slack.Client) error {
	s.logger.Info("preloading all workspace users for web server")

	// Use GetUsers without parameters for now - simpler approach
	users, err := slackClient.GetUsers()
	if err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	totalUsers := 0
	userCacheMu.Lock()
	for _, user := range users {
		// Skip deleted users and bots
		if !user.Deleted && !user.IsBot {
			userCache[user.ID] = user.Name
			userCacheTime[user.ID] = time.Now()
			totalUsers++
		}
	}
	userCacheMu.Unlock()

	s.logger.KV("users", totalUsers).Info("preloaded workspace users for web server")
	return nil
}

func (s *Server) preloadUserCache(userIDs []string, slackClient *slack.Client) {
	var uncachedIDs []string

	// Check which users are not cached or expired
	userCacheMu.RLock()
	for _, userID := range userIDs {
		if _, exists := userCache[userID]; !exists {
			uncachedIDs = append(uncachedIDs, userID)
		} else if cacheTime, timeExists := userCacheTime[userID]; !timeExists || time.Since(cacheTime) >= cacheTimeout {
			uncachedIDs = append(uncachedIDs, userID)
		}
	}
	userCacheMu.RUnlock()

	// Fetch uncached users in batches (Slack doesn't have bulk user API, so we do sequential with rate limiting)
	for _, userID := range uncachedIDs {
		if username, err := s.getUserNameByID(userID, slackClient); err == nil {
			userCacheMu.Lock()
			userCache[userID] = username
			userCacheTime[userID] = time.Now()
			userCacheMu.Unlock()
		}
		// Small delay between bulk user requests
		time.Sleep(100 * time.Millisecond)
	}
}

// parseKarmaMatch parses a regex match and extracts karma information
func (s *Server) parseKarmaMatch(match []string, msgUser string, isPositive bool, slackClient *slack.Client) (points int, fromUser, toUser, reason string) {
	// Get the from user (message sender) using cache
	var err error
	fromUser, err = s.getCachedUsername(msgUser, slackClient)
	if err != nil {
		s.logger.Err(err).KV("user_id", msgUser).Error("failed to get from_user for backfill, using userID as fallback")
		fromUser = msgUser // Use userID as fallback instead of failing completely
	}

	// Parse the match based on the regex groups
	// Give/Take regex: (<@[A-Za-z0-9]+>)\s*(\+{2,})(\s+for\s+(.+))?|(\S+)\s*(\+{2,})(\s+for\s+(.+))?
	// Groups: 1=@user, 2=+++, 3=for clause, 4=reason, 5=username, 6=+++, 7=for clause, 8=reason

	var targetUser, karmaChars string

	if match[1] != "" {
		// @user format
		targetUser = match[1]
		karmaChars = match[2]
		if match[4] != "" {
			reason = match[4]
		}
	} else if match[5] != "" {
		// username format
		targetUser = match[5]
		karmaChars = match[6]
		if match[8] != "" {
			reason = match[8]
		}
	}

	if targetUser == "" || karmaChars == "" {
		return 0, "", "", ""
	}

	// Calculate points based on the number of + or - characters
	points = len(karmaChars) - 1 // -1 because we need at least 2 chars (++ or --)
	if points < 1 {
		points = 1
	}

	// Parse the target user
	if strings.HasPrefix(targetUser, "<@") && strings.HasSuffix(targetUser, ">") {
		// @user format - extract the user ID and convert to username using cache
		userID := strings.TrimSuffix(strings.TrimPrefix(targetUser, "<@"), ">")
		toUser, err = s.getCachedUsername(userID, slackClient)
		if err != nil {
			s.logger.Err(err).KV("user_id", userID).Error("failed to get to_user for backfill, using userID as fallback")
			toUser = userID // Use userID as fallback instead of failing completely
		}
	} else {
		// username format
		toUser = targetUser
	}

	if reason == "" {
		reason = "backfill"
	}

	return points, fromUser, toUser, reason
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func parseFloat(s string) float64 {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0
	}
	return val
}

// Helper functions
func maskToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 8 {
		return strings.Repeat("*", len(token))
	}
	return token[:4] + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
}

// New API handlers for enhanced statistics

func (s *Server) handleAPIKarmaDistribution(w http.ResponseWriter, r *http.Request) {
	year := 0
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if parsed, err := strconv.Atoi(yearStr); err == nil {
			year = parsed
		}
	}

	// Get leaderboard data
	var leaderboard []*database.UserSummary
	var err error

	if year > 0 {
		leaderboard, err = s.db.GetYearlyLeaderboard(year, 1000) // Get more users for distribution
	} else {
		leaderboard, err = s.db.GetCurrentLeaderboard(1000)
	}

	if err != nil {
		s.logger.Err(err).Error("failed to get leaderboard for karma distribution")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Group users into karma ranges
	distribution := map[string]int{
		"0 to 10":     0,
		"11 to 50":    0,
		"51 to 100":   0,
		"101 to 200":  0,
		"201 to 500":  0,
		"501 to 1000": 0,
		"1000+":       0,
	}

	for _, user := range leaderboard {
		points := user.TotalPoints
		switch {
		case points >= 1000:
			distribution["1000+"]++
		case points >= 501:
			distribution["501 to 1000"]++
		case points >= 201:
			distribution["201 to 500"]++
		case points >= 101:
			distribution["101 to 200"]++
		case points >= 51:
			distribution["51 to 100"]++
		case points >= 11:
			distribution["11 to 50"]++
		default:
			distribution["0 to 10"]++
		}
	}

	// Convert to chart-friendly format
	response := make([]map[string]interface{}, 0, len(distribution))
	for rangeStr, count := range distribution {
		if count > 0 { // Only include ranges with users
			response = append(response, map[string]interface{}{
				"range": rangeStr,
				"count": count,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleAPIActivityTimeline(w http.ResponseWriter, r *http.Request) {
	year := 0
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if parsed, err := strconv.Atoi(yearStr); err == nil {
			year = parsed
		}
	}

	var query string
	var args []interface{}

	query = `
			SELECT
				DATE_TRUNC('month', timestamp) as date,
				SUM(CASE WHEN points > 0 THEN points ELSE 0 END) as positive,
				SUM(CASE WHEN points < 0 THEN ABS(points) ELSE 0 END) as negative,
				COUNT(*) as total
			FROM karma_transactions
			WHERE year = $1
			GROUP BY DATE_TRUNC('month', timestamp) ORDER BY DATE_TRUNC('month', timestamp)
		`
	args = append(args, year)

	rows, err := s.db.SQL.Query(query, args...)
	if err != nil {
		s.logger.Err(err).Error("failed to get activity timeline")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var response []map[string]interface{}
	for rows.Next() {
		var date time.Time
		var positive, negative, total int

		err := rows.Scan(&date, &positive, &negative, &total)
		if err != nil {
			s.logger.Err(err).Error("failed to scan activity timeline row")
			continue
		}

		response = append(response, map[string]interface{}{
			"date":     date.Format("2006-01-02"),
			"positive": positive,
			"negative": negative,
			"total":    total,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleAPIPointsOverTime(w http.ResponseWriter, r *http.Request) {
	year := 0
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if parsed, err := strconv.Atoi(yearStr); err == nil {
			year = parsed
		}
	}

	var query string
	var args []interface{}

	if year > 0 {
		query = `
			SELECT
				EXTRACT(MONTH FROM timestamp) as month,
				SUM(points) as totalPoints
			FROM karma_transactions
			WHERE EXTRACT(YEAR FROM timestamp) = $1
			GROUP BY EXTRACT(MONTH FROM timestamp)
			ORDER BY month
		`
		args = append(args, year)
	} else {
		query = `
			SELECT
				EXTRACT(MONTH FROM timestamp) as month,
				SUM(points) as totalPoints
			FROM karma_transactions
			GROUP BY EXTRACT(MONTH FROM timestamp)
			ORDER BY month
		`
	}

	rows, err := s.db.SQL.Query(query, args...)
	if err != nil {
		s.logger.Err(err).Error("failed to get points over time")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var response []map[string]interface{}
	for rows.Next() {
		var month int
		var totalPoints int

		err := rows.Scan(&month, &totalPoints)
		if err != nil {
			s.logger.Err(err).Error("failed to scan points over time row")
			continue
		}

		response = append(response, map[string]interface{}{
			"month":       month,
			"totalPoints": totalPoints,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// checkBotHealth checks if the janet-bot container is healthy
func (s *Server) checkBotHealth() bool {
	// Try to resolve the janet-bot hostname
	// If the container is running and in the same network, this will succeed
	_, err := net.LookupHost("janet-bot")

	if err != nil {
		// Container not reachable or not running
		return false
	}

	// Additional check: try to ping the container's IP
	conn, err := (&net.Dialer{
		Timeout: 1 * time.Second,
	}).Dial("tcp", "janet-bot:22")

	if err != nil {
		// This is expected since SSH isn't running, but it confirms the container is reachable
		// We only care that we can resolve the hostname above
	} else {
		conn.Close()
	}

	return true
}

// fetchMessagesChunked fetches messages from Slack with proper pagination and rate limiting
func (s *Server) fetchMessagesChunked(client *slack.Client, channelID string, totalLimit int, since, until string) ([]slack.Message, error) {
	const maxPerRequest = 1000  // Slack's maximum limit per request
	const rateLimitDelay = 2000 // milliseconds between requests (30 requests per minute = ~2s for bulk operations)

	var allMessages []slack.Message
	var cursor string
	remaining := totalLimit
	unlimited := totalLimit <= 0

	s.logger.KV("channel", channelID).KV("totalLimit", totalLimit).KV("unlimited", unlimited).Info("starting chunked message fetch")

	for unlimited || remaining > 0 {
		// Calculate chunk size (never exceed Slack's max)
		var chunkSize int
		if unlimited {
			chunkSize = maxPerRequest
		} else {
			chunkSize = remaining
			if chunkSize > maxPerRequest {
				chunkSize = maxPerRequest
			}
		}

		// Prepare request parameters
		params := &slack.GetConversationHistoryParameters{
			ChannelID: channelID,
			Limit:     chunkSize,
			Cursor:    cursor,
		}

		// Apply time filters on all requests (convert dates to Unix timestamps)
		if since != "" {
			if sinceTime, err := time.Parse("2006-01-02", since); err == nil {
				params.Oldest = fmt.Sprintf("%.0f", float64(sinceTime.Unix()))
			} else {
				params.Oldest = since // Assume it's already a timestamp
			}
		}
		if until != "" {
			if untilTime, err := time.Parse("2006-01-02", until); err == nil {
				// Add 24 hours to include the entire until date
				untilTime = untilTime.Add(24 * time.Hour)
				params.Latest = fmt.Sprintf("%.0f", float64(untilTime.Unix()))
			} else {
				params.Latest = until // Assume it's already a timestamp
			}
		}

		s.logger.KV("chunkSize", chunkSize).KV("cursor", cursor).Info("fetching message chunk")

		// Make API request with retry logic
		history, err := s.fetchWithRetry(client, params)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch messages chunk: %w", err)
		}

		// Add messages to our collection
		allMessages = append(allMessages, history.Messages...)
		if !unlimited {
			remaining -= len(history.Messages)
		}

		s.logger.KV("fetched", len(history.Messages)).KV("total", len(allMessages)).KV("remaining", remaining).KV("unlimited", unlimited).Info("chunk processed")

		// Check if we need to continue pagination
		if !history.HasMore || len(history.Messages) == 0 {
			s.logger.Info("no more messages available")
			break
		}

		// Update cursor for next request
		cursor = history.ResponseMetaData.NextCursor

		// Rate limiting - wait between requests to avoid hitting limits
		if remaining > 0 {
			time.Sleep(time.Duration(rateLimitDelay) * time.Millisecond)
		}
	}

	s.logger.KV("totalFetched", len(allMessages)).Info("chunked message fetch completed")
	return allMessages, nil
}

// fetchWithRetry implements retry logic for Slack API requests with rate limiting
func (s *Server) fetchWithRetry(client *slack.Client, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
	maxRetries := 3
	baseDelay := 2000 // milliseconds

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Wait for rate limiter before each API call
		ctx := context.Background()
		if err := slackRateLimiter.Wait(ctx); err != nil {
			return nil, err
		}

		history, err := client.GetConversationHistory(params)
		if err == nil {
			return history, nil
		}

		// Check if it's a rate limit error
		if rateLimitErr, ok := err.(*slack.RateLimitedError); ok {
			waitTime := rateLimitErr.RetryAfter
			s.logger.KV("retryAfter", waitTime).Info("rate limited, waiting")
			time.Sleep(waitTime)
			continue
		}

		// For other errors, use exponential backoff
		if attempt < maxRetries-1 {
			delay := time.Duration(baseDelay*(1<<attempt)) * time.Millisecond
			s.logger.KV("attempt", attempt+1).KV("delay", delay).Err(err).Error("request failed, retrying")
			time.Sleep(delay)
			continue
		}

		return nil, err
	}

	return nil, fmt.Errorf("failed after %d attempts", maxRetries)
}

// BulkBackfillRequest represents a request to backfill multiple channels
type BulkBackfillRequest struct {
	Since         string `json:"since"`         // Start date (e.g., "2022-01-01")
	Until         string `json:"until"`         // End date (optional)
	DryRun        bool   `json:"dryRun"`        // Whether to actually save results
	IncludeEmoji  bool   `json:"includeEmoji"`  // Whether to process emoji reactions
	MaxPoints     int    `json:"maxPoints"`     // Maximum points per karma action
	Limit         int    `json:"limit"`         // Messages per channel (0 = unlimited)
	ChannelFilter string `json:"channelFilter"` // Optional regex filter for channel names
}

// BulkBackfillResponse represents the response from a bulk backfill operation
type BulkBackfillResponse struct {
	Status            string                  `json:"status"`
	TotalChannels     int                     `json:"totalChannels"`
	ProcessedChannels int                     `json:"processedChannels"`
	FailedChannels    int                     `json:"failedChannels"`
	TotalMessages     int                     `json:"totalMessages"`
	TotalKarmaFound   int                     `json:"totalKarmaFound"`
	TotalRecordsAdded int                     `json:"totalRecordsAdded"`
	DryRun            bool                    `json:"dryRun"`
	ChannelResults    []ChannelBackfillResult `json:"channelResults"`
	Errors            []string                `json:"errors,omitempty"`
}

// ChannelBackfillResult represents the result for a single channel
type ChannelBackfillResult struct {
	ChannelID         string `json:"channelId"`
	ChannelName       string `json:"channelName"`
	MessagesProcessed int    `json:"messagesProcessed"`
	KarmaFound        int    `json:"karmaFound"`
	RecordsAdded      int    `json:"recordsAdded"`
	DuplicatesSkipped int    `json:"duplicatesSkipped"`
	ErrorsEncountered int    `json:"errorsEncountered"`
	Error             string `json:"error,omitempty"`
}

// Channel caching functions
func (s *Server) getCachedChannelName(channelID string, slackClient *slack.Client) (string, error) {
	channelCacheMu.RLock()
	if channelName, exists := channelCache[channelID]; exists {
		// Check if the cache entry is still valid
		if cacheTime, timeExists := channelCacheTime[channelID]; timeExists && time.Since(cacheTime) < cacheTimeout {
			channelCacheMu.RUnlock()
			return channelName, nil
		}
	}
	channelCacheMu.RUnlock()

	// Cache miss or expired, fetch from API
	channelName, err := s.getChannelName(channelID, slackClient)
	if err != nil {
		return channelID, err // Return channelID as fallback
	}

	// Update cache
	channelCacheMu.Lock()
	channelCache[channelID] = channelName
	channelCacheTime[channelID] = time.Now()
	channelCacheMu.Unlock()

	return channelName, nil
}

func (s *Server) getChannelName(channelID string, slackClient *slack.Client) (string, error) {
	channel, err := slackClient.GetConversationInfo(&slack.GetConversationInfoInput{
		ChannelID: channelID,
	})
	if err != nil {
		s.logger.Err(err).KV("channel_id", channelID).Error("failed to get channel info")
		return "", err
	}
	return channel.Name, nil
}

// Preload channel cache with batch API call
func (s *Server) preloadChannelCache(slackClient *slack.Client, channelIDs []string) error {
	// Get channels that aren't in cache or are expired
	var channelsToFetch []string
	channelCacheMu.RLock()
	for _, channelID := range channelIDs {
		if _, exists := channelCache[channelID]; !exists {
			channelsToFetch = append(channelsToFetch, channelID)
			continue
		}
		// Check if expired
		if cacheTime, timeExists := channelCacheTime[channelID]; !timeExists || time.Since(cacheTime) >= cacheTimeout {
			channelsToFetch = append(channelsToFetch, channelID)
		}
	}
	channelCacheMu.RUnlock()

	if len(channelsToFetch) == 0 {
		return nil // All channels already cached
	}

	s.logger.KV("channels_to_fetch", len(channelsToFetch)).Info("preloading channel cache")

	// Fetch channels in smaller batches to avoid API limits
	batchSize := 50
	for i := 0; i < len(channelsToFetch); i += batchSize {
		end := i + batchSize
		if end > len(channelsToFetch) {
			end = len(channelsToFetch)
		}
		batch := channelsToFetch[i:end]

		// For each channel in batch, get info individually (Slack doesn't have bulk channel info API)
		for _, channelID := range batch {
			channelName, err := s.getChannelName(channelID, slackClient)
			if err != nil {
				s.logger.Err(err).KV("channel_id", channelID).Error("failed to preload channel info")
				continue
			}

			// Update cache
			channelCacheMu.Lock()
			channelCache[channelID] = channelName
			channelCacheTime[channelID] = time.Now()
			channelCacheMu.Unlock()
		}

		// Rate limiting between batches
		if end < len(channelsToFetch) {
			time.Sleep(1 * time.Second)
		}
	}

	return nil
}

// Bulk cache channels from conversations list (only cache specific channels we care about)
func (s *Server) bulkCacheChannelsFromConversations(conversations []slack.Channel) {
	channelCacheMu.Lock()
	defer channelCacheMu.Unlock()

	// Only cache channels we actually care about for backfill operations

	now := time.Now()
	cachedCount := 0
	for _, channel := range conversations {
		if targetChannels[channel.Name] {
			channelCache[channel.ID] = channel.Name
			channelCacheTime[channel.ID] = now
			cachedCount++
		}
	}

	s.logger.KV("channels_cached", cachedCount).KV("total_channels", len(conversations)).Info("bulk cached targeted channels from conversations")
}
