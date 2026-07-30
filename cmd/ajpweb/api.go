package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"atour/internal/ajp"
	"atour/internal/store"
)

type apiServer struct {
	store store.Store
}

type eventItem struct {
	EventID             int    `json:"event_id"`
	Title               string `json:"title"`
	URL                 string `json:"url"`
	Location            string `json:"location"`
	DateText            string `json:"date_text"`
	BracketsUnavailable bool   `json:"brackets_unavailable"`
}

type eventsResponse struct {
	Items []eventItem `json:"items"`
}

type listResponse[T any] struct {
	Total int `json:"total"`
	Items []T `json:"items"`
}

func parseFilter(r *http.Request) store.FilterQuery {
	q := r.URL.Query()
	eventID, _ := strconv.Atoi(q.Get("event_id"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	hide := q.Get("hide_bye")
	return store.FilterQuery{
		Q: q.Get("q"), EventID: eventID,
		Gender: q.Get("gender"), Belt: q.Get("belt"), Style: q.Get("style"),
		HideBye: hide == "1" || hide == "true" || hide == "on",
		Limit: limit, Offset: offset,
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *apiServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rows, err := s.store.ListEvents(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	items := make([]eventItem, len(rows))
	for i, row := range rows {
		items[i] = eventItem{
			EventID:             row.EventID,
			Title:               row.Title,
			URL:                 row.URL,
			Location:            row.Location,
			DateText:            row.DateText,
			BracketsUnavailable: row.BracketsUnavailable,
		}
	}
	writeJSON(w, eventsResponse{Items: items})
}

func (s *apiServer) handleMatches(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	items, total, err := s.store.ListMatches(r.Context(), parseFilter(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if items == nil {
		items = []ajp.MatchRecord{}
	}
	writeJSON(w, listResponse[ajp.MatchRecord]{Total: total, Items: items})
}

func (s *apiServer) handlePlacements(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	items, total, err := s.store.ListPlacements(r.Context(), parseFilter(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if items == nil {
		items = []ajp.PlacementRecord{}
	}
	writeJSON(w, listResponse[ajp.PlacementRecord]{Total: total, Items: items})
}

func newMux(webDir string, s store.Store) http.Handler {
	mux := http.NewServeMux()
	srv := &apiServer{store: s}
	mux.HandleFunc("/api/events", srv.handleEvents)
	mux.HandleFunc("/api/matches", srv.handleMatches)
	mux.HandleFunc("/api/placements", srv.handlePlacements)
	mux.Handle("/", http.FileServer(http.Dir(webDir)))
	return mux
}
