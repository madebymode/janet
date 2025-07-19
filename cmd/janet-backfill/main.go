package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aybabtme/log"
	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
	"github.com/troyxmccall/janet"
	"github.com/troyxmccall/janet/database"
)

var (
	karmaReg = &janet.KarmaRegexps{}
	regexps  = struct {
		Motivate, GivePoints, TakePoints *regexp.Regexp
	}{
		Motivate:   karmaReg.MatchMotivate(),
		GivePoints: karmaReg.MatchGive(),
		TakePoints: karmaReg.MatchTake(),
	}
	// ReactjiConfig matching main bot configuration
	reactjiConfig = &janet.ReactjiConfig{
		Enabled: true,
		UpVote: janet.StringList{
			// All emojis that award +3 points (matching main bot)
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
)

type BackfillConfig struct {
	DatabaseDriver string
	DatabaseURL    string
	SlackToken     string
	Debug          bool
}

type BackfillService struct {
	config      *BackfillConfig
	db          *database.V2DB
	logger      *log.Log
	slackClient *slack.Client
	backfillSvc *database.BackfillService
}

type BackfillParams struct {
	ChannelID    string
	Since        string
	Until        string
	DryRun       bool
	IncludeEmoji bool
	MaxPoints    int
	Limit        int
}

func main() {
	godotenv.Load()

	var (
		debug        = flag.Bool("debug", false, "Enable debug logging")
		dryRun       = flag.Bool("dry-run", false, "Show what would be backfilled without making changes")
		channelID    = flag.String("channel", "", "Slack channel ID to backfill (required)")
		since        = flag.String("since", "", "Start timestamp (YYYY-MM-DD or YYYY-MM-DD HH:MM:SS)")
		until        = flag.String("until", "", "End timestamp (YYYY-MM-DD or YYYY-MM-DD HH:MM:SS)")
		includeEmoji = flag.Bool("include-emoji", false, "Include emoji reactions in backfill")
		maxPoints    = flag.Int("max-points", 5, "Maximum points per karma transaction")
		limit        = flag.Int("limit", 0, "Maximum number of messages to process (0 = unlimited)")
		listChannels = flag.Bool("list-channels", false, "List all available channels and exit")
		helpAndExit  = flag.Bool("help-and-exit", false, "Internal flag to exit without help")
	)
	flag.Parse()

	// Exit silently if no real parameters provided
	if *helpAndExit {
		os.Exit(0)
	}

	logger := log.KV("service", "janet-backfill")
	if *debug {
		logger = logger.KV("debug", true)
	}

	config := loadConfigFromEnv()
	if *debug {
		config.Debug = true
	}

	service, err := NewBackfillService(config, logger)
	if err != nil {
		logger.Err(err).Fatal("failed to create backfill service")
	}

	if *listChannels {
		if err := service.ListChannels(); err != nil {
			logger.Err(err).Fatal("failed to list channels")
		}
		return
	}

	if *channelID == "" {
		fmt.Fprintf(os.Stderr, "Error: -channel is required\nUse -list-channels to see available channels\n")
		flag.Usage()
		os.Exit(1)
	}

	params := BackfillParams{
		ChannelID:    *channelID,
		Since:        *since,
		Until:        *until,
		DryRun:       *dryRun,
		IncludeEmoji: *includeEmoji,
		MaxPoints:    *maxPoints,
		Limit:        *limit,
	}

	if err := service.RunBackfill(params); err != nil {
		logger.Err(err).Fatal("backfill failed")
	}

	logger.Info("backfill completed successfully")
}

func NewBackfillService(config *BackfillConfig, logger *log.Log) (*BackfillService, error) {
	v2db, err := database.NewV2(&database.Config{
		Driver: config.DatabaseDriver,
		URL:    config.DatabaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	logger.Info("connected to database")

	slackClient := slack.New(config.SlackToken, slack.OptionDebug(config.Debug))
	backfillSvc := database.NewBackfillService(v2db)

	return &BackfillService{
		config:      config,
		db:          v2db,
		logger:      logger,
		slackClient: slackClient,
		backfillSvc: backfillSvc,
	}, nil
}

func (bs *BackfillService) ListChannels() error {
	bs.logger.Info("fetching channel list...")

	var channels []slack.Channel
	err := bs.retryOnRateLimit("GetConversations", func() error {
		var err error
		channels, _, err = bs.slackClient.GetConversations(&slack.GetConversationsParameters{
			Types: []string{"public_channel", "private_channel"},
			Limit: 1000,
		})
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to get channels: %w", err)
	}

	fmt.Println("\nAvailable Slack Channels:")
	fmt.Println("=========================")
	for _, channel := range channels {
		channelType := "public"
		if channel.IsPrivate {
			channelType = "private"
		}
		fmt.Printf("%-20s %-10s #%s\n", channel.ID, channelType, channel.Name)
	}
	fmt.Printf("\nTotal: %d channels\n", len(channels))

	return nil
}

func (bs *BackfillService) RunBackfill(params BackfillParams) error {
	startTime := time.Now()

	bs.logger.KV("channel", params.ChannelID).
		KV("since", params.Since).
		KV("until", params.Until).
		KV("dryRun", params.DryRun).
		KV("includeEmoji", params.IncludeEmoji).
		KV("limit", params.Limit).
		Info("starting backfill operation")

	if params.DryRun {
		bs.logger.Info("DRY RUN MODE - no changes will be made")
	}

	// Get channel name for logging
	var channelInfo *slack.Channel
	err := bs.retryOnRateLimit("GetConversationInfo", func() error {
		var err error
		channelInfo, err = bs.slackClient.GetConversationInfo(&slack.GetConversationInfoInput{
			ChannelID: params.ChannelID,
		})
		return err
	})
	if err != nil {
		bs.logger.Err(err).Error("failed to get channel info, continuing with ID only")
	} else {
		bs.logger.KV("name", channelInfo.Name).Info("backfilling channel")
	}

	// Fetch messages with chunking
	allMessages, err := bs.fetchMessagesChunked(params.ChannelID, params.Limit, params.Since, params.Until)
	if err != nil {
		return fmt.Errorf("failed to fetch messages: %w", err)
	}

	bs.logger.KV("count", len(allMessages)).Info("fetched messages")

	// Process messages
	stats, err := bs.processMessagesForBackfill(allMessages, params)
	if err != nil {
		return fmt.Errorf("failed to process messages: %w", err)
	}

	stats.DurationMs = int(time.Since(startTime).Milliseconds())
	bs.printStats(stats)

	return nil
}

func (bs *BackfillService) fetchMessagesChunked(channelID string, totalLimit int, since, until string) ([]slack.Message, error) {
	const maxPerRequest = 1000  // Slack's maximum limit per request
	const rateLimitDelay = 2000 // milliseconds between requests (30 requests per minute = ~2s for bulk operations)

	var allMessages []slack.Message
	var cursor string
	remaining := totalLimit
	unlimited := totalLimit <= 0

	bs.logger.KV("channel", channelID).KV("totalLimit", totalLimit).KV("unlimited", unlimited).Info("starting chunked message fetch")

	// Parse time parameters
	var oldest, latest string
	if since != "" {
		parsedTime, err := parseTimeInput(since)
		if err != nil {
			return nil, fmt.Errorf("invalid since time: %w", err)
		}
		oldest = fmt.Sprintf("%.6f", float64(parsedTime.Unix()))
	}
	if until != "" {
		parsedTime, err := parseTimeInput(until)
		if err != nil {
			return nil, fmt.Errorf("invalid until time: %w", err)
		}
		latest = fmt.Sprintf("%.6f", float64(parsedTime.Unix()))
	}

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

		cursorDisplay := cursor
		if len(cursor) > 20 {
			cursorDisplay = cursor[:20] + "..."
		}
		bs.logger.KV("fetched", len(allMessages)).
			KV("limit", chunkSize).
			KV("remaining", remaining).
			KV("unlimited", unlimited).
			KV("cursor", cursorDisplay).
			Info("fetching message chunk")

		params := &slack.GetConversationHistoryParameters{
			ChannelID: channelID,
			Cursor:    cursor,
			Limit:     chunkSize,
			Oldest:    oldest,
			Latest:    latest,
		}

		history, err := bs.getConversationHistoryWithRetry(params)
		if err != nil {
			return nil, fmt.Errorf("slack API error: %w", err)
		}

		allMessages = append(allMessages, history.Messages...)

		// Update remaining count if not unlimited
		if !unlimited {
			remaining -= len(history.Messages)
		}

		// Break if no more messages or reached end
		if !history.HasMore || len(history.Messages) == 0 {
			break
		}

		cursor = history.ResponseMetaData.NextCursor

		// Rate limiting for bulk operations
		time.Sleep(rateLimitDelay * time.Millisecond)
	}

	return allMessages, nil
}

type BackfillStats struct {
	MessagesProcessed int
	KarmaFound        int
	RecordsAdded      int
	ErrorsEncountered int
	DurationMs        int
}

func (bs *BackfillService) processMessagesForBackfill(messages []slack.Message, params BackfillParams) (*BackfillStats, error) {
	stats := &BackfillStats{}

	// Preload ALL workspace users instead of just ones from messages
	userCache, err := bs.preloadAllWorkspaceUsers()
	if err != nil {
		bs.logger.Err(err).Error("failed to preload workspace users, falling back to individual user lookups")
		// Fallback to old method if workspace user fetch fails
		userCache = make(map[string]string)
	}

	// Helper function to get username with fallback logic
	getUsernameWithFallback := func(userID string) string {
		if username, exists := userCache[userID]; exists && username != "" {
			return username
		}

		// Fallback: try individual lookup
		username, err := bs.getCachedUsername(userID)
		if err != nil {
			bs.logger.Err(err).KV("userID", userID).Error("failed to get username, using userID as fallback")
			// Last resort: use userID as username
			return userID
		}

		// Cache the result for future use
		userCache[userID] = username
		return username
	}

	for _, msg := range messages {
		stats.MessagesProcessed++

		if msg.BotID != "" || msg.SubType == "bot_message" {
			continue
		}

		// Process karma from message text
		karmaTransactions := bs.parseKarmaFromTextWithFallback(msg.Text, getUsernameWithFallback(msg.User), params.MaxPoints, getUsernameWithFallback)
		for _, transaction := range karmaTransactions {
			stats.KarmaFound++

			// Set message context
			transaction.ChannelID = msg.Channel
			transaction.MessageID = msg.Timestamp
			transaction.Timestamp = time.Unix(int64(parseFloat(msg.Timestamp)), 0)

			if err := bs.processKarmaTransaction(transaction, msg, params, stats); err != nil {
				bs.logger.Err(err).Error("failed to process karma transaction")
				stats.ErrorsEncountered++
			}
		}

		// Process emoji reactions if enabled
		if params.IncludeEmoji {
			for _, reaction := range msg.Reactions {
				if bs.isKarmaEmoji(reaction.Name) {
					// Handle bangbang specially
					if reactjiConfig.RepeatPoints.Contains(reaction.Name) {
						bs.processBangBangReaction(reaction, msg, getUsernameWithFallback, params, stats)
					} else {
						// Handle regular emoji reactions
						for _, reactingUserID := range reaction.Users {
							if reactingUserID == msg.User {
								continue // Skip self-reactions
							}

							stats.KarmaFound++

							transaction := &database.BackfillRecord{
								FromUser:        getUsernameWithFallback(reactingUserID),
								ToUser:          getUsernameWithFallback(msg.User),
								Points:          bs.getEmojiPoints(reaction.Name),
								Reason:          fmt.Sprintf("added a :%s: emoji", reaction.Name),
								TransactionType: "reactji",
								EmojiName:       &reaction.Name,
								ChannelID:       msg.Channel,
								MessageID:       msg.Timestamp,
								Timestamp:       time.Unix(int64(parseFloat(msg.Timestamp)), 0),
							}

							if err := bs.processKarmaTransaction(transaction, msg, params, stats); err != nil {
								bs.logger.Err(err).Error("failed to process emoji karma")
								stats.ErrorsEncountered++
							}
						}
					}
				}
			}
		}
	}

	return stats, nil
}

func (bs *BackfillService) processKarmaTransaction(transaction *database.BackfillRecord, msg slack.Message, params BackfillParams, stats *BackfillStats) error {
	if params.DryRun {
		bs.logger.KV("from", transaction.FromUser).
			KV("to", transaction.ToUser).
			KV("points", transaction.Points).
			KV("reason", transaction.Reason).
			KV("type", transaction.TransactionType).
			Info("would add karma transaction")
		return nil
	}

	// Insert the record
	if err := bs.backfillSvc.InsertBackfillRecord(transaction); err != nil {
		return err
	}

	stats.RecordsAdded++
	return nil
}

// processBangBangReaction handles bangbang reactions that repeat the points for all intended users
func (bs *BackfillService) processBangBangReaction(reaction slack.ItemReaction, msg slack.Message, getUsernameFunc func(string) string, params BackfillParams, stats *BackfillStats) {
	// Parse the original message text to find all karma mentions
	textToParse := msg.Text

	// Handle motivates like the main bot does
	if match := regexps.Motivate.FindStringSubmatch(textToParse); len(match) > 0 {
		textToParse = match[1] + "++ for doing good work"
	}

	bs.logger.KV("message_text", textToParse).KV("message_id", msg.Timestamp).Info("processing standalone bangbang for backfill")

	// Check for give karma patterns (both @user++ and username++)
	var allKarmaMatches []struct {
		match       []string
		isGiveKarma bool
		isTakeKarma bool
	}

	if giveMatches := regexps.GivePoints.FindAllStringSubmatch(textToParse, -1); len(giveMatches) > 0 {
		for _, match := range giveMatches {
			allKarmaMatches = append(allKarmaMatches, struct {
				match       []string
				isGiveKarma bool
				isTakeKarma bool
			}{match, true, false})
		}
	}

	if takeMatches := regexps.TakePoints.FindAllStringSubmatch(textToParse, -1); len(takeMatches) > 0 {
		for _, match := range takeMatches {
			allKarmaMatches = append(allKarmaMatches, struct {
				match       []string
				isGiveKarma bool
				isTakeKarma bool
			}{match, false, true})
		}
	}

	bs.logger.KV("karma_matches_found", len(allKarmaMatches)).Info("standalone bangbang karma matches in backfill")

	// Process each user who reacted with bangbang
	for _, reactingUserID := range reaction.Users {
		// Skip self-karma with message author
		if reactingUserID == msg.User {
			continue
		}

		// Get reactor username using fallback function
		fromUser := getUsernameFunc(reactingUserID)

		// Process each karma match found in the original message
		for _, karmaMatch := range allKarmaMatches {
			// Parse karma information from the match
			points, toUser, reason := bs.parseKarmaMatchWithFallback(karmaMatch.match, karmaMatch.isGiveKarma, params.MaxPoints, getUsernameFunc)
			if toUser == "" {
				stats.ErrorsEncountered++
				continue
			}

			// Skip self-reactions (reactor giving karma to themselves)
			if fromUser == toUser {
				continue
			}

			// Apply bangbang effect (double the points)
			bangbangPoints := points
			if karmaMatch.isTakeKarma {
				bangbangPoints = -bangbangPoints
			}

			// Apply max points limit
			if bangbangPoints > params.MaxPoints {
				bangbangPoints = params.MaxPoints
			} else if bangbangPoints < -params.MaxPoints {
				bangbangPoints = -params.MaxPoints
			}

			stats.KarmaFound++

			bangbangReason := fmt.Sprintf("%s added a :bangbang: emoji (doubling existing %d points)", fromUser, points)
			if reason != "" && reason != "good work" && reason != "not so good work" {
				bangbangReason = fmt.Sprintf("%s added a :bangbang: emoji (doubling existing %d points for %s)", fromUser, points, reason)
			}

			bs.logger.KV("from", fromUser).KV("to", toUser).KV("points", bangbangPoints).KV("reason", bangbangReason).Info("standalone bangbang backfill transaction")

			transaction := &database.BackfillRecord{
				FromUser:        fromUser,
				ToUser:          toUser,
				Points:          bangbangPoints,
				Reason:          bangbangReason,
				TransactionType: "reactji",
				EmojiName:       &reaction.Name,
				ChannelID:       msg.Channel,
				MessageID:       msg.Timestamp,
				Timestamp:       time.Unix(int64(parseFloat(msg.Timestamp)), 0),
			}

			if err := bs.processKarmaTransaction(transaction, msg, params, stats); err != nil {
				bs.logger.Err(err).Error("failed to process standalone bangbang karma")
				stats.ErrorsEncountered++
			}
		}
	}
}

// calculateKarmaFromMessage is deprecated - bangbang now splits karma among intended users
// This function is kept for legacy compatibility but should not be used
func (bs *BackfillService) calculateKarmaFromMessage(msg slack.Message, messageAuthor string) int {
	bs.logger.Info("calculateKarmaFromMessage called - this function is deprecated, bangbang should split karma among intended users")
	return 0 // Return 0 to indicate no karma should be processed this way
}

func (bs *BackfillService) parseKarmaFromText(text, fromUser string, maxPoints int) []*database.BackfillRecord {
	// Legacy function for compatibility - uses old method
	return bs.parseKarmaFromTextWithFallback(text, fromUser, maxPoints, nil)
}

func (bs *BackfillService) parseKarmaFromTextWithFallback(text, fromUser string, maxPoints int, getUsernameFunc func(string) string) []*database.BackfillRecord {
	var transactions []*database.BackfillRecord

	// Handle motivates like the main bot does
	if match := regexps.Motivate.FindStringSubmatch(text); len(match) > 0 {
		text = match[1] + "++ for doing good work"
	}

	// Check for give karma patterns (both @user++ and username++)
	if matches := regexps.GivePoints.FindAllStringSubmatch(text, -1); len(matches) > 0 {
		for _, match := range matches {
			points, toUser, reason := bs.parseKarmaMatchWithFallback(match, true, maxPoints, getUsernameFunc)
			if toUser == "" {
				continue
			}

			transaction := &database.BackfillRecord{
				FromUser:        fromUser,
				ToUser:          toUser,
				Points:          points,
				Reason:          reason,
				TransactionType: "message",
				ChannelID:       "",          // Will be set by caller
				MessageID:       "",          // Will be set by caller
				Timestamp:       time.Time{}, // Will be set by caller
			}
			transactions = append(transactions, transaction)
		}
	}

	// Check for take karma patterns (both @user-- and username--)
	if matches := regexps.TakePoints.FindAllStringSubmatch(text, -1); len(matches) > 0 {
		for _, match := range matches {
			points, toUser, reason := bs.parseKarmaMatchWithFallback(match, false, maxPoints, getUsernameFunc)
			if toUser == "" {
				continue
			}

			transaction := &database.BackfillRecord{
				FromUser:        fromUser,
				ToUser:          toUser,
				Points:          -points, // Negative for take karma
				Reason:          reason,
				TransactionType: "message",
				ChannelID:       "",          // Will be set by caller
				MessageID:       "",          // Will be set by caller
				Timestamp:       time.Time{}, // Will be set by caller
			}
			transactions = append(transactions, transaction)
		}
	}

	return transactions
}

func (bs *BackfillService) parseKarmaMatch(match []string, isPositive bool, maxPoints int) (points int, toUser, reason string) {
	// Legacy function for compatibility - uses old method
	return bs.parseKarmaMatchWithFallback(match, isPositive, maxPoints, nil)
}

func (bs *BackfillService) parseKarmaMatchWithFallback(match []string, isPositive bool, maxPoints int, getUsernameFunc func(string) string) (points int, toUser, reason string) {
	// Parse the match based on the regex groups
	// Give/Take regex: (<@[A-Za-z0-9]+>)\s*(\+{2,})(\s+for\s+(.+))?|(\S+)\s*(\+{2,})(\s+for\s+(.+))?
	// Groups: 1=@user, 2=+++, 3=for clause, 4=reason, 5=username, 6=+++, 7=for clause, 8=reason

	var targetUser, karmaChars string

	if match[1] != "" {
		// @user format - extract user ID from <@U1234567> format
		targetUser = match[1]
		karmaChars = match[2]
		if match[4] != "" {
			reason = match[4]
		}

		// Convert <@U1234567> to username
		if strings.HasPrefix(targetUser, "<@") && strings.HasSuffix(targetUser, ">") {
			userID := targetUser[2 : len(targetUser)-1]

			if getUsernameFunc != nil {
				// Use fallback function which handles errors gracefully
				toUser = getUsernameFunc(userID)
			} else {
				// Legacy behavior - fail hard on username resolution errors
				username, err := bs.getCachedUsername(userID)
				if err != nil {
					bs.logger.Err(err).KV("userID", userID).Error("failed to get username for @user karma")
					return 0, "", ""
				}
				toUser = username
			}
		}
	} else if match[5] != "" {
		// username format
		targetUser = match[5]
		karmaChars = match[6]
		if match[8] != "" {
			reason = match[8]
		}
		toUser = targetUser
	}

	// Calculate points from karma characters (++ = 1, +++ = 2, etc.)
	points = len(karmaChars) - 1
	if points > maxPoints {
		points = maxPoints
	}

	// Set default reason if empty
	if reason == "" {
		if isPositive {
			reason = "good work"
		} else {
			reason = "not so good work"
		}
	}

	return points, toUser, reason
}

func (bs *BackfillService) isKarmaEmoji(emojiName string) bool {
	// Check if emoji is in any of the configured reactji categories
	return reactjiConfig.UpVote.Contains(emojiName) ||
		reactjiConfig.DownVote.Contains(emojiName) ||
		reactjiConfig.RepeatPoints.Contains(emojiName)
}

func (bs *BackfillService) getEmojiPoints(emojiName string) int {
	// Use ReactjiConfig to determine points (matching main bot logic)
	switch {
	case reactjiConfig.UpVote.Contains(emojiName):
		return 3 // Matching main bot: +3 points for upvote emojis
	case reactjiConfig.DownVote.Contains(emojiName):
		return -1 // Matching main bot: -1 point for downvote emojis
	case reactjiConfig.RepeatPoints.Contains(emojiName):
		// Bangbang should not use this function - it has special handling
		bs.logger.Info("bangbang emoji should be handled specially, not with fixed points")
		return 3 // Fallback, but this shouldn't be reached
	default:
		// Unknown emoji - shouldn't happen if isKarmaEmoji is used properly
		return 0
	}
}

func (bs *BackfillService) getCachedUsername(userID string) (string, error) {
	var user *slack.User
	err := bs.retryOnRateLimit("GetUserInfo", func() error {
		var err error
		user, err = bs.slackClient.GetUserInfo(userID)
		return err
	})
	if err != nil {
		return "", err
	}
	return user.Name, nil
}

// preloadAllWorkspaceUsers fetches all users from the workspace and returns a map[userID]username
func (bs *BackfillService) preloadAllWorkspaceUsers() (map[string]string, error) {
	bs.logger.Info("preloading all workspace users")

	userCache := make(map[string]string)

	// Use GetUsers without parameters for now - simpler approach
	var users []slack.User
	err := bs.retryOnRateLimit("GetUsers", func() error {
		var err error
		users, err = bs.slackClient.GetUsers()
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	totalUsers := 0
	for _, user := range users {
		// Skip deleted users and bots
		if !user.Deleted && !user.IsBot {
			userCache[user.ID] = user.Name
			totalUsers++
		}
	}

	bs.logger.KV("users", totalUsers).Info("preloaded workspace users")
	return userCache, nil
}

func (bs *BackfillService) printStats(stats *BackfillStats) {
	bs.logger.Info("=== Backfill Statistics ===")
	bs.logger.KV("count", stats.MessagesProcessed).Info("messages processed")
	bs.logger.KV("count", stats.KarmaFound).Info("karma found")
	bs.logger.KV("count", stats.RecordsAdded).Info("records added")
	bs.logger.KV("count", stats.ErrorsEncountered).Info("errors encountered")
	bs.logger.KV("ms", stats.DurationMs).Info("duration")

	if stats.KarmaFound > 0 {
		successRate := float64(stats.RecordsAdded) / float64(stats.KarmaFound) * 100
		bs.logger.KV("percent", fmt.Sprintf("%.1f", successRate)).Info("success rate")
	}
}

func parseTimeInput(timeStr string) (time.Time, error) {
	// Try different time formats
	formats := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %s", timeStr)
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// retryOnRateLimit executes a function with exponential backoff on rate limits
func (bs *BackfillService) retryOnRateLimit(operation string, fn func() error) error {
	const maxRetries = 5
	const baseDelay = 1 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := fn()
		if err != nil {
			// Check if it's a rate limit error (429)
			if slackErr, ok := err.(*slack.SlackErrorResponse); ok {
				if slackErr.Err == "rate_limited" || strings.Contains(slackErr.Err, "429") {
					// Calculate exponential backoff delay
					delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))

					bs.logger.KV("operation", operation).
						KV("attempt", attempt+1).
						KV("maxRetries", maxRetries).
						KV("delaySeconds", delay.Seconds()).
						KV("error", slackErr.Err).
						Info("rate limited by Slack API, retrying with exponential backoff")

					time.Sleep(delay)
					continue
				}
			}

			// Check for HTTP 429 in error message
			if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "Too Many Requests") {
				delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))

				bs.logger.KV("operation", operation).
					KV("attempt", attempt+1).
					KV("maxRetries", maxRetries).
					KV("delaySeconds", delay.Seconds()).
					KV("error", err.Error()).
					Info("HTTP 429 Too Many Requests, retrying with exponential backoff")

				time.Sleep(delay)
				continue
			}

			// For other errors, return immediately
			return err
		}

		// Success
		if attempt > 0 {
			bs.logger.KV("operation", operation).KV("attempt", attempt+1).Info("successfully recovered from rate limit")
		}
		return nil
	}

	return fmt.Errorf("max retries (%d) exceeded for Slack API operation: %s", maxRetries, operation)
}

// getConversationHistoryWithRetry calls Slack API with exponential backoff on rate limits
func (bs *BackfillService) getConversationHistoryWithRetry(params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
	var history *slack.GetConversationHistoryResponse
	err := bs.retryOnRateLimit("GetConversationHistory", func() error {
		var err error
		history, err = bs.slackClient.GetConversationHistory(params)
		return err
	})
	return history, err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func loadConfigFromEnv() *BackfillConfig {
	return &BackfillConfig{
		DatabaseDriver: getEnv("JANET_DATABASE_DRIVER", "postgres"),
		DatabaseURL:    getEnv("JANET_DATABASE_URL", "postgres://janet:password@localhost:5432/janet?sslmode=disable"),
		SlackToken:     getEnv("JANET_SLACK_TOKEN", ""),
		Debug:          parseBool(getEnv("JANET_DEBUG", "false")),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseBool(s string) bool {
	return strings.ToLower(s) == "true" || s == "1" || strings.ToLower(s) == "yes"
}
