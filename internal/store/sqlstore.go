package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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
	rows, err := s.db.QueryContext(ctx, `
SELECT event_id, title, url, location, date_text, brackets_unavailable
FROM events ORDER BY event_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []EventRow
	for rows.Next() {
		var e EventRow
		var bracketsUnavailable int
		if err := rows.Scan(&e.EventID, &e.Title, &e.URL, &e.Location, &e.DateText, &bracketsUnavailable); err != nil {
			return nil, err
		}
		e.BracketsUnavailable = bracketsUnavailable != 0
		out = append(out, e)
	}
	return out, rows.Err()
}

func normalizeLimitOffset(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = 50000
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func appendDivisionFilters(q FilterQuery, where *strings.Builder, args *[]any) {
	if q.EventID != 0 {
		where.WriteString(" AND event_id = ?")
		*args = append(*args, q.EventID)
	}
	if q.Gender != "" {
		where.WriteString(" AND gender = ?")
		*args = append(*args, q.Gender)
	}
	if q.Belt != "" {
		where.WriteString(" AND belt = ?")
		*args = append(*args, q.Belt)
	}
	if q.Style != "" {
		where.WriteString(" AND style = ?")
		*args = append(*args, q.Style)
	}
}

func appendMatchQFilter(q string, where *strings.Builder, args *[]any) {
	q = strings.TrimSpace(q)
	if q == "" {
		return
	}
	like := "%" + strings.ToLower(q) + "%"
	where.WriteString(` AND (
  LOWER(left_name) LIKE ? OR LOWER(left_club) LIKE ? OR
  LOWER(right_name) LIKE ? OR LOWER(right_club) LIKE ?
)`)
	*args = append(*args, like, like, like, like)
}

func appendPlacementQFilter(q string, where *strings.Builder, args *[]any) {
	q = strings.TrimSpace(q)
	if q == "" {
		return
	}
	like := "%" + strings.ToLower(q) + "%"
	where.WriteString(` AND (
  LOWER(name) LIKE ? OR LOWER(club_name) LIKE ? OR LOWER(affiliation_name) LIKE ?
)`)
	*args = append(*args, like, like, like)
}

func (s *SQLStore) ListMatches(ctx context.Context, q FilterQuery) ([]ajp.MatchRecord, int, error) {
	var where strings.Builder
	where.WriteString("WHERE 1=1")
	var args []any
	appendDivisionFilters(q, &where, &args)
	if q.HideBye {
		where.WriteString(" AND is_bye = 0")
	}
	appendMatchQFilter(q.Q, &where, &args)

	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM matches "+where.String(), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit, offset := normalizeLimitOffset(q.Limit, q.Offset)
	queryArgs := append(append([]any{}, args...), limit, offset)
	rows, err := s.db.QueryContext(ctx, `
SELECT event_id, bracket_id, division, match_id, round_name, round, mat_name, mat_match_nr,
  state, is_bye, won_by, score_text, penalty_text,
  left_name, left_club, left_country, left_user_id, left_result,
  right_name, right_club, right_country, right_user_id, right_result,
  winner_side, estimated_start, registrations_count, opponent_count
FROM matches `+where.String()+`
ORDER BY event_id, bracket_id, match_id
LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []ajp.MatchRecord
	for rows.Next() {
		var m ajp.MatchRecord
		var isBye int
		if err := rows.Scan(
			&m.EventID, &m.BracketID, &m.Division, &m.MatchID, &m.RoundName, &m.Round, &m.MatName, &m.MatMatchNr,
			&m.State, &isBye, &m.WonBy, &m.ScoreText, &m.PenaltyText,
			&m.LeftName, &m.LeftClub, &m.LeftCountry, &m.LeftUserID, &m.LeftResult,
			&m.RightName, &m.RightClub, &m.RightCountry, &m.RightUserID, &m.RightResult,
			&m.WinnerSide, &m.EstimatedStart, &m.RegistrationsCount, &m.OpponentCount,
		); err != nil {
			return nil, 0, err
		}
		m.IsBye = isBye != 0
		items = append(items, m)
	}
	return items, total, rows.Err()
}

