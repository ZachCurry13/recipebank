package api

import (
	"strings"

	"github.com/zachcurry13/recipebank/internal/books"
	"github.com/zachcurry13/recipebank/internal/store"
)

// The bookshelf in a backup: books.json (each cookbook with its recipes by
// page, linked to recipes by their ids in recipes.json) and pile.json (the
// cards still to photograph). Restoring joins books by ISBN or title.

type backupBook struct {
	Title   string             `json:"title"`
	Author  string             `json:"author"`
	ISBN    string             `json:"isbn"`
	Cover   string             `json:"cover"`
	Entries []backupBookRecipe `json:"entries"`
}

type backupBookRecipe struct {
	Title  string `json:"title"`
	Page   string `json:"page"`
	Recipe int64  `json:"recipe,omitempty"`
}

type backupCard struct {
	Title  string `json:"title"`
	Note   string `json:"note"`
	Done   bool   `json:"done"`
	Recipe int64  `json:"recipe,omitempty"`
}

// backupShelf is the bookshelf and pile, ready for the zip.
func (s *Server) backupShelf() ([]backupBook, []backupCard, error) {
	list, err := s.Store.ListBooks()
	if err != nil {
		return nil, nil, err
	}
	out := []backupBook{}
	for _, b := range list {
		entries, err := s.Store.BookEntries(b.ID)
		if err != nil {
			return nil, nil, err
		}
		bb := backupBook{Title: b.Title, Author: b.Author, ISBN: b.ISBN, Cover: b.Cover, Entries: []backupBookRecipe{}}
		for _, e := range entries {
			be := backupBookRecipe{Title: e.Title, Page: e.Page}
			if e.RecipeID != nil {
				be.Recipe = *e.RecipeID
			}
			bb.Entries = append(bb.Entries, be)
		}
		out = append(out, bb)
	}
	pile, err := s.Store.ListPile()
	if err != nil {
		return nil, nil, err
	}
	cards := []backupCard{}
	for _, c := range pile {
		bc := backupCard{Title: c.Title, Note: c.Note, Done: c.DoneAt != ""}
		if c.RecipeID != nil {
			bc.Recipe = *c.RecipeID
		}
		cards = append(cards, bc)
	}
	return out, cards, nil
}

// restoreShelf adds the books and cards this RecipeBank doesn't have yet;
// newID maps recipe ids in the file to the ones saved here.
func (s *Server) restoreShelf(list []backupBook, cards []backupCard, newID map[int64]int64, photo func(string) string) (int, error) {
	current, err := s.Store.ListBooks()
	if err != nil {
		return 0, err
	}
	added := 0
	for _, bb := range list {
		bb.Title, bb.ISBN = strings.TrimSpace(bb.Title), books.CleanISBN(bb.ISBN)
		if bb.Title == "" || len(bb.Title) > 200 || len(bb.Author) > 200 {
			continue
		}
		var id int64
		for _, b := range current {
			if bb.ISBN != "" && b.ISBN == bb.ISBN || strings.EqualFold(b.Title, bb.Title) {
				id = b.ID
				break
			}
		}
		if id == 0 {
			if id, err = s.Store.SaveBook(&store.Book{Title: bb.Title, Author: bb.Author, ISBN: bb.ISBN, Cover: photo(bb.Cover)}); err != nil {
				return added, err
			}
			current = append(current, store.Book{ID: id, Title: bb.Title, ISBN: bb.ISBN})
			added++
		}
		var entries []store.BookEntry
		for _, e := range bb.Entries {
			if e.Title = strings.TrimSpace(e.Title); e.Title != "" && len(e.Title) <= 200 && len(e.Page) <= 12 {
				entries = append(entries, store.BookEntry{Title: e.Title, Page: e.Page})
			}
		}
		if len(entries) > 0 {
			if _, err := s.Store.AddBookEntries(id, entries); err != nil {
				return added, err
			}
		}
		for _, e := range bb.Entries {
			if rid, ok := newID[e.Recipe]; ok && e.Recipe > 0 {
				if err := s.Store.LinkRecipe(rid, id, e.Page, e.Title); err != nil {
					return added, err
				}
			}
		}
	}
	pile, err := s.Store.ListPile()
	if err != nil {
		return added, err
	}
	have := map[string]bool{}
	for _, c := range pile {
		have[strings.ToLower(c.Title)] = true
	}
	for _, c := range cards {
		c.Title = strings.TrimSpace(c.Title)
		if c.Title == "" || len(c.Title) > 150 || len(c.Note) > 300 || have[strings.ToLower(c.Title)] {
			continue
		}
		have[strings.ToLower(c.Title)] = true
		id, err := s.Store.AddPile(c.Title, c.Note)
		if err != nil {
			return added, err
		}
		if c.Done {
			var rid *int64
			if n, ok := newID[c.Recipe]; ok && c.Recipe > 0 {
				rid = &n
			}
			if err := s.Store.UpdatePile(id, c.Title, c.Note, true, rid); err != nil {
				return added, err
			}
		}
	}
	return added, nil
}
