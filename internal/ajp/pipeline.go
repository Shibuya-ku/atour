package ajp

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
)

func Run(ctx context.Context, c *Client, calendarPath string) ([]EventResult, error) {
	body, code, err := c.GetBytes(ctx, calendarPath)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, fmt.Errorf("calendar status %d", code)
	}
	events, err := ParseCalendarHTML(body, c.Base())
	if err != nil {
		return nil, err
	}
	events = DedupeEventsByID(FilterChinaEvents(events))
	results := make([]EventResult, 0, len(events))
	for ei, ev := range events {
		log.Printf("event %d/%d id=%d %s", ei+1, len(events), ev.ID, ev.Title)
		er := EventResult{Event: ev}
		brackets, err := c.ListBrackets(ctx, ev.ID)
		if err != nil {
			var unavail ErrBracketsUnavailable
			if errors.As(err, &unavail) {
				er.BracketsUnavailable = true
				results = append(results, er)
				log.Printf("skip brackets event=%d status=%d", ev.ID, unavail.Status)
				continue
			}
			log.Printf("skip event=%d list brackets: %v", ev.ID, err)
			er.BracketsUnavailable = true
			results = append(results, er)
			continue
		}
		targets := FilterBluePlusBrackets(brackets)
		log.Printf("event %d adult brackets=%d/%d", ev.ID, len(targets), len(brackets))
		er.Brackets = fetchBracketsParallel(ctx, c, ev.ID, targets, 4)
		results = append(results, er)
	}
	return results, nil
}

func fetchBracketsParallel(ctx context.Context, c *Client, eventID int, targets []BracketMeta, workers int) []BracketResult {
	if len(targets) == 0 {
		return nil
	}
	if workers < 1 {
		workers = 1
	}
	type job struct {
		idx  int
		meta BracketMeta
	}
	jobs := make(chan job)
	out := make([]BracketResult, len(targets))
	ok := make([]bool, len(targets))
	var wg sync.WaitGroup
	var mu sync.Mutex
	done := 0
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				detail, err := c.FetchBracketDetail(ctx, eventID, j.meta.BracketID, j.meta.Name, j.meta.RegistrationsCount)
				if err != nil {
					log.Printf("skip bracket event=%d bracket=%d: %v", eventID, j.meta.BracketID, err)
					continue
				}
				detail.Meta = j.meta
				out[j.idx] = detail
				ok[j.idx] = true
				mu.Lock()
				done++
				if done%10 == 0 || done == len(targets) {
					log.Printf("event %d progress brackets %d/%d", eventID, done, len(targets))
				}
				mu.Unlock()
			}
		}()
	}
	for i, meta := range targets {
		jobs <- job{idx: i, meta: meta}
	}
	close(jobs)
	wg.Wait()
	kept := make([]BracketResult, 0, len(targets))
	for i, br := range out {
		if ok[i] {
			kept = append(kept, br)
		}
	}
	return kept
}
