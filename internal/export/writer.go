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

// LoadEvents 读取已有 events.json；文件不存在时返回空切片。
func LoadEvents(dir string) ([]ajp.EventResult, error) {
	path := filepath.Join(dir, "events.json")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var results []ajp.EventResult
	if err := json.Unmarshal(b, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// MergeByEventID 按 event_id 合并；后者覆盖同 ID 的旧数据。
func MergeByEventID(base, extra []ajp.EventResult) []ajp.EventResult {
	idx := map[int]int{}
	out := make([]ajp.EventResult, 0, len(base)+len(extra))
	for _, er := range base {
		idx[er.Event.ID] = len(out)
		out = append(out, er)
	}
	for _, er := range extra {
		if i, ok := idx[er.Event.ID]; ok {
			out[i] = er
			continue
		}
		idx[er.Event.ID] = len(out)
		out = append(out, er)
	}
	return out
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
