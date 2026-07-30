package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"atour/internal/ajp"
	"atour/internal/export"
)

func main() {
	base := flag.String("base", "https://ajptour.com", "site base URL")
	calendar := flag.String("calendar", "/en/events-1/events-calendar-2025", "single calendar path")
	calendars := flag.String("calendars", "", "comma-separated calendar paths; overrides -calendar when set")
	out := flag.String("out", "output", "output directory")
	merge := flag.Bool("merge", false, "merge with existing events.json in -out")
	listOnly := flag.Bool("list-only", false, "only list china events / adult White+ divisions")
	flag.Parse()

	paths := []string{*calendar}
	if strings.TrimSpace(*calendars) != "" {
		paths = nil
		for _, p := range strings.Split(*calendars, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				paths = append(paths, p)
			}
		}
	}
	if len(paths) == 0 {
		log.Fatal("no calendar path")
	}

	ctx := context.Background()
	client := ajp.NewClient(*base)
	if *listOnly {
		for _, cal := range paths {
			fmt.Fprintf(os.Stderr, "# %s\n", cal)
			body, code, err := client.GetBytes(ctx, cal)
			if err != nil || code != 200 {
				log.Fatalf("calendar %s: code=%d err=%v", cal, code, err)
			}
			events, err := ajp.ParseCalendarHTML(body, client.Base())
			if err != nil {
				log.Fatal(err)
			}
			events = ajp.DedupeEventsByID(ajp.FilterChinaEvents(events))
			for _, ev := range events {
				fmt.Printf("%d\t%s\t%s\n", ev.ID, ev.Location, ev.Title)
				brackets, err := client.ListBrackets(ctx, ev.ID)
				if err != nil {
					fmt.Printf("  brackets: %v\n", err)
					continue
				}
				for _, b := range ajp.FilterBluePlusBrackets(brackets) {
					fmt.Printf("  %d\t%s\n", b.BracketID, b.Name)
				}
			}
		}
		return
	}

	results, err := ajp.RunMany(ctx, client, paths)
	if err != nil {
		log.Fatal(err)
	}
	if *merge {
		existing, err := export.LoadEvents(*out)
		if err != nil {
			log.Fatal(err)
		}
		results = export.MergeByEventID(existing, results)
		log.Printf("merged with existing: total events=%d", len(results))
	}
	if err := export.WriteAll(*out, results); err != nil {
		log.Fatal(err)
	}
	nMatch := 0
	for _, r := range results {
		for _, b := range r.Brackets {
			nMatch += len(b.Matches)
		}
	}
	fmt.Fprintf(os.Stderr, "events=%d matches=%d out=%s\n", len(results), nMatch, *out)
}
