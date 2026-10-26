package api

import (
	"strings"
	"time"
)

// calItem is one entry in a calendar feed: at a time on a date, or all day
// when Time is "".
type calItem struct {
	UID     string
	Date    string // YYYY-MM-DD
	Time    string // "18:00", or "" for all day
	Minutes int
	Summary string
	Note    string
	Link    string
}

// buildICS writes an iCalendar feed. Times are "floating" (no time zone):
// a meal at 18:00 shows at 6 pm wherever the family's calendar is.
func buildICS(name string, items []calItem, now time.Time) string {
	var b strings.Builder
	line := func(s string) { b.WriteString(fold(s) + "\r\n") }
	line("BEGIN:VCALENDAR")
	line("VERSION:2.0")
	line("PRODID:-//RecipeBank//Meal plan//EN")
	line("CALSCALE:GREGORIAN")
	line("METHOD:PUBLISH")
	line("X-WR-CALNAME:" + icsText(name))
	line("REFRESH-INTERVAL;VALUE=DURATION:PT1H")
	line("X-PUBLISHED-TTL:PT1H")
	stamp := now.UTC().Format("20060102T150405Z")
	for _, it := range items {
		d, err := time.Parse("2006-01-02", it.Date)
		if err != nil {
			continue
		}
		line("BEGIN:VEVENT")
		line("UID:" + it.UID)
		line("DTSTAMP:" + stamp)
		if t, err := time.Parse("15:04", it.Time); err == nil {
			start := d.Add(time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute)
			line("DTSTART:" + start.Format("20060102T150405"))
			line("DTEND:" + start.Add(time.Duration(max(it.Minutes, 15))*time.Minute).Format("20060102T150405"))
		} else {
			line("DTSTART;VALUE=DATE:" + d.Format("20060102"))
			line("DTEND;VALUE=DATE:" + d.AddDate(0, 0, 1).Format("20060102"))
		}
		line("SUMMARY:" + icsText(it.Summary))
		desc := strings.TrimSpace(strings.Join([]string{it.Note, it.Link}, "\n"))
		if desc != "" {
			line("DESCRIPTION:" + icsText(desc))
		}
		if it.Link != "" {
			line("URL:" + it.Link)
		}
		line("TRANSP:TRANSPARENT") // a meal doesn't make anyone "busy"
		line("END:VEVENT")
	}
	line("END:VCALENDAR")
	return b.String()
}

// icsText escapes text for iCalendar (RFC 5545 3.3.11).
func icsText(s string) string {
	return strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\r\n", `\n`, "\n", `\n`, "\r", "").Replace(s)
}

// fold breaks lines longer than 75 bytes, never inside a character.
func fold(s string) string {
	if len(s) <= 75 {
		return s
	}
	var b strings.Builder
	n := 0
	for _, r := range s {
		size := len(string(r))
		if n+size > 75 {
			b.WriteString("\r\n ")
			n = 1
		}
		b.WriteRune(r)
		n += size
	}
	return b.String()
}
