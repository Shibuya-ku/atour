package ajp

import (
	"regexp"
	"strings"
)

var (
	reAdultGender = regexp.MustCompile(`^(Men's|Women's)\s`)
	// 成人腰带：White 及以上（不含 Grey/Yellow 等少儿色带）
	reAdultBelt = regexp.MustCompile(`(?i)/\s*(White|Blue|Purple|Brown|Black)\s*/`)
)

// IsBluePlusAdultDivision 判断 Men's/Women's 且腰带为 White/Blue/Purple/Brown/Black（不含 Youth）。
func IsBluePlusAdultDivision(name string) bool {
	name = strings.TrimSpace(name)
	if strings.HasPrefix(name, "Youth ") {
		return false
	}
	return reAdultGender.MatchString(name) && reAdultBelt.MatchString(name)
}

// OpponentCount 该组别除本人外的对手人数。
func OpponentCount(registrations int) int {
	if registrations <= 1 {
		return 0
	}
	return registrations - 1
}

func FilterChinaEvents(events []EventSummary) []EventSummary {
	out := make([]EventSummary, 0, len(events))
	for _, e := range events {
		if strings.Contains(e.Location, "China") || strings.Contains(e.Title+e.Location, "🇨🇳") {
			out = append(out, e)
		}
	}
	return out
}

func DedupeEventsByID(events []EventSummary) []EventSummary {
	seen := map[int]int{}
	out := make([]EventSummary, 0, len(events))
	for _, e := range events {
		if i, ok := seen[e.ID]; ok {
			if !strings.Contains(out[i].Title, e.Title) {
				out[i].Title = out[i].Title + " | " + e.Title
			}
			continue
		}
		seen[e.ID] = len(out)
		out = append(out, e)
	}
	return out
}
