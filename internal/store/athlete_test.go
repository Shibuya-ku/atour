package store

import (
	"context"
	"errors"
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

func TestAthleteProfileStatsAndEncounters(t *testing.T) {
	ctx := context.Background()
	s, _ := Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	defer s.Close()
	_ = s.Migrate(ctx)
	_ = s.ReplaceAll(ctx,
		[]EventRow{{EventID: 1, Title: "E1", DateText: "d1", Location: "China"}},
		[]ajp.MatchRecord{
			{EventID: 1, BracketID: 10, Division: "Men's / Purple / GI", MatchID: 1,
				LeftName: "A", LeftUserID: 1, LeftClub: "C1",
				RightName: "B", RightUserID: 2, RightClub: "C2",
				WinnerSide: "left", WonBy: "POINTS", ScoreText: "2-0", IsBye: false,
				OpponentCount: 3, RoundName: "Final"},
			{EventID: 1, BracketID: 10, Division: "Men's / Purple / GI", MatchID: 2,
				LeftName: "A", LeftUserID: 1, IsBye: true, OpponentCount: 3},
		},
		[]ajp.PlacementRecord{
			{EventID: 1, BracketID: 10, Division: "Men's / Purple / GI", Placement: 1, UserID: 1, Name: "A", ClubName: "C1"},
			{EventID: 1, BracketID: 10, Division: "Men's / Purple / GI", Placement: 0, UserID: 9, Name: "Z"},
		},
	)
	prof, err := s.AthleteProfile(ctx, []int{1})
	if err != nil {
		t.Fatal(err)
	}
	if prof.Summary.Wins != 1 || prof.Summary.Byes != 1 || prof.Summary.Losses != 0 {
		t.Fatalf("summary=%+v", prof.Summary)
	}
	if prof.Summary.Gold != 1 || prof.Summary.NoPlacement != 0 {
		t.Fatalf("summary=%+v", prof.Summary)
	}
	if len(prof.Encounters) != 1 {
		t.Fatalf("encounters=%d", len(prof.Encounters))
	}
	if prof.Encounters[0].OpponentName != "B" || prof.Encounters[0].Result != "win" {
		t.Fatalf("%+v", prof.Encounters[0])
	}
	if len(prof.Timeline) != 1 {
		t.Fatalf("timeline=%d", len(prof.Timeline))
	}
	if prof.Timeline[0].PlacementLabel != "1" {
		t.Fatalf("placement_label=%q", prof.Timeline[0].PlacementLabel)
	}
	if prof.Summary.Belts["Purple"] != 1 || prof.Summary.Styles["GI"] != 1 {
		t.Fatalf("belts=%v styles=%v", prof.Summary.Belts, prof.Summary.Styles)
	}
}

func TestAthleteProfileNoPlacementAndMergeIDs(t *testing.T) {
	ctx := context.Background()
	s, _ := Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	defer s.Close()
	_ = s.Migrate(ctx)
	_ = s.ReplaceAll(ctx,
		[]EventRow{
			{EventID: 1, Title: "E1", DateText: "d1", Location: "China"},
			{EventID: 2, Title: "E2", DateText: "d2", Location: "China"},
		},
		[]ajp.MatchRecord{
			{EventID: 1, BracketID: 10, Division: "Men's / Blue / GI", MatchID: 1,
				LeftName: "A", LeftUserID: 1, LeftClub: "C1",
				RightName: "X", RightUserID: 99, RightClub: "CX",
				WinnerSide: "left", OpponentCount: 4},
			{EventID: 2, BracketID: 20, Division: "Men's / Purple / NO-GI", MatchID: 1,
				LeftName: "B", LeftUserID: 2, LeftClub: "C2",
				RightName: "Y", RightUserID: 98, RightClub: "CY",
				WinnerSide: "left", OpponentCount: 5},
		},
		[]ajp.PlacementRecord{
			{EventID: 1, BracketID: 10, Division: "Men's / Blue / GI", Placement: 0, UserID: 1, Name: "A", ClubName: "C1"},
			{EventID: 2, BracketID: 20, Division: "Men's / Purple / NO-GI", Placement: 2, UserID: 2, Name: "B", ClubName: "C2"},
		},
	)
	prof, err := s.AthleteProfile(ctx, []int{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	if prof.Summary.Wins != 2 || prof.Summary.NoPlacement != 1 || prof.Summary.Silver != 1 {
		t.Fatalf("summary=%+v", prof.Summary)
	}
	if len(prof.Timeline) != 2 {
		t.Fatalf("timeline=%d", len(prof.Timeline))
	}
	if prof.Timeline[0].EventID != 2 {
		t.Fatalf("timeline order: first event_id=%d", prof.Timeline[0].EventID)
	}
	noPlace := prof.Timeline[1]
	if noPlace.PlacementLabel != "无正式名次" {
		t.Fatalf("placement_label=%q", noPlace.PlacementLabel)
	}
	if len(prof.Encounters) != 2 {
		t.Fatalf("encounters=%d", len(prof.Encounters))
	}
	if len(prof.Identities) != 2 {
		t.Fatalf("identities=%d", len(prof.Identities))
	}
}

func TestAthleteProfileEmptyUserIDs(t *testing.T) {
	s, _ := Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	defer s.Close()
	_, err := s.AthleteProfile(context.Background(), nil)
	if !errors.Is(err, ErrNoUserIDs) {
		t.Fatalf("nil: err=%v", err)
	}
	_, err = s.AthleteProfile(context.Background(), []int{})
	if !errors.Is(err, ErrNoUserIDs) {
		t.Fatalf("empty: err=%v", err)
	}
}
