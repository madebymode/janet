package janet

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/troyxmccall/janet/database"

	"github.com/aybabtme/log"
	"github.com/dustin/go-humanize"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

var (
	regexps = struct {
		Motivate, GivePoints, TakePoints, QueryPoints, Leaderboard, URL, SlackUser, Throwback *regexp.Regexp
	}{
		Motivate:    karmaReg.MatchMotivate(),
		GivePoints:  karmaReg.MatchGive(),
		TakePoints:  karmaReg.MatchTake(),
		QueryPoints: karmaReg.MatchQuery(),
		Leaderboard: regexp.MustCompile(`^goodplace? (?:leaderboard|top|highscores)(?: (\d+))?(?: (all|\d{4}))?$`),
		URL:         regexp.MustCompile(`^janet? (?:url|web|link)?$`),
		SlackUser:   regexp.MustCompile(`^<@([A-Za-z\d]+)>$`),
		Throwback:   karmaReg.MatchThrowback(),
	}
)

// Database interface - now just aliasing V2DB pointer
type Database = *database.V2DB

// ChatService is an abstraction around Slack, mostly designed for use in tests.
type ChatService interface {
	// PostMessage sends a message to a channel with options
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)

	// OpenIMChannel opens a new direct-message channel with the specified user.
	// It returns some status information, and the channel ID.
	OpenIMChannel(user string) (bool, bool, string, error)

	// GetUserInfo retrieves the complete user information for the specified username.
	GetUserInfo(user string) (*slack.User, error)

	// PostEphemeral sends an ephemeral message to a user in a channel.
	PostEphemeral(channelID, userID string, options ...slack.MsgOption) (string, error)

	// UpdateMessage updates an existing message
	UpdateMessage(channelID, timestamp string, options ...slack.MsgOption) (string, string, string, error)
}

// SlackChatService is an implementation of ChatService using github.com/slack-go/slack.
type SlackChatService struct {
	*slack.Client
}

// PostMessage sends a message to a channel with options
func (s SlackChatService) PostMessage(channelID string, options ...slack.MsgOption) (string, string, error) {
	return s.Client.PostMessage(channelID, options...)
}

// UpdateMessage updates an existing message
func (s SlackChatService) UpdateMessage(channelID, timestamp string, options ...slack.MsgOption) (string, string, string, error) {
	return s.Client.UpdateMessage(channelID, timestamp, options...)
}

// OpenIMChannel opens a new direct-message channel with the specified user
func (s SlackChatService) OpenIMChannel(user string) (bool, bool, string, error) {
	channel, _, _, err := s.Client.OpenConversation(&slack.OpenConversationParameters{
		Users: []string{user},
	})
	if err != nil {
		return false, false, "", err
	}
	return true, true, channel.ID, nil
}

// GetUserInfo retrieves the complete user information for the specified username
func (s SlackChatService) GetUserInfo(user string) (*slack.User, error) {
	return s.Client.GetUserInfo(user)
}

// PostEphemeral sends an ephemeral message to a user in a channel
func (s SlackChatService) PostEphemeral(channelID, userID string, options ...slack.MsgOption) (string, error) {
	return s.Client.PostEphemeral(channelID, userID, options...)
}

// UserAliases is a map of alias -> main username
type UserAliases map[string]string

// ReactjiConfig contains the configuration for reactji-based votes
type ReactjiConfig struct {
	Enabled                        bool
	UpVote, DownVote, RepeatPoints StringList
}

// BotPersonality represents the personality of the bot
type BotPersonality struct {
	Username string
	IconURL  string
	IsGood   bool
}

// Config contains all the necessary configs for janet.
type Config struct {
	Slack                       ChatService
	SlackWebClient              *slack.Client
	Debug, Motivate, SelfPoints bool
	MaxPoints, LeaderboardLimit int
	Log                         *log.Log
	UI                          Provider
	DB                          Database
	UserBlacklist               StringList
	Aliases                     UserAliases
	Reactji                     *ReactjiConfig
	WaitGroup                   *sync.WaitGroup
	ReplyType                   string
	GoodPlaceJudgeBotID         string
	GoodPersonality             BotPersonality
	BadPersonality              BotPersonality
}

