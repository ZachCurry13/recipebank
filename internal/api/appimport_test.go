package api

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// Recipes from another app's export are saved like any other: ingredients
// understood, checked for everyone, the photo kept; importing again adds nothing.
func TestImportFromApp(t *testing.T) {
	c, _ := setup(t)
	c.do("POST", "/api/people", map[string]any{"name": "Kid", "heat_max": -1,
		"rules": []map[string]string{{"kind": "allergy", "key": "milk", "severity": "allergic"}}}, nil)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, rec := range []map[string]any{
		{"name": "Shortbread", "ingredients": "2 cups flour\n1 cup butter\n1/2 cup sugar", "directions": "Mix.\nBake 20 minutes.",
			"photo_data": base64.StdEncoding.EncodeToString(jpegBytes(30, 20))},
		{"name": "Fruit salad", "ingredients": "2 apples\n1 cup grapes", "directions": "Chop and toss."},
	} {
		f, _ := zw.Create(rec["name"].(string) + ".paprikarecipe")
		gw := gzip.NewWriter(f)
		json.NewEncoder(gw).Encode(rec)
		gw.Close()
	}
	zw.Close()
	send := func(body []byte) (int, map[string]any) {
		req, _ := http.NewRequest("POST", c.base+"/api/import/app?area=kitchen", bytes.NewReader(body))
		req.Header.Set("X-RecipeBank", "1")
		req.Header.Set("Content-Type", "application/octet-stream")
		resp, err := c.http.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out map[string]any
		json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}
	code, res := send(buf.Bytes())
	if code != 200 || res["app"] != "Paprika" || res["added"] != 2.0 || res["photos"] != 1.0 {
		t.Fatalf("import: %d %v", code, res)
	}
	var lib struct {
		Recipes []struct {
			ID       int64
			Title    string
			Photo    string
			Verdicts []struct{ Status string }
		}
	}
	c.do("GET", "/api/recipes?area=kitchen&who=all", nil, &lib)
	var shortbread int64
	for _, r := range lib.Recipes {
		if r.Title == "Shortbread" {
			shortbread = r.ID
			if r.Photo == "" {
				t.Fatal("the photo wasn't kept")
			}
		}
	}
	var page struct {
		Recipe struct {
			SourceKind  string `json:"source_kind"`
			Ingredients []struct{ Food string }
		}
		Verdicts []struct{ Status string }
	}
	c.do("GET", fmt.Sprintf("/api/recipes/%d", shortbread), nil, &page)
	if page.Recipe.SourceKind != "app" || page.Recipe.Ingredients[1].Food != "butter" {
		t.Fatalf("shortbread: %+v", page.Recipe)
	}
	if len(page.Verdicts) == 0 || page.Verdicts[0].Status != "no" {
		t.Fatalf("butter must be a no for a milk allergy: %+v", page.Verdicts)
	}
	if code, res := send(buf.Bytes()); code != 200 || res["added"] != 0.0 || res["skipped"] != 2.0 {
		t.Fatalf("again: %d %v", code, res)
	}
	if code, _ := send([]byte("not an export")); code != 400 {
		t.Fatalf("junk: %d", code)
	}
}
