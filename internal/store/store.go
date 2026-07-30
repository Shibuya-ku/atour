package store

import (
	"context"

	"atour/internal/ajp"
)

type FilterQuery struct {
	Q       string
	EventID int
	Gender  string
	Belt    string
	Style   string
	HideBye bool
	Limit   int
	Offset  int
}

type EventRow struct {
	EventID             int
	Title               string
	URL                 string
	Location            string
	DateText            string
	BracketsUnavailable bool
}

type Store interface {
	Close() error
	Migrate(ctx context.Context) error
	ListEvents(ctx context.Context) ([]EventRow, error)
	ListMatches(ctx context.Context, q FilterQuery) (items []ajp.MatchRecord, total int, err error)
	ListPlacements(ctx context.Context, q FilterQuery) (items []ajp.PlacementRecord, total int, err error)
	ReplaceAll(ctx context.Context, events []EventRow, matches []ajp.MatchRecord, placements []ajp.PlacementRecord) error
}

func Open(driver, dsn string) (Store, error) {
	return openSQL(driver, dsn)
}
