package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"atour/internal/store"
)

func main() {
	addr := flag.String("addr", ":8787", "listen address")
	web := flag.String("web", "web", "web root directory")
	driver := flag.String("db-driver", "sqlite", "sqlite|mysql")
	dsn := flag.String("dsn", "data/atour.db", "sqlite path or mysql DSN")
	flag.Parse()

	s, err := store.Open(*driver, *dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	webDir, _ := filepath.Abs(*web)
	fmt.Printf("atour web  http://localhost%s\n  web=%s\n  db-driver=%s\n  dsn=%s\n", *addr, webDir, *driver, *dsn)
	log.Fatal(http.ListenAndServe(*addr, newMux(webDir, s)))
}
