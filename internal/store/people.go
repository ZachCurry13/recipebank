package store

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/zachcurry13/recipebank/internal/safety"
)

// Person is someone who eats here (or uses the home recipes), with or
// without an account.
type Person struct {
	ID        int64  `db:"id" json:"id"`
	Name      string `db:"name" json:"name"`
	IsKid     bool   `db:"is_kid" json:"is_kid"`
	IsGuest   bool   `db:"is_guest" json:"is_guest"`
	HeatMax   int    `db:"heat_max" json:"heat_max"`
	CreatedAt string `db:"created_at" json:"created_at"`
	Rules     []Rule `db:"-" json:"rules"`
}

type Rule struct {
	PersonID int64  `db:"person_id" json:"-"`
	Kind     string `db:"kind" json:"kind"` // allergy, diet, dislike, sensitivity
	Key      string `db:"key" json:"key"`
	Severity string `db:"severity" json:"severity,omitempty"`
	// Allow (allergies): foods made from the allergen this person can have,
	// as their doctor says: "soybean oil" for someone allergic to soy.
	Allow     []string `db:"-" json:"allow,omitempty"`
	AllowText string   `db:"allow" json:"-"`
}

// ListPeople returns everyone with their rules, guests last.
func (s *Store) ListPeople() ([]Person, error) {
	ps := []Person{}
	if err := s.DB.Select(&ps, `SELECT id, name, is_kid, is_guest, heat_max, created_at FROM people
		ORDER BY is_guest, name COLLATE NOCASE`); err != nil {
		return nil, err
	}
	var rules []Rule
	if err := s.DB.Select(&rules, `SELECT person_id, kind, key, severity, allow FROM person_rules ORDER BY kind, key`); err != nil {
		return nil, err
	}
	byID := map[int64]*Person{}
	for i := range ps {
		ps[i].Rules = []Rule{}
		byID[ps[i].ID] = &ps[i]
	}
	for _, r := range rules {
		if p := byID[r.PersonID]; p != nil {
			r.Allow = SplitList(r.AllowText)
			p.Rules = append(p.Rules, r)
		}
	}
	return ps, nil
}

func (s *Store) PersonByID(id int64) (*Person, error) {
	ps, err := s.ListPeople()
	if err != nil {
		return nil, err
	}
	for i := range ps {
		if ps[i].ID == id {
			return &ps[i], nil
		}
	}
	return nil, ErrNotFound
}

// SavePerson inserts (ID 0) or updates a person and replaces their rules.
func (s *Store) SavePerson(p *Person) (int64, error) {
	p.Name = strings.TrimSpace(p.Name)
	if p.HeatMax < -1 || p.HeatMax > 5 {
		p.HeatMax = -1
	}
	tx, err := s.DB.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	id := p.ID
	if id == 0 {
		res, err := tx.Exec(`INSERT INTO people (name, is_kid, is_guest, heat_max) VALUES (?, ?, ?, ?)`,
			p.Name, p.IsKid, p.IsGuest, p.HeatMax)
		if err != nil {
			return 0, err
		}
		id, _ = res.LastInsertId()
	} else {
		res, err := tx.Exec(`UPDATE people SET name = ?, is_kid = ?, is_guest = ?, heat_max = ? WHERE id = ?`,
			p.Name, p.IsKid, p.IsGuest, p.HeatMax, id)
		if err != nil {
			return 0, err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return 0, ErrNotFound
		}
	}
	if _, err := tx.Exec(`DELETE FROM person_rules WHERE person_id = ?`, id); err != nil {
		return 0, err
	}
	for _, r := range p.Rules {
		if _, err := tx.Exec(`INSERT OR REPLACE INTO person_rules (person_id, kind, key, severity, allow) VALUES (?, ?, ?, ?, ?)`,
			id, r.Kind, strings.TrimSpace(r.Key), r.Severity, strings.Join(r.Allow, ",")); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}

func (s *Store) DeletePerson(id int64) error {
	res, err := s.DB.Exec(`DELETE FROM people WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// CleanAllow tidies an allergy's "can have anyway" list and says what's
// wrong with it: at most 10 short food names, none that's the allergy itself.
func CleanAllow(r *Rule) error {
	if len(r.Allow) == 0 {
		return nil
	}
	if r.Kind != "allergy" {
		return errors.New("only allergies have foods they can have anyway")
	}
	var out []string
	for _, a := range r.Allow {
		a = strings.Join(strings.Fields(strings.ToLower(a)), " ")
		if a == "" || slicesContains(out, a) {
			continue
		}
		if len(a) < 2 || len(a) > 40 || strings.IndexFunc(a, func(c rune) bool {
			return !unicode.IsLetter(c) && c != ' ' && c != '-' && c != '\''
		}) >= 0 {
			return fmt.Errorf("%q: a food name of 2 to 40 letters, please", a)
		}
		if safety.SameAsAllergen(r.Key, a) {
			return fmt.Errorf("%q is the allergy itself; to allow everything, take the allergy off instead", a)
		}
		out = append(out, a)
	}
	if len(out) > 10 {
		return errors.New("at most 10 foods they can have anyway, for each allergy")
	}
	r.Allow = out
	return nil
}

func slicesContains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// ErrBadRule is returned for a rule the checker wouldn't understand.
var ErrBadRule = errors.New("unknown rule")

// ValidRule checks a rule's kind, key and severity.
func ValidRule(r Rule) error {
	switch r.Kind {
	case "allergy":
		if _, ok := safety.AllergenByKey(r.Key); !ok {
			return ErrBadRule
		}
		if r.Severity != safety.Avoid && r.Severity != safety.Allergic && r.Severity != safety.Severe {
			return ErrBadRule
		}
	case "diet":
		if _, ok := safety.DietByKey(r.Key); !ok {
			return ErrBadRule
		}
	case "dislike", "sensitivity":
		if strings.TrimSpace(r.Key) == "" || len(r.Key) > 60 {
			return ErrBadRule
		}
	default:
		return ErrBadRule
	}
	return nil
}

// Checkable turns a person into what the safety checker needs.
func (p *Person) Checkable() safety.Person {
	out := safety.Person{ID: p.ID, Name: p.Name, IsKid: p.IsKid, HeatMax: p.HeatMax, Allergies: map[string]string{}}
	for _, r := range p.Rules {
		switch r.Kind {
		case "allergy":
			out.Allergies[r.Key] = r.Severity
			if len(r.Allow) > 0 {
				if out.Allowed == nil {
					out.Allowed = map[string][]string{}
				}
				out.Allowed[r.Key] = r.Allow
			}
		case "diet":
			out.Diets = append(out.Diets, r.Key)
		case "dislike":
			out.Dislikes = append(out.Dislikes, r.Key)
		case "sensitivity":
			out.Sensitivities = append(out.Sensitivities, r.Key)
		}
	}
	return out
}
