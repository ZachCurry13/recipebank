package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/store"
)

// The meal plan in a phone's calendar (Me → 📅): a private link the calendar
// app subscribes to, so planned meals show at the person's meal times (and
// events all day), each with a link to the recipe. "New link" stops the old one.

var mealNames = map[string]string{"breakfast": "🍳 Breakfast", "lunch": "🥪 Lunch", "dinner": "🍽 Dinner", "snack": "🍎 Snack"}

func (s *Server) handleGetCalendar(w http.ResponseWriter, r *http.Request) {
	f, on, err := s.Store.CalFeedFor(auth.UserFrom(r).ID)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	out := map[string]any{"on": on, "path": "", "times": store.MealTimes, "events": true, "public": ""}
	if on {
		out["path"], out["times"], out["events"] = "/cal/"+f.Token+".ics", f.Times, f.Events
	}
	// Google Calendar fetches the link from Google's own computers: it needs the outside address.
	if h := s.Store.Setting(store.KeyTunnelHostname); h != "" && s.Store.SettingBool(store.KeyTunnelEnabled) {
		out["public"] = "https://" + h
	}
	writeJSON(w, http.StatusOK, out)
}

// handleNewCalendar makes the link, or a new one in place of the old.
func (s *Server) handleNewCalendar(w http.ResponseWriter, r *http.Request) {
	if _, err := s.Store.NewCalFeed(auth.UserFrom(r).ID); err != nil {
		writeStoreErr(w, err)
		return
	}
	s.handleGetCalendar(w, r)
}

func (s *Server) handleSetCalendar(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Times  map[string]string `json:"times"`
		Events bool              `json:"events"`
	}
	if !readJSON(w, r, &body, 4<<10) {
		return
	}
	if err := s.Store.SetCalFeed(auth.UserFrom(r).ID, body.Times, body.Events); err != nil {
		writeStoreErr(w, err)
		return
	}
	s.handleGetCalendar(w, r)
}

func (s *Server) handleDeleteCalendar(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.DeleteCalFeed(auth.UserFrom(r).ID); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleCalendarFeed is what the calendar app reads (no sign-in: the link is
// the key). It covers two weeks back and three months ahead.
func (s *Server) handleCalendarFeed(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSuffix(chi.URLParam(r, "token"), ".ics")
	f, err := s.Store.CalFeedByToken(token)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u, err := s.Store.UserByID(f.UserID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	now := time.Now()
	from, to := now.AddDate(0, 0, -14).Format(day), now.AddDate(0, 0, 90).Format(day)
	base := requestBase(r)
	items := []calItem{}
	if s.Store.FeatureOn("plan") {
		meals := map[string]bool{}
		for _, m := range strings.Split(u.PlanMeals, ",") {
			meals[m] = true
		}
		entries, err := s.Store.PlanBetween(from, to)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		titles, err := s.Store.RecipeTitles()
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		for _, e := range entries {
			if u.PlanMeals != "" && !meals[e.Meal] {
				continue // they only plan some meals (Me → Keep it simple)
			}
			title, link := e.Title, ""
			if e.RecipeID != nil && titles[*e.RecipeID] != "" {
				title, link = titles[*e.RecipeID], fmt.Sprintf("%s/#/recipe/%d", base, *e.RecipeID)
			}
			if title == "" {
				continue
			}
			items = append(items, calItem{UID: fmt.Sprintf("plan-%d@recipebank", e.ID), Date: e.Date, Time: f.Times[e.Meal],
				Minutes: 60, Summary: mealNames[e.Meal] + ": " + title, Note: e.Note, Link: link})
		}
	}
	if f.Events && s.Store.FeatureOn("events") {
		events, err := s.Store.ListEvents(now.Format(day))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		for _, ev := range events {
			if ev.Date >= from && ev.Date <= to {
				items = append(items, calItem{UID: fmt.Sprintf("event-%d@recipebank", ev.ID), Date: ev.Date,
					Summary: "🎉 " + ev.Name, Link: fmt.Sprintf("%s/#/event/%d", base, ev.ID)})
			}
		}
	}
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Cache-Control", "private, max-age=300")
	_, _ = w.Write([]byte(buildICS("RecipeBank meals", items, now)))
}

// requestBase is the address the request came to (the outside address
// through remote access), for links back into RecipeBank.
func requestBase(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
