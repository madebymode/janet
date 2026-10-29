package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aybabtme/log"
	"github.com/troyxmccall/janet/database"
)

type funFactsDataStore struct {
	DataStore
	facts *database.FunFacts
	err   error
}

func (s funFactsDataStore) GetFunFacts(year int) (*database.FunFacts, error) {
	return s.facts, s.err
}

func (s funFactsDataStore) GetFunFactsByYearMonth(year, month int) (*database.FunFacts, error) {
	return s.facts, s.err
}

func mockFunFacts() *database.FunFacts {
	ch := "general"
	reason := "amazing work on the release"
	return &database.FunFacts{
		MostActiveHour:      14,
		MostActiveHourCount: 42,
		MostActiveDOW:       3,
		MostActiveDOWCount:  18,
		BiggestKarmaEvent: &database.BiggestKarmaEvent{
			FromUser: "alice",
			ToUser:   "bob",
			Points:   5,
			Reason:   &reason,
		},
		MostActiveChannel:      &ch,
		MostActiveChannelCount: 99,
		TotalUniqueChannels:    7,
	}
}

func TestHandleAPIFunFactsRejectsInvalidYear(t *testing.T) {
	h := &Handler{db: funFactsDataStore{facts: mockFunFacts()}, logger: log.KV("test", "facts")}
	req := httptest.NewRequest("GET", "/api/stats/fun-facts?year=bad", nil)
	rr := httptest.NewRecorder()

	h.HandleAPIFunFacts(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleAPIFunFactsRejectsInvalidMonth(t *testing.T) {
	h := &Handler{db: funFactsDataStore{facts: mockFunFacts()}, logger: log.KV("test", "facts")}
	req := httptest.NewRequest("GET", "/api/stats/fun-facts?month=notamonth", nil)
	rr := httptest.NewRecorder()

	h.HandleAPIFunFacts(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleAPIFunFactsReturnsInternalServerErrorOnDBFailure(t *testing.T) {
	h := &Handler{db: funFactsDataStore{err: errors.New("db down")}, logger: log.KV("test", "facts")}
	req := httptest.NewRequest("GET", "/api/stats/fun-facts", nil)
	rr := httptest.NewRecorder()

	h.HandleAPIFunFacts(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandleAPIFunFactsReturnsFunFacts(t *testing.T) {
	h := &Handler{db: funFactsDataStore{facts: mockFunFacts()}, logger: log.KV("test", "facts")}
	req := httptest.NewRequest("GET", "/api/stats/fun-facts", nil)
	rr := httptest.NewRecorder()

	h.HandleAPIFunFacts(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{
		`"most_active_hour":14`,
		`"most_active_dow":3`,
		`"most_active_channel":"general"`,
		`"total_unique_channels":7`,
		`"from_user":"alice"`,
		`"to_user":"bob"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected response to contain %s, got: %s", want, body)
		}
	}
}

func TestHandleAPIFunFactsByYearMonthReturnsFunFacts(t *testing.T) {
	h := &Handler{db: funFactsDataStore{facts: mockFunFacts()}, logger: log.KV("test", "facts")}
	req := httptest.NewRequest("GET", "/api/stats/fun-facts?year=2025&month=3", nil)
	rr := httptest.NewRecorder()

	h.HandleAPIFunFacts(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `"most_active_hour":14`) {
		t.Fatalf("expected fun facts in response, got: %s", body)
	}
}

func TestHandleAPIFunFactsByYearMonthErrorOnDBFailure(t *testing.T) {
	h := &Handler{db: funFactsDataStore{err: errors.New("db down")}, logger: log.KV("test", "facts")}
	req := httptest.NewRequest("GET", "/api/stats/fun-facts?year=2025&month=3", nil)
	rr := httptest.NewRecorder()

	h.HandleAPIFunFacts(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}
