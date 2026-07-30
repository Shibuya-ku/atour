package store

import (
	"regexp"
	"strings"
)

var beltRe = regexp.MustCompile(`(?i)/\s*(White|Blue|Purple|Brown|Black)\s*/`)

func ParseDivision(division string) (gender, belt, style string) {
	d := division
	if strings.HasPrefix(d, "Men's") {
		gender = "Men's"
	} else if strings.HasPrefix(d, "Women's") {
		gender = "Women's"
	}
	if m := beltRe.FindStringSubmatch(d); len(m) == 2 {
		b := m[1]
		belt = strings.ToUpper(b[:1]) + strings.ToLower(b[1:])
	}
	switch {
	case regexp.MustCompile(`(?i)\bNO-GI\b`).MatchString(d):
		style = "NO-GI"
	case regexp.MustCompile(`(?i)\bGI\b`).MatchString(d):
		style = "GI"
	}
	return
}
