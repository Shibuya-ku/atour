package store

import (
	"context"
	"database/sql"
	"fmt"

	"atour/internal/ajp"

	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"
)

var _ Store = (*SQLStore)(nil)

type SQLStore struct {
	db     *sql.DB
	driver string
}

func openSQL(driver, dsn string) (*SQLStore, error) {
	var name string
	switch driver {
	case "sqlite":
		name = "sqlite"
	case "mysql":
		name = "mysql"
	default:
		return nil, fmt.Errorf("unsupported driver %q (want sqlite|mysql)", driver)
	}
	db, err := sql.Open(name, dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if driver == "sqlite" {
		_, _ = db.Exec(`PRAGMA foreign_keys = ON`)
	}
	return &SQLStore{db: db, driver: driver}, nil
}

func (s *SQLStore) Close() error { return s.db.Close() }

func (s *SQLStore) Migrate(ctx context.Context) error {
	stmts := schemaSQLite
	if s.driver == "mysql" {
		stmts = schemaMySQL
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLStore) ListEvents(ctx context.Context) ([]EventRow, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *SQLStore) ListMatches(ctx context.Context, q FilterQuery) ([]ajp.MatchRecord, int, error) {
	return nil, 0, fmt.Errorf("not implemented")
}

func (s *SQLStore) ListPlacements(ctx context.Context, q FilterQuery) ([]ajp.PlacementRecord, int, error) {
	return nil, 0, fmt.Errorf("not implemented")
}

func (s *SQLStore) ReplaceAll(ctx context.Context, events []EventRow, matches []ajp.MatchRecord, placements []ajp.PlacementRecord) error {
	return fmt.Errorf("not implemented")
}
