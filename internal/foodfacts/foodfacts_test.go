package foodfacts

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fakeOFF(t *testing.T, known map[string]string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("User-Agent"), "RecipeBank") {
			t.Errorf("no RecipeBank user agent: %q", r.Header.Get("User-Agent"))
		}
		for code, body := range known {
			if strings.Contains(r.URL.Path, code) {
				w.Write([]byte(body))
				return
			}
		}
		w.Write([]byte(`{"status":0,"status_verbose":"product not found"}`))
	}))
}

func TestLookup(t *testing.T) {
	food := fakeOFF(t, map[string]string{"0051000012616": `{"status":1,"product":{"product_name":"Chicken Broth",
		"brands":"Brand A, Brand B","quantity":"32 oz","allergens_tags":["en:celery","en:gluten"],"traces_tags":["en:milk"]}}`})
	defer food.Close()
	empty := fakeOFF(t, nil)
	defer empty.Close()

	c := Client{Bases: []string{empty.URL, food.URL}}
	p, err := c.Lookup(context.Background(), "0051000012616")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Chicken Broth" || p.Brand != "Brand A" || p.Quantity != "32 oz" {
		t.Errorf("product: %+v", p)
	}
	if strings.Join(p.Allergens, ",") != "celery,gluten,wheat" || strings.Join(p.Traces, ",") != "milk" {
		t.Errorf("allergens %v traces %v", p.Allergens, p.Traces)
	}
	if _, err := c.Lookup(context.Background(), "4006381333931"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown barcode: %v", err)
	}
	if _, err := c.Lookup(context.Background(), "12ab"); err == nil {
		t.Error("accepted a bad barcode")
	}
}
