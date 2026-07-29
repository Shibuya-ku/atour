package ajp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestRunE2E(t *testing.T) {
	calendar, _ := os.ReadFile("../../testdata/calendar_china_snippet.html")
	brackets1489, _ := os.ReadFile("../../testdata/brackets_1489_sample.json")
	render126903, _ := os.ReadFile("../../testdata/render_126903.json")
	placement126903, _ := os.ReadFile("../../testdata/placement_126903.json")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/en/events-1/events-calendar-2026":
			w.Header().Set("Content-Type", "text/html")
			w.Write(calendar)
		case r.URL.Path == "/en/event/1489/schedule/brackets.json":
			w.Header().Set("Content-Type", "application/json")
			w.Write(brackets1489)
		case len(r.URL.Path) > len("/en/event/1489/bracket/") &&
			(r.URL.Path[len(r.URL.Path)-len("/getRenderData"):] == "/getRenderData" ||
				r.URL.Path[len(r.URL.Path)-len("/getPlacementTableData"):] == "/getPlacementTableData") &&
			len(r.URL.Path) > 20 && r.URL.Path[:len("/en/event/1489/bracket/")] == "/en/event/1489/bracket/":
			w.Header().Set("Content-Type", "application/json")
			if len(r.URL.Path) >= len("/getPlacementTableData") &&
				r.URL.Path[len(r.URL.Path)-len("/getPlacementTableData"):] == "/getPlacementTableData" {
				w.Write(placement126903)
			} else {
				w.Write(render126903)
			}
		case r.URL.Path == "/en/event/2/schedule/brackets.json":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"brackets":[]}`))
		case r.URL.Path == "/en/event/1533/schedule/brackets.json" || r.URL.Path == "/en/event/1561/schedule/brackets.json":
			w.WriteHeader(http.StatusForbidden)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, WithMinInterval(0))
	results, err := Run(context.Background(), c, "/en/events-1/events-calendar-2026")
	if err != nil {
		t.Fatal(err)
	}

	var found1489 bool
	for _, er := range results {
		if er.Event.ID != 1489 {
			continue
		}
		found1489 = true
		if len(er.Brackets) < 2 {
			t.Fatalf("want White+Blue brackets, got %d", len(er.Brackets))
		}
		var hasBlue69 bool
		for _, br := range er.Brackets {
			if len(br.Matches) == 0 {
				t.Fatal("bracket has no matches")
			}
			if br.Matches[0].OpponentCount < 0 {
				t.Fatal("missing opponent_count")
			}
			if br.Meta.Name == "Men's GI / Blue / Amateur / 69KG" {
				hasBlue69 = true
				if br.Matches[0].OpponentCount != 5 {
					t.Fatalf("opp=%d", br.Matches[0].OpponentCount)
				}
			}
		}
		if !hasBlue69 {
			t.Fatal("missing blue 69kg bracket")
		}
	}
	if !found1489 {
		t.Fatal("no EventResult with ID 1489")
	}
}
