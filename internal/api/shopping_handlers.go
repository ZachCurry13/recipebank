package api

import (
	"github.com/zachcurry13/recipebank/internal/budget"
	"net/http"
	"slices"
	"strings"

	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/recipe"
	"github.com/zachcurry13/recipebank/internal/safety"
	"github.com/zachcurry13/recipebank/internal/shopping"
	"github.com/zachcurry13/recipebank/internal/store"
)

// handleShopping sends the list, plus things running low that aren't on it.
func (s *Server) handleShopping(w http.ResponseWriter, r *http.Request) {
	items, err := s.Store.ListShopping()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	stock, err := s.Store.ListStock("")
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	listed := map[int64]bool{}
	for _, it := range items {
		if it.StockID != nil && !it.Checked {
			listed[*it.StockID] = true
		}
	}
	low := []store.StockItem{}
	for _, st := range stock {
		if st.Low() && !listed[st.ID] {
			low = append(low, st)
		}
	}
	costs, total := budget.Shopping(items, stock)
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "low": low, "costs": costs,
		"budget": map[string]any{"cost": total, "weekly": s.Store.SettingFloat(store.KeyBudgetWeekly)}})
}

// handleAddShopping adds one thing ({"text": "2 lb apples"} or {"stock_id": 4}).
func (s *Server) handleAddShopping(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Text    string `json:"text"`
		Area    string `json:"area"`
		StockID int64  `json:"stock_id"`
	}
	if !readJSON(w, r, &body, 4<<10) {
		return
	}
	by := auth.UserFrom(r).Username
	if body.StockID > 0 {
		st, err := s.Store.StockItem(body.StockID)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		if _, err := s.addLow(st, by); err != nil {
			writeStoreErr(w, err)
			return
		}
		s.handleShopping(w, r)
		return
	}
	text := strings.TrimSpace(body.Text)
	if text == "" || len(text) > 200 {
		writeErr(w, http.StatusBadRequest, "type what to buy")
		return
	}
	in := recipe.ParseLine(text)
	it := store.ShopItem{Key: safety.FoodKey(in.Food), Name: in.Food, Unit: in.Unit, Area: body.Area, AddedBy: by}
	if in.Qty != nil {
		it.Qty = *in.Qty
	}
	if _, err := s.Store.AddShopping(it, ""); err != nil {
		writeStoreErr(w, err)
		return
	}
	s.handleShopping(w, r)
}

// addLow puts a running-low pantry or supply item on the list, once.
func (s *Server) addLow(st *store.StockItem, by string) (bool, error) {
	if s.Store.OnListForStock(st.ID) {
		return false, nil
	}
	id := st.ID
	_, err := s.Store.AddShopping(store.ShopItem{Name: st.Name, Area: st.Area, StockID: &id, AddedBy: by}, "")
	return err == nil, err
}

// handleShopRecipe adds a recipe's ingredients ({"id": 3, "factor": 2}),
// leaving off what the pantry or closet already has.
func (s *Server) handleShopRecipe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID     int64   `json:"id"`
		Factor float64 `json:"factor"`
		Only   []int   `json:"only"` // the lines the person ticked (null: all but what the house has)
	}
	if !readJSON(w, r, &body, 8<<10) {
		return
	}
	rc, err := s.Store.Recipe(body.ID)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	added, have, err := s.shopRecipe(rc, body.Factor, auth.UserFrom(r).Username, body.Only)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"added": added, "have": have})
}

// haveItem is a recipe line left off because the house has it.
type haveItem struct {
	Line  string `json:"line"`
	Food  string `json:"food"`
	Stock string `json:"stock"`
	Area  string `json:"area"`
}

// shopRecipe puts a recipe's lines on the list. only (when not nil) are the
// lines the person ticked, added even if the house has them; otherwise every
// line the house doesn't seem to have.
func (s *Server) shopRecipe(rc *recipe.Recipe, factor float64, by string, only []int) (int, []haveItem, error) {
	if factor <= 0 {
		factor = 1
	}
	stock, err := s.Store.ListStock("")
	if err != nil {
		return 0, nil, err
	}
	added, have := 0, []haveItem{}
	for i, in := range rc.Ingredients {
		if only != nil {
			if !slices.Contains(only, i) {
				continue
			}
		} else if st := inStock(stock, in.Food); st != nil {
			have = append(have, haveItem{Line: in.Line, Food: in.Food, Stock: st.Name, Area: st.Area})
			continue
		}
		name := shopping.CleanName(in.Food)
		it := store.ShopItem{Key: safety.FoodKey(name), Name: name, Unit: in.Unit, Area: rc.Area, AddedBy: by}
		if in.Qty != nil {
			it.Qty = *in.Qty * factor
		}
		if _, err := s.Store.AddShopping(it, rc.Title); err != nil {
			return added, have, err
		}
		added++
	}
	return added, have, nil
}

// inStock is the item in the house a recipe line names, if any.
func inStock(stock []store.StockItem, food string) *store.StockItem {
	for i := range stock {
		if stock[i].Qty > 0 && safety.SameFood(food, stock[i].Name) {
			return &stock[i]
		}
	}
	return nil
}

// handleShopCheck says, for each of a recipe's lines, what the house seems
// to have already, so the person can tick what to buy.
func (s *Server) handleShopCheck(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	rc, err := s.Store.Recipe(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	stock, err := s.Store.ListStock("")
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	out := make([]*haveItem, len(rc.Ingredients))
	for i, in := range rc.Ingredients {
		if st := inStock(stock, in.Food); st != nil {
			out[i] = &haveItem{Line: in.Line, Food: in.Food, Stock: st.Name, Area: st.Area}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"have": out})
}

func (s *Server) handleUpdateShopping(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	var c store.ShopChange
	if !ok || !readJSON(w, r, &c, 4<<10) {
		return
	}
	if err := s.Store.UpdateShopping(id, c); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleDeleteShopping(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := s.Store.DeleteShopping(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleClearShopping removes ticked lines; running-low items bought go
// back into the pantry or closet (one more each).
func (s *Server) handleClearShopping(w http.ResponseWriter, r *http.Request) {
	done, err := s.Store.ClearChecked()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	restocked := []string{}
	for _, it := range done {
		if it.StockID == nil {
			continue
		}
		if st, err := s.Store.AdjustStock(*it.StockID, max(1, it.Qty)); err == nil {
			restocked = append(restocked, st.Name)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"cleared": len(done), "restocked": restocked})
}
