package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"atour/internal/ajp"
)

func TestOpenMigrateSQLite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.db")
	s, err := Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestReplaceAndFilterMatches(t *testing.T) {
	ctx := context.Background()
	s, err := Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_ = s.Migrate(ctx)

	events := []EventRow{{EventID: 1, Title: "A", DateText: "2025-01-01"}}
	matches := []ajp.MatchRecord{
		{EventID: 1, BracketID: 10, Division: "Men's / Blue / GI", MatchID: 1, LeftName: "Zhang San", LeftClub: "ClubA", RightName: "Li", IsBye: false, OpponentCount: 3},
		{EventID: 1, BracketID: 10, Division: "Men's / Blue / GI", MatchID: 2, LeftName: "Bye", IsBye: true},
		{EventID: 1, BracketID: 11, Division: "Women's / White / NO-GI", MatchID: 3, LeftName: "Wang", RightName: "Zhao"},
	}
	if err := s.ReplaceAll(ctx, events, matches, nil); err != nil {
		t.Fatal(err)
	}

	items, total, err := s.ListMatches(ctx, FilterQuery{Q: "zhang", Gender: "Men's", Belt: "Blue", Style: "GI", HideBye: true})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(items) != 1 || items[0].MatchID != 1 {
		t.Fatalf("total=%d items=%+v", total, items)
	}

	evs, err := s.ListEvents(ctx)
	if err != nil || len(evs) != 1 || evs[0].EventID != 1 {
		t.Fatalf("%+v %v", evs, err)
	}
}

func TestImportJSONDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "events.json"), []byte(`[
	  {"event":{"event_id":9,"title":"T","url":"","location":"China","date_text":"d"},"brackets_unavailable":false,"brackets":[]}
	]`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "matches.json"), []byte(`[
	  {"event_id":9,"bracket_id":1,"division":"Men's / Blue / GI","match_id":1,"left_name":"A","right_name":"B","is_bye":false}
	]`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "placements.json"), []byte(`[
	  {"event_id":9,"bracket_id":1,"division":"Men's / Blue / GI","placement":1,"name":"A"}
	]`), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	s, err := Open("sqlite", filepath.Join(t.TempDir(), "i.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := ImportJSONDir(ctx, s, dir); err != nil {
		t.Fatal(err)
	}
	items, total, err := s.ListMatches(ctx, FilterQuery{})
	if err != nil || total != 1 || items[0].EventID != 9 {
		t.Fatalf("%d %+v %v", total, items, err)
	}
}

func TestParseDivision(t *testing.T) {
	g, b, st := ParseDivision("Men's / Purple / NO-GI Absolute")
	if g != "Men's" || b != "Purple" || st != "NO-GI" {
		t.Fatalf("%q %q %q", g, b, st)
	}
}

func TestOpenMySQLInvalidDSN(t *testing.T) {
	_, err := Open("mysql", "invalid")
	if err == nil {
		t.Fatal("expected error")
	}
}
