package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"atour/internal/ajp"
)

func TestWriteAll(t *testing.T) {
	dir := t.TempDir()
	results := []ajp.EventResult{{
		Event: ajp.EventSummary{ID: 1489, Title: "ERLIAN", Location: "China"},
		Brackets: []ajp.BracketResult{{
			Meta:       ajp.BracketMeta{BracketID: 1, Name: "Men's GI / Blue / Amateur / 69KG"},
			Matches:    []ajp.MatchRecord{{MatchID: 1, LeftName: "A", RightName: "B", WinnerSide: "left"}},
			Placements: []ajp.PlacementRecord{{Placement: 1, Name: "A"}},
		}},
	}}
	if err := WriteAll(dir, results); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"events.json", "matches.json", "placements.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatal(name, err)
		}
	}
	b, _ := os.ReadFile(filepath.Join(dir, "matches.json"))
	var matches []ajp.MatchRecord
	json.Unmarshal(b, &matches)
	if len(matches) != 1 {
		t.Fatalf("%d", len(matches))
	}
}

func TestMergeByEventID(t *testing.T) {
	base := []ajp.EventResult{{Event: ajp.EventSummary{ID: 1, Title: "old"}}}
	extra := []ajp.EventResult{
		{Event: ajp.EventSummary{ID: 1, Title: "new"}},
		{Event: ajp.EventSummary{ID: 2, Title: "b"}},
	}
	out := MergeByEventID(base, extra)
	if len(out) != 2 {
		t.Fatalf("%d", len(out))
	}
	if out[0].Event.Title != "new" || out[1].Event.ID != 2 {
		t.Fatalf("%+v", out)
	}
}

func TestLoadEventsMissing(t *testing.T) {
	got, err := LoadEvents(t.TempDir())
	if err != nil || len(got) != 0 {
		t.Fatalf("got=%v err=%v", got, err)
	}
}
