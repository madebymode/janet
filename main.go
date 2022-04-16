package janet

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/troyxmccall/janet/database"
	"github.com/troyxmccall/janet/munge"
	"github.com/troyxmccall/janet/ui"

	"github.com/aybabtme/log"
	"github.com/dustin/go-humanize"
	"github.com/nlopes/slack"
)

var (
	regexps = struct {
		Motivate, GivePoints, TakePoints, QueryPoints, Leaderboard, URL, SlackUser, Throwback *regexp.Regexp
	}{
		Motivate:    karmaReg.MatchMotivate(),
		GivePoints:  karmaReg.MatchGive(),
		TakePoints:  karmaReg.MatchTake(),
		QueryPoints: karmaReg.MatchQuery(),
		Leaderboard: regexp.MustCompile(`^goodplace? (?:leaderboard|top|highscores) ?([0-9]+)?$`),
		URL:         regexp.MustCompile(`^janet? (?:url|web|link)?$`),
		SlackUser:   regexp.MustCompile(`^<@([A-Za-z0-9]+)>$`),
		Throwback:   karmaReg.MatchThrowback(),
	}
)

// Database is an abstraction around the database, mostly designed for use in tests.
type Database interface {
	// InsertPoints persistently records that points have been given or deducted.
	InsertPoints(points *database.Points) error

	// GetUser returns information about a user, including their current number of points.
	GetUser(name string) (*database.User, error)

	// GetLeaderboard returns the top X users with the most points, in order.
	GetLeaderboard(limit int) (database.Leaderboard, error)

	// GetTotalPoints returns the total number of points transferred across all users.
	GetTotalPoints() (int, error)

	// GetThrowback returns a random karma operation on a specific user.
	GetThrowback(user string) (*database.Throwback, error)
}

// ChatService is an abstraction around Slack, mostly designed for use in tests.
type ChatService interface {
	// IncomingEventsChan returns a channel of real-time events.
	IncomingEventsChan() chan slack.RTMEvent

	// NewOutgoingMessage constructs a new OutgoingMessage using the provided text and channel.
	NewOutgoingMessage(text string, channel string, options ...slack.RTMsgOption) *slack.OutgoingMessage

	// SendMessage sends the provided *OutgoingMessage.
	SendMessage(msg *slack.OutgoingMessage)

	// OpenIMChannel opens a new direct-message channel with the specified user.
	// It returns some status information, and the channel ID.
	OpenIMChannel(user string) (bool, bool, string, error)

	// GetUserInfo retrieves the complete user information for the specified username.
	GetUserInfo(user string) (*slack.User, error)

	// PostEphemeral sends an ephemeral message to a user in a channel.
	PostEphemeral(channelID, userID string, options ...slack.MsgOption) (string, error)
}

// SlackChatService is an implementation of ChatService using github.com/nlopes/slack.
type SlackChatService struct {
	slack.RTM
}

// IncomingEventsChan returns a channel of real-time messaging events.
func (s SlackChatService) IncomingEventsChan() chan slack.RTMEvent {
	return s.IncomingEvents
}

// UserAliases is a map of alias -> main username
type UserAliases map[string]string

// ReactjiConfig contains the configuration for reactji-based votes
type ReactjiConfig struct {
	Enabled          bool
	Upvote, Downvote StringList
}

// Config contains all the necessary configs for janet.
type Config struct {
	Slack                       ChatService
	BadJanetSlack               ChatService
	Debug, Motivate, SelfPoints bool
	MaxPoints, LeaderboardLimit int
	Log                         *log.Log
	UI                          ui.Provider
	DB                          Database
	UserBlacklist               StringList
	Aliases                     UserAliases
	Reactji                     *ReactjiConfig
	WaitGroup                   *sync.WaitGroup
	ReplyType                   string
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

	b.Config.Log.Info("listener called")

	//add listenrs to waitgroup because they need to run at the same timee (concurrently)
	var wg sync.WaitGroup
	wg.Add(2)

	b.GoodJanetListen(wg)
	b.BadJanetListen(wg)

	wg.Wait()
}

