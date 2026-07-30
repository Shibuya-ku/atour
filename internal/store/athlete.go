package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"atour/internal/ajp"
)

var ErrQueryTooShort = errors.New("query too short")
var ErrNoUserIDs = errors.New("no user ids")

const (
	profileTimelineMax   = 200
	profileEncountersMax = 500
)

type AthleteIdentity struct {
	UserID       int      `json:"user_id"`
	Name         string   `json:"name"`
	Clubs        []string `json:"clubs"`
	EventCount   int      `json:"event_count"`
	MatchCount   int      `json:"match_count"`
	LastDateText string   `json:"last_date_text"`
}

type ClubCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type AthleteSummary struct {
	Divisions     int            `json:"divisions"`
	Matches       int            `json:"matches"`
	Wins          int            `json:"wins"`
	Losses        int            `json:"losses"`
	Byes          int            `json:"byes"`
	Gold          int            `json:"gold"`
	Silver        int            `json:"silver"`
	Bronze        int            `json:"bronze"`
	NoPlacement   int            `json:"no_placement"`
	Belts         map[string]int   `json:"belts"`
	Styles        map[string]int   `json:"styles"`
	Clubs         []ClubCount      `json:"clubs"`
}

type TimelineEntry struct {
	EventID        int    `json:"event_id"`
	BracketID      int    `json:"bracket_id"`
	Title          string `json:"title"`
	DateText       string `json:"date_text"`
	Location       string `json:"location"`
	Division       string `json:"division"`
	Club           string `json:"club"`
	Placement      *int   `json:"placement,omitempty"`
	PlacementLabel string `json:"placement_label"`
	OpponentCount  int    `json:"opponent_count"`
	Wins           int    `json:"wins"`
	Losses         int    `json:"losses"`
	Byes           int    `json:"byes"`
}

type Encounter struct {
	EventID      int    `json:"event_id"`
	BracketID    int    `json:"bracket_id"`
	MatchID      int    `json:"match_id"`
	Title        string `json:"title"`
	DateText     string `json:"date_text"`
	Division     string `json:"division"`
	RoundName    string `json:"round_name"`
	OpponentName string `json:"opponent_name"`
	OpponentClub string `json:"opponent_club"`
	Result       string `json:"result"`
	WonBy        string `json:"won_by"`
	ScoreText    string `json:"score_text"`
}

type AthleteProfileResult struct {
	Identities []AthleteIdentity `json:"identities"`
	Summary    AthleteSummary    `json:"summary"`
	Timeline   []TimelineEntry   `json:"timeline"`
	Encounters []Encounter       `json:"encounters"`
	Truncated  bool              `json:"truncated"`
}

type bracketKey struct {
	eventID   int
	bracketID int
}

type timelineAcc struct {
	eventID       int
	bracketID     int
	title         string
	dateText      string
	location      string
	division      string
	club          string
	opponentCount int
	wins          int
	losses        int
	byes          int
	placement     *int
	hasPlacement  bool
}

type identityAcc struct {
	nameCounts map[string]int
	clubs      map[string]struct{}
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

func sideResult(userIDs map[int]bool, m ajp.MatchRecord) (win, loss, bye bool) {
	if m.IsBye {
		if userIDs[m.LeftUserID] || userIDs[m.RightUserID] {
			return false, false, true
		}
		return
	}
	onLeft := userIDs[m.LeftUserID]
	onRight := userIDs[m.RightUserID]
	if !onLeft && !onRight {
		return
	}
	switch m.WinnerSide {
	case "left":
		if onLeft {
			win = true
		} else {
			loss = true
		}
	case "right":
		if onRight {
			win = true
		} else {
			loss = true
		}
	}
	return
}

func encounterResult(onLeft, onRight bool, winnerSide string) string {
	switch winnerSide {
	case "left":
		if onLeft {
			return "win"
		}
		if onRight {
			return "loss"
		}
	case "right":
		if onRight {
			return "win"
		}
		if onLeft {
			return "loss"
		}
	}
	return "unknown"
}

func placementLabel(p *int) string {
	if p == nil {
		return ""
	}
	if *p == 0 {
		return "无正式名次"
	}
	return strconv.Itoa(*p)
}

func buildInPlaceholders(n int) string {
	if n == 0 {
		return ""
	}
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ",")
}

