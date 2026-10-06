package api

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/recipe"
	"github.com/zachcurry13/recipebank/internal/store"
	"github.com/zachcurry13/recipebank/internal/version"
)

// A backup is one .zip: manifest.json, recipes.json (every recipe, its id
// as written in this file), collections.json (by those ids) and photos/.
// Loading it into any RecipeBank adds what that one doesn't have yet.

type backupManifest struct {
	App      string `json:"app"`
	Version  string `json:"version"`
	Exported string `json:"exported"`
	Recipes  int    `json:"recipes"`
}

type backupCollection struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Icon        string  `json:"icon"`
	Area        string  `json:"area"`
	Recipes     []int64 `json:"recipes"`
}

// maxBackup is the largest backup file accepted (photos included).
const maxBackup = 2 << 30

func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request) {
	all, err := s.Store.ListRecipes("")
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	cols, err := s.Store.ListCollections()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	var outCols []backupCollection
	for _, c := range cols {
		ids, err := s.Store.CollectionRecipeIDs(c.ID)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		outCols = append(outCols, backupCollection{c.Name, c.Description, c.Icon, c.Area, ids})
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="recipebank-backup-%s.zip"`, time.Now().Format(day)))
	zw := zip.NewWriter(w)
	defer zw.Close()
	put := func(name string, v any) {
		if f, err := zw.Create(name); err == nil {
			_ = json.NewEncoder(f).Encode(v)
		}
	}
	put("manifest.json", backupManifest{"RecipeBank", version.Version, time.Now().UTC().Format(time.RFC3339), len(all)})
	put("recipes.json", all)
	put("collections.json", outCols)
	seen := map[string]bool{}
	for _, rc := range all {
		for _, name := range append([]string{rc.Photo}, rc.SourcePhotos...) {
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			if p, ok := s.photoPath(name); ok {
				if data, err := os.ReadFile(p); err == nil {
					if f, err := zw.Create("photos/" + name); err == nil {
						_, _ = f.Write(data)
					}
				}
			}
		}
	}
}

// handleRestore loads a backup .zip (the request body). Recipes already here
// (same title and ingredient lines) are skipped; photos are checked like
// any upload; collections are joined by name.
func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	tmp, err := os.CreateTemp(s.CacheDir, "restore-*.zip")
	if err != nil {
		tmp, err = os.CreateTemp("", "restore-*.zip")
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "couldn't make room for the backup")
		return
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	if _, err := io.Copy(tmp, http.MaxBytesReader(w, r.Body, maxBackup)); err != nil {
		writeErr(w, http.StatusBadRequest, "the backup couldn't be read (up to 2 GB)")
		return
	}
	res, err := s.restore(tmp.Name(), auth.UserFrom(r).Username)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

type restoreResult struct {
	Added       int `json:"added"`
	Skipped     int `json:"skipped"`
	Photos      int `json:"photos"`
	Collections int `json:"collections"`
}

func (s *Server) restore(path, by string) (restoreResult, error) {
	var res restoreResult
	zr, err := zip.OpenReader(path)
	if err != nil {
		return res, errors.New("that isn't a RecipeBank backup (.zip)")
	}
	defer zr.Close()
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	var man backupManifest
	var recipes []recipe.Recipe
	var cols []backupCollection
	if readZipJSON(files["manifest.json"], 1<<20, &man) != nil || man.App != "RecipeBank" ||
		readZipJSON(files["recipes.json"], 512<<20, &recipes) != nil {
		return res, errors.New("that isn't a RecipeBank backup")
	}
	_ = readZipJSON(files["collections.json"], 16<<20, &cols)

	existing, err := s.Store.ListRecipes("")
	if err != nil {
		return res, err
	}
	have := map[string]int64{}
	for _, rc := range existing {
		have[recipeKey(&rc)] = rc.ID
	}
	newID := map[int64]int64{}
	savedAs := map[string]string{} // each photo in the backup is saved once
	photo := func(name string) string {
		if n, ok := savedAs[name]; ok {
			return n
		}
		f := files["photos/"+name]
		if name == "" || f == nil || f.UncompressedSize64 > maxPhoto {
			return ""
		}
		rd, err := f.Open()
		if err != nil {
			return ""
		}
		defer rd.Close()
		data, err := io.ReadAll(io.LimitReader(rd, maxPhoto+1))
		if err != nil {
			return ""
		}
		saved, err := s.savePhoto(data)
		if err != nil {
			return ""
		}
		savedAs[name] = saved
		res.Photos++
		return saved
	}
	for i := range recipes {
		rc := recipes[i]
		old := rc.ID
		if id, ok := have[recipeKey(&rc)]; ok {
			newID[old] = id
			res.Skipped++
			continue
		}
		rc.ID = 0
		rc.Photo = photo(rc.Photo)
		var kept []string
		for _, p := range rc.SourcePhotos {
			if n := photo(p); n != "" {
				kept = append(kept, n)
			}
		}
		rc.SourcePhotos = kept
		if rc.VersionOf != nil {
			if id, ok := newID[*rc.VersionOf]; ok {
				rc.VersionOf = &id
			} else {
				rc.VersionOf = nil
			}
		}
		if rc.CreatedBy == "" {
			rc.CreatedBy = by
		}
		id, err := s.Store.SaveRecipe(&rc)
		if err != nil {
			return res, err
		}
		newID[old] = id
		have[recipeKey(&rc)] = id
		res.Added++
	}

	current, err := s.Store.ListCollections()
	if err != nil {
		return res, err
	}
	for _, c := range cols {
		var colID int64
		for _, cc := range current {
			if strings.EqualFold(cc.Name, c.Name) && cc.Area == c.Area {
				colID = cc.ID
			}
		}
		if colID == 0 {
			nc := store.Collection{Name: c.Name, Description: c.Description, Icon: c.Icon, Area: c.Area, CreatedBy: by}
			if colID, err = s.Store.SaveCollection(&nc); err != nil {
				return res, err
			}
			res.Collections++
		}
		var ids []int64
		for _, old := range c.Recipes {
			if id, ok := newID[old]; ok {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			if err := s.Store.AddToCollection(colID, ids); err != nil {
				return res, err
			}
		}
	}
	if res.Photos > 0 || res.Added > 0 {
		s.cleanPhotos()
	}
	return res, nil
}

// recipeKey says when two recipes are the same: title and ingredient lines.
func recipeKey(rc *recipe.Recipe) string {
	var b strings.Builder
	b.WriteString(strings.ToLower(strings.TrimSpace(rc.Title)))
	for _, in := range rc.Ingredients {
		b.WriteString("|" + strings.ToLower(strings.TrimSpace(in.Line)))
	}
	return b.String()
}

func readZipJSON(f *zip.File, limit int64, into any) error {
	if f == nil {
		return errors.New("missing")
	}
	rd, err := f.Open()
	if err != nil {
		return err
	}
	defer rd.Close()
	return json.NewDecoder(io.LimitReader(rd, limit)).Decode(into)
}
