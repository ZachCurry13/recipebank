package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// A receipt's lines are matched to pantry and supply items; saving the ticked
// ones adds the amounts, keeps the price for one, and adds the rest as new.
func TestScanReceipt(t *testing.T) {
	c, srv := setup(t)
	ai := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		answer := `{"items":[{"name":"whole milk","qty":2,"size":"1 gal","price":7.0,"area":"kitchen"},
			{"name":"dish soap","qty":1,"price":3.5,"area":"home"},{"name":"bananas","qty":0,"price":-1,"area":"fruit"},{"name":""}]}`
		resp, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": answer}}}})
		w.Write(resp)
	}))
	defer ai.Close()
	useAI(srv.Store, ai.URL)
	var milk struct{ ID int64 }
	c.do("POST", "/api/stock", map[string]any{"area": "kitchen", "name": "Milk", "qty": 1}, &milk)

	var read struct {
		Lines []struct {
			Name  string  `json:"name"`
			Qty   float64 `json:"qty"`
			Price float64 `json:"price"`
			Area  string  `json:"area"`
			Match *struct {
				ID int64 `json:"id"`
			} `json:"match"`
		} `json:"lines"`
	}
	if code := c.do("POST", "/api/stock/receipt", photoBody(600, 1200), &read); code != 200 || len(read.Lines) != 3 {
		t.Fatalf("receipt: %d %+v", code, read)
	}
	if m := read.Lines[0].Match; m == nil || m.ID != milk.ID {
		t.Fatalf("milk should match the pantry's Milk: %+v", read.Lines[0])
	}
	if read.Lines[1].Match != nil || read.Lines[1].Area != "home" {
		t.Fatalf("soap is new, in supplies: %+v", read.Lines[1])
	}
	if b := read.Lines[2]; b.Qty != 1 || b.Price != 0 || b.Area != "kitchen" {
		t.Fatalf("odd values should be made sensible: %+v", b)
	}

	var res struct{ Added, Updated int }
	c.do("POST", "/api/stock/receipt/apply", map[string]any{"lines": []map[string]any{
		{"name": "whole milk", "qty": 2, "price": 7.0, "size": "1 gal", "area": "kitchen", "match_id": milk.ID},
		{"name": "Dish soap", "qty": 1, "price": 3.5, "area": "home"},
		{"name": "", "qty": 1},
	}}, &res)
	if res.Added != 1 || res.Updated != 1 {
		t.Fatalf("apply: %+v", res)
	}
	var kitchen, home []struct {
		Name  string
		Qty   float64
		Price float64
		Size  string
	}
	c.do("GET", "/api/stock?area=kitchen", nil, &kitchen)
	c.do("GET", "/api/stock?area=home", nil, &home)
	if len(kitchen) != 1 || kitchen[0].Qty != 3 || kitchen[0].Price != 3.5 || kitchen[0].Size != "1 gal" {
		t.Fatalf("milk: %+v", kitchen)
	}
	if len(home) != 1 || home[0].Name != "Dish soap" || home[0].Price != 3.5 {
		t.Fatalf("soap: %+v", home)
	}
	if code := c.do("POST", "/api/stock/receipt/apply", map[string]any{"lines": []any{}}, nil); code != 400 {
		t.Fatalf("nothing ticked: %d", code)
	}
}