// A Bot is an instance of janet.
type Bot struct {
	Config    *Config
	WaitGroup *sync.WaitGroup
}

// New returns a pointer to an new instance of janet.
func New(config *Config) *Bot {
	return &Bot{
		Config: config,
	}
}

func (b *Bot) Listen() {
	b.Config.Log.Info("goodplace judge listener called")
	// Socket mode implementation would go here
	// For now, this is a placeholder - the actual socket mode implementation
	// would need to be added when upgrading the slack library
	select {} // Block forever
}

// ListenWithSocketMode handles socket mode events
func (b *Bot) ListenWithSocketMode(client *socketmode.Client) {
	b.Config.Log.Info("goodplace judge socket mode listener started")

	go func() {
		for evt := range client.Events {
			switch evt.Type {
			case socketmode.EventTypeConnecting:
				b.Config.Log.Info("connecting to slack with socket mode...")
			case socketmode.EventTypeConnectionError:
				b.Config.Log.Error("connection failed. retrying later...")
			case socketmode.EventTypeConnected:
				b.Config.Log.Info("connected to slack with socket mode.")
			case socketmode.EventTypeSlashCommand:
				b.Config.Log.Info("received slash command event")
				client.Ack(*evt.Request)
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					b.Config.Log.Error("ignored event")
					continue
				}

				client.Ack(*evt.Request)

				switch eventsAPIEvent.Type {
				case slackevents.CallbackEvent:
					innerEvent := eventsAPIEvent.InnerEvent
					switch ev := innerEvent.Data.(type) {
					case *slackevents.AppMentionEvent:
						// Convert to MessageEvent for processing
						msgEvent := &slack.MessageEvent{
							Msg: slack.Msg{
								Type:            "message",
								Channel:         ev.Channel,
								User:            ev.User,
								Text:            ev.Text,
								Timestamp:       ev.TimeStamp,
								ThreadTimestamp: ev.ThreadTimeStamp,
							},
						}
						b.handleMessageEvent(msgEvent)
					case *slackevents.MessageEvent:
						// Convert to slack.MessageEvent for compatibility
						msgEvent := &slack.MessageEvent{
							Msg: slack.Msg{
								Type:            ev.Type,
								Channel:         ev.Channel,
								User:            ev.User,
								Text:            ev.Text,
								Timestamp:       ev.TimeStamp,
								ThreadTimestamp: ev.ThreadTimeStamp,
							},
						}
						b.handleMessageEvent(msgEvent)
					case *slackevents.ReactionAddedEvent:
						// Convert to slack.ReactionAddedEvent for compatibility
						reactionEvent := &slack.ReactionAddedEvent{
							Type:     ev.Type,
							User:     ev.User,
							Reaction: ev.Reaction,
							Item: slack.ReactionItem{
								Type:      ev.Item.Type,
								Channel:   ev.Item.Channel,
								Timestamp: ev.Item.Timestamp,
							},
							ItemUser:       ev.ItemUser,
							EventTimestamp: ev.EventTimestamp,
						}
						b.handleReactionAddedEvent(reactionEvent)
					case *slackevents.ReactionRemovedEvent:
						// Convert to slack.ReactionRemovedEvent for compatibility
						reactionEvent := &slack.ReactionRemovedEvent{
							Type:     ev.Type,
							User:     ev.User,
							Reaction: ev.Reaction,
							Item: slack.ReactionItem{
								Type:      ev.Item.Type,
								Channel:   ev.Item.Channel,
								Timestamp: ev.Item.Timestamp,
							},
							ItemUser:       ev.ItemUser,
							EventTimestamp: ev.EventTimestamp,
						}
						b.handleReactionRemovedEvent(reactionEvent)
					default:
						b.Config.Log.Info("unsupported events API event received")
					}
				default:
					b.Config.Log.Info("unsupported events API event received")
				}
			default:
				b.Config.Log.Info("unexpected event type received")
			}
		}
	}()

	client.Run()
}

