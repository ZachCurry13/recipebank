package recipe

import "strings"

// Misreads finds lines that were corrected rather than added or removed:
// where a run of lines was replaced by the same number of lines, each pair
// whose foods look alike ("bell pepper flakes" → "red pepper flakes", "cumen"
// → "cumin") is a likely misread, returned as [wrong, right]. Amount-only
// changes and different foods aren't.
func Misreads(before, after []Ingredient) [][2]string {
	key := func(in Ingredient) string { return strings.Join(strings.Fields(strings.ToLower(in.Line)), " ") }
	n, m := len(before), len(after)
	L := make([][]int, n+1)
	for i := range L {
		L[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if key(before[i]) == key(after[j]) {
				L[i][j] = L[i+1][j+1] + 1
			} else {
				L[i][j] = max(L[i+1][j], L[i][j+1])
			}
		}
	}
	var out [][2]string
	var gone, added []Ingredient
	flush := func() {
		if len(gone) == len(added) {
			for k := range gone {
				w, r := foodOf(gone[k]), foodOf(added[k])
				if w != "" && r != "" && w != r && len(w) <= 40 && len(r) <= 40 && alike(w, r) {
					out = append(out, [2]string{w, r})
				}
			}
		}
		gone, added = nil, nil
	}
	i, j := 0, 0
	for i < n || j < m {
		switch {
		case i < n && j < m && key(before[i]) == key(after[j]):
			flush()
			i, j = i+1, j+1
		case j >= m || (i < n && L[i+1][j] >= L[i][j+1]):
			gone = append(gone, before[i])
			i++
		default:
			added = append(added, after[j])
			j++
		}
	}
	flush()
	return out
}

func foodOf(in Ingredient) string {
	f := in.Food
	if f == "" {
		f = ParseLine(in.Line).Food
	}
	return strings.ToLower(strings.TrimSpace(f))
}

// alike: the foods share a word, or are spelled nearly the same. Adding
// detail ("cheese" → "cheddar cheese") isn't a misread.
func alike(a, b string) bool {
	if within(a, b) || within(b, a) {
		return false
	}
	for _, w := range strings.Fields(a) {
		if len(w) >= 3 && strings.Contains(" "+b+" ", " "+w+" ") {
			return true
		}
	}
	return editDistance(a, b) <= max(1, min(len(a), len(b))/4)
}

func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

// within: every word of a is in b.
func within(a, b string) bool {
	have := map[string]bool{}
	for _, w := range strings.Fields(b) {
		have[w] = true
	}
	for _, w := range strings.Fields(a) {
		if !have[w] {
			return false
		}
	}
	return true
}
