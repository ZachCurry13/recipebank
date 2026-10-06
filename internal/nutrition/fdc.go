// Package nutrition estimates nutrition per serving from USDA FoodData
// Central (public domain): nutrients per 100 g of each food, and the weight
// of a cup, a spoon or one piece to turn recipe amounts into grams.
package nutrition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// DefaultBase is FoodData Central's API.
const DefaultBase = "https://api.nal.usda.gov/fdc/v1"

// ErrNotFound means FoodData Central has nothing for a food.
var ErrNotFound = errors.New("not in FoodData Central")

// Nutrients are amounts per 100 g (Food) or per serving (an estimate).
type Nutrients struct {
	Kcal     float64 `json:"kcal"`
	Protein  float64 `json:"protein_g"`
	Fat      float64 `json:"fat_g"`
	Carbs    float64 `json:"carbs_g"`
	Fiber    float64 `json:"fiber_g"`
	Sugar    float64 `json:"sugar_g"`
	SodiumMg float64 `json:"sodium_mg"`
}

// Portion is a household measure: "1 cup" weighs 128 g.
type Portion struct {
	Text  string  `json:"text"`
	Grams float64 `json:"grams"`
}

// Food is what FoodData Central knows about one food.
type Food struct {
	FdcID    int64     `json:"fdc_id"`
	Name     string    `json:"name"`
	Per100g  Nutrients `json:"per_100g"`
	Portions []Portion `json:"portions"`
}

// Client looks foods up; the key is sent in a header, never in the address.
type Client struct {
	HTTP  *http.Client
	Base  string
	Key   string
	Agent string
}

// FDC nutrient numbers.
const (
	nEnergy, nEnergyAtwater, nEnergyAtwater2 = 1008, 2047, 2048
	nProtein, nFat, nCarbs, nFiber           = 1003, 1004, 1005, 1079
	nSugar, nSugar2, nSodium                 = 2000, 1063, 1093
)

type rawNutrient struct {
	NutrientID int64   `json:"nutrientId"`
	Value      float64 `json:"value"`
	Amount     float64 `json:"amount"` // the single-food endpoint's name for value
	Nutrient   struct {
		ID int64 `json:"id"`
	} `json:"nutrient"`
}

// Lookup finds a food (plain foods first) with its nutrients and portions.
func (c Client) Lookup(ctx context.Context, food string) (Food, error) {
	terms := searchTerms(food)
	q := url.Values{"query": {terms}, "dataType": {"SR Legacy", "Foundation"}, "pageSize": {"10"}}
	var res struct {
		Foods []struct {
			FdcID       int64         `json:"fdcId"`
			Description string        `json:"description"`
			Nutrients   []rawNutrient `json:"foodNutrients"`
			Measures    []struct {
				Text  string  `json:"disseminationText"`
				Grams float64 `json:"gramWeight"`
			} `json:"foodMeasures"`
		} `json:"foods"`
	}
	if err := c.get(ctx, "/foods/search?"+q.Encode(), &res); err != nil {
		return Food{}, err
	}
	if len(res.Foods) == 0 {
		return Food{}, ErrNotFound
	}
	best := 0
	for i, r := range res.Foods {
		if score(terms, r.Description) > score(terms, res.Foods[best].Description) {
			best = i
		}
	}
	f := res.Foods[best]
	out := Food{FdcID: f.FdcID, Name: f.Description, Per100g: nutrients(f.Nutrients)}
	for _, m := range f.Measures {
		if m.Grams > 0 && m.Text != "" && !strings.EqualFold(m.Text, "Quantity not specified") {
			out.Portions = append(out.Portions, Portion{Text: strings.ToLower(m.Text), Grams: m.Grams})
		}
	}
	if len(out.Portions) == 0 {
		out.Portions, _ = c.portions(ctx, f.FdcID) // a food without measures still counts by weight
	}
	return out, nil
}

// portions reads a food's household measures from its own page.
func (c Client) portions(ctx context.Context, id int64) ([]Portion, error) {
	var res struct {
		Portions []struct {
			Amount   float64 `json:"amount"`
			Modifier string  `json:"modifier"`
			Desc     string  `json:"portionDescription"`
			Grams    float64 `json:"gramWeight"`
			Unit     struct {
				Name string `json:"name"`
			} `json:"measureUnit"`
		} `json:"foodPortions"`
	}
	if err := c.get(ctx, fmt.Sprintf("/food/%d", id), &res); err != nil {
		return nil, err
	}
	var out []Portion
	for _, p := range res.Portions {
		text := p.Desc
		if text == "" || strings.EqualFold(text, "Quantity not specified") {
			unit := p.Unit.Name
			if unit == "undetermined" {
				unit = ""
			}
			text = strings.Join(strings.Fields(fmt.Sprintf("%g %s %s", max(p.Amount, 1), unit, p.Modifier)), " ")
		}
		if p.Grams > 0 {
			out = append(out, Portion{Text: strings.ToLower(text), Grams: p.Grams})
		}
	}
	return out, nil
}

func nutrients(list []rawNutrient) Nutrients {
	v := map[int64]float64{}
	for _, n := range list {
		id, val := n.NutrientID, n.Value
		if id == 0 {
			id = n.Nutrient.ID
		}
		if val == 0 {
			val = n.Amount
		}
		if _, seen := v[id]; !seen {
			v[id] = val
		}
	}
	first := func(ids ...int64) float64 {
		for _, id := range ids {
			if x, ok := v[id]; ok {
				return x
			}
		}
		return 0
	}
	return Nutrients{Kcal: first(nEnergy, nEnergyAtwater, nEnergyAtwater2), Protein: first(nProtein), Fat: first(nFat),
		Carbs: first(nCarbs), Fiber: first(nFiber), Sugar: first(nSugar, nSugar2), SodiumMg: first(nSodium)}
}

func (c Client) get(ctx context.Context, path string, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.Base, "/")+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", c.Key)
	req.Header.Set("User-Agent", c.Agent)
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return errors.New("couldn't reach USDA FoodData Central")
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized:
		return errors.New("USDA FoodData Central didn't accept the key (Admin → House settings)")
	case resp.StatusCode == http.StatusTooManyRequests:
		return errors.New("USDA FoodData Central's hourly limit was reached; try again later")
	case resp.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("USDA FoodData Central answered %s", resp.Status)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(into)
}
