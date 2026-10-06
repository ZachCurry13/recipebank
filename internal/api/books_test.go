package api

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A cookbook found by its barcode gets Open Library's title, author and
// cover; its recipes by page are searchable, and a saved recipe can say
// which page it's from.
func TestBookshelf(t *testing.T) {
	c, srv := setup(t)
	var cover bytes.Buffer
	jpeg.Encode(&cover, image.NewRGBA(image.Rect(0, 0, 20, 30)), nil)
	var ol *httptest.Server
	ol = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/b/"):
			w.Write(cover.Bytes())
		case strings.Contains(r.URL.RawQuery, "9780743246262"):
			fmt.Fprintf(w, `{"ISBN:9780743246262": {"title": "Family Favorites", "authors": [{"name": "A. Cook"}],
				"cover": {"medium": "%s/b/id/1-M.jpg"}}}`, ol.URL)
		default:
			w.Write([]byte(`{}`))
		}
	}))
	defer ol.Close()
	srv.BookBase = ol.URL
	srv.Fetch = ol.Client()

	var found struct {
		Book struct{ Title, Author, ISBN string }
	}
	if code := c.do("GET", "/api/books/lookup?isbn=978-0-7432-4626-2", nil, &found); code != 200 || found.Book.Title != "Family Favorites" {
		t.Fatalf("lookup: %d %+v", code, found)
	}
	if code := c.do("GET", "/api/books/lookup?isbn=4006381333931", nil, nil); code != 400 {
		t.Fatalf("a product barcode isn't a book: %d", code)
	}
	if code := c.do("GET", "/api/books/lookup?isbn=0743246268", nil, nil); code != 404 {
		t.Fatalf("unknown book: %d", code)
	}
	var book struct {
		ID    int64
		Cover string
	}
	c.do("POST", "/api/books", map[string]any{"title": found.Book.Title, "author": found.Book.Author, "isbn": found.Book.ISBN}, &book)
	if book.ID == 0 || !photoName.MatchString(book.Cover) {
		t.Fatalf("saved with its cover: %+v", book)
	}
	// The daily clean-up keeps covers, however old.
	old := time.Now().Add(-72 * time.Hour)
	os.Chtimes(filepath.Join(srv.PhotoDir, book.Cover), old, old)
	srv.CleanPhotos()
	if _, err := os.Stat(filepath.Join(srv.PhotoDir, book.Cover)); err != nil {
		t.Fatalf("the cover was cleaned up: %v", err)
	}
	var again struct{ Existing struct{ ID int64 } }
	if c.do("GET", "/api/books/lookup?isbn=9780743246262", nil, &again); again.Existing.ID != book.ID {
		t.Fatalf("on the shelf already: %+v", again)
	}

	var added struct{ Added int }
	c.do("POST", fmt.Sprintf("/api/books/%d/entries", book.ID), map[string]any{"entries": []map[string]string{
		{"title": "Lasagna", "page": "112"}, {"title": "Garlic bread", "page": "9"}, {"title": "Lasagna", "page": "112"}, {"title": " "}}}, &added)
	if added.Added != 2 {
		t.Fatalf("entries: %+v", added)
	}
	var rc struct{ ID int64 }
	c.do("POST", "/api/recipes", map[string]any{"title": "Mom's lasagna", "ingredients": []map[string]string{{"line": "1 lb pasta"}}}, &rc)
	if code := c.do("PUT", fmt.Sprintf("/api/recipes/%d/book", rc.ID), map[string]any{"book_id": book.ID, "page": "112"}, nil); code != 200 {
		t.Fatalf("link: %d", code)
	}
	var page struct {
		Book struct {
			BookTitle string `json:"book_title"`
			Page      string
			Title     string
		}
	}
	c.do("GET", fmt.Sprintf("/api/recipes/%d", rc.ID), nil, &page)
	if page.Book.BookTitle != "Family Favorites" || page.Book.Page != "112" || page.Book.Title != "Lasagna" {
		t.Fatalf("the recipe's page: %+v", page.Book)
	}
	var shelf struct {
		Book    struct{ Entries, Saved int }
		Entries []struct {
			Title    string
			Page     string
			RecipeID *int64 `json:"recipe_id"`
		}
	}
	c.do("GET", fmt.Sprintf("/api/books/%d", book.ID), nil, &shelf)
	if shelf.Book.Entries != 2 || shelf.Book.Saved != 1 || shelf.Entries[0].Page != "9" || shelf.Entries[1].RecipeID == nil {
		t.Fatalf("book: %+v", shelf)
	}
	var hits struct {
		Entries []struct {
			Title     string
			BookTitle string `json:"book_title"`
		}
	}
	c.do("GET", "/api/books/search?q=lasag", nil, &hits)
	if len(hits.Entries) != 1 || hits.Entries[0].BookTitle != "Family Favorites" {
		t.Fatalf("search: %+v", hits)
	}
	// Unlinking leaves the entry; deleting the book leaves the recipe.
	c.do("PUT", fmt.Sprintf("/api/recipes/%d/book", rc.ID), map[string]any{"book_id": 0}, nil)
	c.do("GET", fmt.Sprintf("/api/books/%d", book.ID), nil, &shelf)
	if shelf.Book.Saved != 0 || shelf.Book.Entries != 2 {
		t.Fatalf("unlinked: %+v", shelf.Book)
	}
	c.do("DELETE", fmt.Sprintf("/api/books/%d", book.ID), nil, nil)
	if code := c.do("GET", fmt.Sprintf("/api/recipes/%d", rc.ID), nil, nil); code != 200 {
		t.Fatalf("recipe after the book went: %d", code)
	}
}

// Cards waiting to be photographed are ticked off with the recipe saved from them.
func TestPile(t *testing.T) {
	c, _ := setup(t)
	var card struct{ ID int64 }
	c.do("POST", "/api/pile", map[string]any{"title": "Grandma's pie card", "note": "blue tin"}, &card)
	if code := c.do("POST", "/api/pile", map[string]any{"title": ""}, nil); code != 400 {
		t.Fatalf("a name is needed: %d", code)
	}
	var rc struct{ ID int64 }
	c.do("POST", "/api/recipes", map[string]any{"title": "Apple pie", "ingredients": []map[string]string{{"line": "6 apples"}}}, &rc)
	if code := c.do("PUT", fmt.Sprintf("/api/pile/%d", card.ID), map[string]any{"title": "Grandma's pie card", "note": "blue tin",
		"done": true, "recipe_id": rc.ID}, nil); code != 204 {
		t.Fatalf("tick: %d", code)
	}
	var pile struct {
		Pile []struct {
			Title    string
			RecipeID *int64 `json:"recipe_id"`
			DoneAt   string `json:"done_at"`
		}
	}
	c.do("GET", "/api/pile", nil, &pile)
	if len(pile.Pile) != 1 || pile.Pile[0].DoneAt == "" || pile.Pile[0].RecipeID == nil || *pile.Pile[0].RecipeID != rc.ID {
		t.Fatalf("done: %+v", pile)
	}
	c.do("PUT", fmt.Sprintf("/api/pile/%d", card.ID), map[string]any{"title": "Grandma's pie card", "done": false}, nil)
	c.do("GET", "/api/pile", nil, &pile)
	if pile.Pile[0].DoneAt != "" || pile.Pile[0].RecipeID != nil {
		t.Fatalf("unticked: %+v", pile)
	}
}
