// Package books looks cookbooks up by ISBN in Open Library (openlibrary.org),
// a free, open catalog: title, authors and a cover.
package books

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// DefaultBase is Open Library's address; tests point the client at a fake.
const DefaultBase = "https://openlibrary.org"

const userAgent = "RecipeBank (+https://github.com/ZachCurry13/recipebank)"

// ErrNotFound means Open Library doesn't know the ISBN.
var ErrNotFound = errors.New("not found")

// Book is what Open Library knows about a book.
type Book struct {
	ISBN     string `json:"isbn"`
	Title    string `json:"title"`
	Author   string `json:"author"`
	CoverURL string `json:"-"` // a covers.openlibrary.org address, fetched by the server only
}

// Client looks books up.
type Client struct {
	HTTP *http.Client
	Base string
}

// CleanISBN strips spaces and dashes; "" when it isn't a valid ISBN-10 or
// ISBN-13 (the barcode on a book is its ISBN-13, starting 978 or 979).
func CleanISBN(s string) string {
	s = strings.ToUpper(strings.NewReplacer("-", "", " ", "").Replace(strings.TrimSpace(s)))
	switch len(s) {
	case 13:
		if !strings.HasPrefix(s, "978") && !strings.HasPrefix(s, "979") {
			return ""
		}
		sum := 0
		for i, c := range s {
			if c < '0' || c > '9' {
				return ""
			}
			d := int(c - '0')
			if i%2 == 1 {
				d *= 3
			}
			sum += d
		}
		if sum%10 == 0 {
			return s
		}
	case 10:
		sum := 0
		for i, c := range s {
			d := int(c - '0')
			switch {
			case c == 'X' && i == 9:
				d = 10
			case c < '0' || c > '9':
				return ""
			}
			sum += d * (10 - i)
		}
		if sum%11 == 0 {
			return s
		}
	}
	return ""
}

// Lookup finds a book by ISBN.
func (c Client) Lookup(ctx context.Context, isbn string) (Book, error) {
	isbn = CleanISBN(isbn)
	if isbn == "" {
		return Book{}, errors.New("that isn't a book's ISBN (the barcode on the back starts with 978 or 979)")
	}
	base := strings.TrimRight(c.Base, "/")
	if base == "" {
		base = DefaultBase
	}
	key := "ISBN:" + isbn
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		base+"/api/books?format=json&jscmd=data&bibkeys="+url.QueryEscape(key), nil)
	if err != nil {
		return Book{}, err
	}
	req.Header.Set("User-Agent", userAgent)
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return Book{}, fmt.Errorf("couldn't reach Open Library: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Book{}, fmt.Errorf("Open Library answered %s", resp.Status)
	}
	var body map[string]struct {
		Title    string `json:"title"`
		Subtitle string `json:"subtitle"`
		Authors  []struct {
			Name string `json:"name"`
		} `json:"authors"`
		Cover struct {
			Medium string `json:"medium"`
		} `json:"cover"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return Book{}, errors.New("Open Library's answer couldn't be read")
	}
	b, ok := body[key]
	if !ok || strings.TrimSpace(b.Title) == "" {
		return Book{}, ErrNotFound
	}
	var authors []string
	for _, a := range b.Authors {
		if n := strings.TrimSpace(a.Name); n != "" && len(authors) < 3 {
			authors = append(authors, n)
		}
	}
	title := strings.TrimSpace(b.Title)
	if sub := strings.TrimSpace(b.Subtitle); sub != "" && len(title)+len(sub) < 120 {
		title += ": " + sub
	}
	return Book{ISBN: isbn, Title: title, Author: strings.Join(authors, ", "), CoverURL: coverFrom(base, b.Cover.Medium)}, nil
}

// coverFrom keeps a cover address only when it's Open Library's own.
func coverFrom(base, cover string) string {
	u, err := url.Parse(cover)
	b, berr := url.Parse(base)
	if cover == "" || err != nil || berr != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return ""
	}
	site := func(h string) string {
		parts := strings.Split(h, ".")
		if len(parts) > 2 {
			parts = parts[len(parts)-2:]
		}
		return strings.Join(parts, ".")
	}
	if site(u.Hostname()) != site(b.Hostname()) {
		return ""
	}
	return u.String()
}
