package slack

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/slack-go/slack"
	"github.com/troyxmccall/janet"
	"github.com/troyxmccall/janet/database"
)

// Service handles Slack integration operations
type Service struct {
	bot           *janet.Bot
	slackClient   *slack.Client
	channelCache  []slack.Channel
	cacheFetchedAt time.Time
	messageCache   map[string]messageCacheEntry
	messageCacheMu sync.RWMutex
	attachmentsDir string
	attachmentsURL string
	slackToken     string
}

var ignoredPlusRegex = regexp.MustCompile(`\+{4,}`)

// NewService creates a new Slack service
func NewService(bot *janet.Bot, opts ServiceOptions) *Service {
	return &Service{
		bot:            bot,
		slackClient:    nil,
		messageCache:   make(map[string]messageCacheEntry),
		attachmentsDir: opts.AttachmentsDir,
		attachmentsURL: strings.TrimRight(opts.AttachmentsURL, "/"),
		slackToken:     opts.SlackToken,
	}
}

// NewWebService creates a new Slack service with a standalone client for web-only mode
func NewWebService(slackClient *slack.Client, opts ServiceOptions) *Service {
	return &Service{
		bot:            nil,
		slackClient:    slackClient,
		messageCache:   make(map[string]messageCacheEntry),
		attachmentsDir: opts.AttachmentsDir,
		attachmentsURL: strings.TrimRight(opts.AttachmentsURL, "/"),
		slackToken:     opts.SlackToken,
	}
}

// ServiceOptions controls Slack metadata enrichment behavior.
type ServiceOptions struct {
	AttachmentsDir string
	AttachmentsURL string
	SlackToken     string
}

type messageCacheEntry struct {
	details   *MessageDetails
	expiresAt time.Time
}

func (s *Service) getCachedMessageDetails(channelID, messageID string) (*MessageDetails, bool) {
	key := channelID + ":" + messageID
	s.messageCacheMu.RLock()
	entry, ok := s.messageCache[key]
	s.messageCacheMu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.details, true
}

func (s *Service) setCachedMessageDetails(channelID, messageID string, details *MessageDetails) {
	key := channelID + ":" + messageID
	s.messageCacheMu.Lock()
	s.messageCache[key] = messageCacheEntry{
		details:   details,
		expiresAt: time.Now().Add(15 * time.Minute),
	}
	s.messageCacheMu.Unlock()
}

