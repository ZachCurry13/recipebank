// Package safety answers "can this person eat (or use) this recipe?" with
// word lists and rules, never with the AI alone: allergens (with hidden
// names), diets, dislikes, heat, skin sensitivities, and Home & Care hazards.
// Unknown is never safe for a real allergy.
package safety

import (
	"strings"
	"unicode"
)

var accents = strings.NewReplacer("é", "e", "è", "e", "ê", "e", "ë", "e", "à", "a", "â", "a", "î", "i", "ï", "i",
	"ô", "o", "ö", "o", "û", "u", "ü", "u", "ñ", "n", "ç", "c", "’", "'", "‘", "'")

var irregular = map[string]string{"leaves": "leaf", "halves": "half", "loaves": "loaf", "knives": "knife", "chives": "chive"}

// words turns text into lower-case, singular words.
func words(s string) []string {
	s = accents.Replace(strings.ToLower(s))
	f := strings.FieldsFunc(s, func(r rune) bool { return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'') })
	out := f[:0]
	for _, w := range f {
		if w = strings.Trim(w, "'"); w != "" {
			out = append(out, singular(w))
		}
	}
	return out
}

// singular is a rough plural remover, applied the same way to recipes and
// word lists so both sides agree.
func singular(w string) string {
	if s, ok := irregular[w]; ok {
		return s
	}
	switch {
	case len(w) <= 3:
		return w
	case strings.HasSuffix(w, "ies") && len(w) > 4:
		return w[:len(w)-3] + "y"
	case strings.HasSuffix(w, "oes"), strings.HasSuffix(w, "ches"), strings.HasSuffix(w, "shes"),
		strings.HasSuffix(w, "sses"), strings.HasSuffix(w, "xes"):
		return w[:len(w)-2]
	case strings.HasSuffix(w, "ss"), strings.HasSuffix(w, "us"), strings.HasSuffix(w, "is"):
		return w
	case strings.HasSuffix(w, "s"):
		return w[:len(w)-1]
	}
	return w
}

// phrases is a compiled word list, indexed by each phrase's first word.
type phrases struct {
	byFirst map[string][][]string
}

func compile(list ...string) phrases {
	ps := phrases{byFirst: map[string][][]string{}}
	for _, s := range list {
		if w := words(s); len(w) > 0 {
			ps.byFirst[w[0]] = append(ps.byFirst[w[0]], w)
		}
	}
	return ps
}

// text is an ingredient's words; masked words ("") never match.
type text []string

func newText(s string) text { return text(words(s)) }

func (t text) clone() text { return append(text(nil), t...) }

// at reports whether phrase p starts at word i.
func (t text) at(i int, p []string) bool {
	if i+len(p) > len(t) {
		return false
	}
	for j, w := range p {
		if t[i+j] != w {
			return false
		}
	}
	return true
}

// first returns the first phrase of ps found in t, or "".
func (t text) first(ps phrases) string {
	for i, w := range t {
		if w == "" {
			continue
		}
		for _, p := range ps.byFirst[w] {
			if t.at(i, p) {
				return strings.Join(p, " ")
			}
		}
	}
	return ""
}

func (t text) has(ps phrases) bool { return t.first(ps) != "" }

// mask blanks every occurrence of ps (e.g. "coconut milk" before looking for
// "milk"). All matches are found first, so a short phrase ("gluten free")
// can't hide a longer one ("gluten free flour").
func (t text) mask(ps phrases) {
	type hit struct{ at, n int }
	var hits []hit
	for i, w := range t {
		for _, p := range ps.byFirst[w] {
			if w != "" && t.at(i, p) {
				hits = append(hits, hit{i, len(p)})
			}
		}
	}
	for _, h := range hits {
		for j := 0; j < h.n; j++ {
			t[h.at+j] = ""
		}
	}
}
