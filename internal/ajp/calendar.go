package ajp

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

var (
	reEventHref = regexp.MustCompile(`/event/(\d+)`)
	reDateAtLoc = regexp.MustCompile(`((?:January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{1,2}(?:\s*-\s*\d{1,2})?)\s+@\s+(.+)$`)
)

func ParseCalendarHTML(page []byte, baseURL string) ([]EventSummary, error) {
	doc, err := html.Parse(bytes.NewReader(page))
	if err != nil {
		return nil, err
	}
	baseURL = stringsTrimRightSlash(baseURL)
	var out []EventSummary
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			href := attr(n, "href")
			m := reEventHref.FindStringSubmatch(href)
			if m != nil {
				id, _ := strconv.Atoi(m[1])
				title := strings.TrimSpace(textContent(n))
				card := buildEventCardText(n)
				loc, date := extractDateLocation(card)
				url := href
				if strings.HasPrefix(url, "/") {
					url = baseURL + url
				}
				if title != "" && id > 0 {
					out = append(out, EventSummary{
						ID: id, Title: title, URL: url, Location: loc, DateText: date,
					})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if len(out) == 0 {
		return nil, fmt.Errorf("no events parsed from calendar html")
	}
	return out, nil
}

// buildEventCardText 拼接标题链接及其后兄弟节点文本，直到下一个赛事链接。
func buildEventCardText(a *html.Node) string {
	parts := []string{strings.TrimSpace(textContent(a))}
	for sib := a.NextSibling; sib != nil; sib = sib.NextSibling {
		if isEventLinkNode(sib) {
			break
		}
		if txt := strings.TrimSpace(textContent(sib)); txt != "" {
			parts = append(parts, txt)
		}
	}
	return strings.Join(parts, " ")
}

func isEventLinkNode(n *html.Node) bool {
	if n.Type != html.ElementNode || n.Data != "a" {
		return false
	}
	return reEventHref.MatchString(attr(n, "href"))
}

func extractDateLocation(card string) (loc, date string) {
	if mm := reDateAtLoc.FindStringSubmatch(card); mm != nil {
		return strings.TrimSpace(mm[2]), strings.TrimSpace(mm[1])
	}
	if i := strings.Index(card, "@"); i >= 0 {
		loc = strings.TrimSpace(card[i+1:])
	}
	return loc, date
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func textContent(n *html.Node) string {
	if n == nil {
		return ""
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}