func (s *SQLStore) AthleteProfile(ctx context.Context, userIDs []int) (AthleteProfileResult, error) {
	if len(userIDs) == 0 {
		return AthleteProfileResult{}, ErrNoUserIDs
	}

	idSet := make(map[int]bool, len(userIDs))
	for _, id := range userIDs {
		idSet[id] = true
	}

	matches, err := s.loadProfileMatches(ctx, userIDs)
	if err != nil {
		return AthleteProfileResult{}, err
	}
	placements, err := s.loadProfilePlacements(ctx, userIDs)
	if err != nil {
		return AthleteProfileResult{}, err
	}

	timelineMap := make(map[bracketKey]*timelineAcc)
	clubCounts := make(map[string]int)
	identities := make(map[int]*identityAcc)
	for _, id := range userIDs {
		identities[id] = &identityAcc{
			nameCounts: make(map[string]int),
			clubs:      make(map[string]struct{}),
		}
	}

	var summary AthleteSummary
	summary.Belts = make(map[string]int)
	summary.Styles = make(map[string]int)

	for _, m := range matches {
		win, loss, bye := sideResult(idSet, m)
		if !win && !loss && !bye {
			continue
		}

		key := bracketKey{eventID: m.EventID, bracketID: m.BracketID}
		acc := timelineMap[key]
		if acc == nil {
			acc = &timelineAcc{
				eventID:       m.EventID,
				bracketID:     m.BracketID,
				division:      m.Division,
				opponentCount: m.OpponentCount,
			}
			timelineMap[key] = acc
		}
		if acc.division == "" {
			acc.division = m.Division
		}
		if m.OpponentCount > acc.opponentCount {
			acc.opponentCount = m.OpponentCount
		}
		acc.wins += boolCount(win)
		acc.losses += boolCount(loss)
		acc.byes += boolCount(bye)

		if !m.IsBye {
			summary.Matches++
		}
		summary.Wins += boolCount(win)
		summary.Losses += boolCount(loss)
		summary.Byes += boolCount(bye)

		onLeft := idSet[m.LeftUserID]
		onRight := idSet[m.RightUserID]
		recordIdentitySide(identities, m.LeftUserID, m.LeftName, m.LeftClub, onLeft)
		recordIdentitySide(identities, m.RightUserID, m.RightName, m.RightClub, onRight)

		if onLeft && strings.TrimSpace(m.LeftClub) != "" {
			clubCounts[strings.TrimSpace(m.LeftClub)]++
			if acc.club == "" {
				acc.club = strings.TrimSpace(m.LeftClub)
			}
		}
		if onRight && strings.TrimSpace(m.RightClub) != "" {
			clubCounts[strings.TrimSpace(m.RightClub)]++
			if acc.club == "" {
				acc.club = strings.TrimSpace(m.RightClub)
			}
		}
	}

	for _, p := range placements {
		key := bracketKey{eventID: p.EventID, bracketID: p.BracketID}
		acc := timelineMap[key]
		if acc == nil {
			acc = &timelineAcc{
				eventID:       p.EventID,
				bracketID:     p.BracketID,
				division:      p.Division,
				opponentCount: p.OpponentCount,
			}
			timelineMap[key] = acc
		}
		if acc.division == "" {
			acc.division = p.Division
		}
		if p.OpponentCount > acc.opponentCount {
			acc.opponentCount = p.OpponentCount
		}
		placement := p.Placement
		acc.placement = &placement
		acc.hasPlacement = true
		if strings.TrimSpace(p.ClubName) != "" {
			clubCounts[strings.TrimSpace(p.ClubName)]++
			if acc.club == "" {
				acc.club = strings.TrimSpace(p.ClubName)
			}
		}

		recordIdentitySide(identities, p.UserID, p.Name, p.ClubName, true)

		switch p.Placement {
		case 1:
			summary.Gold++
		case 2:
			summary.Silver++
		case 3:
			summary.Bronze++
		case 0:
			summary.NoPlacement++
		}
	}

	eventMeta, err := s.loadEventMeta(ctx, timelineMap)
	if err != nil {
		return AthleteProfileResult{}, err
	}
	for key, acc := range timelineMap {
		if meta, ok := eventMeta[key.eventID]; ok {
			acc.title = meta.title
			acc.dateText = meta.dateText
			acc.location = meta.location
		}
	}

	timeline := make([]TimelineEntry, 0, len(timelineMap))
	for _, acc := range timelineMap {
		entry := TimelineEntry{
			EventID:       acc.eventID,
			BracketID:     acc.bracketID,
			Title:         acc.title,
			DateText:      acc.dateText,
			Location:      acc.location,
			Division:      acc.division,
			Club:          acc.club,
			OpponentCount: acc.opponentCount,
			Wins:          acc.wins,
			Losses:        acc.losses,
			Byes:          acc.byes,
		}
		if acc.hasPlacement {
			entry.Placement = acc.placement
			entry.PlacementLabel = placementLabel(acc.placement)
		}
		timeline = append(timeline, entry)

		_, belt, style := ParseDivision(acc.division)
		if belt != "" {
			summary.Belts[belt]++
		}
		if style != "" {
			summary.Styles[style]++
		}
	}
	sort.Slice(timeline, func(i, j int) bool {
		if timeline[i].EventID != timeline[j].EventID {
			return timeline[i].EventID > timeline[j].EventID
		}
		return timeline[i].BracketID < timeline[j].BracketID
	})
	summary.Divisions = len(timeline)

	encounters := buildEncounters(matches, idSet, eventMeta)
	sort.Slice(encounters, func(i, j int) bool {
		if encounters[i].EventID != encounters[j].EventID {
			return encounters[i].EventID > encounters[j].EventID
		}
		if encounters[i].BracketID != encounters[j].BracketID {
			return encounters[i].BracketID < encounters[j].BracketID
		}
		return encounters[i].MatchID < encounters[j].MatchID
	})

	result := AthleteProfileResult{
		Identities: buildProfileIdentities(userIDs, identities),
		Summary:    summary,
	}
	result.Summary.Clubs = clubCountsToSlice(clubCounts)

	if len(timeline) > profileTimelineMax {
		timeline = timeline[:profileTimelineMax]
		result.Truncated = true
	}
	if len(encounters) > profileEncountersMax {
		encounters = encounters[:profileEncountersMax]
		result.Truncated = true
	}
	result.Timeline = timeline
	result.Encounters = encounters
	return result, nil
}

