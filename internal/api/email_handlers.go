package api

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/mail"
	"github.com/zachcurry13/recipebank/internal/recipe"
	"github.com/zachcurry13/recipebank/internal/store"
)

// mailsPerHour caps how many emails one person can send.
const mailsPerHour = 20

type mailLimiter struct {
	mu   sync.Mutex
	sent map[int64][]time.Time
}

func (l *mailLimiter) allow(user int64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.sent == nil {
		l.sent = map[int64][]time.Time{}
	}
	recent := l.sent[user][:0]
	for _, t := range l.sent[user] {
		if time.Since(t) < time.Hour {
			recent = append(recent, t)
		}
	}
	if len(recent) >= mailsPerHour {
		l.sent[user] = recent
		return false
	}
	l.sent[user] = append(recent, time.Now())
	return true
}

func (s *Server) mailConfig() mail.Config {
	return mail.Config{Host: s.Store.Setting(store.KeySMTPHost), Port: s.Store.Setting(store.KeySMTPPort),
		Username: s.Store.Setting(store.KeySMTPUser), Password: s.Store.Setting(store.KeySMTPPassword),
		From: s.Store.Setting(store.KeySMTPFrom)}
}

func (s *Server) emailReady() bool { return s.mailConfig().Validate() == nil }

// mayEmail: parents always; everyone else when Admin → Email allows it.
func (s *Server) mayEmail(u *store.User) bool {
	return u != nil && (u.CanManage() || s.Store.Setting(store.KeyEmailWho) == "everyone")
}

// sendMail sends m for the signed-in person, within their hourly limit.
func (s *Server) sendMail(w http.ResponseWriter, r *http.Request, m mail.Message) {
	u := auth.UserFrom(r)
	if !s.mayEmail(u) {
		writeErr(w, http.StatusForbidden, "only parents can send email (Admin → Email)")
		return
	}
	if _, err := mail.Address(m.To); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if !s.mails.allow(u.ID) {
		writeErr(w, http.StatusTooManyRequests, fmt.Sprintf("that's %d emails this hour; try again later", mailsPerHour))
		return
	}
	send := s.SendMail
	if send == nil {
		send = mail.Send
	}
	if err := send(s.mailConfig(), m); err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleEmailTest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		To string `json:"to"`
	}
	if !readJSON(w, r, &body, 1<<10) {
		return
	}
	s.sendMail(w, r, mail.Message{To: body.To, Subject: "RecipeBank can send email",
		Text: "This is a test from RecipeBank: email is set up.", HTML: "<p>This is a test from RecipeBank: email is set up.</p>"})
}

