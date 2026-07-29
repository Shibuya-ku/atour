package ajp

import (
	"context"
	"os"
	"testing"
)

func TestParseCalendarHTMLChina(t *testing.T) {
	b, err := os.ReadFile("../../testdata/calendar_china_snippet.html")
	if err != nil {
		t.Fatal(err)
	}
	events, err := ParseCalendarHTML(b, "https://ajptour.com")
	if err != nil {
		t.Fatal(err)
	}
	china := DedupeEventsByID(FilterChinaEvents(events))
	ids := map[int]bool{}
	for _, e := range china {
		ids[e.ID] = true
	}
	for _, id := range []int{1489, 1533, 1561} {
		if !ids[id] {
			t.Fatalf("missing %d in %+v", id, china)
		}
	}
	if ids[2] {
		t.Fatal("brazil should be excluded after FilterChinaEvents")
	}
}

func TestParseLiveCalendarChina(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	c := NewClient("https://ajptour.com")
	body, code, err := c.GetBytes(context.Background(), "/en/events-1/events-calendar-2026")
	if err != nil || code != 200 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	events, err := ParseCalendarHTML(body, "https://ajptour.com")
	if err != nil {
		t.Fatal(err)
	}
	china := DedupeEventsByID(FilterChinaEvents(events))
	if len(china) < 3 {
		t.Fatalf("expected >=3 china events, got %+v", china)
	}
}