type eventMetaRow struct {
	title    string
	dateText string
	location string
}

func boolCount(b bool) int {
	if b {
		return 1
	}
	return 0
}

func recordIdentitySide(identities map[int]*identityAcc, userID int, name, club string, onSide bool) {
	if !onSide {
		return
	}
	acc, ok := identities[userID]
	if !ok {
		return
	}
	if name != "" {
		acc.nameCounts[name]++
	}
	if c := strings.TrimSpace(club); c != "" {
		acc.clubs[c] = struct{}{}
	}
}

func buildProfileIdentities(userIDs []int, identities map[int]*identityAcc) []AthleteIdentity {
	out := make([]AthleteIdentity, 0, len(userIDs))
	for _, id := range userIDs {
		acc := identities[id]
		if acc == nil {
			out = append(out, AthleteIdentity{UserID: id})
			continue
		}
		name := ""
		best := 0
		for n, c := range acc.nameCounts {
			if c > best || (c == best && (name == "" || n < name)) {
				name = n
				best = c
			}
		}
		clubs := make([]string, 0, len(acc.clubs))
		for c := range acc.clubs {
			clubs = append(clubs, c)
		}
		sort.Strings(clubs)
		out = append(out, AthleteIdentity{UserID: id, Name: name, Clubs: clubs})
	}
	return out
}

