package ajp

import "testing"

func TestIsBluePlusAdultDivision(t *testing.T) {
	cases := map[string]bool{
		"Men's GI / Blue / Amateur / 69KG":            true,
		"Women's NO-GI / Black / Professional / 52KG": true,
		"Men's GI / White / Amateur / 69KG":           true,
		"Youth Men's GI / Blue / 55KG":                false,
		"Boys GI / Kids 3 / Blue / 30KG":              false,
		"Men's GI / Grey / Amateur / 69KG":            false,
	}
	for name, want := range cases {
		if got := IsBluePlusAdultDivision(name); got != want {
			t.Fatalf("%s: got %v want %v", name, got, want)
		}
	}
}

func TestOpponentCount(t *testing.T) {
	if OpponentCount(6) != 5 {
		t.Fatalf("got %d", OpponentCount(6))
	}
	if OpponentCount(1) != 0 || OpponentCount(0) != 0 {
		t.Fatal("expected 0")
	}
}

func TestFilterChinaAndDedupe(t *testing.T) {
	in := []EventSummary{
		{ID: 1489, Title: "ERLIAN GI", Location: "Erenhot, China"},
		{ID: 1489, Title: "ERLIAN NO-GI", Location: "Erenhot, China"},
		{ID: 2, Title: "RIO", Location: "Rio de Janeiro, Brazil"},
	}
	china := FilterChinaEvents(in)
	if len(china) != 2 {
		t.Fatalf("%d", len(china))
	}
	deduped := DedupeEventsByID(china)
	if len(deduped) != 1 || deduped[0].ID != 1489 {
		t.Fatalf("%+v", deduped)
	}
}
