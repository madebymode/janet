package main

import (
	"embed"
	stdlog "log"
	"os"

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
	slackService := slack.NewService(srv.GetBot())

	// Initialize handlers
	handlerService := handlers.NewHandler(srv.GetDB(), srv.GetBot(), srv.GetLogger(), slackService)

	// Register handlers with server
	srv.RegisterHandlers(handlerService)

	// Start server
	if err := srv.Start(); err != nil {
		stdlog.Fatal("Server error:", err)
	}
}