func (b *Bot) handleReactionAddedEvent(ev *slack.ReactionAddedEvent) {
	if !b.Config.Reactji.Enabled {
		return
	}

	var (
		points int
		reason string
	)
	switch {
	case b.Config.Reactji.UpVote.Contains(ev.Reaction):
		points = +3 // Changed from +1 to +3 to match backfill service
	case b.Config.Reactji.DownVote.Contains(ev.Reaction):
		points = -1
	case b.Config.Reactji.RepeatPoints.Contains(ev.Reaction):
		b.handleBangBangPoints(ev)
		return
	default:
		return
	}

	reason = fmt.Sprintf("added a :%s: emoji", ev.Reaction)
	b.handleReactionEvent(ev, reason, points)
}

func (b *Bot) handleReactionRemovedEvent(ev *slack.ReactionRemovedEvent) {
	if !b.Config.Reactji.Enabled {
		return
	}

	var (
		points int
		reason string
	)
	switch {
	case b.Config.Reactji.UpVote.Contains(ev.Reaction):
		points = -3 // Changed from -1 to -3 to match backfill service
	case b.Config.Reactji.DownVote.Contains(ev.Reaction):
		points = +1
	default:
		return
	}

	reason = fmt.Sprintf("removed a :%s: emoji", ev.Reaction)
	b.handleReactionEvent((*slack.ReactionAddedEvent)(ev), reason, points)
}

func (b *Bot) handleBangBangPoints(ev *slack.ReactionAddedEvent) {
	// Get the original message to analyze karma content
	message, err := b.Config.SlackWebClient.GetConversationHistory(&slack.GetConversationHistoryParameters{
		ChannelID: ev.Item.Channel,
		Latest:    ev.Item.Timestamp,
		Limit:     1,
		Inclusive: true,
	})
	if err != nil {
		b.Config.Log.Err(err).Error("failed to get message for bangbang processing")
		return
	}

	if len(message.Messages) == 0 {
		b.Config.Log.Error("no message found for bangbang processing")
		return
	}

	originalMessage := message.Messages[0]
	textToParse := originalMessage.Text

	// Handle motivates like the main bot does
	if match := regexps.Motivate.FindStringSubmatch(textToParse); len(match) > 0 {
		textToParse = match[1] + "++ for doing good work"
	}

	// Get bangbang reactor username
	reactorUsername, err := b.GetUserNameByID(ev.User)
	if err != nil {
		b.Config.Log.Err(err).Error("failed to get reactor username for bangbang")
		return
	}

	var goodJanetResponse strings.Builder
	var badJanetResponse strings.Builder

	// Check for give karma patterns (both @user++ and username++) using shared regexps
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

	// Process each karma match found in the original message
	for _, karmaMatch := range allKarmaMatches {
		// Parse karma information from the match
		points, toUser := b.parseKarmaMatch(karmaMatch.match, karmaMatch.isGiveKarma)
		if toUser == "" {
			continue
		}

		// Skip self-reactions
		if reactorUsername == toUser {
			continue
		}

		// Apply bangbang effect (double the points)
		bangbangPoints := points
		if karmaMatch.isTakeKarma {
			bangbangPoints = -bangbangPoints
		}

		// Apply max points limit
		if bangbangPoints > b.Config.MaxPoints {
			bangbangPoints = b.Config.MaxPoints
		} else if bangbangPoints < -b.Config.MaxPoints {
			bangbangPoints = -b.Config.MaxPoints
		}

		// Process the karma transaction
		tx := &database.Transaction{
			FromUser:        reactorUsername,
			ToUser:          toUser,
			Points:          bangbangPoints,
			Reason:          fmt.Sprintf("added a :bangbang: emoji (doubling existing %d points)", points),
			TransactionType: "reactji",
			ChannelID:       &ev.Item.Channel,
			MessageID:       &ev.Item.Timestamp,
			Timestamp:       time.Now(),
		}
		err := b.Config.DB.InsertTransaction(tx)
		if err != nil {
			b.Config.Log.Err(err).Error("failed to insert bangbang transaction")
			continue
		}

		// Get updated user points for response
		user, err := b.Config.DB.GetUser(toUser)
		if err != nil {
			b.Config.Log.Err(err).Error("failed to get user points for bangbang response")
			continue
		}

		pointsMsg := fmt.Sprintf("%s now has %d points", toUser, user.TotalPoints)

		// Add to appropriate response builder
		if karmaMatch.isGiveKarma {
			if len(goodJanetResponse.String()) > 0 {
				goodJanetResponse.WriteString("\n")
			}
			goodJanetResponse.WriteString(pointsMsg)
		} else {
			if len(badJanetResponse.String()) > 0 {
				badJanetResponse.WriteString("\n")
			}
			badJanetResponse.WriteString(pointsMsg)
		}
	}

	// Get thread history to find existing bot messages
	history, _, _, err := b.Config.SlackWebClient.GetConversationReplies(&slack.GetConversationRepliesParameters{
		ChannelID: ev.Item.Channel,
		Timestamp: ev.Item.Timestamp,
		Inclusive: true,
	})
	if err != nil {
		b.Config.Log.Err(err).Error("failed to get thread history for bangbang")
		return
	}

	// Find the latest message from GoodPlace Judge bot
	var lastMsgFromBot *slack.Message
	for _, msg := range history {
		if msg.BotID == b.Config.GoodPlaceJudgeBotID {
			lastMsgFromBot = &msg
			break // Get the first (most recent) match
		}
	}

	// Form response messages
	_ = fmt.Sprintf("bc %s added a :bangbang: emoji \n", reactorUsername)

	if len(goodJanetResponse.String()) > 0 {
		responseMsg := goodJanetResponse.String()

		// Update existing bot message or post new one
		if lastMsgFromBot != nil {
			currentMsg := lastMsgFromBot.Text
			newMsg := currentMsg + "\n" + responseMsg
			_, _, _, err = b.Config.SlackWebClient.UpdateMessage(ev.Item.Channel, lastMsgFromBot.Timestamp, slack.MsgOptionText(newMsg, false))
		} else {
			_, _, err = b.Config.Slack.PostMessage(ev.Item.Channel,
				slack.MsgOptionText(responseMsg, false),
				slack.MsgOptionTS(ev.Item.Timestamp),
				slack.MsgOptionUsername(b.Config.GoodPersonality.Username),
				slack.MsgOptionIconURL(b.Config.GoodPersonality.IconURL))
		}

		if err != nil {
			b.Config.Log.Err(err).Error("failed to post/update good janet bangbang message")
		}
	}

	if len(badJanetResponse.String()) > 0 {
		responseMsg := badJanetResponse.String()

		// Update existing bot message or post new one
		if lastMsgFromBot != nil {
			currentMsg := lastMsgFromBot.Text
			newMsg := currentMsg + "\n" + responseMsg
			_, _, _, err = b.Config.SlackWebClient.UpdateMessage(ev.Item.Channel, lastMsgFromBot.Timestamp, slack.MsgOptionText(newMsg, false))
		} else {
			_, _, err = b.Config.Slack.PostMessage(ev.Item.Channel,
				slack.MsgOptionText(responseMsg, false),
				slack.MsgOptionTS(ev.Item.Timestamp),
				slack.MsgOptionUsername(b.Config.BadPersonality.Username),
				slack.MsgOptionIconURL(b.Config.BadPersonality.IconURL))
		}

		if err != nil {
			b.Config.Log.Err(err).Error("failed to post/update bad janet bangbang message")
		}
	}
}

