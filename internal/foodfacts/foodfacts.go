// Package foodfacts looks up packaged products by barcode in the open
// databases run by Open Food Facts: foods, and (for Home & Care) beauty and
// household products. Only the barcode is sent.
package foodfacts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

const userAgent = "RecipeBank/0.2 (+https://github.com/ZachCurry13/recipebank)"

// Product is what a label says.
type Product struct {
	Barcode         string   `json:"barcode"`
	Name            string   `json:"name"`
	Brand           string   `json:"brand"`
	Quantity        string   `json:"quantity"` // as printed: "500 g"
	Allergens       []string `json:"allergens"`
	Traces          []string `json:"traces"`
	IngredientsText string   `json:"ingredients_text"`
	Source          string   `json:"source"` // which database answered
}

// Client looks products up. Bases are tried in order; tests point them at a fake.
type Client struct {
	HTTP  *http.Client
	Bases []string
}

// FoodBases and HomeBases are the databases for the pantry and the supply closet.
var (
	FoodBases = []string{"https://world.openfoodfacts.org"}
	HomeBases = []string{"https://world.openbeautyfacts.org", "https://world.openproductsfacts.org", "https://world.openfoodfacts.org"}
)

// ErrNotFound means no database knows the barcode.
var ErrNotFound = errors.New("that barcode isn't in the open product databases yet")

var barcodeRE = regexp.MustCompile(`^\d{8}$|^\d{12,14}$`)

// ValidBarcode reports whether s looks like an EAN/UPC product barcode.
func ValidBarcode(s string) bool { return barcodeRE.MatchString(s) }

// Lookup asks each database in turn.
func (c Client) Lookup(ctx context.Context, barcode string) (Product, error) {
	if !ValidBarcode(barcode) {
		return Product{}, errors.New("a barcode is 8, 12, 13 or 14 digits")
	}
	// A UPC (12 digits) is stored as an EAN-13 with a leading 0, and the other way round.
	codes := []string{barcode}
	if len(barcode) == 12 {
		codes = append(codes, "0"+barcode)
	} else if len(barcode) == 13 && barcode[0] == '0' {
		codes = append(codes, barcode[1:])
	}
	var lastErr error = ErrNotFound
	for _, base := range c.Bases {
		for _, code := range codes {
			p, err := c.one(ctx, base, code)
			if err == nil {
				p.Barcode = barcode
				return p, nil
			}
			if !errors.Is(err, ErrNotFound) {
				lastErr = err
			}
		}
	}
	return Product{}, lastErr
}

func (c Client) one(ctx context.Context, base, barcode string) (Product, error) {
	url := fmt.Sprintf("%s/api/v2/product/%s.json?fields=product_name,brands,quantity,allergens_tags,traces_tags,ingredients_text", strings.TrimRight(base, "/"), barcode)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Product{}, err
	}
	req.Header.Set("User-Agent", userAgent)
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return Product{}, fmt.Errorf("couldn't reach the product database: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Product{}, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return Product{}, fmt.Errorf("the product database answered %s", resp.Status)
	}
	var body struct {
		Status  int `json:"status"`
		Product struct {
			Name        string   `json:"product_name"`
			Brands      string   `json:"brands"`
			Quantity    string   `json:"quantity"`
			Allergens   []string `json:"allergens_tags"`
			Traces      []string `json:"traces_tags"`
			Ingredients string   `json:"ingredients_text"`
		} `json:"product"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&body); err != nil {
		return Product{}, errors.New("the product database's answer couldn't be read")
	}
	if body.Status != 1 || strings.TrimSpace(body.Product.Name) == "" {
		return Product{}, ErrNotFound
	}
	brand, _, _ := strings.Cut(body.Product.Brands, ",")
	return Product{Barcode: barcode, Name: strings.TrimSpace(body.Product.Name), Brand: strings.TrimSpace(brand),
		Quantity: strings.TrimSpace(body.Product.Quantity), Allergens: Keys(body.Product.Allergens),
		Traces: Keys(body.Product.Traces), IngredientsText: strings.TrimSpace(body.Product.Ingredients), Source: base}, nil
}

// tagKeys turns Open Food Facts allergen tags into RecipeBank's allergen
// keys. "Gluten" may be wheat or another grain, so it counts as both.
var tagKeys = map[string][]string{
	"en:milk": {"milk"}, "en:eggs": {"egg"}, "en:fish": {"fish"}, "en:crustaceans": {"shellfish"},
	"en:nuts": {"treenut"}, "en:peanuts": {"peanut"}, "en:gluten": {"gluten", "wheat"}, "en:soybeans": {"soy"},
	"en:sesame-seeds": {"sesame"}, "en:celery": {"celery"}, "en:mustard": {"mustard"}, "en:lupin": {"lupin"},
	"en:molluscs": {"mollusc"}, "en:sulphur-dioxide-and-sulphites": {"sulphite"},
}

// Keys maps allergen tags to allergen keys, without repeats.
func Keys(tags []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, t := range tags {
		for _, k := range tagKeys[strings.ToLower(strings.TrimSpace(t))] {
			if !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
	}
	return out
}
