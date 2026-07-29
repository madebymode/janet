package server

import (
	"path/filepath"
	"testing"
)

func TestConfigNormalizeAppliesSafeDefaults(t *testing.T) {
	cfg := &Config{
		DatabaseURL:      "postgres://janet:janet@localhost:5432/janet?sslmode=disable",
		MaxPoints:        -5,
		LeaderboardLimit: 1000,
		ReplyType:        "",
		RateLimitRPS:     -1,
		RateLimitBurst:   -1,
		UserBlacklist:    []string{" alice ", "", "alice"},
		UserAliases: map[string]string{
			" alice ": " bob ",
			"":        "nobody",
		},
	}

	if err := cfg.normalize(); err != nil {
		t.Fatalf("normalize returned error: %v", err)
	}

	if cfg.MaxPoints != 5 {
		t.Fatalf("expected MaxPoints default 5, got %d", cfg.MaxPoints)
	}
	if cfg.LeaderboardLimit != 10 {
		t.Fatalf("expected LeaderboardLimit default 10, got %d", cfg.LeaderboardLimit)
	}
	if cfg.ReplyType != "thread" {
		t.Fatalf("expected ReplyType thread, got %q", cfg.ReplyType)
	}
	if cfg.RateLimitRPS != 20 {
		t.Fatalf("expected RateLimitRPS default 20, got %v", cfg.RateLimitRPS)
	}
	if cfg.RateLimitBurst != 60 {
		t.Fatalf("expected RateLimitBurst default 60, got %d", cfg.RateLimitBurst)
	}
	if len(cfg.UserBlacklist) != 1 || cfg.UserBlacklist[0] != "alice" {
		t.Fatalf("unexpected blacklist %#v", cfg.UserBlacklist)
	}
	if cfg.UserAliases["alice"] != "bob" {
		t.Fatalf("unexpected aliases %#v", cfg.UserAliases)
	}
}

func TestConfigNormalizeRejectsBadReplyType(t *testing.T) {
	cfg := &Config{
		DatabaseURL: "postgres://janet:janet@localhost:5432/janet?sslmode=disable",
		ReplyType:   "invalid",
	}

	if err := cfg.normalize(); err == nil {
		t.Fatal("expected invalid reply type error")
	}
}

func TestLoadConfigRejectsMissingExplicitConfig(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing.json")

	if _, err := LoadConfig(missingPath); err == nil {
		t.Fatal("expected missing config file error")
	}
}