// GoodJanetListen Listen starts listening for Slack messages and calls the
// appropriate handlers.
func (b *Bot) GoodJanetListen(wg sync.WaitGroup) {

	b.Config.Log.Info("good-janet listener called")

	go func() {
		for msg := range b.Config.Slack.IncomingEventsChan() {
			switch ev := msg.Data.(type) {
			case *slack.ReactionAddedEvent:
				go b.handleReactionAddedEvent(msg.Data.(*slack.ReactionAddedEvent))
			case *slack.ReactionRemovedEvent:
				go b.handleReactionRemovedEvent(msg.Data.(*slack.ReactionRemovedEvent))
			case *slack.MessageEvent:
				go b.handleMessageEvent(msg.Data.(*slack.MessageEvent))
			case *slack.ConnectedEvent:
				b.Config.Log.Info("janet connected to slack")
				if b.Config.Debug {
					b.Config.Log.KV("info", ev.Info).Info("got slack info")
					b.Config.Log.KV("connections", ev.ConnectionCount).Info("got connection count")
				}
			case *slack.RTMError:
				b.Config.Log.Err(ev).Error("slack rtm error")
			case *slack.InvalidAuthEvent:
				wg.Done()
				b.Config.Log.Fatal("invalid slack token")
			default:
				b.Config.Log.KV("data", msg.Data).KV("event", reflect.TypeOf(msg.Data)).Info("unexpected slack api event")
			}
		}
	}()
}