func clubCountsToSlice(counts map[string]int) []ClubCount {
	out := make([]ClubCount, 0, len(counts))
	for name, count := range counts {
		out = append(out, ClubCount{Name: name, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func buildEncounters(matches []ajp.MatchRecord, idSet map[int]bool, eventMeta map[int]eventMetaRow) []Encounter {
	var out []Encounter
	for _, m := range matches {
		if m.IsBye {
			continue
		}
		onLeft := idSet[m.LeftUserID]
		onRight := idSet[m.RightUserID]
		if !onLeft && !onRight {
			continue
		}
		meta := eventMeta[m.EventID]
		enc := Encounter{
			EventID:   m.EventID,
			BracketID: m.BracketID,
			MatchID:   m.MatchID,
			Title:     meta.title,
			DateText:  meta.dateText,
			Division:  m.Division,
			RoundName: m.RoundName,
			Result:    encounterResult(onLeft, onRight, m.WinnerSide),
			WonBy:     m.WonBy,
			ScoreText: m.ScoreText,
		}
		if onLeft {
			enc.OpponentName = m.RightName
			enc.OpponentClub = m.RightClub
		} else {
			enc.OpponentName = m.LeftName
			enc.OpponentClub = m.LeftClub
		}
		out = append(out, enc)
	}
	return out
}

func (s *SQLStore) loadProfileMatches(ctx context.Context, userIDs []int) ([]ajp.MatchRecord, error) {
	ph := buildInPlaceholders(len(userIDs))
	args := make([]any, len(userIDs)*2)
	for i, id := range userIDs {
		args[i] = id
		args[len(userIDs)+i] = id
	}
	query := fmt.Sprintf(`
SELECT m.event_id, m.bracket_id, m.division, m.match_id, m.round_name, m.round, m.mat_name, m.mat_match_nr,
  m.state, m.is_bye, m.won_by, m.score_text, m.penalty_text,
  m.left_name, m.left_club, m.left_country, m.left_user_id, m.left_result,
  m.right_name, m.right_club, m.right_country, m.right_user_id, m.right_result,
  m.winner_side, m.estimated_start, m.registrations_count, m.opponent_count
FROM matches m
WHERE m.left_user_id IN (%s) OR m.right_user_id IN (%s)
ORDER BY m.event_id DESC, m.bracket_id, m.match_id`, ph, ph)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		m.IsBye = isBye != 0
		items = append(items, m)
	}
	return items, rows.Err()
}

func (s *SQLStore) loadProfilePlacements(ctx context.Context, userIDs []int) ([]ajp.PlacementRecord, error) {
	ph := buildInPlaceholders(len(userIDs))
	args := make([]any, len(userIDs))
	for i, id := range userIDs {
		args[i] = id
	}
	query := fmt.Sprintf(`
SELECT p.event_id, p.bracket_id, p.division, p.placement, p.user_id, p.name, p.club_name, p.affiliation_name,
  p.registrations_count, p.opponent_count
FROM placements p
WHERE p.user_id IN (%s)
ORDER BY p.event_id DESC, p.bracket_id`, ph)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ajp.PlacementRecord
	for rows.Next() {
		var p ajp.PlacementRecord
		if err := rows.Scan(
			&p.EventID, &p.BracketID, &p.Division, &p.Placement, &p.UserID, &p.Name, &p.ClubName, &p.AffiliationName,
			&p.RegistrationsCount, &p.OpponentCount,
		); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

func (s *SQLStore) loadEventMeta(ctx context.Context, timelineMap map[bracketKey]*timelineAcc) (map[int]eventMetaRow, error) {
	if len(timelineMap) == 0 {
		return map[int]eventMetaRow{}, nil
	}
	eventIDs := make(map[int]struct{})
	for key := range timelineMap {
		eventIDs[key.eventID] = struct{}{}
	}
	ids := make([]int, 0, len(eventIDs))
	for id := range eventIDs {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	ph := buildInPlaceholders(len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	query := fmt.Sprintf(`SELECT event_id, title, date_text, location FROM events WHERE event_id IN (%s)`, ph)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	meta := make(map[int]eventMetaRow, len(ids))
	for rows.Next() {
		var eventID int
		var row eventMetaRow
		if err := rows.Scan(&eventID, &row.title, &row.dateText, &row.location); err != nil {
			return nil, err
		}
		meta[eventID] = row
	}
	return meta, rows.Err()
}
