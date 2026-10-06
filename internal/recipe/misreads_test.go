package recipe

import (
	"reflect"
	"testing"
)

func lines(ls ...string) []Ingredient {
	var out []Ingredient
	for _, l := range ls {
		out = append(out, Ingredient{Line: l})
	}
	return out
}

func TestMisreads(t *testing.T) {
	cases := []struct {
		name          string
		before, after []Ingredient
		want          [][2]string
	}{
		{"a look-alike word", lines("1 c. rice", "bell pepper flakes", "2 c. broth"),
			lines("1 c. rice", "1 tsp red pepper flakes", "2 c. broth", "1 c. cheese"),
			[][2]string{{"bell pepper flakes", "red pepper flakes"}}},
		{"a spelling", lines("1 t. cumen"), lines("1 tsp cumin"), [][2]string{{"cumen", "cumin"}}},
		{"only the amount", lines("1 c. sugar"), lines("1 T. sugar"), nil},
		{"a different food", lines("1 c. rice"), lines("2 carrots"), nil},
		{"lines added, not fixed", lines("1 c. rice"), lines("1 c. rice", "1 c. peas", "1 onion"), nil},
		{"one line became two", lines("salt and peper"), lines("1 tsp salt", "1/2 tsp pepper"), nil},
		{"more detail", lines("1 c. cheese"), lines("1 c. cheddar cheese"), nil},
	}
	for _, c := range cases {
		if got := Misreads(c.before, c.after); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
