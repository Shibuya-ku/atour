package ajp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientGetJSONSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/en/event/1/schedule/brackets.json" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if ua := r.Header.Get("User-Agent"); ua == "" {
			t.Fatal("missing UA")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"brackets":[{"id":1,"name":"Men's GI / Blue / Amateur / 69KG","bracket_id":9}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, WithMinInterval(0))
	var out bracketsJSON
	if err := c.GetJSON(context.Background(), "/en/event/1/schedule/brackets.json", &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Brackets) != 1 || out.Brackets[0].BracketID != 9 {
		t.Fatalf("%+v", out)
	}
}

func TestClientRetriesOn500(t *testing.T) {
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n < 3 {
			w.WriteHeader(500)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, WithMinInterval(0), WithRetry(3, time.Millisecond))
	var dest map[string]any
	if err := c.GetJSON(context.Background(), "/x", &dest); err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("retries=%d", n)
	}
}
