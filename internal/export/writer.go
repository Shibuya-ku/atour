package export

import (
	"encoding/json"
	"os"
	"path/filepath"

	"atour/internal/ajp"
)

func WriteAll(dir string, results []ajp.EventResult) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var matches []ajp.MatchRecord
	var placements []ajp.PlacementRecord
	for _, er := range results {
		for _, br := range er.Brackets {
			matches = append(matches, br.Matches...)
			placements = append(placements, br.Placements...)
		}
	}
	if err := writeJSON(filepath.Join(dir, "events.json"), results); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, "matches.json"), matches); err != nil {
		return err
	}
	return writeJSON(filepath.Join(dir, "placements.json"), placements)
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
