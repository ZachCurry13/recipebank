package store

// A second AI just for photos (recipe cards, receipts, the fridge, index
// pages): a bigger model on another computer, or an online service. The main
// AI keeps the text work, and reads photos itself when this one is off.
const (
	KeyPhotoProvider = "photo_provider" // "" = none, "openai" (any OpenAI-compatible API) or "anthropic"
	KeyPhotoBaseURL  = "photo_base_url"
	KeyPhotoAPIKey   = "photo_api_key" // a secret: never sent to the browser
	KeyPhotoModel    = "photo_model"   // tried in order, comma-separated
	KeyPhotoJSONMode = "photo_json_mode"
)

func init() {
	SecretKeys[KeyPhotoAPIKey] = true
	for _, k := range []string{KeyPhotoProvider, KeyPhotoBaseURL, KeyPhotoModel} {
		Defaults[k] = ""
	}
	Defaults[KeyPhotoJSONMode] = "true"
}

// PhotoAIConfig is the photo AI's settings; not Ready when there's none.
func (s *Store) PhotoAIConfig() AIConfig {
	provider := s.Setting(KeyPhotoProvider)
	if provider != "openai" && provider != "anthropic" {
		return AIConfig{}
	}
	models := SplitList(s.Setting(KeyPhotoModel))
	return AIConfig{Provider: provider, BaseURL: s.Setting(KeyPhotoBaseURL), APIKey: s.Setting(KeyPhotoAPIKey),
		JSONMode: s.SettingBool(KeyPhotoJSONMode), Models: models, Vision: models}
}

// AnyAI reports whether some AI is set up: the main one, or just one for photos.
func (s *Store) AnyAI() bool {
	return s.AIConfig().Ready() || s.PhotoAIConfig().Ready()
}
