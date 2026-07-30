package store

import (
	"context"
	"path/filepath"
	"testing"
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