// calculateKarmaFromMessage parses a message and calculates total karma points being given in the message
func (b *Bot) calculateKarmaFromMessage(messageText, messageAuthor string) int {
	totalPoints := 0

	// Handle motivates like the main bot does
	if match := regexps.Motivate.FindStringSubmatch(messageText); len(match) > 0 {
		messageText = match[1] + "++ for doing good work"
	}

	// Check for give karma patterns - sum ALL karma being given
	if matches := regexps.GivePoints.FindAllStringSubmatch(messageText, -1); len(matches) > 0 {
		for _, match := range matches {
			points, toUser := b.parseKarmaMatch(match, true)
			if toUser != "" { // Any valid karma given
				totalPoints += points
			}
		}
	}

	// Check for take karma patterns - subtract karma being taken
	if matches := regexps.TakePoints.FindAllStringSubmatch(messageText, -1); len(matches) > 0 {
		for _, match := range matches {
			points, toUser := b.parseKarmaMatch(match, false)
			if toUser != "" { // Any valid karma taken
				totalPoints += points // points is already negative from parseKarmaMatch
			}
		}
	}

	return totalPoints
}

// parseKarmaMatch parses a regex match and extracts karma information for bangbang calculation
func (b *Bot) parseKarmaMatch(match []string, isPositive bool) (int, string) {
	var targetUser, karmaChars string

	if match[1] != "" {
		// @user format
		targetUser = match[1]
		karmaChars = match[2]
	} else if match[5] != "" {
		// username format
		targetUser = match[5]
		karmaChars = match[6]
	}

	if targetUser == "" || karmaChars == "" {
		return 0, ""
	}

	// Calculate points from karma characters (++ = 1, +++ = 2, etc.)
	points := len(karmaChars) - 1
	if points < 1 {
		points = 1
	}
	if points > b.Config.MaxPoints {
		points = b.Config.MaxPoints
	}

	if !isPositive {
		points = -points
	}

	// Handle @user format - extract username
	var toUser string
	if strings.HasPrefix(targetUser, "<@") && strings.HasSuffix(targetUser, ">") {
		userID := targetUser[2 : len(targetUser)-1]
		username, err := b.GetUserNameByID(userID)
		if err != nil {
			return 0, ""
		}
		toUser = username
	} else {
		toUser = targetUser
	}

	return points, toUser
}

