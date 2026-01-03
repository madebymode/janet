package main

import (
	"embed"
	stdlog "log"
	"os"

	slackgo "github.com/slack-go/slack"
	"github.com/troyxmccall/janet/janet-server/handlers"
	"github.com/troyxmccall/janet/janet-server/server"
	"github.com/troyxmccall/janet/janet-server/slack"
)

//go:embed web/static/* web/templates/*
var webFS embed.FS

func main() {
	configPath := ""
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	// Create server with embedded filesystem
	srv, err := server.NewServer(configPath, webFS)
	if err != nil {
		stdlog.Fatal("Failed to create server:", err)
	}

	// Initialize Slack service
	var slackService handlers.SlackService
	if srv.GetBot() != nil {
		// Bot is enabled, use bot's Slack client
		slackService = slack.NewService(srv.GetBot())
		srv.GetLogger().Info("using bot's Slack client")
	} else {
		// Bot is disabled, create standalone Slack client for API calls
		// Use JANET_WEB_TOKEN if available, otherwise fall back to JANET_SLACK_TOKEN
		webToken := os.Getenv("JANET_WEB_TOKEN")
		if webToken == "" {
			webToken = os.Getenv("JANET_SLACK_TOKEN")
		}

		if webToken != "" {
			slackClient := slackgo.New(webToken)
			slackService = slack.NewWebService(slackClient)
			srv.GetLogger().Info("using standalone Slack client for web mode")
		} else {
			// No Slack client available
			slackService = slack.NewWebService(nil)
			srv.GetLogger().Error("no Slack token available, Slack features disabled")
		}
	}

	// Initialize handlers
	handlerService := handlers.NewHandler(srv.GetDB(), srv.GetBot(), srv.GetLogger(), slackService)

	// Register handlers with server
	srv.RegisterHandlers(handlerService)

	// Start server
	if err := srv.Start(); err != nil {
		stdlog.Fatal("Server error:", err)
	}
}