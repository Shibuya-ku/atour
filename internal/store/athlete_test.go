package store

import (
	"context"
	"path/filepath"
	"testing"

	"atour/internal/ajp"
)

func TestSearchAthletesGroupsByUserID(t *testing.T) {
	ctx := context.Background()
	s, err := Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_ = s.Migrate(ctx)
	_ = s.ReplaceAll(ctx,
		[]EventRow{{EventID: 1, Title: "E", DateText: "1 Jan 2025", Location: "China"}},
		[]ajp.MatchRecord{
			{EventID: 1, BracketID: 10, Division: "Men's / Blue / GI", MatchID: 1,
				LeftName: "Zhiyuan Kong", LeftClub: "ClubA", LeftUserID: 100,
				RightName: "Other", RightUserID: 200, WinnerSide: "left"},
			{EventID: 1, BracketID: 11, Division: "Men's / Blue / NO-GI", MatchID: 2,
				LeftName: "Zhiyuan Kong", LeftClub: "ClubB", LeftUserID: 100,
				RightName: "X", RightUserID: 201, WinnerSide: "right"},
		},
		[]ajp.PlacementRecord{
			{EventID: 1, BracketID: 10, Division: "Men's / Blue / GI", Placement: 1,
				UserID: 100, Name: "Zhiyuan Kong", ClubName: "ClubA"},
		},
	)
	items, err := s.SearchAthletes(ctx, "kong", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].UserID != 100 {
		t.Fatalf("%+v", items)
	}
	if items[0].MatchCount < 1 || items[0].EventCount < 1 {
		t.Fatalf("%+v", items[0])
	}
	if len(items[0].Clubs) < 1 {
		t.Fatalf("clubs=%v", items[0].Clubs)
	}
}

func TestSearchAthletesRejectsShortQuery(t *testing.T) {
	ctx := context.Background()
	s, _ := Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	defer s.Close()
	_ = s.Migrate(ctx)
	_, err := s.SearchAthletes(ctx, "a", 50)
	if err == nil {
		t.Fatal("expected error")
	}
}
