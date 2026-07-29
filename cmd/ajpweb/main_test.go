package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestMuxServesWebAndData(t *testing.T) {
	root := t.TempDir()
	web := filepath.Join(root, "web")
	data := filepath.Join(root, "output")
	os.MkdirAll(web, 0o755)
	os.MkdirAll(data, 0o755)
	os.WriteFile(filepath.Join(web, "index.html"), []byte("<html>ok</html>"), 0o644)
	os.WriteFile(filepath.Join(data, "matches.json"), []byte(`[]`), 0o644)

	mux := newMux(web, data)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != 200 || rec.Body.String() != "<html>ok</html>" {
		t.Fatalf("web: code=%d body=%q", rec.Code, rec.Body.String())
	}

	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/data/matches.json", nil))
	if rec2.Code != 200 || rec2.Body.String() != `[]` {
		t.Fatalf("data: code=%d body=%q", rec2.Code, rec2.Body.String())
	}
}