// EnrichUsersWithSlackInfo enriches user summaries with Slack profile information
func (s *Service) EnrichUsersWithSlackInfo(users []*database.UserSummary) []*database.UserSummary {
	if s.bot == nil {
		return users // Return unchanged if no bot available
	}

	for _, user := range users {
		// Try to get Slack user info
		if slackUser, err := s.getSlackUserByUsername(user.Username); err == nil {
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

// getSlackUserByUsername attempts to find a Slack user by their username
func (s *Service) getSlackUserByUsername(username string) (*slack.User, error) {
	if s.bot == nil {
		return nil, fmt.Errorf("bot not available")
	}

	// Skip obviously corrupted usernames
	if strings.Contains(username, "@") || strings.Contains(username, "+++++") ||
	   strings.HasPrefix(username, "*") || strings.HasPrefix(username, "<") ||
	   strings.HasPrefix(username, ":") || strings.Contains(username, "http") {
		return nil, fmt.Errorf("corrupted username: %s", username)
	}

	// Try to get user info directly by username
	return s.bot.GetSlackUserInfo(username)
}

// GetMessagePermalink gets a permalink to a Slack message
func (s *Service) GetMessagePermalink(channelID, messageID string) (string, error) {
	var client *slack.Client

	// Use standalone client if available, otherwise use bot's client
	if s.slackClient != nil {
		client = s.slackClient
	} else if s.bot != nil && s.bot.Config.SlackWebClient != nil {
		client = s.bot.Config.SlackWebClient
	} else {
		return "", fmt.Errorf("no slack client available")
	}

	params := &slack.PermalinkParameters{
		Channel: channelID,
		Ts:      messageID,
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		permalink, err := client.GetPermalink(params)
		if err == nil {
			return permalink, nil
		}
		if rateLimitErr, ok := err.(*slack.RateLimitedError); ok {
			time.Sleep(rateLimitErr.RetryAfter + (1 * time.Second))
			lastErr = err
			continue
		}
		return "", err
	}

	return "", lastErr
}

// GetMessageText fetches the text content of a Slack message
func (s *Service) GetMessageText(channelID, messageID string) (string, error) {
	var client *slack.Client

	// Use standalone client if available, otherwise use bot's client
	if s.slackClient != nil {
		client = s.slackClient
	} else if s.bot != nil && s.bot.Config.SlackWebClient != nil {
		client = s.bot.Config.SlackWebClient
	} else {
		return "", fmt.Errorf("no slack client available")
	}

	var history *slack.GetConversationHistoryResponse
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		// Fetch conversation history for this specific message
		history, err = client.GetConversationHistory(&slack.GetConversationHistoryParameters{
			ChannelID: channelID,
			Latest:    messageID,
			Inclusive: true,
			Limit:     1,
		})
		if err == nil {
			break
		}
		if rateLimitErr, ok := err.(*slack.RateLimitedError); ok {
			time.Sleep(rateLimitErr.RetryAfter + (1 * time.Second))
			continue
		}
		return "", fmt.Errorf("failed to get conversation history: %w", err)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get conversation history: %w", err)
	}

	if len(history.Messages) == 0 {
		return "", fmt.Errorf("message not found")
	}

	return history.Messages[0].Text, nil
}

// MessageDetails captures Slack message metadata used in the UI.
type MessageDetails struct {
	Text         string
	Permalink    string
	AuthorID     string
	AuthorName   string
	AuthorAvatar string
	ImageURL     string
	AttachmentURL  string
	AttachmentMime string
	HasImage     bool
	HasAttachment bool
	IsReply      bool
	IsIgnored    bool
	ReactionCount int
}

// GetMessageDetails fetches message text, permalink, author, and image URL.
func (s *Service) GetMessageDetails(channelID, messageID string) (*MessageDetails, error) {
	if details, ok := s.getCachedMessageDetails(channelID, messageID); ok {
		return details, nil
	}

	permalink, err := s.GetMessagePermalink(channelID, messageID)
	if err != nil {
		return nil, err
	}

	message, err := s.getMessage(channelID, messageID)
	if err != nil {
		return nil, err
	}

	details := &MessageDetails{
		Text:      message.Text,
		Permalink: permalink,
	}
	if message.ThreadTimestamp != "" && message.ThreadTimestamp != message.Timestamp {
		details.IsReply = true
	}
	if ignoredPlusRegex.MatchString(message.Text) {
		details.IsIgnored = true
	}

	imageURL, hasImage := s.getMessageImageURL(message)
	details.HasImage = hasImage
	if imageURL != "" {
		details.ImageURL = imageURL
	}

	attachmentURL, attachmentMime, hasAttachment := s.getMessageAttachment(message)
	details.HasAttachment = hasAttachment
	if attachmentURL != "" {
		details.AttachmentURL = attachmentURL
		details.AttachmentMime = attachmentMime
	}

	details.ReactionCount = countMessageReactions(message)

	if message.User != "" {
		details.AuthorID = message.User
		if user, err := s.getUserInfoWithRetry(message.User); err == nil {
			details.AuthorName = user.Name
			details.AuthorAvatar = user.Profile.Image72
		}
	} else if message.Username != "" {
		details.AuthorName = message.Username
	}

	s.setCachedMessageDetails(channelID, messageID, details)
	return details, nil
}

func (s *Service) getMessage(channelID, messageID string) (slack.Message, error) {
	var client *slack.Client

	if s.slackClient != nil {
		client = s.slackClient
	} else if s.bot != nil && s.bot.Config.SlackWebClient != nil {
		client = s.bot.Config.SlackWebClient
	} else {
		return slack.Message{}, fmt.Errorf("no slack client available")
	}

	var history *slack.GetConversationHistoryResponse
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		history, err = client.GetConversationHistory(&slack.GetConversationHistoryParameters{
			ChannelID: channelID,
			Latest:    messageID,
			Inclusive: true,
			Limit:     1,
		})
		if err == nil {
			break
		}
		if rateLimitErr, ok := err.(*slack.RateLimitedError); ok {
			time.Sleep(rateLimitErr.RetryAfter + (1 * time.Second))
			continue
		}
		return slack.Message{}, fmt.Errorf("failed to get conversation history: %w", err)
	}
	if err != nil {
		return slack.Message{}, fmt.Errorf("failed to get conversation history: %w", err)
	}

	if len(history.Messages) == 0 {
		return slack.Message{}, fmt.Errorf("message not found")
	}

	return history.Messages[0], nil
}

func (s *Service) getUserInfoWithRetry(userID string) (*slack.User, error) {
	var client *slack.Client

	if s.slackClient != nil {
		client = s.slackClient
	} else if s.bot != nil && s.bot.Config.SlackWebClient != nil {
		client = s.bot.Config.SlackWebClient
	} else {
		return nil, fmt.Errorf("no slack client available")
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		user, err := client.GetUserInfo(userID)
		if err == nil {
			return user, nil
		}
		if rateLimitErr, ok := err.(*slack.RateLimitedError); ok {
			time.Sleep(rateLimitErr.RetryAfter + (1 * time.Second))
			lastErr = err
			continue
		}
		return nil, err
	}

	return nil, lastErr
}

func (s *Service) getMessageImageURL(message slack.Message) (string, bool) {
	for _, file := range message.Files {
		if !strings.HasPrefix(file.Mimetype, "image/") {
			continue
		}
		hasImage := true
		if localURL := s.downloadSlackFile(file); localURL != "" {
			return localURL, hasImage
		}
		if file.PermalinkPublic != "" {
			return file.PermalinkPublic, hasImage
		}
		return "", hasImage
	}

	for _, attachment := range message.Attachments {
		if attachment.ImageURL != "" {
			return attachment.ImageURL, true
		}
		if attachment.ThumbURL != "" {
			return attachment.ThumbURL, true
		}
	}

	return "", false
}

func (s *Service) getMessageAttachment(message slack.Message) (string, string, bool) {
	for _, file := range message.Files {
		if strings.HasPrefix(file.Mimetype, "image/") {
			continue
		}
		if !strings.HasPrefix(file.Mimetype, "video/") && !strings.HasPrefix(file.Mimetype, "audio/") {
			continue
		}
		hasAttachment := true
		if localURL := s.downloadSlackFile(file); localURL != "" {
			return localURL, file.Mimetype, hasAttachment
		}
		if file.PermalinkPublic != "" {
			return file.PermalinkPublic, file.Mimetype, hasAttachment
		}
		return "", file.Mimetype, hasAttachment
	}

	return "", "", false
}

func (s *Service) downloadSlackFile(file slack.File) string {
	if s.attachmentsDir == "" || s.attachmentsURL == "" {
		return ""
	}
	if s.slackToken == "" {
		return ""
	}

	downloadURL := file.URLPrivateDownload
	if downloadURL == "" {
		downloadURL = file.URLPrivate
	}
	if downloadURL == "" {
		return ""
	}

	filename := s.buildAttachmentFilename(file)
	if filename == "" {
		return ""
	}

	localURL, err := s.downloadFile(downloadURL, filename)
	if err != nil {
		return ""
	}

	return localURL
}

func countMessageReactions(message slack.Message) int {
	total := 0
	for _, reaction := range message.Reactions {
		total += reaction.Count
	}
	return total
}

func (s *Service) buildAttachmentFilename(file slack.File) string {
	ext := sanitizeExtension(file.Filetype)
	if ext == "" {
		ext = "img"
	}

	if file.ID == "" {
		return ""
	}

	return file.ID + "." + ext
}

func sanitizeExtension(ext string) string {
	ext = strings.ToLower(ext)
	for _, r := range ext {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			continue
		}
		return ""
	}
	return ext
}

