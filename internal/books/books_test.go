package books

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCleanISBN(t *testing.T) {
	cases := map[string]string{
		"978-0-7432-4626-2": "9780743246262",
		"9780743246262":     "9780743246262",
		"0743246268":        "0743246268",
		"0-8044-2957-X":     "080442957X",
		"9780743246261":     "", // wrong check digit
		"4006381333931":     "", // a product barcode, not a book
		"12345":             "",
	}
	for in, want := range cases {
		if got := CleanISBN(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}

func TestLookup(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "ISBN%3A9780743246262") || !strings.HasPrefix(r.Header.Get("User-Agent"), "RecipeBank") {
			w.Write([]byte(`{}`))
			return
		}
		w.Write([]byte(`{"ISBN:9780743246262": {"title": "Joy of Cooking", "subtitle": "75th Anniversary Edition",
			"authors": [{"name": "Irma S. Rombauer"}, {"name": "Marion Rombauer Becker"}],
			"cover": {"medium": "` + srv.URL + `/b/id/1-M.jpg"}}}`))
	}))
	defer srv.Close()
	c := Client{Base: srv.URL}
	b, err := c.Lookup(context.Background(), "978-0-7432-4626-2")
	if err != nil {
		t.Fatal(err)
	}
	if b.Title != "Joy of Cooking: 75th Anniversary Edition" || b.Author != "Irma S. Rombauer, Marion Rombauer Becker" ||
		b.CoverURL != srv.URL+"/b/id/1-M.jpg" {
		t.Fatalf("%+v", b)
	}
	if _, err := c.Lookup(context.Background(), "0743246268"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown: %v", err)
	}
	if _, err := c.Lookup(context.Background(), "4006381333931"); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("not an ISBN: %v", err)
	}
	if coverFrom("https://openlibrary.org", "https://evil.example/x.jpg") != "" ||
		coverFrom("https://openlibrary.org", "https://covers.openlibrary.org/b/id/1-M.jpg") == "" {
		t.Fatal("covers only from Open Library")
	}
}
