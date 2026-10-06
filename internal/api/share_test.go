package api

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

// The phone's share menu opens "Add a recipe" with the link (or text) filled in.
func TestShareTarget(t *testing.T) {
	c, _ := setup(t)
	noFollow := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	where := func(q url.Values) string {
		resp, err := noFollow.Get(c.base + "/share?" + q.Encode())
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.Header.Get("Location")
	}
	if got := where(url.Values{"text": {"Look at this https://example.com/r/12?x=1."}}); got != "/#/add?way=link&url="+url.QueryEscape("https://example.com/r/12?x=1") {
		t.Fatalf("a link in the text: %s", got)
	}
	if got := where(url.Values{"url": {"https://example.com/a"}, "text": {"ignored"}}); got != "/#/add?way=link&url="+url.QueryEscape("https://example.com/a") {
		t.Fatalf("a link: %s", got)
	}
	if got := where(url.Values{"title": {"Soup"}, "text": {"Ingredients: water"}}); got != "/#/add?way=text&text="+url.QueryEscape("Soup\nIngredients: water") {
		t.Fatalf("text: %s", got)
	}
	if got := where(url.Values{}); got != "/#/add" {
		t.Fatalf("nothing: %s", got)
	}
}

// The cookbook is every Kitchen recipe by course and title, or a collection.
func TestCookbook(t *testing.T) {
	c, _ := setup(t)
	add := func(title, course string) int64 {
		var rc struct{ ID int64 }
		c.do("POST", "/api/recipes", map[string]any{"title": title, "course": course, "ingredients": []map[string]string{{"line": "1 cup rice"}}}, &rc)
		return rc.ID
	}
	pie, soup, bread := add("Pie", "dessert"), add("Soup", "main"), add("Bread", "dessert")
	c.do("POST", "/api/recipes", map[string]any{"title": "Cleaner", "area": "home"}, nil)
	var book struct {
		Title   string
		Recipes []struct{ ID int64 }
	}
	c.do("GET", "/api/cookbook", nil, &book)
	if len(book.Recipes) != 3 || book.Recipes[0].ID != bread || book.Recipes[1].ID != pie || book.Recipes[2].ID != soup {
		t.Fatalf("all: %+v", book)
	}
	var col struct{ ID int64 }
	c.do("POST", "/api/collections", map[string]string{"name": "Holidays"}, &col)
	c.do("POST", fmt.Sprintf("/api/collections/%d/recipes", col.ID), map[string]any{"ids": []int64{soup, pie}}, nil)
	c.do("GET", fmt.Sprintf("/api/cookbook?collection=%d", col.ID), nil, &book)
	if book.Title != "Holidays" || len(book.Recipes) != 2 {
		t.Fatalf("collection: %+v", book)
	}
}
