package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atour/internal/ajp"
	"atour/internal/store"
)

func TestAPIMatches(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "t.db")
	s, err := store.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_ = s.Migrate(ctx)
	_ = s.ReplaceAll(ctx,
		[]store.EventRow{{EventID: 1, Title: "E", Location: "China"}},
		[]ajp.MatchRecord{{EventID: 1, BracketID: 1, Division: "Men's / Blue / GI", MatchID: 7, LeftName: "Ann"}},
		nil,
	)

	web := filepath.Join(t.TempDir(), "web")
	os.MkdirAll(web, 0o755)
	os.WriteFile(filepath.Join(web, "index.html"), []byte("<html>ok</html>"), 0o644)

	mux := newMux(web, s)

	recEv := httptest.NewRecorder()
	mux.ServeHTTP(recEv, httptest.NewRequest(http.MethodGet, "/api/events", nil))
	if recEv.Code != 200 {
		t.Fatalf("events code=%d body=%s", recEv.Code, recEv.Body.String())
	}
	evBody := recEv.Body.String()
	if strings.Contains(evBody, `"total"`) {
		t.Fatalf("events must not include total: %s", evBody)
	}
	if !strings.Contains(evBody, `"event_id":1`) {
		t.Fatalf("events: %s", evBody)
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/matches?q=ann", nil))
	if rec.Code != 200 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"match_id":7`) {
		t.Fatalf("%s", rec.Body.String())
	}

	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec2.Code != 200 {
		t.Fatal(rec2.Code)
	}
}
