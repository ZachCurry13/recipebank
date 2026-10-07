package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/zachcurry13/recipebank/internal/safety"
	"github.com/zachcurry13/recipebank/internal/store"
)

// The guest page of an event (/e/{token}, no account): guests say they're
// coming, with their allergies and diets, and what they're bringing, and see
// the menu marked for them. They never see the family's or other guests'
// allergies, only how many are coming. Their browser keeps a key (header
// X-Guest-Key) to change what they said.

const (
	maxGuests      = 60
	maxGuestDishes = 60
)

// guestEvent finds the event a link opens, or writes 410 when it's run out.
func (s *Server) guestEvent(w http.ResponseWriter, r *http.Request) (*store.EventLink, *store.Event, bool) {
	l, err := s.Store.EventLinkByToken(chi.URLParam(r, "token"))
	if err != nil || l == nil || !s.Store.FeatureOn("events") {
		writeErr(w, http.StatusGone, "This link has run out. Ask whoever sent it for a new one.")
		return nil, nil, false
	}
	e, err := s.Store.Event(l.EventID)
	if err != nil {
		writeErr(w, http.StatusGone, "This event is gone.")
		return nil, nil, false
	}
	return l, e, true
}

// guestFromRequest is the guest whose browser sent its key, or nil.
func (s *Server) guestFromRequest(r *http.Request, eventID int64) *store.EventGuest {
	g, err := s.Store.GuestByKey(eventID, r.Header.Get("X-Guest-Key"))
	if err != nil {
		return nil
	}
	return g
}

func (s *Server) handleGuestPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
	page, err := fs.ReadFile(s.Web, "guest.html")
	if err != nil {
		http.Error(w, "not available", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(page)
}

type guestDish struct {
	ID       int64    `json:"id"`
	Title    string   `json:"title"`
	Brings   string   `json:"brings"`
	Contains []string `json:"contains"` // allergen names its cook gave
	Recipe   bool     `json:"recipe"`   // a family recipe (checked from its ingredients)
	Mine     bool     `json:"mine"`
	Status   string   `json:"status,omitempty"` // for this guest: ok, no or unsure
}

func (s *Server) handleGuestView(w http.ResponseWriter, r *http.Request) {
	l, e, ok := s.guestEvent(w, r)
	if !ok {
		return
	}
	me := s.guestFromRequest(r, e.ID)
	dishes, err := s.Store.Dishes(e.ID)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	guests, err := s.Store.EventGuests(e.ID)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	var p *safety.Person
	if me != nil {
		p = &safety.Person{ID: -me.ID, Name: me.Name, HeatMax: -1, Allergies: me.Allergies, Diets: me.Diets}
	}
	out := []guestDish{}
	for _, d := range dishes {
		gd := guestDish{ID: d.ID, Title: d.Title, Brings: d.Brings, Contains: []string{}, Mine: me != nil && d.GuestID != nil && *d.GuestID == me.ID}
		for _, k := range d.Contains {
			if a, ok := safety.AllergenByKey(k); ok {
				gd.Contains = append(gd.Contains, a.Label)
			}
		}
		var rcOK bool
		if d.RecipeID != nil {
			if rc, err := s.Store.Recipe(*d.RecipeID); err == nil {
				rcOK, gd.Recipe = true, true
				if gd.Title == "" {
					gd.Title = rc.Title
				}
				if p != nil {
					gd.Status = safety.Check(rc, *p).Status
				}
			}
		}
		if !rcOK && p != nil {
			gd.Status = safety.DishStatus(d.Contains, *p)
		}
		if gd.Brings == "" {
			gd.Brings = "the hosts"
		}
		out = append(out, gd)
	}
	type allergen struct{ Key, Label string }
	var als []allergen
	for _, a := range safety.AllergenList(s.Store.Setting(store.KeyAllergenList)) {
		als = append(als, allergen{a.Key, a.Label})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"event": map[string]string{"name": e.Name, "date": e.Date, "notes": e.Notes}, "until": l.ExpiresAt,
		"dishes": out, "coming": len(e.Who) + e.Extra + len(guests), "me": me,
		"allergens": als, "diets": safety.Diets,
	})
}

// guestBody is what a guest says about themselves.
type guestBody struct {
	Name      string            `json:"name"`
	Allergies map[string]string `json:"allergies"`
	Diets     []string          `json:"diets"`
	Note      string            `json:"note"`
}

