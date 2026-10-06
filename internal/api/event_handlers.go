package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/store"
)

// Events: holiday dinners and potlucks, with who's coming and the menu.

type eventCard struct {
	store.Event
	Dishes int `json:"dishes"`
	People int `json:"people"` // everyone coming, extra guests included
}

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	today := r.URL.Query().Get("today")
	if _, err := time.Parse(day, today); err != nil {
		today = time.Now().Format(day)
	}
	events, err := s.Store.ListEvents(today)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	people, err := s.Store.ListPeople()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	known := map[int64]bool{}
	for _, p := range people {
		known[p.ID] = true
	}
	out := []eventCard{}
	for _, e := range events {
		coming := 0
		for _, id := range e.Who {
			if known[id] { // someone since taken off the Family page doesn't count
				coming++
			}
		}
		dishes, err := s.Store.Dishes(e.ID)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		out = append(out, eventCard{Event: e, Dishes: len(dishes), People: coming + e.Extra})
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": out})
}

// handleSaveEvent adds (POST) or changes (PUT) an event.
func (s *Server) handleSaveEvent(w http.ResponseWriter, r *http.Request) {
	var e store.Event
	if !readJSON(w, r, &e, 16<<10) {
		return
	}
	e.ID = 0
	if id, ok := pathID(r, "id"); ok {
		e.ID = id
	}
	e.Name = strings.TrimSpace(e.Name)
	if e.Name == "" || len(e.Name) > 80 {
		writeErr(w, http.StatusBadRequest, "give the event a name (up to 80 letters)")
		return
	}
	if _, err := time.Parse(day, e.Date); e.Date != "" && err != nil {
		writeErr(w, http.StatusBadRequest, "that date isn't valid")
		return
	}
	if e.Extra < 0 || e.Extra > 500 || len(e.Notes) > 2000 {
		writeErr(w, http.StatusBadRequest, "up to 500 more guests and 2,000 letters of notes")
		return
	}
	people, err := s.Store.ListPeople()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	known := map[int64]bool{}
	for _, p := range people {
		known[p.ID] = true
	}
	who := []int64{}
	for _, id := range e.Who {
		if known[id] {
			who = append(who, id)
		}
	}
	e.Who = who
	e.CreatedBy = auth.UserFrom(r).Username
	id, err := s.Store.SaveEvent(&e)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (s *Server) handleDeleteEvent(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := s.Store.DeleteEvent(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// eventsOn lists the events on a date, for Tonight.
func (s *Server) eventsOn(date string) []store.Event {
	events, err := s.Store.ListEvents(date)
	out := []store.Event{}
	if err != nil {
		return out
	}
	for _, e := range events {
		if e.Date == date {
			out = append(out, e)
		}
	}
	return out
}