// processKarmaTransaction creates and processes a karma transaction
func (b *Bot) processKarmaTransaction(fromUser, toUser string, points int, reason, transactionType string, channelID, messageID *string) {
	// add the actor's username to the reason
	reason = fmt.Sprintf("%s %s", fromUser, reason)

	// create and insert transaction
	tx := &database.Transaction{
		FromUser:        fromUser,
		ToUser:          toUser,
		Points:          points,
		Reason:          reason,
		TransactionType: transactionType,
		ChannelID:       channelID,
		MessageID:       messageID,
		Timestamp:       time.Now(),
	}
	err := b.Config.DB.InsertTransaction(tx)
	if b.handleError(err, nil) {
		return
	}
}

// at this point there is no difference between ReactionAddedEvent and ReactionRemovedEvent
func (b *Bot) handleReactionEvent(ev *slack.ReactionAddedEvent, reason string, points int) {
	// look up usernames
	from, err := b.GetUserNameByID(ev.User)
	if b.handleError(err, nil) {
		return
	}
	to, err := b.GetUserNameByID(ev.ItemUser)
	if b.handleError(err, nil) {
		return
	}

	// add the actor's username to the reason
	reason = fmt.Sprintf("%s %s", from, reason)

	// give points
	tx := &database.Transaction{
		FromUser:        from,
		ToUser:          to,
		Points:          points,
		Reason:          reason,
		TransactionType: "reactji",
		ChannelID:       &ev.Item.Channel,
		MessageID:       &ev.Item.Timestamp,
		Timestamp:       time.Now(),
	}
	err = b.Config.DB.InsertTransaction(tx)
	if b.handleError(err, nil) {
		return
	}
}

