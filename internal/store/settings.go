package store

import (
	"strconv"
	"strings"
	"time"
)

// Setting keys editable from the admin page.
const (
	KeyLLMProvider       = "llm_provider" // "openai" (any OpenAI-compatible API) or "anthropic" (Claude)
	KeyLLMBaseURL        = "llm_base_url"
	KeyLLMAPIKey         = "llm_api_key"
	KeyLLMModel          = "llm_model"
	KeyLLMFallbackModel  = "llm_fallback_model" // more models to try, comma-separated
	KeyLLMVisionModel    = "llm_vision_model"   // for photos; "" = the main model
	KeyLLMJSONMode       = "llm_json_mode"
	KeyLLMTimeoutSeconds = "llm_timeout_seconds" // 0 = automatic

	KeySessionDays   = "session_days"   // "keep me signed in" length; renewed while in use
	KeyAllergenList  = "allergen_list"  // "us" (the 9 major US allergens) or "eu" (the EU's 14)
	KeyDefaultUnits  = "default_units"  // "us" or "metric"
	KeyHouseholdPets = "household_pets" // comma-separated: dog, cat, bird, small, fish
)

var Defaults = map[string]string{
	KeyLLMProvider:       "openai",
	KeyLLMBaseURL:        "https://api.openai.com/v1",
	KeyLLMModel:          "gpt-4o-mini",
	KeyLLMFallbackModel:  "",
	KeyLLMVisionModel:    "",
	KeyLLMJSONMode:       "true",
	KeyLLMTimeoutSeconds: "0",
	KeySessionDays:       "30",
	KeyAllergenList:      "us",
	KeyDefaultUnits:      "us",
	KeyHouseholdPets:     "",
}

// SecretKeys are never returned to the browser in clear text.
var SecretKeys = map[string]bool{KeyLLMAPIKey: true}

// AllSettings returns stored values merged over defaults.
func (s *Store) AllSettings() (map[string]string, error) {
	out := make(map[string]string, len(Defaults))
	for k, v := range Defaults {
		out[k] = v
	}
	var rows []struct {
		Key   string `db:"key"`
		Value string `db:"value"`
	}
	if err := s.DB.Select(&rows, `SELECT key, value FROM settings`); err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.Key] = r.Value
	}
	return out, nil
}

func (s *Store) Setting(key string) string {
	var v string
	if err := s.DB.Get(&v, `SELECT value FROM settings WHERE key = ?`, key); err != nil {
		return Defaults[key]
	}
	return v
}

func (s *Store) SettingInt(key string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s.Setting(key)))
	return n
}

func (s *Store) SettingBool(key string) bool {
	b, _ := strconv.ParseBool(s.Setting(key))
	return b
}

func (s *Store) SetSetting(key, value string) error {
	_, err := s.DB.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// Pets returns the household's pets (for Home & Care safety checks).
func (s *Store) Pets() []string {
	return SplitList(s.Setting(KeyHouseholdPets))
}

// AIConfig is how to reach the AI.
type AIConfig struct {
	Provider string // "openai" (any OpenAI-compatible API) or "anthropic"
	BaseURL  string
	APIKey   string
	JSONMode bool
	Models   []string // tried in order
	Vision   []string // models for photos, tried in order
}

// AIConfig returns the AI settings; Ready is false until a model is set.
func (s *Store) AIConfig() AIConfig {
	models := SplitList(s.Setting(KeyLLMModel) + "," + s.Setting(KeyLLMFallbackModel))
	vision := SplitList(s.Setting(KeyLLMVisionModel))
	if len(vision) == 0 {
		vision = models
	}
	return AIConfig{
		Provider: s.Setting(KeyLLMProvider), BaseURL: s.Setting(KeyLLMBaseURL), APIKey: s.Setting(KeyLLMAPIKey),
		JSONMode: s.SettingBool(KeyLLMJSONMode), Models: models, Vision: vision,
	}
}

// Ready reports whether enough is set to ask the AI anything.
func (c AIConfig) Ready() bool {
	if len(c.Models) == 0 {
		return false
	}
	if c.Provider == "anthropic" {
		return c.APIKey != ""
	}
	return strings.TrimSpace(c.BaseURL) != ""
}

// RecordUsage logs one AI call's tokens.
func (s *Store) RecordUsage(model string, in, out int, took time.Duration) error {
	_, err := s.DB.Exec(`INSERT INTO token_usage (model, prompt_tokens, completion_tokens, seconds) VALUES (?, ?, ?, ?)`,
		model, in, out, took.Seconds())
	return err
}

// TokensThisMonth is how many tokens the AI used since the 1st.
func (s *Store) TokensThisMonth() int {
	var n int
	_ = s.DB.Get(&n, `SELECT COALESCE(SUM(prompt_tokens + completion_tokens), 0) FROM token_usage
		WHERE at >= date('now', 'start of month')`)
	return n
}

// SplitList splits a comma-separated list, trimming and dropping blanks and repeats.
func SplitList(list string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, m := range strings.Split(list, ",") {
		if m = strings.TrimSpace(m); m != "" && !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
}
