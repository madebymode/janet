package slack

import (
	"testing"

	goslack "github.com/slack-go/slack"
)

func TestCountMessageReactions(t *testing.T) {
	message := goslack.Message{
		Msg: goslack.Msg{
			Reactions: []goslack.ItemReaction{
				{Count: 2},
				{Count: 5},
			},
		},
	}

	if got := countMessageReactions(message); got != 7 {
		t.Fatalf("expected 7 reactions, got %d", got)
	}
}

func TestMessageCacheRoundTrip(t *testing.T) {
	svc := newService(nil, nil, ServiceOptions{}, nil)
	details := &MessageDetails{Text: "hello"}

	svc.setCachedMessageDetails("C1", "123.456", details)
	got, ok := svc.getCachedMessageDetails("C1", "123.456")
	if !ok {
		t.Fatal("expected cached details")
	}
	if got.Text != "hello" {
		t.Fatalf("expected cached text hello, got %q", got.Text)
	}
}
