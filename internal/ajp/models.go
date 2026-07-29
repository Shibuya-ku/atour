package ajp

import (
	"encoding/json"
	"fmt"
)

type EventSummary struct {
	ID       int    `json:"event_id"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Location string `json:"location"`
	DateText string `json:"date_text"`
}

type BracketMeta struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	Mats               string `json:"mats"`
	BracketBundleID    int    `json:"bracket_bundle_id"`
	EstimatedStart     string `json:"estimated_start"`
	RegistrationsCount int    `json:"registrations_count"`
	BracketID          int    `json:"bracket_id"`
}

type bracketsJSON struct {
	Brackets []BracketMeta `json:"brackets"`
}

type Player struct {
	RegistrationID int    `json:"registration_id"`
	UserProfileURL string `json:"profile_link"`
	Name           string `json:"name"`
	Firstname      string `json:"firstname"`
	Lastname       string `json:"lastname"`
	Club           string `json:"club"`
	ClubID         int    `json:"club_id"`
	Country        string `json:"country"`
}

type Seat struct {
	Type      string  `json:"type"` // registration | bye
	Name      string  `json:"name"`
	Result    string  `json:"result"` // won | lost
	IsWinner  bool    `json:"isWinner"`
	Placement *int    `json:"placement"`
	Club      string  `json:"club"`
	Country   string  `json:"country"`
	Player    *Player `json:"player"`
}

type MatchScore struct {
	Score   string `json:"score"`
	Penalty string `json:"penalty"`
}

type RawMatch struct {
	ID                 int             `json:"id"`
	State              string          `json:"state"`
	MatMatchNr         string          `json:"mat_match_nr"`
	MatchNr            int             `json:"match_nr"`
	MatName            string          `json:"mat_name"`
	WonBy              string          `json:"wonBy"`
	Score              json.RawMessage `json:"score"` // object or []
	Round              int             `json:"round"`
	Name               string          `json:"name"`
	IsBye              bool            `json:"isBye"`
	EstimatedStarttime string          `json:"estimated_starttime"`
	Seats              map[string]Seat `json:"seats"`
	BracketID          int             `json:"bracket_id"`
	EventID            int             `json:"eventId"`
}

type RenderData struct {
	State struct {
		Matches map[string]RawMatch `json:"matches"`
	} `json:"state"`
	BracketInfo json.RawMessage `json:"bracketInfo"`
}

func (rd *RenderData) UnmarshalJSON(data []byte) error {
	type wire struct {
		State struct {
			Matches json.RawMessage `json:"matches"`
		} `json:"state"`
		BracketInfo json.RawMessage `json:"bracketInfo"`
	}
	var w wire
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	rd.BracketInfo = w.BracketInfo
	rd.State.Matches = make(map[string]RawMatch)
	if len(w.State.Matches) == 0 {
		return nil
	}
	if w.State.Matches[0] == '[' {
		var arr []RawMatch
		if err := json.Unmarshal(w.State.Matches, &arr); err != nil {
			return err
		}
		for i, m := range arr {
			rd.State.Matches[fmt.Sprintf("%d", i)] = m
		}
		return nil
	}
	return json.Unmarshal(w.State.Matches, &rd.State.Matches)
}

type Placement struct {
	ID              int     `json:"id"`
	Placement       int     `json:"placement"`
	UserID          int     `json:"user_id"`
	Name            string  `json:"name"`
	ClubID          int     `json:"club_id"`
	RegistrationID  int     `json:"registration_id"`
	ClubName        string  `json:"club_name"`
	AffiliationName *string `json:"affiliation_name"`
}

type PlacementTableData struct {
	PlacementTableState struct {
		Placements []Placement `json:"placements"`
	} `json:"placementTableState"`
}

// 导出用扁平结构
type MatchRecord struct {
	EventID        int    `json:"event_id"`
	BracketID      int    `json:"bracket_id"`
	Division       string `json:"division"`
	MatchID        int    `json:"match_id"`
	RoundName      string `json:"round_name"`
	Round          int    `json:"round"`
	MatName        string `json:"mat_name"`
	MatMatchNr     string `json:"mat_match_nr"`
	State          string `json:"state"`
	IsBye          bool   `json:"is_bye"`
	WonBy          string `json:"won_by"`
	ScoreText      string `json:"score_text"`
	PenaltyText    string `json:"penalty_text"`
	LeftName       string `json:"left_name"`
	LeftClub       string `json:"left_club"`
	LeftCountry    string `json:"left_country"`
	LeftUserID     int    `json:"left_user_id"`
	LeftResult     string `json:"left_result"`
	RightName      string `json:"right_name"`
	RightClub      string `json:"right_club"`
	RightCountry   string `json:"right_country"`
	RightUserID    int    `json:"right_user_id"`
	RightResult    string `json:"right_result"`
	WinnerSide         string `json:"winner_side"` // left|right|""
	EstimatedStart     string `json:"estimated_start"`
	RegistrationsCount int    `json:"registrations_count"`
	OpponentCount      int    `json:"opponent_count"` // 该组别总对手数 = registrations_count-1
}

type PlacementRecord struct {
	EventID            int    `json:"event_id"`
	BracketID          int    `json:"bracket_id"`
	Division           string `json:"division"`
	Placement          int    `json:"placement"`
	UserID             int    `json:"user_id"`
	Name               string `json:"name"`
	ClubName           string `json:"club_name"`
	AffiliationName    string `json:"affiliation_name"`
	RegistrationsCount int    `json:"registrations_count"`
	OpponentCount      int    `json:"opponent_count"`
}

type BracketResult struct {
	Meta       BracketMeta       `json:"meta"`
	Matches    []MatchRecord     `json:"matches"`
	Placements []PlacementRecord `json:"placements"`
}

type EventResult struct {
	Event               EventSummary    `json:"event"`
	BracketsUnavailable bool            `json:"brackets_unavailable"`
	Brackets            []BracketResult `json:"brackets"`
}
