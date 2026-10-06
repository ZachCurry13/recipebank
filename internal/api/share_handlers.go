package api

import (
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/zachcurry13/recipebank/internal/recipe"
)

var linkRE = regexp.MustCompile(`https?://[^\s<>"]+`)

// handleShare is where a phone's share menu sends a link or text (the app's
// share target): it opens "Add a recipe" with the link, or the text, filled in.
func (s *Server) handleShare(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	link := q.Get("url")
	if link == "" {
		link = linkRE.FindString(q.Get("text"))
	}
	to := "/#/add"
	switch {
	case link != "":
		to += "?way=link&url=" + url.QueryEscape(strings.TrimRight(link, ".,)"))
	case strings.TrimSpace(q.Get("text")) != "":
		text := strings.TrimSpace(q.Get("title") + "\n" + q.Get("text"))
		if len(text) > 6000 {
			text = text[:6000]
		}
		to += "?way=text&text=" + url.QueryEscape(text)
	}
	http.Redirect(w, r, to, http.StatusSeeOther)
}

// handleCookbook returns whole recipes to print as the family cookbook: all
// Kitchen recipes by course and title, or one collection in its order.
func (s *Server) handleCookbook(w http.ResponseWriter, r *http.Request) {
	all, err := s.Store.ListRecipes("kitchen")
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	title := "Our family cookbook"
	list := all
	if id := queryID(r, "collection"); id > 0 {
		c, err := s.Store.Collection(id)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		ids, err := s.Store.CollectionRecipeIDs(id)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		byID := map[int64]recipe.Recipe{}
		for _, rc := range all {
			byID[rc.ID] = rc
		}
		list = []recipe.Recipe{}
		for _, id := range ids {
			if rc, ok := byID[id]; ok {
				list = append(list, rc)
			}
		}
		title = c.Name
	} else {
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].Course != list[j].Course {
				return list[i].Course < list[j].Course
			}
			return strings.ToLower(list[i].Title) < strings.ToLower(list[j].Title)
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"title": title, "recipes": list})
}
