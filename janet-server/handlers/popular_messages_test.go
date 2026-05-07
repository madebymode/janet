package handlers

import (
	"net/http/httptest"
	"testing"
)

func TestParsePopularMessagesQuery(t *testing.T) {
	req := httptest.NewRequest("GET", "/?limit=99&offset=-1&year=2026&user=alice&min_reactions=7&has_media=1&funny_bias=true&include_meta=on", nil)

	query, err := parsePopularMessagesQuery(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if query.Limit != 15 {
		t.Fatalf("expected clamped limit 15, got %d", query.Limit)
	}
	if query.Offset != 0 {
		t.Fatalf("expected clamped offset 0, got %d", query.Offset)
	}
	if query.Year != 2026 {
		t.Fatalf("expected year 2026, got %d", query.Year)
	}
	if query.FilterUser != "alice" {
		t.Fatalf("expected filter user alice, got %q", query.FilterUser)
	}
	if query.MinReactions != 7 || !query.MediaOnly || !query.FunnyBias || !query.IncludeMeta {
		t.Fatalf("unexpected parsed query %#v", query)
	}
}

func TestPaginate(t *testing.T) {
	start, end := paginate(50, 10, 12)
	if start != 12 || end != 12 {
		t.Fatalf("expected capped pagination 12,12 got %d,%d", start, end)
	}

	start, end = paginate(3, 5, 10)
	if start != 3 || end != 8 {
		t.Fatalf("expected 3,8 got %d,%d", start, end)
	}
}

func TestMinPopularQueuePosition(t *testing.T) {
	h := &Handler{}
	items := []map[string]interface{}{
		{"pending_details": false, "queue_position": 5},
		{"pending_details": true, "queue_position": 4},
		{"pending_details": true, "queue_position": 2},
	}

	if got := h.minPopularQueuePosition(items); got != 2 {
		t.Fatalf("expected min queue position 2, got %d", got)
	}
}
