package api

import (
	"net/http"
	"strings"

	"github.com/zachcurry13/recipebank/internal/llm"
	"github.com/zachcurry13/recipebank/internal/safety"
	"github.com/zachcurry13/recipebank/internal/store"
)

// Scanning a receipt: the AI reads what was bought, each line is matched to
// a pantry or supply item, the person ticks what's right, and saving adds
// the amounts and remembers the prices (for the budget).

type receiptLine struct {
	llm.ReceiptItem
	Match *receiptMatch `json:"match"` // the item it adds to, or nil for a new one
}

type receiptMatch struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func (s *Server) handleReceipt(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Images []string `json:"images"`
	}
	if !readJSON(w, r, &body, 30<<20) {
		return
	}
	images, ok := readImages(w, body.Images, 3)
	if !ok {
		return
	}
	var items []llm.ReceiptItem
	if _, err := s.askPhotos(r.Context(), llm.ReceiptPrompt, images, func(out string) (perr error) {
		items, perr = llm.ParseReceipt(out)
		return perr
	}); err != nil {
		writeErr(w, http.StatusBadGateway, importErr(err))
		return
	}
	stock, err := s.Store.ListStock("")
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	lines := []receiptLine{}
	for _, it := range items {
		lines = append(lines, receiptLine{ReceiptItem: it, Match: matchStock(stock, it.Area, it.Name)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"lines": lines})
}

// matchStock finds the pantry or supply item a receipt line is.
func matchStock(stock []store.StockItem, area, name string) *receiptMatch {
	for _, it := range stock {
		if it.Area == area && (safety.SameFood(name, it.Name) || safety.SameFood(it.Name, name) || safety.Covers(it.Name, name)) {
			return &receiptMatch{it.ID, it.Name}
		}
	}
	return nil
}

// handleReceiptApply saves the ticked lines: matched items get the amount
// added and the price (for one) remembered; the rest become new items.
func (s *Server) handleReceiptApply(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Lines []struct {
			Name    string  `json:"name"`
			Brand   string  `json:"brand"`
			Qty     float64 `json:"qty"`
			Size    string  `json:"size"`
			Price   float64 `json:"price"` // for the whole line
			Area    string  `json:"area"`
			MatchID int64   `json:"match_id"`
		} `json:"lines"`
	}
	if !readJSON(w, r, &body, 256<<10) {
		return
	}
	if len(body.Lines) == 0 || len(body.Lines) > 100 {
		writeErr(w, http.StatusBadRequest, "tick 1 to 100 lines")
		return
	}
	added, updated := 0, 0
	for _, l := range body.Lines {
		name := strings.TrimSpace(l.Name)
		if name == "" || len(name) > 120 || l.Qty <= 0 || l.Qty > 100 || l.Price < 0 || l.Price > 10000 {
			continue
		}
		each := 0.0
		if l.Price > 0 {
			each = l.Price / l.Qty
		}
		if l.MatchID > 0 {
			it, err := s.Store.StockItem(l.MatchID)
			if err == nil {
				it.Qty += l.Qty
				if each > 0 {
					it.Price = each
				}
				if it.Size == "" && strings.TrimSpace(l.Size) != "" {
					it.Size = l.Size
				}
				if _, err := s.Store.SaveStock(it); err != nil {
					writeStoreErr(w, err)
					return
				}
				updated++
				continue
			}
		}
		area := "kitchen"
		if l.Area == "home" {
			area = "home"
		}
		if _, err := s.Store.SaveStock(&store.StockItem{Area: area, Name: name, Brand: strings.TrimSpace(l.Brand), Qty: l.Qty,
			Price: each, Size: strings.TrimSpace(l.Size)}); err != nil {
			writeStoreErr(w, err)
			return
		}
		added++
	}
	writeJSON(w, http.StatusOK, map[string]int{"added": added, "updated": updated})
}
