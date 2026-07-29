package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"atour/internal/ajp"
	"atour/internal/export"
)

func main() {
	base := flag.String("base", "https://ajptour.com", "site base URL")
	calendar := flag.String("calendar", "/en/events-1/events-calendar-2025", "calendar path")
	out := flag.String("out", "output", "output directory")
	listOnly := flag.Bool("list-only", false, "only list china events / adult White+ divisions")
	flag.Parse()

	ctx := context.Background()
	client := ajp.NewClient(*base)
	if *listOnly {
		body, code, err := client.GetBytes(ctx, *calendar)
		if err != nil || code != 200 {
			log.Fatalf("calendar: code=%d err=%v", code, err)
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
		return
	}

	results, err := ajp.Run(ctx, client, *calendar)
	if err != nil {
		log.Fatal(err)
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
