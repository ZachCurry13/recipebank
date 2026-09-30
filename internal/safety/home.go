package safety

import (
	"strings"

	"github.com/zachcurry13/recipebank/internal/recipe"
)

// Hazard is a Home & Care warning about a whole recipe.
type Hazard struct {
	Level       string `json:"level"` // danger, caution, info
	Kind        string `json:"kind"`  // mix, kids, pets, surface, storage, mouth
	Text        string `json:"text"`
	Ingredients []int  `json:"ingredients"`
}

var essentialOils = []string{"essential oil", "tea tree", "tea tree oil", "melaleuca", "peppermint oil", "spearmint oil",
	"lavender oil", "eucalyptus", "eucalyptus oil", "clove oil", "cinnamon oil", "cinnamon bark oil", "lemon oil",
	"orange oil", "sweet orange oil", "lime oil", "grapefruit oil", "bergamot", "wintergreen", "wintergreen oil",
	"pine oil", "rosemary oil", "thyme oil", "oregano oil", "ylang ylang", "pennyroyal", "camphor", "birch oil",
	"lemongrass oil", "citronella", "anise oil", "juniper oil", "cedarwood oil", "sweet birch", "myrrh", "frankincense"}

// Chemical groups that must never meet.
var (
	bleach   = compile("bleach", "chlorine bleach", "sodium hypochlorite", "chlorine")
	bleachNo = compile("oxygen bleach", "non chlorine bleach", "bleach free", "chlorine free")
	ammonia  = compile("ammonia", "ammonium hydroxide", "ammonia based")
	acids    = compile("vinegar", "lemon juice", "lime juice", "citric acid", "acetic acid", "muriatic acid",
		"hydrochloric acid", "toilet bowl cleaner", "lemon", "lime")
	alcohols = compile("rubbing alcohol", "isopropyl", "isopropyl alcohol", "ethanol", "vodka", "grain alcohol", "everclear")
	peroxide = compile("hydrogen peroxide", "peroxide")
	bases    = compile("baking soda", "washing soda", "sodium bicarbonate", "sodium carbonate")
	castile  = compile("castile soap", "castile")
	borax    = compile("borax", "sodium borate", "boric acid")
	xylitol  = compile("xylitol")
	fluoride = compile("fluoride")
	water    = compile("water", "distilled water", "boiled water", "tap water", "hydrosol", "aloe", "aloe vera")
	oils     = compile(essentialOils...)
	strongEO = compile("wintergreen", "wintergreen oil", "camphor", "eucalyptus", "eucalyptus oil", "pennyroyal", "sweet birch", "birch oil")
	catEO    = oils // cats can't break down essential oils at all
	dogEO    = compile("tea tree", "tea tree oil", "melaleuca", "pennyroyal", "wintergreen", "pine oil", "cinnamon oil",
		"cinnamon bark oil", "peppermint oil", "ylang ylang", "sweet birch", "birch oil", "eucalyptus", "eucalyptus oil",
		"clove oil", "lemon oil", "orange oil", "lime oil", "grapefruit oil", "bergamot")
)

// found lists the ingredient lines matching any of ps (after look-alikes are masked).
func found(r *recipe.Recipe, ps phrases, not phrases) []int {
	var out []int
	for i, in := range r.Ingredients {
		t := newText(in.Line + " \n " + in.Food)
		t.mask(not)
		if t.has(ps) {
			out = append(out, i)
		}
	}
	return out
}

