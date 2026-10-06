package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/zachcurry13/recipebank/internal/books"
	"github.com/zachcurry13/recipebank/internal/store"
)

// The bookshelf: the family's cookbooks (looked up by the barcode's ISBN in
// Open Library, or typed), each with its recipes by page, so "where's that
// lasagna?" finds "page 112" even before anyone types the recipe in.

func (s *Server) books() books.Client { return books.Client{HTTP: s.Fetch, Base: s.BookBase} }

func (s *Server) handleListBooks(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListBooks()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"books": list})
}

func (s *Server) handleGetBook(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	b, err := s.Store.GetBook(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	entries, err := s.Store.BookEntries(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"book": b, "entries": entries})
}

// handleLookupBook finds a cookbook by ISBN: on the shelf already, or in Open Library.
func (s *Server) handleLookupBook(w http.ResponseWriter, r *http.Request) {
	isbn := books.CleanISBN(r.URL.Query().Get("isbn"))
	if isbn == "" {
		writeErr(w, http.StatusBadRequest, "That isn't a book's barcode: it starts with 978 or 979 (an ISBN).")
		return
	}
	if b, err := s.Store.BookByISBN(isbn); err == nil {
		writeJSON(w, http.StatusOK, map[string]any{"existing": b})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	b, err := s.books().Lookup(ctx, isbn)
	if errors.Is(err, books.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "Open Library doesn't know that book yet. Type its title instead.")
		return
	}
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"book": b})
}

// handleSaveBook adds (POST) or changes (PUT) a cookbook. A new one with an
// ISBN gets Open Library's cover when there is one; "image" is a photo of
// the cover instead.
func (s *Server) handleSaveBook(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title  string `json:"title"`
		Author string `json:"author"`
		ISBN   string `json:"isbn"`
		Image  string `json:"image"`
	}
	if !readJSON(w, r, &body, 12<<20) {
		return
	}
	b := &store.Book{Title: strings.TrimSpace(body.Title), Author: strings.TrimSpace(body.Author), ISBN: books.CleanISBN(body.ISBN)}
	if b.Title == "" || len(b.Title) > 200 || len(b.Author) > 200 {
		writeErr(w, http.StatusBadRequest, "a title (and author) under 200 characters is needed")
		return
	}
	if id, ok := pathID(r, "id"); ok {
		old, err := s.Store.GetBook(id)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		b.ID, b.Cover = id, old.Cover
	} else if other, err := s.Store.BookByISBN(b.ISBN); err == nil {
		writeJSON(w, http.StatusOK, other) // on the shelf already
		return
	}
	if body.Image != "" {
		data, _, err := decodeDataURL(body.Image)
		if err == nil {
			b.Cover, err = s.savePhoto(data)
		}
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	} else if b.ID == 0 && b.ISBN != "" {
		b.Cover = s.bookCover(r.Context(), b.ISBN)
	}
	id, err := s.Store.SaveBook(b)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	saved, err := s.Store.GetBook(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

// bookCover saves Open Library's cover for an ISBN; "" when there's none.
func (s *Server) bookCover(ctx context.Context, isbn string) string {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	b, err := s.books().Lookup(ctx, isbn)
	if err != nil || b.CoverURL == "" {
		return ""
	}
	data, _, err := s.fetch(ctx, b.CoverURL, 1<<20)
	if err != nil {
		return ""
	}
	name, err := s.savePhoto(data)
	if err != nil {
		return ""
	}
	return name
}

func (s *Server) handleDeleteBook(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	if err := s.Store.DeleteBook(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleAddBookEntries adds recipes to a cookbook's list: [{title, page}].
func (s *Server) handleAddBookEntries(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	var body struct {
		Entries []store.BookEntry `json:"entries"`
	}
	if !readJSON(w, r, &body, 256<<10) {
		return
	}
	if _, err := s.Store.GetBook(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	var clean []store.BookEntry
	for _, e := range body.Entries {
		e.Title, e.Page = strings.TrimSpace(e.Title), strings.TrimSpace(e.Page)
		if e.Title != "" && len(e.Title) <= 200 && len(e.Page) <= 12 {
			clean = append(clean, e)
		}
	}
	if len(clean) == 0 || len(clean) > 500 {
		writeErr(w, http.StatusBadRequest, "add 1 to 500 recipes, each with a name")
		return
	}
	n, err := s.Store.AddBookEntries(id, clean)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"added": n})
}

func (s *Server) handleDeleteBookEntry(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	if err := s.Store.DeleteBookEntry(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleRecipeBook says which cookbook page a saved recipe is from (book_id 0: none).
func (s *Server) handleRecipeBook(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	var body struct {
		BookID int64  `json:"book_id"`
		Page   string `json:"page"`
	}
	if !readJSON(w, r, &body, 4<<10) {
		return
	}
	rc, err := s.Store.Recipe(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	if body.Page = strings.TrimSpace(body.Page); len(body.Page) > 12 {
		writeErr(w, http.StatusBadRequest, "a page number is short")
		return
	}
	if body.BookID > 0 {
		if _, err := s.Store.GetBook(body.BookID); err != nil {
			writeStoreErr(w, err)
			return
		}
	}
	if err := s.Store.LinkRecipe(rc.ID, body.BookID, body.Page, rc.Title); err != nil {
		writeStoreErr(w, err)
		return
	}
	e, err := s.Store.EntryForRecipe(rc.ID)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"book": e})
}

// handleSearchBooks finds recipes in the family's cookbooks by name.
func (s *Server) handleSearchBooks(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 100 {
		q = q[:100]
	}
	out, err := s.Store.SearchBookEntries(q, 20)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": out})
}
