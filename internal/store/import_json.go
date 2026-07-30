package store

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"atour/internal/ajp"
)

func ImportJSONDir(ctx context.Context, s Store, dir string) error {
	b, err := os.ReadFile(filepath.Join(dir, "events.json"))
	if err != nil {
		return err
	}
	var results []ajp.EventResult
	if err := json.Unmarshal(b, &results); err != nil {
		return err
	}

	events := make([]EventRow, 0, len(results))
	for _, er := range results {
		events = append(events, EventRow{
			EventID:             er.Event.ID,
			Title:               er.Event.Title,
			URL:                 er.Event.URL,
			Location:            er.Event.Location,
			DateText:            er.Event.DateText,
			BracketsUnavailable: er.BracketsUnavailable,
		})
	}

	matches, err := loadMatchesJSON(dir, results)
	if err != nil {
		return err
	}
	placements, err := loadPlacementsJSON(dir, results)
	if err != nil {
		return err
	}

	if err := s.Migrate(ctx); err != nil {
		return err
	}
	return s.ReplaceAll(ctx, events, matches, placements)
}

func loadMatchesJSON(dir string, results []ajp.EventResult) ([]ajp.MatchRecord, error) {
	path := filepath.Join(dir, "matches.json")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return flattenMatches(results), nil
		}
		return nil, err
	}
	var matches []ajp.MatchRecord
	if err := json.Unmarshal(b, &matches); err != nil {
		return nil, err
	}
	return matches, nil
}

func loadPlacementsJSON(dir string, results []ajp.EventResult) ([]ajp.PlacementRecord, error) {
	path := filepath.Join(dir, "placements.json")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return flattenPlacements(results), nil
		}
		return nil, err
	}
	var placements []ajp.PlacementRecord
	if err := json.Unmarshal(b, &placements); err != nil {
		return nil, err
	}
	return placements, nil
}

func flattenMatches(results []ajp.EventResult) []ajp.MatchRecord {
	var matches []ajp.MatchRecord
	for _, er := range results {
		for _, br := range er.Brackets {
			matches = append(matches, br.Matches...)
		}
	}
	return matches
}

func flattenPlacements(results []ajp.EventResult) []ajp.PlacementRecord {
	var placements []ajp.PlacementRecord
	for _, er := range results {
		for _, br := range er.Brackets {
			placements = append(placements, br.Placements...)
		}
	}
	return placements
}
