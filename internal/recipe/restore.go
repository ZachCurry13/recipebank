package recipe

import (
	"regexp"
	"strings"
)

// RestoreLines puts back the ingredient lines as the person wrote them. AI
// models, small ones above all, shorten lines while organizing a recipe even
// when told not to: "1 1/2 cups chocolate chips (half milk chocolate) 340g"
// came back as "1 1/2 cups chocolate chips" plus a line "340g", and the note
// that says milk was lost. So each line the AI gives is matched to the
// source text (pasted text, a page, or what a card was read as): a line the
// source has more of gets its whole source line back, and a piece of a line
// already restored is dropped. Lines not found are kept as the AI gave them.
func RestoreLines(source string, ings []Ingredient, steps []Step) []Ingredient {
	var cands []string // the source's lines that could be ingredients
	stepStarts := map[string]bool{}
	for _, st := range steps {
		if k := lineKey(st.Text); len(k) >= 12 {
			stepStarts[k[:12]] = true
		}
	}
	for _, l := range strings.Split(source, "\n") {
		l = strings.Join(strings.Fields(l), " ")
		l = strings.TrimLeft(l, "-•*·▪◦ ")
		k := lineKey(l)
		if l == "" || len(l) > 240 || (len(k) >= 12 && stepStarts[k[:12]]) {
			continue
		}
		cands = append(cands, l)
	}
	used := map[int]bool{}
	out := make([]Ingredient, 0, len(ings))
	for _, in := range ings {
		k := lineKey(in.Line)
		at := -1
		if k != "" {
			for pass := 0; pass < 3 && at < 0; pass++ {
				for i, c := range cands {
					ck := lineKey(c)
					switch {
					case pass == 0 && ck == k, // the same line
						pass == 1 && strings.HasPrefix(ck, k+" "),                     // the AI cut its end off
						pass == 2 && hasAmount.MatchString(k) && containsWords(ck, k): // a piece of it ("340g")
						at = i
					}
					if at >= 0 {
						break
					}
				}
			}
		}
		switch {
		case at < 0:
			out = append(out, in)
		case used[at]:
			// A piece of a line that's already back in full.
		default:
			used[at] = true
			if lineKey(cands[at]) != k {
				in.Line, in.Food, in.Qty, in.QtyMax, in.Unit, in.Note = cands[at], "", nil, nil, "", "" // parsed again from the whole line
			}
			out = append(out, in)
		}
	}
	return out
}

var hasAmount = regexp.MustCompile(`\d`)

// lineKey compares lines loosely: lower case, single spaces, no end punctuation.
func lineKey(s string) string {
	s = strings.ToLower(strings.Join(strings.Fields(s), " "))
	return strings.TrimRight(s, " .,;:")
}

// containsWords says whether k appears in s as whole words.
func containsWords(s, k string) bool {
	i := strings.Index(" "+s+" ", " "+k+" ")
	if i >= 0 {
		return true
	}
	// "340g)" or "(340g": allow brackets around it.
	clean := strings.NewReplacer("(", " ", ")", " ", ",", " ").Replace(s)
	return strings.Contains(" "+strings.Join(strings.Fields(clean), " ")+" ", " "+k+" ")
}