// handleEmailRecipe sends a recipe as written, with an optional note.
func (s *Server) handleEmailRecipe(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	var body struct {
		To   string `json:"to"`
		Note string `json:"note"`
	}
	if !ok || !readJSON(w, r, &body, 8<<10) {
		return
	}
	rc, err := s.Store.Recipe(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	text, htm := recipeEmail(rc, strings.TrimSpace(body.Note), auth.UserFrom(r).Username)
	s.sendMail(w, r, mail.Message{To: body.To, Subject: rc.Title + " (a recipe from RecipeBank)", Text: text, HTML: htm})
}

func recipeEmail(rc *recipe.Recipe, note, by string) (string, string) {
	var t, h strings.Builder
	e := html.EscapeString
	if len(note) > 1000 {
		note = note[:1000]
	}
	if note != "" {
		fmt.Fprintf(&t, "%s\n\n", note)
		fmt.Fprintf(&h, "<p>%s</p>", strings.ReplaceAll(e(note), "\n", "<br>"))
	}
	fmt.Fprintf(&t, "%s\n", rc.Title)
	fmt.Fprintf(&h, "<h1>%s</h1>", e(rc.Title))
	var facts []string
	if rc.Servings > 0 {
		facts = append(facts, "Serves "+strconv.FormatFloat(rc.Servings, 'f', -1, 64))
	}
	if rc.TotalMin > 0 {
		facts = append(facts, fmt.Sprintf("%d min", rc.TotalMin))
	}
	if len(facts) > 0 {
		fmt.Fprintf(&t, "%s\n", strings.Join(facts, " · "))
		fmt.Fprintf(&h, "<p>%s</p>", e(strings.Join(facts, " · ")))
	}
	t.WriteString("\nIngredients\n")
	h.WriteString("<h2>Ingredients</h2><ul>")
	for _, in := range rc.Ingredients {
		fmt.Fprintf(&t, "- %s\n", in.Line)
		fmt.Fprintf(&h, "<li>%s</li>", e(in.Line))
	}
	t.WriteString("\nSteps\n")
	h.WriteString("</ul><h2>Steps</h2><ol>")
	for i, st := range rc.Steps {
		fmt.Fprintf(&t, "%d. %s\n", i+1, st.Text)
		fmt.Fprintf(&h, "<li>%s</li>", e(st.Text))
	}
	h.WriteString("</ol>")
	if rc.Notes != "" {
		fmt.Fprintf(&t, "\nNotes\n%s\n", rc.Notes)
		fmt.Fprintf(&h, "<h2>Notes</h2><p>%s</p>", strings.ReplaceAll(e(rc.Notes), "\n", "<br>"))
	}
	if rc.SourceURL != "" {
		fmt.Fprintf(&t, "\nFrom: %s\n", rc.SourceURL)
		fmt.Fprintf(&h, "<p>From: %s</p>", e(rc.SourceURL))
	}
	fmt.Fprintf(&t, "\nSent from RecipeBank by %s.\n", by)
	fmt.Fprintf(&h, "<p><small>Sent from RecipeBank by %s.</small></p>", e(by))
	return t.String(), h.String()
}

// handleEmailShopping sends what's still to buy, by store section.
func (s *Server) handleEmailShopping(w http.ResponseWriter, r *http.Request) {
	var body struct {
		To string `json:"to"`
	}
	if !readJSON(w, r, &body, 1<<10) {
		return
	}
	items, err := s.Store.ListShopping()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	var t, h strings.Builder
	n, section := 0, "\x00"
	h.WriteString("<h1>Shopping list</h1>")
	for _, it := range items {
		if it.Checked {
			continue
		}
		if it.Section != section {
			if section != "\x00" {
				h.WriteString("</ul>")
			}
			section = it.Section
			fmt.Fprintf(&t, "\n%s\n", strings.ToUpper(orOther(section)))
			fmt.Fprintf(&h, "<h2>%s</h2><ul>", html.EscapeString(orOther(section)))
		}
		line := shopLine(it)
		fmt.Fprintf(&t, "[ ] %s\n", line)
		fmt.Fprintf(&h, "<li>%s</li>", html.EscapeString(line))
		n++
	}
	if n == 0 {
		writeErr(w, http.StatusBadRequest, "the list is empty")
		return
	}
	h.WriteString("</ul>")
	by := auth.UserFrom(r).Username
	s.sendMail(w, r, mail.Message{To: body.To, Subject: fmt.Sprintf("Shopping list (%d things)", n),
		Text: fmt.Sprintf("Shopping list (%d things)\n%s\nSent from RecipeBank by %s.\n", n, t.String(), by),
		HTML: h.String() + fmt.Sprintf("<p><small>Sent from RecipeBank by %s.</small></p>", html.EscapeString(by))})
}

func orOther(section string) string {
	if section == "" {
		return "Other"
	}
	return section
}

// shopLine is "2 cups flour (for Pancakes)".
func shopLine(it store.ShopItem) string {
	line := it.Name
	if it.Qty > 0 {
		q := strconv.FormatFloat(float64(int(it.Qty*100+0.5))/100, 'f', -1, 64)
		line = strings.TrimSpace(q+" "+it.Unit) + " " + it.Name
	}
	if it.Note != "" {
		line += " (" + it.Note + ")"
	}
	return line
}