func (b *Bot) handleMessageEvent(ev *slack.MessageEvent) {
	b.Config.Log.KV("channel", ev.Channel).KV("user", ev.User).KV("text", ev.Text).Info("received message")
	if ev.Type != "message" {
		return
	}

	// convert motivates into janet syntax
	if b.Config.Motivate {
		if match := regexps.Motivate.FindStringSubmatch(ev.Text); len(match) > 0 {
			ev.Text = match[1] + "++ for doing good work"
		}
	}

	textToParse := ev.Text

	re := regexp.MustCompile(`(<@[A-Za-z0-9]+>(\s)?([+]{2,})?([-]{2,})?)`)

	splits := re.FindAllString(textToParse, -1)

	var goodJanetResponse strings.Builder
	var badJanetResponse strings.Builder

	if splits != nil {
		for _, split := range splits {

			splitText := split

			switch {
			case regexps.GivePoints.MatchString(splitText):
				goodJanetResponse.WriteString(b.applyPoints(ev, true, splitText))
				if len(goodJanetResponse.String()) > 0 {
					goodJanetResponse.WriteString("\n")
				}

			case regexps.TakePoints.MatchString(splitText):
				badJanetResponse.WriteString(b.applyPoints(ev, false, splitText))
				if len(badJanetResponse.String()) > 0 {
					badJanetResponse.WriteString("\n")
				}
			case regexps.Throwback.MatchString(ev.Text):
				b.getThrowback(ev)

			case regexps.QueryPoints.MatchString(ev.Text):
				b.queryPoints(ev)
			}
		}

		if len(goodJanetResponse.String()) > 0 {
			responseMsg := goodJanetResponse.String()
			_, _, err := b.Config.Slack.PostMessage(ev.Channel, slack.MsgOptionText(responseMsg, false), slack.MsgOptionTS(ev.Timestamp), slack.MsgOptionUsername(b.Config.GoodPersonality.Username), slack.MsgOptionIconURL(b.Config.GoodPersonality.IconURL))
			if err != nil {
				b.Config.Log.Error(err.Error())
			}
		}

		if len(badJanetResponse.String()) > 0 {
			responseMsg := badJanetResponse.String()
			_, _, err := b.Config.Slack.PostMessage(ev.Channel, slack.MsgOptionText(responseMsg, false), slack.MsgOptionTS(ev.Timestamp), slack.MsgOptionUsername(b.Config.BadPersonality.Username), slack.MsgOptionIconURL(b.Config.BadPersonality.IconURL))
			if err != nil {
				b.Config.Log.Error(err.Error())
			}
		}

	} else {

		switch {
		case regexps.URL.MatchString(ev.Text):
			b.printURL(ev)
		case regexps.Leaderboard.MatchString(ev.Text):
			b.printLeaderboard(ev)
		}

	}
}

func (b *Bot) getReplyThread(message *slack.MessageEvent) string {
	var thread string

	switch b.Config.ReplyType {
	case "message":
		thread = message.ThreadTimestamp
	case "thread":
		if message.ThreadTimestamp != "" {
			thread = message.ThreadTimestamp
		} else {
			thread = message.Timestamp
		}
	}

	return thread
}

// SendReply sends a reply to a message, either as a new message in the channel or a thread (configurable)
func (b *Bot) SendReply(reply string, message *slack.MessageEvent, isGoodPersonality bool) {
	switch b.Config.ReplyType {
	case "ephemeral":
		b.SendReplyEphemeral(reply, message)
	default:
		b.SendMessage(reply, message.Channel, b.getReplyThread(message), isGoodPersonality)
	}
}

// SendReplyEphemeral sends a reply to a message as an ephemeral message to the user
func (b *Bot) SendReplyEphemeral(reply string, message *slack.MessageEvent) {
	b.SendMessageEphemeral(message.Channel, message.User, reply, message.ThreadTimestamp)
}

// SendMessageEphemeral sends an ephemeral message to a user
func (b *Bot) SendMessageEphemeral(reply, channel, user, thread string) {
	b.Config.Slack.PostEphemeral(channel, user, slack.MsgOptionText(reply, false), slack.MsgOptionTS(thread))
}

// SendMessage sends a message to a Slack channel with personality-based username and avatar.
func (b *Bot) SendMessage(message, channel, thread string, isGoodPersonality bool) {
	var personality BotPersonality
	if isGoodPersonality {
		personality = b.Config.GoodPersonality
	} else {
		personality = b.Config.BadPersonality
	}

	b.Config.Log.Info("sending message as " + personality.Username)

	options := []slack.MsgOption{
		slack.MsgOptionText(message, false),
		slack.MsgOptionUsername(personality.Username),
		slack.MsgOptionIconURL(personality.IconURL),
	}

	if thread != "" {
		options = append(options, slack.MsgOptionTS(thread))
	}

	_, _, err := b.Config.Slack.PostMessage(channel, options...)
	if err != nil {
		b.Config.Log.Err(err).Error("failed to send message")
		return
	}

	// Randomly append a quote
	if appendQuoteToMessage() {
		var quote string
		if isGoodPersonality {
			quote = goodJanetQuote()
		} else {
			quote = badJanetQuote()
		}

		quoteOptions := []slack.MsgOption{
			slack.MsgOptionText(quote, false),
			slack.MsgOptionUsername(personality.Username),
			slack.MsgOptionIconURL(personality.IconURL),
		}

		if thread != "" {
			quoteOptions = append(quoteOptions, slack.MsgOptionTS(thread))
		}

		_, _, err := b.Config.Slack.PostMessage(channel, quoteOptions...)
		if err != nil {
			b.Config.Log.Err(err).Error("failed to send quote message")
		}
	}
}

