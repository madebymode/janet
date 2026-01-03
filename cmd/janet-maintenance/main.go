package main

import (
	"flag"
	"fmt"
	stdlog "log"
	"os"

	"github.com/joho/godotenv"
	"github.com/troyxmccall/janet/database"
)

func main() {
	_ = godotenv.Load()

	var fillChannelIDs bool
	flag.BoolVar(&fillChannelIDs, "fill-channel-ids", false, "fill missing channel_id values from other rows with the same message_id")
	flag.Parse()

	if !fillChannelIDs {
		stdlog.Fatal("no action specified, use -fill-channel-ids")
	}

	driver := getEnv("JANET_DATABASE_DRIVER", "postgres")
	url := getEnv("JANET_DATABASE_URL", "")
	if url == "" {
		stdlog.Fatal("JANET_DATABASE_URL is required")
	}

	db, err := database.NewV2(&database.Config{
		Driver: driver,
		URL:    url,
	})
	if err != nil {
		stdlog.Fatal("failed to initialize database:", err)
	}
	defer db.Close()

	updated, err := db.BackfillChannelIDsFromTransactions()
	if err != nil {
		stdlog.Fatal("failed to backfill channel ids:", err)
	}

	fmt.Printf("updated %d rows\n", updated)
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
