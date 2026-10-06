package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// A backup from one RecipeBank loads into another: recipes, their photos,
// collections and "our version" links; loading it again adds nothing.
func TestBackupAndRestore(t *testing.T) {
	src, _ := setup(t)
	var photo struct{ Name string }
	src.do("POST", "/api/photos", map[string]string{"image": "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(jpegBytes(40, 30))}, &photo)
	var soup, ours, cleaner struct{ ID int64 }
	src.do("POST", "/api/recipes", map[string]any{"title": "Grandma's soup", "photo": photo.Name, "source_kind": "photo",
		"source_photos": []string{photo.Name}, "ingredients": []map[string]string{{"line": "1 onion"}}}, &soup)
	src.do("POST", fmt.Sprintf("/api/recipes/%d/version", soup.ID), map[string]any{}, &ours)
	src.do("POST", "/api/recipes", map[string]any{"title": "Glass cleaner", "area": "home",
		"ingredients": []map[string]string{{"line": "1 cup vinegar"}}}, &cleaner)
	var col struct{ ID int64 }
	src.do("POST", "/api/collections", map[string]string{"name": "Winter"}, &col)
	src.do("POST", fmt.Sprintf("/api/collections/%d/recipes", col.ID), map[string]any{"ids": []int64{soup.ID}}, nil)

	resp, err := src.http.Get(src.base + "/api/admin/backup")
	if err != nil || resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "application/zip" {
		t.Fatalf("download: %v %v", err, resp)
	}
	zipped, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	dst, _ := setup(t)
	load := func() restoreResult {
		req, _ := http.NewRequest("POST", dst.base+"/api/admin/backup", bytes.NewReader(zipped))
		req.Header.Set("X-RecipeBank", "1")
		req.Header.Set("Content-Type", "application/zip")
		resp, err := dst.http.Do(req)
		if err != nil || resp.StatusCode != 200 {
			t.Fatalf("restore: %v %v", err, resp)
		}
		defer resp.Body.Close()
		var res restoreResult
		_ = json.NewDecoder(resp.Body).Decode(&res)
		return res
	}
	if res := load(); res.Added != 3 || res.Skipped != 0 || res.Photos != 1 || res.Collections != 1 {
		t.Fatalf("first load: %+v", res)
	}
	var kitchen struct {
		Recipes []struct {
			ID    int64
			Title string
			Photo string
		}
	}
	dst.do("GET", "/api/recipes?area=kitchen", nil, &kitchen)
	if len(kitchen.Recipes) != 2 {
		t.Fatalf("kitchen: %+v", kitchen.Recipes)
	}
	for _, r := range kitchen.Recipes {
		var page struct {
			Recipe struct {
				Photo        string   `json:"photo"`
				VersionOf    *int64   `json:"version_of"`
				SourcePhotos []string `json:"source_photos"`
			}
		}
		dst.do("GET", fmt.Sprintf("/api/recipes/%d", r.ID), nil, &page)
		if page.Recipe.Photo == "" {
			t.Fatalf("%s lost its photo", r.Title)
		}
		if code := dst.do("GET", "/api/photos/"+page.Recipe.Photo, nil, nil); code != 200 {
			t.Fatalf("%s's photo isn't there: %d", r.Title, code)
		}
		if r.Title == "Grandma's soup" && len(page.Recipe.SourcePhotos) != 1 {
			t.Fatalf("card photos: %+v", page.Recipe)
		}
		if r.Title != "Grandma's soup" && page.Recipe.VersionOf == nil {
			t.Fatalf("our version lost its link: %+v", page.Recipe)
		}
	}
	var cl struct {
		Collections []struct {
			Name  string
			Count int
		}
	}
	dst.do("GET", "/api/collections", nil, &cl)
	if len(cl.Collections) != 1 || cl.Collections[0].Name != "Winter" || cl.Collections[0].Count != 1 {
		t.Fatalf("collections: %+v", cl)
	}
	if res := load(); res.Added != 0 || res.Skipped != 3 || res.Collections != 0 {
		t.Fatalf("second load: %+v", res)
	}

	req, _ := http.NewRequest("POST", dst.base+"/api/admin/backup", bytes.NewReader([]byte("not a zip")))
	req.Header.Set("X-RecipeBank", "1")
	if resp, _ := dst.http.Do(req); resp == nil || resp.StatusCode != 400 {
		t.Fatalf("not a backup: %v", resp)
	}
}
