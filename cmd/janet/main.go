package main

import (
	"flag"
	"strings"

	"github.com/troyxmccall/janet"
	"github.com/troyxmccall/janet/database"
	janetui "github.com/troyxmccall/janet/ui"
	"github.com/troyxmccall/janet/ui/blankui"
	"github.com/troyxmccall/janet/ui/webui"

	"github.com/aybabtme/log"
	"github.com/nlopes/slack"
	"github.com/troyxmccall/envy"
)

// cli flags
var (
	token            = flag.String("token", "", "slack RTM token for Good Janet")
	badJanetToken    = flag.String("badJanetToken", "", "slack RTM token for Bad Janet")
	dbpath           = flag.String("db", "./db.sqlite3", "path to sqlite database")
	maxpoints        = flag.Int("maxpoints", 5, "the maximum amount of points that users can give/take at once")
	leaderboardlimit = flag.Int("leaderboardlimit", 10, "the default amount of users to list in the leaderboard")
	debug            = flag.Bool("debug", false, "set debug mode")
	webuitotp        = flag.String("webui.totp", "", "totp key")
	webuipath        = flag.String("webui.path", "", "path to web UI files")
	webuilistenaddr  = flag.String("webui.listenaddr", "", "address to listen and serve the web ui on")
	webuiurl         = flag.String("webui.url", "", "url address for accessing the web ui")
	motivate         = flag.Bool("motivate", true, "toggle motivate.im support")
	blacklist        = make(janet.StringList, 0)
	reactji          = flag.Bool("reactji", true, "use reactji as karma operations")
	upvotereactji    = make(janet.StringList, 0)
	downvotereactji  = make(janet.StringList, 0)
	repeatreactji    = make(janet.StringList, 0)
	aliases          = make(janet.StringList, 0)
	selfkarma        = flag.Bool("selfkarma", false, "allow users to add/remove karma to themselves")
	replytype        = flag.String("replytype", "thread", "how to reply to commands (message, thread)")
)

func main() {
	// logging

	ll := log.KV("version", janet.Version)

	// cli flags

	flag.Var(&blacklist, "blacklist", "blacklist users from having karma operations applied on them")
	flag.Var(&aliases, "alias", "alias different users to one user")
	flag.Var(&upvotereactji, "reactji.upvote", "a list of reactjis to use for upvotes")
	flag.Var(&downvotereactji, "reactji.downvote", "a list of reactjis to use for downvotes")
	flag.Var(&repeatreactji, "reactji.repeat", "a list of reactjis to use for repeating points")

	envy.Parse("KB")
	flag.Parse()

	// startup

	ll.Info("starting both janets")
	// reactjis

	// reactji defaults
	if len(upvotereactji) == 0 {
		upvotereactji.Set("+1")
		upvotereactji.Set("thumbsup")
		upvotereactji.Set("thumbsup_all")
		upvotereactji.Set("joy")
		upvotereactji.Set("100")
	}
	if len(downvotereactji) == 0 {
		downvotereactji.Set("-1")
		downvotereactji.Set("thumbsdown")
	}

	if len(repeatreactji) == 0 {
		repeatreactji.Set("bangbang")
	}

	reactjiConfig := &janet.ReactjiConfig{
		Enabled:      *reactji,
		UpVote:       upvotereactji,
		DownVote:     downvotereactji,
		RepeatPoints: repeatreactji,
	}

	// format aliases
	aliasMap := make(janet.UserAliases, 0)
	for k := range aliases {
		users := strings.Split(k, "++")
		if len(users) <= 1 {
			ll.Fatal("invalid alias format. see documentation")
		}

		user := users[0]
		for _, alias := range users[1:] {
			aliasMap[alias] = user
		}
	}

	// database

	db, err := database.New(&database.Config{
		Path: *dbpath,
	})

	if err != nil {
		ll.KV("path", *dbpath).Err(err).Fatal("could not open sqlite db")
	}

	// slack

	if *token == "" {
		ll.Fatal("please pass the slack RTM token (see `janet -h` for help)")
	}

	if *badJanetToken == "" {
		ll.Fatal("please pass a slack RTM token for badJanet (see `janet -h` for help)")
	}

	//TODO: figure out a way to fix this
	//our current logging library does not implement
	//log.Logger
	//slack.SetLogger(*ll)
	SlackConnection := slack.New(*token, slack.OptionDebug(*debug)).NewRTM()
	go SlackConnection.ManageConnection()

	badJanetSlackConnection := slack.New(*badJanetToken, slack.OptionDebug(*debug)).NewRTM()
	go badJanetSlackConnection.ManageConnection()

	// janet

	var ui janetui.Provider
	if *webuipath != "" && *webuilistenaddr != "" {
		ui, err = webui.New(&webui.Config{
			ListenAddr:       *webuilistenaddr,
			URL:              *webuiurl,
			FilesPath:        *webuipath,
			TOTP:             *webuitotp,
			LeaderboardLimit: *leaderboardlimit,
			Log:              ll.KV("provider", "webui"),
			Debug:            *debug,
			DB:               db,
		})

		if err != nil {
			ll.Err(err).Fatal("could not initialize web ui")
		}
	} else {
		ui = blankui.New()
	}
	go ui.Listen()

	bot := janet.New(&janet.Config{
		Slack:            &janet.SlackChatService{*SlackConnection},
		BadJanetSlack:    &janet.SlackChatService{*badJanetSlackConnection},
		UI:               ui,
		Debug:            *debug,
		MaxPoints:        *maxpoints,
		LeaderboardLimit: *leaderboardlimit,
		Log:              ll,
		DB:               db,
		UserBlacklist:    blacklist,
		Reactji:          reactjiConfig,
		Motivate:         *motivate,
		Aliases:          aliasMap,
		SelfPoints:       *selfkarma,
		ReplyType:        *replytype,
	})

	bot.Listen()
}
