package api

import (
	"io"
	"net/http"
	"os"

	"github.com/zachcurry13/recipebank/internal/apps"
	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/recipe"
)

// maxAppExport is the largest export from another app accepted (photos included).
const maxAppExport = 1 << 30

// handleImportApp reads an export from another recipe app (the request
// body) and saves its recipes, skipping ones already here. They're checked
// for everyone like any other recipe when they're opened.
func (s *Server) handleImportApp(w http.ResponseWriter, r *http.Request) {
	area := recipe.AreaKitchen
	if r.URL.Query().Get("area") == recipe.AreaHome {
		area = recipe.AreaHome
	}
	tmp, err := os.CreateTemp(s.CacheDir, "import-*")
	if err != nil {
		tmp, err = os.CreateTemp("", "import-*")
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "couldn't make room for the file")
		return
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	if _, err := io.Copy(tmp, http.MaxBytesReader(w, r.Body, maxAppExport)); err != nil {
		writeErr(w, http.StatusBadRequest, "the file couldn't be read (up to 1 GB)")
		return
	}
	tmp.Close()
	app, found, err := apps.Read(tmp.Name())
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	existing, err := s.Store.ListRecipes("")
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	have := map[string]bool{}
	for i := range existing {
		have[recipeKey(&existing[i])] = true
	}
	by := auth.UserFrom(r).Username
	res := struct {
		App     string `json:"app"`
		Added   int    `json:"added"`
		Skipped int    `json:"skipped"`
		Photos  int    `json:"photos"`
	}{App: app}
	for _, f := range found {
		rc := f.Recipe
		rc.Area, rc.SourceKind, rc.CreatedBy = area, "app", by
		rc.Clean()
		if have[recipeKey(&rc)] {
			res.Skipped++
			continue
		}
		if len(f.Photo) > 0 {
			if name, err := s.savePhoto(f.Photo); err == nil {
				rc.Photo = name
				res.Photos++
			}
		}
		if _, err := s.Store.SaveRecipe(&rc); err != nil {
			writeStoreErr(w, err)
			return
		}
		have[recipeKey(&rc)] = true
		res.Added++
	}
	writeJSON(w, http.StatusOK, res)
}