func (s *Service) downloadFile(url, filename string) (string, error) {
	if err := os.MkdirAll(s.attachmentsDir, 0o755); err != nil {
		return "", err
	}

	destPath := filepath.Join(s.attachmentsDir, filename)
	if _, err := os.Stat(destPath); err == nil {
		return s.attachmentsURL + "/" + filename, nil
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.slackToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected status downloading file: %s", resp.Status)
	}

	tmpPath := destPath + ".part"
	fileHandle, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}
	defer fileHandle.Close()

	if _, err := io.Copy(fileHandle, resp.Body); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}

	return s.attachmentsURL + "/" + filename, nil
}

// FindChannelByMessageID finds the channel_id for a message by checking all accessible channels
// This is a fallback when channel_id is not stored in the database
func (s *Service) FindChannelByMessageID(messageID string) (string, error) {
	var client *slack.Client

	// Use standalone client if available, otherwise use bot's client
	if s.slackClient != nil {
		client = s.slackClient
	} else if s.bot != nil && s.bot.Config.SlackWebClient != nil {
		client = s.bot.Config.SlackWebClient
	} else {
		return "", fmt.Errorf("no slack client available")
	}

	// Use cached channel list if available and recent (within 5 minutes)
	// This prevents hitting rate limits by re-fetching the channel list for every message
	var allChannels []slack.Channel
	cacheAge := time.Since(s.cacheFetchedAt)

	if len(s.channelCache) > 0 && cacheAge < 5*time.Minute {
		// Use cached channels
		allChannels = s.channelCache
	} else {
		// Fetch fresh channel list with pagination
		cursor := ""

		for {
			params := &slack.GetConversationsParameters{
				Types:           []string{"public_channel", "private_channel"},
				ExcludeArchived: false,
				Limit:           200,
				Cursor:          cursor,
			}

			channels, nextCursor, err := client.GetConversations(params)
			if err != nil {
				// Handle rate limit errors
				if rateLimitErr, ok := err.(*slack.RateLimitedError); ok {
					// Wait for the retry after duration plus buffer
					waitDuration := rateLimitErr.RetryAfter + (1 * time.Second)
					time.Sleep(waitDuration)
					// Retry this page (don't advance cursor)
					continue
				}
				return "", fmt.Errorf("failed to list channels: %w", err)
			}

			allChannels = append(allChannels, channels...)

			if nextCursor == "" {
				break
			}
			cursor = nextCursor

			// Add small delay between pagination requests to avoid rate limits
			time.Sleep(100 * time.Millisecond)
		}

		// Cache the channel list
		s.channelCache = allChannels
		s.cacheFetchedAt = time.Now()
	}

	// Try to fetch the message from each channel using the exact timestamp
	// This mirrors how the bot fetches messages: GetConversationHistory with Latest=timestamp
	for i := 0; i < len(allChannels); i++ {
		channel := allChannels[i]

		// Rate limiting: Slack allows ~50 requests/min for Tier 3 methods (conversations.history)
		// Add a small delay between requests to avoid hitting limits
		// 100ms delay = max 600 requests/min, well under the limit
		if i > 0 {
			time.Sleep(100 * time.Millisecond)
		}

		history, err := client.GetConversationHistory(&slack.GetConversationHistoryParameters{
			ChannelID: channel.ID,
			Latest:    messageID,  // Use message timestamp as Latest
			Inclusive: true,       // Include the message at Latest timestamp
			Limit:     1,
		})

		if err != nil {
			// Check if it's a rate limit error
			if rateLimitErr, ok := err.(*slack.RateLimitedError); ok {
				// Wait for the retry after duration plus a small buffer
				time.Sleep(rateLimitErr.RetryAfter + (1 * time.Second))
				// Retry this channel
				i--
				continue
			}
			// Channel might be inaccessible or other error, skip it
			continue
		}

		// Check if we found the exact message by comparing timestamps
		if len(history.Messages) > 0 && history.Messages[0].Timestamp == messageID {
			// Found it! Return the channel ID
			return channel.ID, nil
		}
	}

	return "", fmt.Errorf("message not found in any accessible channel")
}

