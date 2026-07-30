package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"atour/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: ajpdb import [flags]")
	}
	switch os.Args[1] {
	case "import":
		fs := flag.NewFlagSet("import", flag.ExitOnError)
		from := fs.String("from", "output", "JSON directory")
		driver := fs.String("driver", "sqlite", "sqlite|mysql")
		dsn := fs.String("dsn", "data/atour.db", "DSN or sqlite file path")
		_ = fs.Parse(os.Args[2:])
		if *driver == "sqlite" {
			if err := os.MkdirAll(filepath.Dir(*dsn), 0o755); err != nil {
				log.Fatal(err)
			}
		}
		s, err := store.Open(*driver, *dsn)
		if err != nil {
			log.Fatal(err)
		}
		defer s.Close()
		if err := store.ImportJSONDir(context.Background(), s, *from); err != nil {
			log.Fatal(err)
		}
		fmt.Println("ok")
	default:
		log.Fatal("unknown command")
	}
}
