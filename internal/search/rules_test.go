package search

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	r := Parse("quick dairy-free dinners everyone can eat")
	if strings.Join(r.Diets, ",") != "dairyfree" || r.MaxMin != 30 || !r.Everyone || !r.Quick {
		t.Errorf("%+v", r)
	}
	r = Parse("Vegan soups under 45 minutes")
	if strings.Join(r.Diets, ",") != "vegan" || r.MaxMin != 45 || r.Everyone {
		t.Errorf("%+v", r)
	}
	if r := Parse("Lent Friday suppers"); strings.Join(r.Diets, ",") != "meatless" {
		t.Errorf("%+v", r)
	}
	if k := Keywords("something quick with chicken and rice the kids like"); strings.Join(k, " ") != "chicken rice" {
		t.Errorf("keywords %v", k)
	}
}
