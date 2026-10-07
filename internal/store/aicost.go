package store

import "strconv"

// Prices of online AI services, in dollars per million tokens (0 = free,
// like a model on your own computer), for "about $X this month".
const (
	KeyLLMPriceIn    = "llm_price_in"
	KeyLLMPriceOut   = "llm_price_out"
	KeyPhotoPriceIn  = "photo_price_in"
	KeyPhotoPriceOut = "photo_price_out"
)

func init() {
	for _, k := range []string{KeyLLMPriceIn, KeyLLMPriceOut, KeyPhotoPriceIn, KeyPhotoPriceOut} {
		Defaults[k] = "0"
	}
}

// AICost is what the AIs cost this month, from their token use and prices.
type AICost struct {
	Main  float64 `json:"main"`
	Photo float64 `json:"photo"`
	Total float64 `json:"total"`
}

// CostThisMonth adds up each model's tokens since the 1st at the price of
// the AI that uses it (the photo AI's models first: they're its only job).
func (s *Store) CostThisMonth() AICost {
	var rows []struct {
		Model string `db:"model"`
		In    int64  `db:"in_tokens"`
		Out   int64  `db:"out_tokens"`
	}
	_ = s.DB.Select(&rows, `SELECT model, COALESCE(SUM(prompt_tokens), 0) AS in_tokens, COALESCE(SUM(completion_tokens), 0) AS out_tokens
		FROM token_usage WHERE at >= date('now', 'start of month') GROUP BY model`)
	price := func(k string) float64 { v, _ := strconv.ParseFloat(s.Setting(k), 64); return v }
	photo := map[string]bool{}
	for _, m := range s.PhotoAIConfig().Models {
		photo[m] = true
	}
	var c AICost
	for _, r := range rows {
		if photo[r.Model] {
			c.Photo += (float64(r.In)*price(KeyPhotoPriceIn) + float64(r.Out)*price(KeyPhotoPriceOut)) / 1e6
		} else {
			c.Main += (float64(r.In)*price(KeyLLMPriceIn) + float64(r.Out)*price(KeyLLMPriceOut)) / 1e6
		}
	}
	c.Total = c.Main + c.Photo
	return c
}
