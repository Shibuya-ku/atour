package store

import (
	"context"
	"errors"
	"sort"
	"strings"
)

var ErrQueryTooShort = errors.New("query too short")

type AthleteIdentity struct {
	UserID       int
	Name         string
	Clubs        []string
	EventCount   int
	MatchCount   int
	LastDateText string
}

const searchAthletesMaxLimit = 50

type athleteHit struct {
	userID    int
	name      string
	club      string
	eventID   int
	dateText  string
	fromMatch bool
}

type athleteAgg struct {
	userID      int
	nameCounts  map[string]int
	clubs       map[string]struct{}
	eventIDs    map[int]struct{}
	matchCount  int
	lastEventID int
	lastDate    string
}

func (s *SQLStore) SearchAthletes(ctx context.Context, q string, limit int) ([]AthleteIdentity, error) {
	q = strings.TrimSpace(q)
	if len(q) < 2 {
		return nil, ErrQueryTooShort
	}
	if limit <= 0 {
		limit = searchAthletesMaxLimit
	}
	if limit > searchAthletesMaxLimit {
		limit = searchAthletesMaxLimit
	}

	like := "%" + strings.ToLower(q) + "%"
	hits, err := s.collectAthleteSearchHits(ctx, like)
	if err != nil {
		return nil, err
	}

	byUser := aggregateAthleteHits(hits)
	out := make([]AthleteIdentity, 0, len(byUser))
	for _, agg := range byUser {
		out = append(out, agg.toIdentity())
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].MatchCount != out[j].MatchCount {
			return out[i].MatchCount > out[j].MatchCount
		}
		return out[i].UserID < out[j].UserID
	})

	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *SQLStore) collectAthleteSearchHits(ctx context.Context, like string) ([]athleteHit, error) {
	var hits []athleteHit

	matchRows, err := s.db.QueryContext(ctx, `
SELECT m.left_user_id, m.left_name, m.left_club, m.event_id, e.date_text
FROM matches m
LEFT JOIN events e ON e.event_id = m.event_id
WHERE LOWER(m.left_name) LIKE ?`, like)
	if err != nil {
		return nil, err
	}
	for matchRows.Next() {
		var h athleteHit
		if err := matchRows.Scan(&h.userID, &h.name, &h.club, &h.eventID, &h.dateText); err != nil {
			matchRows.Close()
			return nil, err
		}
		h.fromMatch = true
		hits = append(hits, h)
	}
	if err := matchRows.Close(); err != nil {
		return nil, err
	}
	if err := matchRows.Err(); err != nil {
		return nil, err
	}

	matchRows, err = s.db.QueryContext(ctx, `
SELECT m.right_user_id, m.right_name, m.right_club, m.event_id, e.date_text
FROM matches m
LEFT JOIN events e ON e.event_id = m.event_id
WHERE LOWER(m.right_name) LIKE ?`, like)
	if err != nil {
		return nil, err
	}
	for matchRows.Next() {
		var h athleteHit
		if err := matchRows.Scan(&h.userID, &h.name, &h.club, &h.eventID, &h.dateText); err != nil {
			matchRows.Close()
			return nil, err
		}
		h.fromMatch = true
		hits = append(hits, h)
	}
	if err := matchRows.Close(); err != nil {
		return nil, err
	}
	if err := matchRows.Err(); err != nil {
		return nil, err
	}

	placementRows, err := s.db.QueryContext(ctx, `
SELECT p.user_id, p.name, p.club_name, p.event_id, e.date_text
FROM placements p
LEFT JOIN events e ON e.event_id = p.event_id
WHERE LOWER(p.name) LIKE ?`, like)
	if err != nil {
		return nil, err
	}
	defer placementRows.Close()
	for placementRows.Next() {
		var h athleteHit
		if err := placementRows.Scan(&h.userID, &h.name, &h.club, &h.eventID, &h.dateText); err != nil {
			return nil, err
		}
		hits = append(hits, h)
	}
	return hits, placementRows.Err()
}

func aggregateAthleteHits(hits []athleteHit) map[int]*athleteAgg {
	byUser := make(map[int]*athleteAgg)
	for _, h := range hits {
		agg, ok := byUser[h.userID]
		if !ok {
			agg = &athleteAgg{
				userID:     h.userID,
				nameCounts: make(map[string]int),
				clubs:      make(map[string]struct{}),
				eventIDs:   make(map[int]struct{}),
			}
			byUser[h.userID] = agg
		}
		if h.name != "" {
			agg.nameCounts[h.name]++
		}
		if club := strings.TrimSpace(h.club); club != "" {
			agg.clubs[club] = struct{}{}
		}
		if h.eventID != 0 {
			agg.eventIDs[h.eventID] = struct{}{}
		}
		if h.fromMatch {
			agg.matchCount++
		}
		// LastDateText：date_text 非 ISO，用最大 event_id 对应的 date_text 近似「最近」。
		if h.eventID > agg.lastEventID {
			agg.lastEventID = h.eventID
			agg.lastDate = h.dateText
		}
	}
	return byUser
}

func (a *athleteAgg) toIdentity() AthleteIdentity {
	name := ""
	best := 0
	for n, c := range a.nameCounts {
		if c > best || (c == best && (name == "" || n < name)) {
			name = n
			best = c
		}
	}
	clubs := make([]string, 0, len(a.clubs))
	for c := range a.clubs {
		clubs = append(clubs, c)
	}
	sort.Strings(clubs)
	return AthleteIdentity{
		UserID:       a.userID,
		Name:         name,
		Clubs:        clubs,
		EventCount:   len(a.eventIDs),
		MatchCount:   a.matchCount,
		LastDateText: a.lastDate,
	}
}