// Hazards lists what to know before making or using a Home & Care recipe.
// pets are the household's pets ("dog", "cat", "bird", …).
func Hazards(r *recipe.Recipe, pets []string) []Hazard {
	out := []Hazard{}
	add := func(level, kind, text string, lines ...[]int) {
		var ing []int
		for _, l := range lines {
			ing = append(ing, l...)
		}
		out = append(out, Hazard{Level: level, Kind: kind, Text: text, Ingredients: nonNil(ing)})
	}
	bl, am, ac := found(r, bleach, bleachNo), found(r, ammonia, phrases{}), found(r, acids, phrases{})
	al, px, ba := found(r, alcohols, phrases{}), found(r, peroxide, phrases{}), found(r, bases, phrases{})
	eo := found(r, oils, phrases{})

	// Mixes
	if len(bl) > 0 && len(am) > 0 {
		add("danger", "mix", "Never mix bleach and ammonia: together they make a poisonous gas (chloramine).", bl, am)
	}
	if len(bl) > 0 && len(ac) > 0 {
		add("danger", "mix", "Never mix bleach with vinegar, lemon or other acids: together they make chlorine gas.", bl, ac)
	}
	if len(bl) > 0 && len(al) > 0 {
		add("danger", "mix", "Never mix bleach and rubbing alcohol: together they make chloroform and other harmful gases.", bl, al)
	}
	if len(px) > 0 && len(ac) > 0 {
		add("danger", "mix", "Don't mix hydrogen peroxide and vinegar in one bottle: it makes peracetic acid, which can burn skin, eyes and lungs.", px, ac)
	}
	if len(bl) > 0 {
		add("caution", "mix", "Use bleach on its own with windows open, and never mix it with any other cleaner.", bl)
	}
	if len(ba) > 0 && len(ac) > 0 {
		add("info", "mix", "Baking soda and vinegar fizz: mix them in an open container and never close the bottle while it's fizzing (it can burst).", ba, ac)
	}
	if c := found(r, castile, phrases{}); len(c) > 0 && len(ac) > 0 {
		add("info", "mix", "Castile soap and vinegar (or lemon) cancel each other out: the soap turns curdy. Use them one after the other instead.", c, ac)
	}

	// Kids and swallowing
	oral := isOral(r)
	if w := found(r, strongEO, phrases{}); len(w) > 0 {
		add("danger", "kids", "Wintergreen, camphor and eucalyptus oils can poison a child who swallows even a little. Keep the bottle locked away; don't use these on young kids.", w)
	}
	if b := found(r, borax, phrases{}); len(b) > 0 {
		add("caution", "kids", "Borax is harmful if swallowed: keep it away from kids and pets.", b)
	}
	if oral {
		add("caution", "mouth", "Don't swallow. Kids under 6 need a grown-up watching and only a pea-sized amount.")
		if len(px) > 0 {
			add("caution", "mouth", "Use only 3% hydrogen peroxide, diluted, and spit it out.", px)
		}
		if len(al) > 0 {
			add("caution", "mouth", "Contains alcohol: not for young kids, who may swallow it.", al)
		}
		if strings.Contains(strings.ToLower(r.Title+" "+r.Course), "toothpaste") && len(found(r, fluoride, phrases{})) == 0 {
			add("info", "mouth", "Home-made toothpaste has no fluoride, which dentists recommend against cavities. Ask your dentist.")
		}
	}

	// Pets
	has := func(p string) bool {
		for _, x := range pets {
			if strings.EqualFold(strings.TrimSpace(x), p) {
				return true
			}
		}
		return false
	}
	if has("dog") {
		if x := found(r, xylitol, phrases{}); len(x) > 0 {
			add("danger", "pets", "Xylitol is very poisonous to dogs, even a lick of toothpaste. Keep it where the dog can't reach.", x)
		}
		if d := found(r, dogEO, phrases{}); len(d) > 0 {
			add("caution", "pets", "Some of these essential oils are toxic to dogs: don't use them on the dog, its bed or bowls, and air the room.", d)
		}
	}
	if has("cat") && len(found(r, catEO, phrases{})) > 0 {
		add("danger", "pets", "Essential oils are toxic to cats, even as vapor or on surfaces they walk on and lick. Avoid them in a home with cats.", found(r, catEO, phrases{}))
	}
	if has("bird") && (len(eo) > 0 || len(bl) > 0 || len(am) > 0) {
		add("caution", "pets", "Birds are very sensitive to fumes: use this far from the bird, with windows open.", eo, bl, am)
	}
	if (has("dog") || has("cat") || has("small")) && (len(bl) > 0 || len(am) > 0) {
		add("caution", "pets", "Keep pets out until the surface is rinsed and dry.", bl, am)
	}

	// Surfaces and storage
	if len(ac) > 0 && !oral {
		add("info", "surface", "Acids (vinegar, lemon, citric acid) etch marble, granite and other natural stone, and can dull waxed wood. Don't use it there.", ac)
	}
	if len(bl) > 0 {
		add("info", "surface", "Bleach takes the color out of fabric and can pit metal. Rinse after using.", bl)
	}
	if len(px) > 0 {
		add("info", "storage", "Keep hydrogen peroxide in a dark bottle: light breaks it down. It can lighten fabric and carpet: test a hidden spot first.", px)
	}
	if len(found(r, water, phrases{})) > 0 && len(al) == 0 {
		add("info", "storage", "Mixes with water can grow mold or bacteria: make small batches and use them within 1 to 2 weeks.")
	}
	if len(r.Ingredients) > 1 {
		add("info", "storage", "Label the bottle (what's in it and the date) and store it out of kids' reach. Never use a food or drink bottle.")
	}
	return out
}

// isOral: toothpaste, mouthwash and the like.
func isOral(r *recipe.Recipe) bool {
	s := strings.ToLower(r.Title + " " + r.Course)
	for _, w := range []string{"toothpaste", "tooth powder", "mouthwash", "mouth rinse", "oral", "gargle", "breath", "teeth", "tooth"} {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

func nonNil(l []int) []int {
	if l == nil {
		return []int{}
	}
	return l
}
