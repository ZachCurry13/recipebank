package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zachcurry13/recipebank/internal/push"
	"github.com/zachcurry13/recipebank/internal/safety"
	"github.com/zachcurry13/recipebank/internal/store"
)

// Reminders sends the daily notifications at the hours set in Admin (the
// server's time zone, TZ): food to use soon in the morning, tonight's plan in
// the afternoon. Each goes out once a day, even after a restart.
func (s *Server) Reminders(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		s.remindIfDue(time.Now())
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (s *Server) remindIfDue(now time.Time) {
	today := now.Format(day)
	due := func(hourKey, lastKey string) bool {
		h := s.Store.SettingInt(hourKey)
		if h < 0 || h > 23 || now.Hour() < h || s.Store.Setting(lastKey) == today {
			return false
		}
		_ = s.Store.SetSetting(lastKey, today)
		return true
	}
	if due(store.KeyMorningHour, "reminder_morning_last") {
		if m, ok := s.useSoonMessage(now); ok {
			s.Push.Notify("useby", 0, m)
		}
	}
	if due(store.KeyTonightHour, "reminder_tonight_last") {
		if m, ok := s.tonightMessage(now); ok {
			s.Push.Notify("tonight", 0, m)
		}
	}
}

func (s *Server) useSoonMessage(now time.Time) (push.Message, bool) {
	items := s.useSoon(now)
	if len(items) == 0 {
		return push.Message{}, false
	}
	var parts []string
	for _, it := range items {
		switch d := daysUntil(now, it.UseBy); {
		case d < 0:
			parts = append(parts, it.Name+" (past its date)")
		case d == 0:
			parts = append(parts, it.Name+" (today)")
		case d == 1:
			parts = append(parts, it.Name+" (tomorrow)")
		default:
			parts = append(parts, it.Name)
		}
	}
	return push.Message{Title: "🥫 Use soon", Body: strings.Join(parts, ", "), URL: "/#/pantry", Tag: "useby"}, true
}

func daysUntil(now time.Time, date string) int {
	d, err := time.Parse(day, date)
	if err != nil {
		return 99
	}
	n, _ := time.Parse(day, now.Format(day))
	return int(d.Sub(n).Hours() / 24)
}

// tonightMessage says what's planned for dinner and who it doesn't suit.
func (s *Server) tonightMessage(now time.Time) (push.Message, bool) {
	date := now.Format(day)
	pc, err := s.loadPlanContext(date, date)
	if err != nil {
		return push.Message{}, false
	}
	entries, err := s.Store.PlanBetween(date, date)
	if err != nil {
		return push.Message{}, false
	}
	ids, _ := pc.who(date)
	diners := pc.diners(ids)
	for _, e := range entries {
		if e.Meal != "dinner" {
			continue
		}
		title := e.Title
		var problems []string
		if e.RecipeID != nil {
			if rc := s.recipeFor(pc, *e.RecipeID); rc != nil {
				title = rc.Title
				for _, p := range diners {
					if v := safety.Check(rc, p); v.Status != safety.OK && len(v.Reasons) > 0 {
						word := "not for"
						if v.Status == safety.Unsure {
							word = "check for"
						}
						problems = append(problems, fmt.Sprintf("%s %s (%s)", word, p.Name, v.Reasons[0].Text))
					}
				}
			}
		}
		body := "Tonight: " + title
		if len(problems) > 0 {
			body += ". " + strings.Join(problems, "; ")
		}
		return push.Message{Title: "🌙 Dinner", Body: body, URL: "/#/tonight", Tag: "tonight"}, true
	}
	return push.Message{Title: "🌙 Dinner", Body: "Nothing planned for dinner yet. Tap for ideas everyone can eat.", URL: "/#/tonight", Tag: "tonight"}, true
}