func (s *SQLStore) ListPlacements(ctx context.Context, q FilterQuery) ([]ajp.PlacementRecord, int, error) {
	var where strings.Builder
	where.WriteString("WHERE 1=1")
	var args []any
	appendDivisionFilters(q, &where, &args)
	appendPlacementQFilter(q.Q, &where, &args)

	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM placements "+where.String(), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit, offset := normalizeLimitOffset(q.Limit, q.Offset)
	queryArgs := append(append([]any{}, args...), limit, offset)
	rows, err := s.db.QueryContext(ctx, `
SELECT event_id, bracket_id, division, placement, user_id, name, club_name, affiliation_name,
  registrations_count, opponent_count
FROM placements `+where.String()+`
ORDER BY event_id, division, placement
LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []ajp.PlacementRecord
	for rows.Next() {
		var p ajp.PlacementRecord
		if err := rows.Scan(
			&p.EventID, &p.BracketID, &p.Division, &p.Placement, &p.UserID, &p.Name, &p.ClubName, &p.AffiliationName,
			&p.RegistrationsCount, &p.OpponentCount,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, p)
	}
	return items, total, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *SQLStore) ReplaceAll(ctx context.Context, events []EventRow, matches []ajp.MatchRecord, placements []ajp.PlacementRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, stmt := range []string{
		`DELETE FROM placements`,
		`DELETE FROM matches`,
		`DELETE FROM events`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	eventStmt, err := tx.PrepareContext(ctx, `
INSERT INTO events (event_id, title, url, location, date_text, brackets_unavailable)
VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer eventStmt.Close()

	for _, e := range events {
		if _, err := eventStmt.ExecContext(ctx, e.EventID, e.Title, e.URL, e.Location, e.DateText, boolToInt(e.BracketsUnavailable)); err != nil {
			return err
		}
	}

	matchStmt, err := tx.PrepareContext(ctx, `
INSERT INTO matches (
  event_id, bracket_id, division, gender, belt, style, match_id,
  round_name, round, mat_name, mat_match_nr, state, is_bye,
  won_by, score_text, penalty_text,
  left_name, left_club, left_country, left_user_id, left_result,
  right_name, right_club, right_country, right_user_id, right_result,
  winner_side, estimated_start, registrations_count, opponent_count
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer matchStmt.Close()

	for _, m := range matches {
		gender, belt, style := ParseDivision(m.Division)
		if _, err := matchStmt.ExecContext(ctx,
			m.EventID, m.BracketID, m.Division, gender, belt, style, m.MatchID,
			m.RoundName, m.Round, m.MatName, m.MatMatchNr, m.State, boolToInt(m.IsBye),
			m.WonBy, m.ScoreText, m.PenaltyText,
			m.LeftName, m.LeftClub, m.LeftCountry, m.LeftUserID, m.LeftResult,
			m.RightName, m.RightClub, m.RightCountry, m.RightUserID, m.RightResult,
			m.WinnerSide, m.EstimatedStart, m.RegistrationsCount, m.OpponentCount,
		); err != nil {
			return err
		}
	}

	placementStmt, err := tx.PrepareContext(ctx, `
INSERT INTO placements (
  event_id, bracket_id, division, gender, belt, style, placement, user_id,
  name, club_name, affiliation_name, registrations_count, opponent_count
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer placementStmt.Close()

	for _, p := range placements {
		gender, belt, style := ParseDivision(p.Division)
		if _, err := placementStmt.ExecContext(ctx,
			p.EventID, p.BracketID, p.Division, gender, belt, style, p.Placement, p.UserID,
			p.Name, p.ClubName, p.AffiliationName, p.RegistrationsCount, p.OpponentCount,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