// DMUser sends a message directly to a Slack user.
func (b *Bot) DMUser(message, user string, isGoodPersonality bool) {
	_, _, channel, err := b.Config.Slack.OpenIMChannel(user)
	if err != nil {
		b.Config.Log.Err(err).KV("user", user).Error("could not open IM channel with user")
		return
	}

	b.SendMessage(message, channel, "", isGoodPersonality)
}

func (b *Bot) handleError(err error, message *slack.MessageEvent) bool {
	if err == nil {
		return false
	}

	b.Config.Log.Err(err).Error("error")
	if b.Config.Debug && message != nil {
		text := fmt.Sprintf("i had a problem: %v", err)
		b.SendReply(text, message, false)
	}

	return true
}

func (b *Bot) printURL(ev *slack.MessageEvent) {
	url, err := b.Config.UI.GetURL("/")
	if b.handleError(err, ev) {
		return
	}

	// ui is disabled
	if url == "" {
		return
	}

	b.SendReply(url, ev, true)
}

func (b *Bot) applyPoints(ev *slack.MessageEvent, isGoodPersonality bool, splitText string) string {
	personality := "good"
	if !isGoodPersonality {
		personality = "bad"
	}
	b.Config.Log.Info(personality)

	match := regexps.GivePoints.FindStringSubmatch(splitText)
	if len(match) == 0 {
		match = regexps.TakePoints.FindStringSubmatch(splitText)
	}
	if len(match) == 0 {
		return ""
	}

	// forgive me
	if match[1] != "" {
		// we matched the first alt expression
		match = match[:4]
	} else {
		// we matched the second alt expression
		match = append(match[:1], match[4:]...)
	}

	points := min(len(match[2])-1, b.Config.MaxPoints)
	if match[2][0] == '-' {
		points *= -1
	}
	reason := match[3]

	from, err := b.GetUserNameByID(ev.User)
	if b.handleError(err, ev) {
		return ""
	}
	to, err := b.ParseUser(match[1])
	if b.handleError(err, ev) {
		return ""
	}

	to = strings.ToLower(to)
	from = strings.ToLower(from)

	if !b.Config.SelfPoints && to == from {
		return "giving points to yourself is a classic bad place move"
	}

	if b.Config.UserBlacklist.Contains(to) {
		b.Config.Log.KV("user", to).Info("user is blacklisted, ignoring karma command")
		return ""
	}

	err = b.Config.DB.InsertTransaction(&database.Transaction{
		FromUser:        from,
		ToUser:          to,
		Points:          points,
		Reason:          reason,
		TransactionType: "manual",
		ChannelID:       &ev.Channel,
		MessageID:       &ev.Timestamp,
		Timestamp:       time.Now(),
	})
	if b.handleError(err, ev) {
		return ""
	}

	user, err := b.Config.DB.GetUser(to)
	if b.handleError(err, ev) {
		return ""
	}

	text := fmt.Sprintf("%s has %d points", Munge(to), user.TotalPoints)
	if reason != "" {
		text = fmt.Sprintf("%s (%d for %s)", text, points, reason)
	}
	return text
}

func (b *Bot) getThrowback(ev *slack.MessageEvent) {
	match := regexps.Throwback.FindStringSubmatch(ev.Text)
	if len(match) == 0 {
		return
	}

	var (
		user string
		err  error
	)
	if match[1] != "" {
		user, err = b.ParseUser(match[1])
		if b.handleError(err, ev) {
			return
		}
		user = strings.ToLower(user)
	} else {
		user, err = b.GetUserNameByID(ev.User)
		if b.handleError(err, ev) {
			return
		}
	}

	throwback, err := b.Config.DB.GetThrowback(user)
	if err == database.ErrNoSuchUser {
		b.SendReply(fmt.Sprintf("could not find any karma operations for %s", user), ev, true)
		return
	}

	if b.handleError(err, ev) {
		return
	}

	date := humanize.Time(throwback.Timestamp)
	if throwback.Reason != "" {
		throwback.Reason = fmt.Sprintf(" for %s", throwback.Reason)
	}
	text := fmt.Sprintf("%s received %d points from %s %s%s", Munge(throwback.To), throwback.Points.Points, Munge(throwback.From), date, throwback.Reason)

	b.SendReply(text, ev, true)
}

