package runtime

import "testing"

func TestBotOptionsNormalizeDefaultsAndValidation(t *testing.T) {
	opts := BotOptions{
		SlackToken:       "xoxb-token",
		SlackSocketToken: "xapp-token",
		MaxPoints:        -1,
		LeaderboardLimit: 1000,
		ReplyType:        "",
		UserBlacklist:    []string{" alice ", "", "alice", "bob"},
		UserAliases: map[string]string{
			" alice ": " bob ",
			"":        "nobody",
		},
	}

	if err := opts.Normalize(); err != nil {
		t.Fatalf("Normalize returned error: %v", err)
	}

	if opts.MaxPoints != 5 {
		t.Fatalf("expected MaxPoints default 5, got %d", opts.MaxPoints)
	}
	if opts.LeaderboardLimit != 10 {
		t.Fatalf("expected LeaderboardLimit default 10, got %d", opts.LeaderboardLimit)
	}
	if opts.ReplyType != "thread" {
		t.Fatalf("expected ReplyType thread, got %q", opts.ReplyType)
	}
	if len(opts.UserBlacklist) != 2 {
		t.Fatalf("expected deduplicated blacklist, got %#v", opts.UserBlacklist)
	}
	if opts.UserAliases["alice"] != "bob" {
		t.Fatalf("expected normalized alias map, got %#v", opts.UserAliases)
	}
}

func TestBotOptionsNormalizeRequiresTokens(t *testing.T) {
	opts := BotOptions{}
	if err := opts.Normalize(); err == nil {
		t.Fatal("expected missing token error")
	}
}