// FindChannelByMessageAuthorAndTimestamp searches Slack for a message by author and timestamp.
// Requires a user token with search:read scope.
func (s *Service) FindChannelByMessageAuthorAndTimestamp(authorUsername, messageID string) (string, error) {
	if authorUsername == "" {
		return "", fmt.Errorf("author username required")
	}

	var client *slack.Client
	if s.slackClient != nil {
		client = s.slackClient
	} else if s.bot != nil && s.bot.Config.SlackWebClient != nil {
		client = s.bot.Config.SlackWebClient
	} else {
		return "", fmt.Errorf("no slack client available")
	}

	ts, err := strconv.ParseFloat(messageID, 64)
	if err != nil {
		return "", fmt.Errorf("invalid message timestamp: %w", err)
	}

	msgTime := time.Unix(int64(ts), 0)
	after := msgTime.Add(-24 * time.Hour).Format("2006-01-02")
	before := msgTime.Add(24 * time.Hour).Format("2006-01-02")
	query := fmt.Sprintf("from:%s after:%s before:%s", authorUsername, after, before)

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		result, err := client.SearchMessages(query, slack.SearchParameters{
			Count:     50,
			Page:      1,
			Highlight: false,
		})
		if err != nil {
			if rateLimitErr, ok := err.(*slack.RateLimitedError); ok {
				time.Sleep(rateLimitErr.RetryAfter + (1 * time.Second))
				lastErr = err
				continue
			}
			return "", err
		}

		for _, match := range result.Matches {
			if match.Timestamp == messageID {
				if match.Channel.ID != "" {
					msg, err := s.getMessage(match.Channel.ID, messageID)
					if err != nil {
						continue
					}
					if len(msg.Reactions) > 0 {
						return match.Channel.ID, nil
					}
				}
			}
		}

		return "", fmt.Errorf("message not found in search results")
	}

	return "", lastErr
}
