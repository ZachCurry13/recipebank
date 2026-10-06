// Package apps reads recipes exported from other recipe apps: Paprika
// (.paprikarecipes), Tandoor (its export .zip) and Mealie (recipe .json
// files or a backup .zip), and recipe JSON in the schema.org format many
// apps and sites use. Each becomes a RecipeBank recipe, with its photo.
package apps

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"strings"

	"github.com/zachcurry13/recipebank/internal/recipe"
)

// Found is one recipe read from an export, and its photo when it had one.
type Found struct {
	Recipe recipe.Recipe
	Photo  []byte
}

// Limits keep a strange file from using up memory.
const (
	maxEntries = 20000
	maxJSON    = 32 << 20
	maxPhoto   = 8 << 20
	maxRecipes = 5000
)

// ErrUnknown means the file isn't an export RecipeBank knows.
var ErrUnknown = errors.New("That file isn't an export RecipeBank can read: use Paprika's .paprikarecipes, Tandoor's or Mealie's export .zip, or recipe .json")

// Read reads an export file; app says which app it came from.
func Read(file string) (app string, found []Found, err error) {
	f, err := os.Open(file)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()
	head := make([]byte, 4)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	switch {
	case bytes.HasPrefix(head, []byte("PK")):
		f.Close()
		zr, err := zip.OpenReader(file)
		if err != nil {
			return "", nil, ErrUnknown
		}
		defer zr.Close()
		return readZip(&zr.Reader)
	case bytes.HasPrefix(head, []byte{0x1f, 0x8b}): // one Paprika recipe
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return "", nil, err
		}
		r, err := paprika(f)
		if err != nil {
			return "", nil, err
		}
		return "Paprika", []Found{r}, nil
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", nil, err
	}
	data, err := io.ReadAll(io.LimitReader(f, maxJSON))
	if err != nil {
		return "", nil, err
	}
	found, err = fromJSONFile(data)
	if err != nil {
		return "", nil, err
	}
	return "Recipe JSON", found, nil
}

func readZip(zr *zip.Reader) (string, []Found, error) {
	if len(zr.File) > maxEntries {
		return "", nil, errors.New("that file has too many parts to be a recipe export")
	}
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	var app string
	var found []Found
	add := func(r Found, err error) {
		if err == nil && len(found) < maxRecipes {
			found = append(found, r)
		}
	}
	for _, f := range zr.File {
		name := strings.ToLower(f.Name)
		switch {
		case strings.HasSuffix(name, ".paprikarecipe"):
			app = "Paprika"
			rd, err := f.Open()
			if err != nil {
				continue
			}
			r, err := paprika(rd)
			rd.Close()
			add(r, err)
		case strings.HasSuffix(name, ".zip"): // Tandoor: a zip of recipe zips
			data, err := readAll(f, maxJSON+maxPhoto)
			if err != nil {
				continue
			}
			inner, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				continue
			}
			r, err := tandoorZip(inner)
			if err == nil {
				app = "Tandoor"
			}
			add(r, err)
		case path.Base(name) == "recipe.json" && len(zr.File) <= 3: // one Tandoor recipe
			r, err := tandoorZip(zr)
			if err == nil {
				app = "Tandoor"
			}
			add(r, err)
		case path.Base(name) == "database.json":
			data, err := readAll(f, 256<<20)
			if err != nil {
				continue
			}
			list, err := mealieBackup(data, files)
			if err == nil {
				app = "Mealie"
				for _, r := range list {
					add(r, nil)
				}
			}
		case strings.HasSuffix(name, ".json"):
			data, err := readAll(f, maxJSON)
			if err != nil {
				continue
			}
			list, err := fromJSONFile(data)
			if err != nil {
				continue
			}
			if app == "" {
				app = "Mealie"
			}
			for _, r := range list {
				if r.Photo == nil {
					r.Photo = photoNear(files, f.Name)
				}
				add(r, nil)
			}
		}
	}
	if len(found) == 0 {
		return "", nil, ErrUnknown
	}
	return app, found, nil
}

// photoNear finds a Mealie recipe's photo: images/original.* next to its JSON.
func photoNear(files map[string]*zip.File, jsonName string) []byte {
	dir := path.Dir(jsonName)
	for _, name := range []string{"images/original.webp", "images/original.jpg", "images/original.png", "images/min-original.webp"} {
		if f := files[path.Join(dir, name)]; f != nil {
			if data, err := readAll(f, maxPhoto); err == nil {
				return data
			}
		}
	}
	return nil
}

// readAll reads one part of a zip, refusing anything bigger than limit.
func readAll(f *zip.File, limit int64) ([]byte, error) {
	if f.UncompressedSize64 > uint64(limit) {
		return nil, errors.New("too big")
	}
	rd, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rd.Close()
	data, err := io.ReadAll(io.LimitReader(rd, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, errors.New("too big")
	}
	return data, nil
}

// fromJSONFile reads a recipe JSON file: one recipe, a list, or {"recipes": [...]}.
func fromJSONFile(data []byte) ([]Found, error) {
	var v any
	if err := json.Unmarshal(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")), &v); err != nil {
		return nil, ErrUnknown
	}
	var items []any
	switch t := v.(type) {
	case []any:
		items = t
	case map[string]any:
		if list, ok := t["recipes"].([]any); ok {
			items = list
		} else if g, ok := t["@graph"].([]any); ok {
			items = g
		} else {
			items = []any{t}
		}
	}
	var out []Found
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		if r, ok := genericRecipe(m); ok {
			out = append(out, Found{Recipe: r})
		}
	}
	if len(out) == 0 {
		return nil, ErrUnknown
	}
	return out, nil
}

// paprika reads one gzip-compressed Paprika recipe.
func paprika(rd io.Reader) (Found, error) {
	gz, err := gzip.NewReader(rd)
	if err != nil {
		return Found{}, err
	}
	defer gz.Close()
	data, err := io.ReadAll(io.LimitReader(gz, maxJSON))
	if err != nil {
		return Found{}, err
	}
	return paprikaRecipe(data)
}
