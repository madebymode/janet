package handlers

import (
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestParseOptionalYearValue(t *testing.T) {
	currentMax := max(maxYear, time.Now().Year()+1)

	tests := []struct {
		name    string
		value   string
		want    int
		wantErr bool
	}{
		{name: "empty", value: "", want: 0},
		{name: "valid", value: "2026", want: 2026},
		{name: "too low", value: "2019", wantErr: true},
		{name: "too high", value: "9999", wantErr: true},
		{name: "current max", value: strconv.Itoa(currentMax), want: currentMax},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOptionalYearValue(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error=%v, got %v", tt.wantErr, err)
			}
			if got != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, got)
			}
		})
	}
}

func TestParseIntAndBoolQuery(t *testing.T) {
	req := httptest.NewRequest("GET", "/?limit=200&offset=-3&funny_bias=yes&include_meta=0", nil)

	if got := parseIntQuery(req, "limit", 15, 1, 100); got != 100 {
		t.Fatalf("expected clamped limit 100, got %d", got)
	}
	if got := parseIntQuery(req, "offset", 10, 0, 100); got != 0 {
		t.Fatalf("expected clamped offset 0, got %d", got)
	}
	if !parseBoolQuery(req, "funny_bias") {
		t.Fatal("expected funny_bias true")
	}
	if parseBoolQuery(req, "include_meta") {
		t.Fatal("expected include_meta false")
	}
}

func TestSanitizeUsernameFilter(t *testing.T) {
	got, err := sanitizeUsernameFilter("  alice.smith  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "alice.smith" {
		t.Fatalf("expected trimmed username, got %q", got)
	}

	if _, err := sanitizeUsernameFilter("<@U123>"); err == nil {
		t.Fatal("expected invalid username error")
	}
}