func (b *guestBody) check() string {
	b.Name, b.Note = strings.TrimSpace(b.Name), strings.TrimSpace(b.Note)
	if b.Name == "" || len(b.Name) > 40 || len(b.Note) > 200 {
		return "your name (up to 40 letters) is needed, and a note of up to 200"
	}
	if b.Allergies == nil {
		b.Allergies = map[string]string{}
	}
	for k, sev := range b.Allergies {
		if _, ok := safety.AllergenByKey(k); !ok || (sev != safety.Avoid && sev != safety.Allergic && sev != safety.Severe) {
			return "unknown allergy: " + k
		}
	}
	for _, d := range b.Diets {
		if _, ok := safety.DietByKey(d); !ok {
			return "unknown diet: " + d
		}
	}
	return ""
}

// handleGuestMe adds the guest, or changes what they said (with their key).
func (s *Server) handleGuestMe(w http.ResponseWriter, r *http.Request) {
	if !s.guestWrites.allow(clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "too many changes: try again in a few minutes")
		return
	}
	_, e, ok := s.guestEvent(w, r)
	if !ok {
		return
	}
	var body guestBody
	if !readJSON(w, r, &body, 8<<10) {
		return
	}
	if msg := body.check(); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	g := s.guestFromRequest(r, e.ID)
	if g == nil {
		guests, err := s.Store.EventGuests(e.ID)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		if len(guests) >= maxGuests {
			writeErr(w, http.StatusBadRequest, "this event has as many guests as it can take here")
			return
		}
		g = &store.EventGuest{EventID: e.ID}
	}
	g.Name, g.Allergies, g.Diets, g.Note = body.Name, body.Allergies, body.Diets, body.Note
	if err := s.Store.SaveGuest(g); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"key": g.Key, "me": g})
}

// handleGuestLeave takes the guest (and what they bring) off the event.
func (s *Server) handleGuestLeave(w http.ResponseWriter, r *http.Request) {
	_, e, ok := s.guestEvent(w, r)
	if !ok {
		return
	}
	g := s.guestFromRequest(r, e.ID)
	if g == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err := s.Store.DeleteGuest(e.ID, g.ID); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleGuestDish adds what a guest is bringing, with the allergens it has.
func (s *Server) handleGuestDish(w http.ResponseWriter, r *http.Request) {
	if !s.guestWrites.allow(clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "too many changes: try again in a few minutes")
		return
	}
	_, e, ok := s.guestEvent(w, r)
	if !ok {
		return
	}
	g := s.guestFromRequest(r, e.ID)
	if g == nil {
		writeErr(w, http.StatusForbidden, "say you're coming first")
		return
	}
	var body struct {
		Title    string   `json:"title"`
		Contains []string `json:"contains"`
	}
	if !readJSON(w, r, &body, 4<<10) {
		return
	}
	body.Title = strings.TrimSpace(body.Title)
	if body.Title == "" || len(body.Title) > 80 {
		writeErr(w, http.StatusBadRequest, "say what you're bringing (up to 80 letters)")
		return
	}
	for _, k := range body.Contains {
		if _, ok := safety.AllergenByKey(k); !ok {
			writeErr(w, http.StatusBadRequest, "unknown allergen: "+k)
			return
		}
	}
	if n, err := s.Store.CountGuestDishes(e.ID); err != nil || n >= maxGuestDishes {
		writeErr(w, http.StatusBadRequest, "this event's menu is as long as it can be here")
		return
	}
	id, err := s.Store.AddDish(store.Dish{EventID: e.ID, Title: body.Title, Brings: g.Name, Contains: body.Contains, GuestID: &g.ID})
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

// handleGuestDropDish takes back a dish the guest added.
func (s *Server) handleGuestDropDish(w http.ResponseWriter, r *http.Request) {
	_, e, ok := s.guestEvent(w, r)
	if !ok {
		return
	}
	g := s.guestFromRequest(r, e.ID)
	id, _ := pathID(r, "id")
	dishes, err := s.Store.Dishes(e.ID)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	for _, d := range dishes {
		if d.ID == id && g != nil && d.GuestID != nil && *d.GuestID == g.ID {
			if err := s.Store.DeleteDish(e.ID, id); err != nil {
				writeStoreErr(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	writeErr(w, http.StatusNotFound, "not found")
}
