package ajp

import (
	"encoding/json"
	"os"
	"testing"
)

func TestFilterBluePlusBrackets(t *testing.T) {
	in := []BracketMeta{
		{Name: "Men's GI / Blue / Amateur / 69KG", BracketID: 1},
		{Name: "Youth Men's GI / Blue / 55KG", BracketID: 2},
		{Name: "Men's GI / White / Amateur / 69KG", BracketID: 3},
		{Name: "Men's GI / Grey / Amateur / 69KG", BracketID: 4},
	}
	out := FilterBluePlusBrackets(in)
	if len(out) != 2 {
		t.Fatalf("%+v", out)
	}
	ids := map[int]bool{}
	for _, b := range out {
		ids[b.BracketID] = true
	}
	if !ids[1] || !ids[3] {
		t.Fatalf("%+v", out)
	}
}

func TestFlattenMatchesFromFixture(t *testing.T) {
	b, _ := os.ReadFile("../../testdata/render_126903.json")
	var rd RenderData
	if err := json.Unmarshal(b, &rd); err != nil {
		t.Fatal(err)
	}
	recs, err := FlattenMatches(1489, "Men's GI / Blue / Amateur / 69KG", rd, 6)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) < 6 {
		t.Fatalf("matches=%d", len(recs))
	}
	var final bool
	for _, m := range recs {
		if m.OpponentCount != 5 || m.RegistrationsCount != 6 {
			t.Fatalf("opponent fields %+v", m)
		}
		if m.RoundName == "Final" && m.WonBy == "points" {
			final = true
			if m.LeftName == "" || m.RightName == "" {
				t.Fatal("missing names")
			}
			if m.WinnerSide == "" {
				t.Fatal("missing winner")
			}
		}
	}
	if !final {
		t.Fatal("no final match")
	}
}

func TestFlattenPlacementsFromFixture(t *testing.T) {
	b, _ := os.ReadFile("../../testdata/placement_126903.json")
	var pd PlacementTableData
	json.Unmarshal(b, &pd)
	recs := FlattenPlacements(1489, "Men's GI / Blue / Amateur / 69KG", 126903, pd, 6)
	if len(recs) != 6 {
		t.Fatalf("%d", len(recs))
	}
	if recs[0].Placement != 1 || recs[0].Name == "" {
		t.Fatalf("%+v", recs[0])
	}
	if recs[0].OpponentCount != 5 {
		t.Fatalf("opp=%d", recs[0].OpponentCount)
	}
}
