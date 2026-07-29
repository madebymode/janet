package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aybabtme/log"
)

type statsDataStore struct {
	DataStore
	err error
}

func (s statsDataStore) GetTotalPointsCumulative() (int, error) {
	if s.err != nil {
		return 0, s.err
	}
	return 120, nil
}

func (s statsDataStore) GetTotalUsersCumulative() (int, error) {
	if s.err != nil {
		return 0, s.err
	}
	return 12, nil
}

func (s statsDataStore) GetTotalTransactionsCumulative() (int, error) {
	if s.err != nil {
		return 0, s.err
	}
	return 30, nil
}

func (s statsDataStore) GetTotalPointsByYear(int) (int, error) {
	return s.GetTotalPointsCumulative()
}

func (s statsDataStore) GetTotalUsersByYear(int) (int, error) {
	return s.GetTotalUsersCumulative()
}

func (s statsDataStore) GetTotalTransactionsByYear(int) (int, error) {
	return s.GetTotalTransactionsCumulative()
}

func TestHandleAPIStatsV2RejectsInvalidYear(t *testing.T) {
	h := &Handler{db: statsDataStore{}, logger: log.KV("test", "stats")}
	req := httptest.NewRequest("GET", "/api/stats?year=bad", nil)
	rr := httptest.NewRecorder()

	h.HandleAPIStatsV2(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleAPIStatsV2ReturnsInternalServerErrorOnDBFailure(t *testing.T) {
	h := &Handler{db: statsDataStore{err: errors.New("database unavailable")}, logger: log.KV("test", "stats")}
	req := httptest.NewRequest("GET", "/api/stats", nil)
	rr := httptest.NewRecorder()

	h.HandleAPIStatsV2(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandleAPIStatsV2ReturnsStats(t *testing.T) {
	h := &Handler{db: statsDataStore{}, logger: log.KV("test", "stats")}
	req := httptest.NewRequest("GET", "/api/stats", nil)
	rr := httptest.NewRecorder()

	h.HandleAPIStatsV2(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{`"totalPoints":120`, `"totalUsers":12`, `"totalTransactions":30`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected response to contain %s, got %s", want, body)
		}
	}
}