func (b *Bot) BadJanetListen(wg sync.WaitGroup) {

	b.Config.Log.Info("bad-janet listener called")

	go func() {
		for msg := range b.Config.BadJanetSlack.IncomingEventsChan() {
			switch ev := msg.Data.(type) {
			case *slack.MessageEvent:
				b.Config.Log.Info("bad-janet got a message")
				//go b.handleMessageEvent(msg.Data.(*slack.MessageEvent))
			case *slack.ConnectedEvent:
				b.Config.Log.Info("bad-janet connected to slack")
				if b.Config.Debug {
					b.Config.Log.KV("info", ev.Info).Info("got bad-janet slack info")
					b.Config.Log.KV("connections", ev.ConnectionCount).Info("got bad-janet connection count")
				}
			case *slack.RTMError:
				b.Config.Log.Err(ev).Error("badJanet slack rtm error")
			case *slack.InvalidAuthEvent:
				wg.Done()
				b.Config.Log.Fatal("badJanet invalid slack token")
			default:
				b.Config.Log.KV("data", msg.Data).KV("event", reflect.TypeOf(msg.Data)).Info("unexpected slack api event")
			}
		}
	}()
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
func (b *Bot) SendReply(reply string, message *slack.MessageEvent, whichJanet string) {
	switch b.Config.ReplyType {
	case "ephemeral":
		b.SendReplyEphemeral(reply, message)
	default:
		b.SendMessage(reply, message.Channel, b.getReplyThread(message), whichJanet)
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

// SendMessage sends a message to a Slack channel.
func (b *Bot) SendMessage(message, channel, thread string, whichJanet string) {

	b.Config.Log.Info("sending message as")

	if whichJanet == "badJanet" {
		//b.Config.Log.Info(whichJanet)

		msg := b.Config.BadJanetSlack.NewOutgoingMessage(message, channel)
		msg.ThreadTimestamp = thread
		b.Config.BadJanetSlack.SendMessage(msg)

		appendMessage := appendQuoteToMessage()
		if appendMessage {
			msg := b.Config.BadJanetSlack.NewOutgoingMessage(badJanetQuote(), channel)
			msg.ThreadTimestamp = thread
			b.Config.BadJanetSlack.SendMessage(msg)
		}
	} else {

		msg := b.Config.Slack.NewOutgoingMessage(message, channel)
		msg.ThreadTimestamp = thread
		b.Config.Slack.SendMessage(msg)

		appendMessage := appendQuoteToMessage()
		if appendMessage {
			msg := b.Config.Slack.NewOutgoingMessage(goodJanetQuote(), channel)
			msg.ThreadTimestamp = thread
			b.Config.Slack.SendMessage(msg)
		}
	}
}

// DMUser sends a message directly to a Slack user.
func (b *Bot) DMUser(message, user string, whichJanet string) {
	_, _, channel, err := b.Config.Slack.OpenIMChannel(user)
	if err != nil {
		b.Config.Log.Err(err).KV("user", user).Error("could not open IM channel with user")
		return
	}

	if whichJanet == "badJanet" {
		_, _, channel, err = b.Config.BadJanetSlack.OpenIMChannel(user)
	}

	b.SendMessage(message, channel, "", whichJanet)
}

func (b *Bot) handleError(err error, message *slack.MessageEvent) bool {
	if err == nil {
		return false
	}

	b.Config.Log.Err(err).Error("error")
	if message != nil {
		var text string
		if b.Config.Debug {
			text = err.Error()
		} else {
			text = "hi guys im broken."
		}

		b.SendReply(text, message, "")
	}

	return true
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
	case b.Config.Reactji.Upvote.Contains(ev.Reaction):
		points = +1
	case b.Config.Reactji.Downvote.Contains(ev.Reaction):
		points = -1
	default:
		return
	}

	reason = fmt.Sprintf("added a :%s: reactji", ev.Reaction)
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
	case b.Config.Reactji.Upvote.Contains(ev.Reaction):
		points = -1
	case b.Config.Reactji.Downvote.Contains(ev.Reaction):
		points = +1
	default:
		return
	}

	reason = fmt.Sprintf("removed a :%s: reactji", ev.Reaction)
	b.handleReactionEvent((*slack.ReactionAddedEvent)(ev), reason, points)
}

// at this point there is no difference between ReactionAddedEvent and ReactionRemovedEvent
func (b *Bot) handleReactionEvent(ev *slack.ReactionAddedEvent, reason string, points int) {
	// look up usernames
	from, err := b.getUserNameByID(ev.User)
	if b.handleError(err, nil) {
		return
	}
	to, err := b.getUserNameByID(ev.ItemUser)
	if b.handleError(err, nil) {
		return
	}

	// add the actor's username to the reason
	reason = fmt.Sprintf("%s %s", from, reason)

	from, to = strings.ToLower(from), strings.ToLower(to)

	// insert points
	record := &database.Points{
		From:   from,
		To:     to,
		Points: points,
		Reason: reason,
	}

	err = b.Config.DB.InsertPoints(record)
	if b.handleError(err, nil) {
		return
	}

	pointsMsg, err := b.getUserPointsMessage(to, reason, points)
	if b.handleError(err, nil) {
		return
	}

	whichJanet := ""
	if points < 0 {
		whichJanet = "badJanet"
	} else {
		whichJanet = "goodJanet"
	}

	// reply as ephemeral message
	b.SendMessageEphemeral(pointsMsg, ev.Item.Channel, ev.User, whichJanet)
}

func (b *Bot) handleMessageEvent(ev *slack.MessageEvent) {
	if ev.Type != "message" {
		return
	}

	// convert motivates into janet syntax
	if b.Config.Motivate {
		if match := regexps.Motivate.FindStringSubmatch(ev.Text); len(match) > 0 {
			ev.Text = match[1] + "++ for doing good work"
		}
	}

	re := regexp.MustCompile("(<@[A-Za-z0-9]+>(\\s)?([\\+]{2,})?([\\-]{2,})?)")
	splits := re.FindAllString(ev.Text, -1)

	var goodJanetResponse strings.Builder
	var badJanetResponse strings.Builder

	if splits != nil {
		for _, split := range splits {

			splitText := split

			switch {
			case regexps.GivePoints.MatchString(splitText):
				goodJanetResponse.WriteString(b.applyPoints(ev, "", splitText))
				if len(goodJanetResponse.String()) > 0 {
					goodJanetResponse.WriteString("\n")
				}

			case regexps.TakePoints.MatchString(splitText):
				whichJanet := "badJanet"
				badJanetResponse.WriteString(b.applyPoints(ev, whichJanet, splitText))
				if len(badJanetResponse.String()) > 0 {
					badJanetResponse.WriteString("\n")
				}
			case regexps.Throwback.MatchString(ev.Text):
				b.getThrowback(ev)

			case regexps.QueryPoints.MatchString(ev.Text):
				b.queryPoints(ev)
			}
		}
		//send combined messages as good janet
		if len(goodJanetResponse.String()) > 0 {
			b.SendReply(goodJanetResponse.String(), ev, "")
		}

		//send combined messages as bad janet
		if len(badJanetResponse.String()) > 0 {
			b.SendReply(badJanetResponse.String(), ev, "badJanet")
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

func (b *Bot) printURL(ev *slack.MessageEvent) {
	url, err := b.Config.UI.GetURL("/")
	if b.handleError(err, ev) {
		return
	}

	// ui is disabled
	if url == "" {
		return
	}

	b.SendReply(url, ev, "")
}

func (b *Bot) applyPoints(ev *slack.MessageEvent, whichJanet, splitText string) string {
	b.Config.Log.Info(whichJanet)

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

	from, err := b.getUserNameByID(ev.User)
	if b.handleError(err, ev) {
		return ""
	}
	to, err := b.parseUser(match[1])
	if b.handleError(err, ev) {
		return ""
	}
	to = strings.ToLower(to)

	if _, blacklisted := b.Config.UserBlacklist[to]; blacklisted {
		b.Config.Log.KV("user", to).Info("user is blacklisted, ignoring karma command")
		return ""
	}

	points := min(len(match[2])-1, b.Config.MaxPoints)
	if match[2][0] == '-' {
		points *= -1
	}
	reason := match[3]

	if !b.Config.SelfPoints && from == to {
		b.SendReply("you are not allowed to modify your own points.", ev, "badJanet")
		return ""
	}

	record := &database.Points{
		From:   from,
		To:     to,
		Points: points,
		Reason: reason,
	}

	err = b.Config.DB.InsertPoints(record)
	if b.handleError(err, ev) {
		return ""
	}

	pointsMsg, err := b.getUserPointsMessage(to, reason, points)
	if b.handleError(err, ev) {
		return ""
	}

	return pointsMsg
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
		user, err = b.parseUser(match[1])
		if b.handleError(err, ev) {
			return
		}
		user = strings.ToLower(user)
	} else {
		user, err = b.getUserNameByID(ev.User)
		if b.handleError(err, ev) {
			return
		}
	}

	throwback, err := b.Config.DB.GetThrowback(user)
	if err == database.ErrNoSuchUser {
		b.SendReply(fmt.Sprintf("could not find any karma operations for %s", user), ev, "")
		return
	}

	if b.handleError(err, ev) {
		return
	}

	date := humanize.Time(throwback.Timestamp)
	if throwback.Reason != "" {
		throwback.Reason = fmt.Sprintf(" for %s", throwback.Reason)
	}
	text := fmt.Sprintf("%s received %d points from %s %s%s", munge.Munge(throwback.To), throwback.Points.Points, munge.Munge(throwback.From), date, throwback.Reason)

	b.SendReply(text, ev, "")
}

func (b *Bot) getUserPointsMessage(name, reason string, points int) (string, error) {
	user, err := b.Config.DB.GetUser(name)
	if err != nil {
		return "", err
	}

	text := fmt.Sprintf("%s now has %d points (", name, user.Points)

	if points > 0 {
		text += "+"
	}
	text = fmt.Sprintf("%s%d", text, points)

	if reason != "" {
		text += fmt.Sprintf(" for %s", reason)
	}
	text += ")"

	return text, nil
}

func (b *Bot) printLeaderboard(ev *slack.MessageEvent) {
	match := regexps.Leaderboard.FindStringSubmatch(ev.Text)
	if len(match) == 0 {
		return
	}

	limit := b.Config.LeaderboardLimit
	if match[1] != "" {
		var err error
		limit, err = strconv.Atoi(match[1])
		if b.handleError(err, ev) {
			return
		}
	}

	text := fmt.Sprintf("*top %d leaderboard*\n", limit)

	url, err := b.Config.UI.GetURL(fmt.Sprintf("/leaderboard/%d", limit))
	if b.handleError(err, ev) {
		return
	}
	if url != "" {
		text = fmt.Sprintf("%s%s\n", text, url)
	}

	leaderboard, err := b.Config.DB.GetLeaderboard(limit)
	if b.handleError(err, ev) {
		return
	}

	for i, user := range leaderboard {
		text += fmt.Sprintf("%d. %s == %d\n", i+1, munge.Munge(user.Name), user.Points)
	}

	b.SendReply(text, ev, "")
}

func (b *Bot) parseUser(user string) (string, error) {
	if match := regexps.SlackUser.FindStringSubmatch(user); len(match) > 0 {
		var err error
		user, err = b.getUserNameByID(match[1])
		if err != nil {
			return "", err
		}
	}

	// check if it is aliased
	if alias, ok := b.Config.Aliases[user]; ok {
		user = alias
	}

	return user, nil
}

func (b *Bot) getUserNameByID(id string) (string, error) {
	userInfo, err := b.Config.Slack.GetUserInfo(id)
	if err != nil {
		return "", err
	}

	return userInfo.Name, nil
}

func (b *Bot) queryPoints(ev *slack.MessageEvent) {
	match := regexps.QueryPoints.FindStringSubmatch(ev.Text)
	if len(match) == 0 {
		return
	}

	name, err := b.parseUser(match[1])
	if b.handleError(err, ev) {
		return
	}
	name = strings.ToLower(name)

	user, err := b.Config.DB.GetUser(name)
	switch {
	case err == database.ErrNoSuchUser:
		// override debug mode
		b.SendReply(err.Error(), ev, "")
	case b.handleError(err, ev):
	default:
		b.SendReply(fmt.Sprintf("%s == %d", user.Name, user.Points), ev, "")
	}
}
