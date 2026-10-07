package api

import (
	"net/http"
	"time"

	"github.com/zachcurry13/recipebank/internal/auth"
)

// The host's side of an event's guest link: make it, see it, stop it, and
// take a guest off.

func (s *Server) handleGetEventLink(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	l, err := s.Store.EventLinkFor(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"link": l})
}

// handleMakeEventLink opens a guest link until the day after the event (30
// days when it has no date yet).
func (s *Server) handleMakeEventLink(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	e, err := s.Store.Event(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	until := time.Now().AddDate(0, 0, 30)
	if e.Date != "" {
		d, err := time.Parse(day, e.Date)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "the event's date can't be read")
			return
		}
		until = d.AddDate(0, 0, 2) // to the end of the day after
		if until.Before(time.Now()) {
			writeErr(w, http.StatusBadRequest, "this event is over")
			return
		}
	}
	if l, err := s.Store.EventLinkFor(id); err == nil && l != nil {
		writeJSON(w, http.StatusOK, map[string]any{"link": l})
		return
	}
	l, err := s.Store.CreateEventLink(id, auth.UserFrom(r).Username, until)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"link": l})
}

func (s *Server) handleStopEventLink(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	if err := s.Store.StopEventLink(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleDropEventGuest takes a guest who came through the link off the event.
func (s *Server) handleDropEventGuest(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	gid, _ := pathID(r, "gid")
	if err := s.Store.DeleteGuest(id, gid); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
