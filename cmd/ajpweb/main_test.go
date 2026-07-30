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

func TestAthleteSearchAndProfile(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "t.db")
	s, err := store.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_ = s.Migrate(ctx)
	_ = s.ReplaceAll(ctx,
		[]store.EventRow{{EventID: 1, Title: "E", DateText: "1 Jan 2025", Location: "China"}},
		[]ajp.MatchRecord{
			{EventID: 1, BracketID: 10, Division: "Men's / Blue / GI", MatchID: 1,
				LeftName: "Zhiyuan Kong", LeftClub: "ClubA", LeftUserID: 100,
				RightName: "Other", RightUserID: 200, WinnerSide: "left"},
		},
		[]ajp.PlacementRecord{
			{EventID: 1, BracketID: 10, Division: "Men's / Blue / GI", Placement: 1,
				UserID: 100, Name: "Zhiyuan Kong", ClubName: "ClubA"},
		},
	)

	web := filepath.Join(t.TempDir(), "web")
	os.MkdirAll(web, 0o755)
	mux := newMux(web, s)

	recShort := httptest.NewRecorder()
	mux.ServeHTTP(recShort, httptest.NewRequest(http.MethodGet, "/api/athletes/search?q=a", nil))
	if recShort.Code != 400 {
		t.Fatalf("search short q: code=%d body=%s", recShort.Code, recShort.Body.String())
	}

	recSearch := httptest.NewRecorder()
	mux.ServeHTTP(recSearch, httptest.NewRequest(http.MethodGet, "/api/athletes/search?q=kong", nil))
	if recSearch.Code != 200 {
		t.Fatalf("search: code=%d body=%s", recSearch.Code, recSearch.Body.String())
	}
	searchBody := recSearch.Body.String()
	if !strings.Contains(searchBody, `"items"`) || !strings.Contains(searchBody, `"user_id":100`) {
		t.Fatalf("search: %s", searchBody)
	}

	recProf := httptest.NewRecorder()
	mux.ServeHTTP(recProf, httptest.NewRequest(http.MethodGet, "/api/athletes/profile?user_ids=100", nil))
	if recProf.Code != 200 {
		t.Fatalf("profile: code=%d body=%s", recProf.Code, recProf.Body.String())
	}
	profBody := recProf.Body.String()
	if !strings.Contains(profBody, `"summary"`) || !strings.Contains(profBody, `"gold":1`) {
		t.Fatalf("profile: %s", profBody)
	}

	recNoIDs := httptest.NewRecorder()
	mux.ServeHTTP(recNoIDs, httptest.NewRequest(http.MethodGet, "/api/athletes/profile", nil))
	if recNoIDs.Code != 400 {
		t.Fatalf("profile no ids: code=%d body=%s", recNoIDs.Code, recNoIDs.Body.String())
	}
}
