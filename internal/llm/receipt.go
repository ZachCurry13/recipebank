package llm

import (
	"encoding/json"
	"errors"
	"strings"
)

// ReceiptPrompt asks the vision model for what a store receipt bought.
const ReceiptPrompt = `This photo shows a store receipt. List each item bought. Reply with JSON only:
{"items": [{"name": "plain product name", "brand": "", "qty": 1, "size": "", "price": 0, "area": "kitchen"}]}
- name: what it is in plain words, expanding receipt abbreviations only when you're sure ("GV WHL MLK" → "whole milk").
- qty: how many were bought (1 if not shown). size: the package size if printed ("1 gal", "16 oz"), else "".
- price: what was paid for that line in total, as a number. area: "kitchen" for food and drink, "home" for
  cleaning, bathroom and household things.
Leave out taxes, totals, payments, bag fees, coupons and anything you can't read.
If the photo isn't a receipt, reply {"items": []}.`

// ReceiptItem is one line of a receipt.
type ReceiptItem struct {
	Name  string  `json:"name"`
	Brand string  `json:"brand"`
	Qty   float64 `json:"qty"`
	Size  string  `json:"size"`
	Price float64 `json:"price"`
	Area  string  `json:"area"`
}

// ParseReceipt reads the answer: at most 80 sensible lines.
func ParseReceipt(out string) ([]ReceiptItem, error) {
	var a struct {
		Items []ReceiptItem `json:"items"`
	}
	if err := json.Unmarshal([]byte(jsonObject(out)), &a); err != nil {
		return nil, errors.New("the AI's answer wasn't valid JSON")
	}
	var items []ReceiptItem
	for _, it := range a.Items {
		it.Name, it.Brand, it.Size = clip(it.Name, 120), clip(it.Brand, 60), clip(it.Size, 30)
		if it.Name == "" {
			continue
		}
		if it.Qty <= 0 || it.Qty > 100 {
			it.Qty = 1
		}
		if it.Price < 0 || it.Price > 10000 {
			it.Price = 0
		}
		if it.Area != "home" {
			it.Area = "kitchen"
		}
		it.Name = strings.TrimSpace(it.Name)
		if items = append(items, it); len(items) == 80 {
			break
		}
	}
	if len(items) == 0 {
		return nil, errors.New("no items could be read from the receipt: try a flatter, closer photo")
	}
	return items, nil
}
