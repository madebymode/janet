package slack

import (
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/aybabtme/log"
	goslack "github.com/slack-go/slack"
	"github.com/troyxmccall/janet"
)

type slackClient interface {
	GetUsers(...goslack.GetUsersOption) ([]goslack.User, error)
	GetPermalink(params *goslack.PermalinkParameters) (string, error)
	GetConversationHistory(params *goslack.GetConversationHistoryParameters) (*goslack.GetConversationHistoryResponse, error)
	GetUserInfo(user string) (*goslack.User, error)
	GetConversations(params *goslack.GetConversationsParameters) ([]goslack.Channel, string, error)
	SearchMessages(query string, params goslack.SearchParameters) (*goslack.SearchMessages, error)
}

// Service handles Slack integration operations.
type Service struct {
	bot            *janet.Bot
	slackClient    slackClient
	channelCache   []goslack.Channel
	cacheFetchedAt time.Time
	messageCache   map[string]messageCacheEntry
	messageCacheMu sync.RWMutex
	userCache      map[string]goslack.User
	userCacheMu    sync.RWMutex
	attachmentsDir string
	attachmentsURL string
	slackToken     string
	logger         *log.Log
}

// ServiceOptions controls Slack metadata enrichment behavior.
type ServiceOptions struct {
	AttachmentsDir string
	AttachmentsURL string
	SlackToken     string
	Logger         *log.Log
}

type messageCacheEntry struct {
	details   *MessageDetails
	expiresAt time.Time
}

var ignoredPlusRegex = regexp.MustCompile(`\+{4,}`)

// NewService creates a new Slack service.
func NewService(bot *janet.Bot, opts ServiceOptions) *Service {
	logger := opts.Logger
	if logger == nil && bot != nil && bot.Config != nil {
		logger = bot.Config.Log
	}
	return newService(bot, nil, opts, logger)
}

// NewWebService creates a new Slack service with a standalone client for web-only mode.
func NewWebService(slackClient slackClient, opts ServiceOptions) *Service {
	return newService(nil, slackClient, opts, opts.Logger)
}

func newService(bot *janet.Bot, slackClient slackClient, opts ServiceOptions, logger *log.Log) *Service {
	return &Service{
		bot:            bot,
		slackClient:    slackClient,
		messageCache:   make(map[string]messageCacheEntry),
		userCache:      make(map[string]goslack.User),
		attachmentsDir: opts.AttachmentsDir,
		attachmentsURL: strings.TrimRight(opts.AttachmentsURL, "/"),
		slackToken:     opts.SlackToken,
		logger:         logger,
	}
}

func (s *Service) client() slackClient {
	if s.slackClient != nil {
		return s.slackClient
	}
	if s.bot != nil && s.bot.Config.SlackWebClient != nil {
		return s.bot.Config.SlackWebClient
	}
	return nil
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
