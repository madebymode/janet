package server

import (
	"net/http/httptest"
	"testing"
)

func TestGetRealClientIPUsesFirstValidForwardedAddress(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "garbage, 203.0.113.7, 10.0.0.2")

	if got := getRealClientIP(req); got != "203.0.113.7" {
		t.Fatalf("expected forwarded IP, got %q", got)
	}
}

func TestGetRealClientIPFallsBackToRemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.10:4321"
	req.Header.Set("X-Forwarded-For", "garbage")

	if got := getRealClientIP(req); got != "192.0.2.10" {
		t.Fatalf("expected remote addr fallback, got %q", got)
	}
}
