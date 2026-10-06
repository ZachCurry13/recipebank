package api

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zachcurry13/recipebank/internal/store"
)

func TestStockAndPantryHints(t *testing.T) {
	c, srv := setup(t)
	var off *httptest.Server
	off = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/front.jpg" {
			w.Write(jpegBytes(40, 60))
			return
		}
		if strings.Contains(r.URL.Path, "0051000012616") {
			w.Write([]byte(`{"status":1,"product":{"product_name":"Chicken Broth","brands":"Brand A","allergens_tags":["en:celery"],"traces_tags":[],
				"image_front_small_url":"` + off.URL + `/front.jpg"}}`))
			return
		}
		if strings.Contains(r.URL.Path, "0012000001536") {
			w.Write([]byte(`{"status":1,"product":{"product_name":"Cola","image_front_small_url":"https://elsewhere.example/x.jpg"}}`))
			return
		}
		w.Write([]byte(`{"status":0}`))
	}))
	defer off.Close()
	srv.Products = productBases{FoodBases: []string{off.URL}, HomeBases: []string{off.URL}}

	// A barcode the databases know fills in the label.
	var found struct {
		Product struct {
			Name      string   `json:"name"`
			Allergens []string `json:"allergens"`
			Photo     string   `json:"photo"`
		} `json:"product"`
	}
	if code := c.do("GET", "/api/stock/lookup?barcode=0051000012616&area=kitchen", nil, &found); code != 200 || found.Product.Name != "Chicken Broth" {
		t.Fatalf("lookup: %d %+v", code, found)
	}
	// The package photo comes along so the person can see it's the right thing.
	if !strings.HasPrefix(found.Product.Photo, "data:image/jpeg;base64,") {
		t.Fatalf("no package photo: %.40q", found.Product.Photo)
	}
	var cola struct {
		Product struct {
			Name  string `json:"name"`
			Photo string `json:"photo"`
		} `json:"product"`
	}
	if code := c.do("GET", "/api/stock/lookup?barcode=0012000001536&area=kitchen", nil, &cola); code != 200 || cola.Product.Name != "Cola" || cola.Product.Photo != "" {
		t.Fatalf("a photo from a site that isn't the database's: %d %+v", code, cola)
	}
	if code := c.do("GET", "/api/stock/lookup?barcode=4006381333931&area=kitchen", nil, nil); code != 404 {
		t.Fatalf("unknown barcode: %d", code)
	}
	var item store.StockItem
	c.do("POST", "/api/stock", map[string]any{"area": "kitchen", "name": "Brand A Chicken Broth", "qty": 2, "barcode": "0051000012616",
		"allergens": found.Product.Allergens, "label_source": "off", "low_at": 1}, &item)
	if item.ID == 0 || item.Qty != 2 {
		t.Fatalf("saved: %+v", item)
	}
	var again struct {
		Existing *store.StockItem `json:"existing"`
	}
	c.do("GET", "/api/stock/lookup?barcode=0051000012616&area=kitchen", nil, &again)
	if again.Existing == nil || again.Existing.ID != item.ID {
		t.Fatalf("scanning an item in the house should find it: %+v", again)
	}

	// A recipe with broth, for someone with a severe milk allergy: the
	// pantry's labelled broth is offered to settle the "not sure".
	c.do("POST", "/api/people", map[string]any{"name": "Kid", "heat_max": -1,
		"rules": []map[string]string{{"kind": "allergy", "key": "milk", "severity": "severe"}}}, nil)
	var saved struct{ ID int64 }
	c.do("POST", "/api/recipes", map[string]any{"title": "Soup", "ingredients": []map[string]string{{"line": "4 cups chicken broth"}, {"line": "1 onion"}}}, &saved)
	var page struct {
		Pantry []pantryHint `json:"pantry"`
	}
	c.do("GET", "/api/recipes/"+itoa(saved.ID), nil, &page)
	if len(page.Pantry) != 1 || page.Pantry[0].Ingredient != 0 || page.Pantry[0].StockID != item.ID {
		t.Fatalf("pantry hints: %+v", page.Pantry)
	}

	// Kids can use things up, not add them.
	c.do("POST", "/api/admin/users", map[string]string{"username": "kid", "password": "password123", "role": "kid"}, nil)
	jar, _ := cookiejar.New(nil)
	kid := &client{t: t, base: c.base, http: &http.Client{Jar: jar}}
	kid.do("POST", "/api/auth/login", map[string]string{"username": "kid", "password": "password123"}, nil)
	var used store.StockItem
	if code := kid.do("POST", "/api/stock/"+itoa(item.ID)+"/adjust", map[string]float64{"delta": -1}, &used); code != 200 || used.Qty != 1 || !used.Low() {
		t.Fatalf("kid used one: %d %+v", code, used)
	}
	if code := kid.do("POST", "/api/stock", map[string]any{"name": "x"}, nil); code != 403 {
		t.Fatalf("kid added stock: %d", code)
	}
}
