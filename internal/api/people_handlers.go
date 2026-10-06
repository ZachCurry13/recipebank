package api

import (
	"net/http"
	"strings"

	"github.com/zachcurry13/recipebank/internal/safety"
	"github.com/zachcurry13/recipebank/internal/store"
)

// handleInfo sends the lists the web page builds its forms from.
func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	list := s.Store.Setting(store.KeyAllergenList)
	writeJSON(w, http.StatusOK, map[string]any{
		"allergens":     safety.AllergenList(list),
		"all_allergens": safety.Allergens,
		"allergen_list": list,
		"diets":         safety.Diets,
		"pets":          s.Store.Pets(),
		"ai_ready":      s.Store.AIConfig().Ready(),
		"email_ready":   s.emailReady(),
		"email_who":     s.Store.Setting(store.KeyEmailWho),
		"currency":      s.Store.Setting(store.KeyCurrency),
		"default_units": s.Store.Setting(store.KeyDefaultUnits),
	})
}

func (s *Server) handleListPeople(w http.ResponseWriter, r *http.Request) {
	ps, err := s.Store.ListPeople()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ps)
}

func (s *Server) handleSavePerson(w http.ResponseWriter, r *http.Request) {
	var p store.Person
	if !readJSON(w, r, &p, 32<<10) {
		return
	}
	p.ID = 0
	if id, ok := pathID(r, "id"); ok {
		p.ID = id
	}
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" || len(p.Name) > 60 {
		writeErr(w, http.StatusBadRequest, "a name (up to 60 letters) is needed")
		return
	}
	for _, rule := range p.Rules {
		if store.ValidRule(rule) != nil {
			writeErr(w, http.StatusBadRequest, "unknown rule: "+rule.Kind+" "+rule.Key)
			return
		}
	}
	id, err := s.Store.SavePerson(&p)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	saved, err := s.Store.PersonByID(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) handleDeletePerson(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := s.Store.DeletePerson(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

var knownPets = map[string]bool{"dog": true, "cat": true, "bird": true, "small": true, "fish": true}

// handleHousehold saves the household's pets.
func (s *Server) handleHousehold(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Pets []string `json:"pets"`
	}
	if !readJSON(w, r, &body, 4<<10) {
		return
	}
	var keep []string
	for _, p := range body.Pets {
		if knownPets[p] {
			keep = append(keep, p)
		}
	}
	if err := s.Store.SetSetting(store.KeyHouseholdPets, strings.Join(keep, ",")); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"pets": s.Store.Pets()})
}

// diners returns the people a check is for: the ids in ?who=1,2 or, when
// none are given, everyone who isn't a guest.
func (s *Server) diners(r *http.Request) ([]safety.Person, error) {
	ps, err := s.Store.ListPeople()
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, id := range store.SplitList(r.URL.Query().Get("who")) {
		want[id] = true
	}
	out := []safety.Person{}
	for i := range ps {
		p := &ps[i]
		if (len(want) == 0 && !p.IsGuest) || want[itoa(p.ID)] {
			out = append(out, p.Checkable())
		}
	}
	return out, nil
}