func (b *Bot) printLeaderboard(ev *slack.MessageEvent) {
	match := regexps.Leaderboard.FindStringSubmatch(ev.Text)
	if len(match) == 0 {
		return
	}

	limit := b.Config.LeaderboardLimit
	yearParam := ""

	// Handle ambiguous case: single number could be limit or year
	// If match[1] is present but match[2] is empty, check if match[1] is a 4-digit year
	if match[1] != "" && match[2] == "" {
		// Check if it's a 4-digit number (year format)
		if len(match[1]) == 4 {
			// Treat as year
			yearParam = match[1]
		} else {
			// Treat as limit
			var err error
			limit, err = strconv.Atoi(match[1])
			if b.handleError(err, ev) {
				return
			}
		}
	} else {
		// Both present or only match[2] present
		if match[1] != "" {
			var err error
			limit, err = strconv.Atoi(match[1])
			if b.handleError(err, ev) {
				return
			}
		}
		yearParam = match[2]
	}

	// Parse year parameter
	var board []*database.UserSummary
	var err error
	var yearText string

	if yearParam == "" {
		// No year specified, use current year
		board, err = b.Config.DB.GetLeaderboardByCurrentYear(limit)
		yearText = fmt.Sprintf("%d", time.Now().Year())
	} else if yearParam == "all" {
		// "all" specified, use cumulative/all-time
		board, err = b.Config.DB.GetLeaderboardCumulative(limit)
		yearText = "all-time"
	} else {
		// Specific year
		year, parseErr := strconv.Atoi(yearParam)
		if b.handleError(parseErr, ev) {
			return
		}
		board, err = b.Config.DB.GetLeaderboardByYear(year, limit)
		yearText = yearParam
	}

	if b.handleError(err, ev) {
		return
	}

	text := fmt.Sprintf("*top %d leaderboard (%s)*\n", limit, yearText)

	url, err := b.Config.UI.GetURL(fmt.Sprintf("/leaderboard/%d", limit))
	if b.handleError(err, ev) {
		return
	}
	if url != "" {
		text = fmt.Sprintf("%s%s\n", text, url)
	}

	for i, user := range board {
		text = fmt.Sprintf("%s%d. %s (%d)\n", text, i+1, Munge(user.Username), user.TotalPoints)
	}

	b.SendReply(text, ev, true)
}

func (b *Bot) ParseUser(user string) (string, error) {
	user = strings.Trim(user, "<>@ ")

	var name string
	var err error

	// check if it's a UID
	if !regexps.SlackUser.MatchString(fmt.Sprintf("<@%s>", user)) {
		if alias, ok := b.Config.Aliases[user]; ok {
			return alias, nil
		}
		return user, nil
	}

	// it's a UID, look it up
	name, err = b.GetUserNameByID(user)
	if err != nil {
		return "", err
	}
	if alias, ok := b.Config.Aliases[name]; ok {
		return alias, nil
	}
	return name, nil
}

func (b *Bot) GetUserNameByID(id string) (string, error) {
	user, err := b.Config.Slack.GetUserInfo(id)
	if err != nil {
		return "", err
	}
	return user.Name, nil
}

// GetSlackUserInfo returns complete Slack user information for filtering and display
func (b *Bot) GetSlackUserInfo(id string) (*slack.User, error) {
	return b.Config.Slack.GetUserInfo(id)
}

func (b *Bot) queryPoints(ev *slack.MessageEvent) {
	match := regexps.QueryPoints.FindStringSubmatch(ev.Text)
	if len(match) == 0 {
		return
	}

	name, err := b.ParseUser(match[1])
	if b.handleError(err, ev) {
		return
	}
	name = strings.ToLower(name)

	user, err := b.Config.DB.GetUser(name)
	switch {
	case err == database.ErrNoSuchUser:
		// override debug mode
		b.SendReply(err.Error(), ev, true)
	case b.handleError(err, ev):
	default:
		b.SendReply(fmt.Sprintf("%s == %d", name, user.TotalPoints), ev, true)
	}
}
