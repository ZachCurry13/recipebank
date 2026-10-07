package api

import (
	"net/http"
	"sort"
	"strings"
)

// 🔍 Search everything (the top bar): recipes (Kitchen and Home & Care), what's
// in the house, the shopping list, the cookbooks' recipes, events, collections
// and people, each linking to its page. Plain word matching: every word typed
// must be there. Features turned off for the house aren't searched.

type findHit struct {
	Title string `json:"title"`
	Sub   string `json:"sub,omitempty"`
	Link  string `json:"link"`
}

type findGroup struct {
	Key   string    `json:"key"`
	Label string    `json:"label"`
	Hits  []findHit `json:"hits"`
}

const findPerGroup = 8

func (s *Server) handleFind(w http.ResponseWriter, r *http.Request) {
	words := strings.Fields(strings.ToLower(r.URL.Query().Get("q")))
	if len(words) == 0 || len(words) > 12 {
		writeJSON(w, http.StatusOK, map[string]any{"groups": []findGroup{}})
		return
	}
	has := func(text ...string) bool {
		all := strings.ToLower(strings.Join(text, " "))
		for _, w := range words {
			if !strings.Contains(all, w) {
				return false
			}
		}
		return true
	}
	on := s.Store.FeaturesOn()
	var groups []findGroup
	add := func(key, label string, hits []findHit) {
		if len(hits) > findPerGroup {
			hits = hits[:findPerGroup]
		}
		if len(hits) > 0 {
			groups = append(groups, findGroup{key, label, hits})
		}
	}

	if all, err := s.Store.ListRecipes(""); err == nil {
		type ranked struct {
			hit   findHit
			title bool
		}
		var rs []ranked
		for i := range all {
			rc := &all[i]
			if rc.Area == "home" && !on["home"] || !has(rc.Text()) {
				continue
			}
			sub := "Kitchen"
			if rc.Area == "home" {
				sub = "Home & Care"
			}
			if rc.Course != "" {
				sub += " · " + rc.Course
			}
			rs = append(rs, ranked{findHit{rc.Title, sub, "#/recipe/" + itoa(rc.ID)}, has(rc.Title)})
		}
		sort.SliceStable(rs, func(a, b int) bool { return rs[a].title && !rs[b].title }) // title matches first
		var hits []findHit
		for _, x := range rs {
			hits = append(hits, x.hit)
		}
		add("recipes", "Recipes", hits)
	}
	if on["pantry"] {
		if items, err := s.Store.ListStock(""); err == nil {
			var hits []findHit
			for _, it := range items {
				if it.Area == "home" && !on["home"] || !has(it.Name, it.Brand) {
					continue
				}
				link, sub := "#/pantry", "Pantry"
				if it.Area == "home" {
					link, sub = "#/supplies", "Supplies"
				}
				if it.Location != "" {
					sub += " · " + it.Location
				}
				hits = append(hits, findHit{it.Name, sub, link})
			}
			add("pantry", "In the house", hits)
		}
	}
	if on["shopping"] {
		if items, err := s.Store.ListShopping(); err == nil {
			var hits []findHit
			for _, it := range items {
				if !it.Checked && has(it.Name) {
					hits = append(hits, findHit{it.Name, "On the list", "#/shopping"})
				}
			}
			add("shopping", "Shopping list", hits)
		}
	}
	if on["books"] {
		var hits []findHit
		if list, err := s.Store.ListBooks(); err == nil {
			for _, b := range list {
				if has(b.Title, b.Author) {
					hits = append(hits, findHit{b.Title, "Cookbook", "#/book/" + itoa(b.ID)})
				}
			}
		}
		if entries, err := s.Store.SearchBookEntries(strings.Join(words, " "), findPerGroup); err == nil {
			for _, e := range entries {
				sub := e.BookTitle
				if e.Page != "" {
					sub += ", page " + e.Page
				}
				hits = append(hits, findHit{e.Title, sub, "#/book/" + itoa(e.BookID)})
			}
		}
		add("books", "In your cookbooks", hits)
	}
	if on["events"] {
		if list, err := s.Store.ListEvents(""); err == nil {
			var hits []findHit
			for _, e := range list {
				if has(e.Name, e.Notes) {
					hits = append(hits, findHit{e.Name, e.Date, "#/event/" + itoa(e.ID)})
				}
			}
			add("events", "Events", hits)
		}
	}
	if on["collections"] {
		if list, err := s.Store.ListCollections(); err == nil {
			var hits []findHit
			for _, c := range list {
				if has(c.Name, c.Description) {
					hits = append(hits, findHit{strings.TrimSpace(c.Icon + " " + c.Name), "Collection", "#/collection/" + itoa(c.ID)})
				}
			}
			add("collections", "Collections", hits)
		}
	}
	if people, err := s.Store.ListPeople(); err == nil {
		var hits []findHit
		for _, p := range people {
			if has(p.Name) {
				hits = append(hits, findHit{p.Name, "Family", "#/family"})
			}
		}
		add("people", "People", hits)
	}
	if groups == nil {
		groups = []findGroup{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": groups})
}
