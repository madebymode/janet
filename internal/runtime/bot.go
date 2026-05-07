package runtime

import (
	"fmt"
	"strings"

	"github.com/aybabtme/log"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
	"github.com/troyxmccall/janet"
	"github.com/troyxmccall/janet/database"
)

type BotOptions struct {
	SlackToken          string
	SlackSocketToken    string
	GoodPlaceJudgeBotID string

	MaxPoints        int
	LeaderboardLimit int
	ReplyType        string
	Debug            bool
	SelfKarma        bool
	Motivate         bool
	ReactjiEnabled   bool

	GoodJanetUsername string
	GoodJanetIconURL  string
	BadJanetUsername  string
	BadJanetIconURL   string

	UserBlacklist []string
	UserAliases   map[string]string
}

type SlackBotRuntime struct {
	Bot          *janet.Bot
	Client       *slack.Client
	SocketClient *socketmode.Client
}

func (o *BotOptions) Normalize() error {
	o.SlackToken = strings.TrimSpace(o.SlackToken)
	o.SlackSocketToken = strings.TrimSpace(o.SlackSocketToken)
	o.GoodPlaceJudgeBotID = strings.TrimSpace(o.GoodPlaceJudgeBotID)

	if o.SlackToken == "" {
		return fmt.Errorf("JANET_SLACK_TOKEN is required")
	}
	if o.SlackSocketToken == "" {
		return fmt.Errorf("JANET_SLACK_SOCKET_TOKEN is required")
	}

	if o.MaxPoints <= 0 || o.MaxPoints > 100 {
		o.MaxPoints = 5
	}
	if o.LeaderboardLimit <= 0 || o.LeaderboardLimit > 100 {
		o.LeaderboardLimit = 10
	}

	switch strings.ToLower(strings.TrimSpace(o.ReplyType)) {
	case "", "thread":
		o.ReplyType = "thread"
	case "message":
		o.ReplyType = "message"
	default:
		return fmt.Errorf("invalid reply type %q", o.ReplyType)
	}

	o.GoodJanetUsername = defaultString(o.GoodJanetUsername, "Good Janet")
	o.BadJanetUsername = defaultString(o.BadJanetUsername, "Bad Janet")
	o.GoodJanetIconURL = strings.TrimSpace(o.GoodJanetIconURL)
	o.BadJanetIconURL = strings.TrimSpace(o.BadJanetIconURL)
	o.UserBlacklist = normalizeStringList(o.UserBlacklist)
	o.UserAliases = normalizeAliases(o.UserAliases)

	return nil
}

func NewSlackBotRuntime(opts BotOptions, db *database.V2DB, logger *log.Log) (*SlackBotRuntime, error) {
	if err := opts.Normalize(); err != nil {
		return nil, err
	}

	slackClient := slack.New(
		opts.SlackToken,
		slack.OptionDebug(opts.Debug),
		slack.OptionAppLevelToken(opts.SlackSocketToken),
	)
	socketClient := socketmode.New(slackClient, socketmode.OptionDebug(opts.Debug))

	blacklistMap := make(janet.StringList, len(opts.UserBlacklist))
	for _, user := range opts.UserBlacklist {
		blacklistMap[user] = struct{}{}
	}

	userAliases := make(janet.UserAliases, len(opts.UserAliases))
	for alias, main := range opts.UserAliases {
		userAliases[alias] = main
	}

	botConfig := &janet.Config{
		Slack:               &janet.SlackChatService{Client: slackClient},
		SlackWebClient:      slackClient,
		Debug:               opts.Debug,
		MaxPoints:           opts.MaxPoints,
		LeaderboardLimit:    opts.LeaderboardLimit,
		Log:                 logger.KV("component", "bot"),
		UI:                  &janet.BlankUIProvider{},
		DB:                  db,
		UserBlacklist:       blacklistMap,
		Aliases:             userAliases,
		Reactji:             defaultReactjiConfig(opts.ReactjiEnabled),
		Motivate:            opts.Motivate,
		SelfPoints:          opts.SelfKarma,
		ReplyType:           opts.ReplyType,
		GoodPlaceJudgeBotID: opts.GoodPlaceJudgeBotID,
		GoodPersonality: janet.BotPersonality{
			Username: opts.GoodJanetUsername,
			IconURL:  opts.GoodJanetIconURL,
			IsGood:   true,
		},
		BadPersonality: janet.BotPersonality{
			Username: opts.BadJanetUsername,
			IconURL:  opts.BadJanetIconURL,
			IsGood:   false,
		},
	}

	return &SlackBotRuntime{
		Bot:          janet.New(botConfig),
		Client:       slackClient,
		SocketClient: socketClient,
	}, nil
}

func defaultReactjiConfig(enabled bool) *janet.ReactjiConfig {
	return &janet.ReactjiConfig{
		Enabled: enabled,
		UpVote: janet.StringList{
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
		DownVote: janet.StringList{
			"thumbsdown": struct{}{},
			"-1":         struct{}{},
		},
		RepeatPoints: janet.StringList{
			"bangbang":    struct{}{},
			"exclamation": struct{}{},
			"!!!":         struct{}{},
		},
	}
}

func normalizeStringList(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func normalizeAliases(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}

	result := make(map[string]string, len(values))
	for alias, main := range values {
		alias = strings.TrimSpace(alias)
		main = strings.TrimSpace(main)
		if alias == "" || main == "" {
			continue
		}
		result[alias] = main
	}
	return result
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
