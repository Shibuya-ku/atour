package ajp

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
)

var reProfileID = regexp.MustCompile(`/profile/(\d+)`)

func (c *Client) ListBrackets(ctx context.Context, eventID int) ([]BracketMeta, error) {
	var raw bracketsJSON
	path := fmt.Sprintf("/en/event/%d/schedule/brackets.json", eventID)
	body, code, err := c.GetBytes(ctx, path)
	if err != nil {
		return nil, err
	}
	if code == 403 || code == 404 {
		return nil, ErrBracketsUnavailable{EventID: eventID, Status: code}
	}
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf("brackets.json status %d", code)
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	return raw.Brackets, nil
}

type ErrBracketsUnavailable struct {
	EventID int
	Status  int
}

func (e ErrBracketsUnavailable) Error() string {
	return fmt.Sprintf("brackets unavailable for event %d (status %d)", e.EventID, e.Status)
}

func FilterBluePlusBrackets(in []BracketMeta) []BracketMeta {
	out := make([]BracketMeta, 0)
	for _, b := range in {
		if IsBluePlusAdultDivision(b.Name) {
			out = append(out, b)
		}
	}
	return out
}

func ParseUserIDFromProfile(url string) int {
	m := reProfileID.FindStringSubmatch(url)
	if m == nil {
		return 0
	}
	var id int
	fmt.Sscanf(m[1], "%d", &id)
	return id
}

func scoreTexts(raw json.RawMessage) (score, penalty string) {
	if len(raw) == 0 || string(raw) == "[]" || string(raw) == "null" {
		return "", ""
	}
	var obj map[string]string
	if err := json.Unmarshal(raw, &obj); err == nil {
		return obj["score"], obj["penalty"]
	}
	return "", ""
}

func FlattenMatches(eventID int, division string, rd RenderData, registrationsCount int) ([]MatchRecord, error) {
	opp := OpponentCount(registrationsCount)
	out := make([]MatchRecord, 0, len(rd.State.Matches))
	for _, m := range rd.State.Matches {
		sc, pe := scoreTexts(m.Score)
		left, right := m.Seats["left"], m.Seats["right"]
		rec := MatchRecord{
			EventID: eventID, BracketID: m.BracketID, Division: division,
			MatchID: m.ID, RoundName: m.Name, Round: m.Round,
			MatName: m.MatName, MatMatchNr: m.MatMatchNr,
			State: m.State, IsBye: m.IsBye, WonBy: m.WonBy,
			ScoreText: sc, PenaltyText: pe,
			LeftName: left.Name, LeftClub: left.Club, LeftCountry: left.Country, LeftResult: left.Result,
			RightName: right.Name, RightClub: right.Club, RightCountry: right.Country, RightResult: right.Result,
			EstimatedStart: m.EstimatedStarttime,
			RegistrationsCount: registrationsCount,
			OpponentCount:      opp,
		}
		if left.Player != nil {
			rec.LeftUserID = ParseUserIDFromProfile(left.Player.UserProfileURL)
			if rec.LeftClub == "" {
				rec.LeftClub = left.Player.Club
			}
			if rec.LeftCountry == "" {
				rec.LeftCountry = left.Player.Country
			}
		}
		if right.Player != nil {
			rec.RightUserID = ParseUserIDFromProfile(right.Player.UserProfileURL)
			if rec.RightClub == "" {
				rec.RightClub = right.Player.Club
			}
			if rec.RightCountry == "" {
				rec.RightCountry = right.Player.Country
			}
		}
		switch {
		case left.IsWinner:
			rec.WinnerSide = "left"
		case right.IsWinner:
			rec.WinnerSide = "right"
		}
		if m.EventID != 0 {
			rec.EventID = m.EventID
		}
		out = append(out, rec)
	}
	return out, nil
}

func FlattenPlacements(eventID int, division string, bracketID int, pd PlacementTableData, registrationsCount int) []PlacementRecord {
	opp := OpponentCount(registrationsCount)
	out := make([]PlacementRecord, 0, len(pd.PlacementTableState.Placements))
	for _, p := range pd.PlacementTableState.Placements {
		aff := ""
		if p.AffiliationName != nil {
			aff = *p.AffiliationName
		}
		out = append(out, PlacementRecord{
			EventID: eventID, BracketID: bracketID, Division: division,
			Placement: p.Placement, UserID: p.UserID, Name: p.Name,
			ClubName: p.ClubName, AffiliationName: aff,
			RegistrationsCount: registrationsCount,
			OpponentCount:      opp,
		})
	}
	return out
}

func (c *Client) FetchBracketDetail(ctx context.Context, eventID, bracketID int, division string, registrationsCount int) (BracketResult, error) {
	var rd RenderData
	var pd PlacementTableData
	rPath := fmt.Sprintf("/en/event/%d/bracket/%d/getRenderData", eventID, bracketID)
	pPath := fmt.Sprintf("/en/event/%d/bracket/%d/getPlacementTableData", eventID, bracketID)
	if err := c.GetJSON(ctx, rPath, &rd); err != nil {
		return BracketResult{}, err
	}
	_ = c.GetJSON(ctx, pPath, &pd)
	matches, err := FlattenMatches(eventID, division, rd, registrationsCount)
	if err != nil {
		return BracketResult{}, err
	}
	return BracketResult{
		Meta: BracketMeta{
			BracketID:          bracketID,
			Name:               division,
			RegistrationsCount: registrationsCount,
		},
		Matches:    matches,
		Placements: FlattenPlacements(eventID, division, bracketID, pd, registrationsCount),
	}, nil
}
