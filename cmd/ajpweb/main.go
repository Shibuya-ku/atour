package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
)

func newMux(webDir, dataDir string) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/data/", http.StripPrefix("/data/", http.FileServer(http.Dir(dataDir))))
	mux.Handle("/", http.FileServer(http.Dir(webDir)))
	return mux
}

func main() {
	addr := flag.String("addr", ":8787", "listen address")
	web := flag.String("web", "web", "web root directory")
	data := flag.String("data", "output", "JSON data directory")
	flag.Parse()

	webDir, _ := filepath.Abs(*web)
	dataDir, _ := filepath.Abs(*data)
	fmt.Printf("atour web  http://localhost%s\n  web=%s\n  data=%s\n", *addr, webDir, dataDir)
	log.Fatal(http.ListenAndServe(*addr, newMux(webDir, dataDir)))
}
