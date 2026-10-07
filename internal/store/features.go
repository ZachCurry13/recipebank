package store

import "strings"

// Features an admin can turn off for the whole house, so the app shows only
// what the family uses. All are on until turned off; their data stays.
var Features = []string{"plan", "shopping", "make", "home", "pantry", "collections", "books", "events",
	"budget", "nutrition", "email", "share", "kids"}

// FeatureKey is a feature's setting ("feature_plan").
func FeatureKey(f string) string { return "feature_" + f }

func init() {
	for _, f := range Features {
		Defaults[FeatureKey(f)] = "true"
	}
}

// FeaturesOn says which features are on.
func (s *Store) FeaturesOn() map[string]bool {
	out := make(map[string]bool, len(Features))
	for _, f := range Features {
		out[f] = s.SettingBool(FeatureKey(f))
	}
	return out
}

// FeatureOn says whether one feature is on.
func (s *Store) FeatureOn(f string) bool { return s.SettingBool(FeatureKey(f)) }

// PlanMeals and HidePages are what a person may choose to see less of.
var (
	PlanMeals = []string{"breakfast", "lunch", "dinner", "snack"}
	HidePages = []string{"plan", "shopping", "make", "home", "pantry", "supplies", "collections", "books", "events", "add", "family"}
)

// SetSimpler saves a person's own simpler view: the meals their plan shows
// ("" = all) and the pages left out of their menu.
func (s *Store) SetSimpler(id int64, planMeals, hiddenPages []string) error {
	_, err := s.DB.Exec(`UPDATE users SET plan_meals = ?, hidden_pages = ? WHERE id = ?`,
		strings.Join(planMeals, ","), strings.Join(hiddenPages, ","), id)
	return err
}
