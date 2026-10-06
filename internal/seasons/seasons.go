// Package seasons defines the seasonal shelves and feast days (Advent
// baking, Lent Fridays, summer grilling…): when each is in season and the
// words that find its recipes. Movable feasts follow Easter and Advent each
// year (the calendar is from NovelCheck).
package seasons

import (
	"time"

	"github.com/zachcurry13/recipebank/internal/recipe"
	"github.com/zachcurry13/recipebank/internal/safety"
)

// Season is one seasonal shelf.
type Season struct {
	Key   string `json:"key"`
	Icon  string `json:"icon"`
	Name  string `json:"name"`
	Area  string `json:"area"`  // kitchen or home
	Theme string `json:"theme"` // what the AI is asked for
	// Diet, when set, is what every recipe on the shelf must be (Lent Fridays: meatless).
	Diet  string `json:"-"`
	words safety.Words
	span  func(year int) (from, to time.Time) // inclusive days
}

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

// fixed is a span of calendar days; one ending before it starts runs into the next year.
func fixed(m1 time.Month, d1 int, m2 time.Month, d2 int) func(int) (time.Time, time.Time) {
	return func(y int) (time.Time, time.Time) {
		from, to := day(y, m1, d1), day(y, m2, d2)
		if to.Before(from) {
			to = day(y+1, m2, d2)
		}
		return from, to
	}
}

// fromEaster is a span counted in days from Easter Sunday.
func fromEaster(a, b int) func(int) (time.Time, time.Time) {
	return func(y int) (time.Time, time.Time) { e := Easter(y); return e.AddDate(0, 0, a), e.AddDate(0, 0, b) }
}

// Easter is Easter Sunday of year (the Gregorian computus).
func Easter(year int) time.Time {
	a, b, c := year%19, year/100, year%100
	d, e := b/4, b%4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i, k := c/4, c%4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	return day(year, time.Month(month), (h+l-7*m+114)%31+1)
}

// AdventSunday is the first Sunday of Advent: the Sunday from November 27 to December 3.
func AdventSunday(year int) time.Time {
	d := day(year, time.November, 27)
	for d.Weekday() != time.Sunday {
		d = d.AddDate(0, 0, 1)
	}
	return d
}

// Thanksgiving is the US Thanksgiving: the fourth Thursday of November.
func Thanksgiving(year int) time.Time {
	d := day(year, time.November, 1)
	for d.Weekday() != time.Thursday {
		d = d.AddDate(0, 0, 1)
	}
	return d.AddDate(0, 0, 21)
}

func s(key, icon, name, area, theme string, span func(int) (time.Time, time.Time), words []string, not ...string) Season {
	return Season{Key: key, Icon: icon, Name: name, Area: area, Theme: theme, span: span, words: safety.NewWords(words, not)}
}

// All seasonal shelves and feasts, in calendar order from the new year.
var All = []Season{
	s("epiphany", "👑", "Epiphany", "kitchen", "Epiphany (Three Kings) treats: king cake, galette des rois, rosca de reyes",
		fixed(time.January, 1, time.January, 6), []string{"king cake", "galette des rois", "rosca", "three kings", "epiphany"}),
	s("candlemas", "🕯️", "Candlemas crêpes", "kitchen", "Candlemas (February 2) crêpes and pancakes",
		fixed(time.January, 28, time.February, 2), []string{"crepe", "crepes", "pancake", "galette"}),
	s("valentines", "💝", "Valentine's", "kitchen", "Valentine's treats and a special dinner: chocolate, strawberries, red velvet",
		fixed(time.February, 7, time.February, 14), []string{"chocolate", "heart", "strawberry", "red velvet", "fondue", "truffle", "valentine"}),
	s("mardigras", "🎭", "Mardi Gras", "kitchen", "Mardi Gras and Fat Tuesday: pancakes, paczki, king cake, beignets, gumbo, jambalaya",
		fromEaster(-53, -47), []string{"pancake", "paczki", "king cake", "beignet", "gumbo", "jambalaya", "doughnut", "donut", "fat tuesday", "mardi gras"}),
	{Key: "lent", Icon: "🐟", Name: "Lent Fridays", Area: "kitchen", Diet: "meatless",
		Theme: "meatless meals for Fridays in Lent: fish, eggs, beans, soups, cheese and vegetables", span: fromEaster(-46, -3)},
	s("stpatrick", "☘️", "St. Patrick's Day", "kitchen", "St. Patrick's Day: soda bread, corned beef and cabbage, colcannon, Irish stew, shepherd's pie",
		fixed(time.March, 12, time.March, 17), []string{"soda bread", "corned beef", "colcannon", "irish stew", "shepherd's pie", "cottage pie", "cabbage", "irish"}),
	s("stjoseph", "🪚", "St. Joseph's Day", "kitchen", "St. Joseph's Day (March 19) table: zeppole, sfinge, fava beans, breads and meatless dishes",
		fixed(time.March, 15, time.March, 19), []string{"zeppole", "sfinge", "fava", "st joseph", "saint joseph", "st. joseph"}),
	s("springclean", "🧹", "Spring cleaning", "home", "spring cleaning: all-purpose sprays, glass and window cleaners, scrubs, laundry and freshening",
		fixed(time.March, 20, time.April, 30), []string{"cleaner", "spray", "scrub", "window", "glass", "all purpose", "degreaser", "laundry", "freshener"}),
	s("easter", "🐣", "Easter", "kitchen", "Easter dinner and breads: ham, lamb, hot cross buns, Easter bread, deviled eggs, carrot cake",
		fromEaster(-6, 7), []string{"ham", "lamb", "hot cross bun", "deviled egg", "easter", "paska", "kulich", "babka", "carrot cake", "quiche", "asparagus"}),
	s("grilling", "🔥", "Summer grilling", "kitchen", "summer grilling and cookouts: burgers, kebabs, ribs, corn on the cob, potato salad, coleslaw",
		fixed(time.May, 20, time.September, 7), []string{"grill", "grilled", "grilling", "bbq", "barbecue", "burger", "kebab", "skewer", "corn on the cob",
			"potato salad", "coleslaw", "smoked", "ribs", "hot dog", "lemonade", "popsicle"}),
	s("school", "🎒", "Back-to-school lunches", "kitchen", "back-to-school lunchbox and after-school snacks kids can help make",
		fixed(time.August, 10, time.September, 15), []string{"lunchbox", "lunch box", "muffin", "granola", "wrap", "sandwich", "energy bite", "snack", "pinwheel"}),
	s("harvest", "🍎", "Fall harvest", "kitchen", "fall harvest cooking: apples, pumpkin, squash, soups, stews, cider and pies",
		fixed(time.September, 22, time.November, 30), []string{"pumpkin", "apple", "squash", "butternut", "cinnamon", "stew", "chili", "cider", "pie",
			"sweet potato", "pear", "cranberry", "maple"}),
	s("halloween", "🎃", "Halloween", "kitchen", "Halloween treats and spooky dinners: pumpkin, caramel apples, monster cookies",
		fixed(time.October, 15, time.October, 31), []string{"pumpkin", "caramel apple", "halloween", "monster", "mummy", "spooky", "ghost", "candy corn"}),
	{Key: "thanksgiving", Icon: "🦃", Name: "Thanksgiving", Area: "kitchen",
		Theme: "Thanksgiving dinner: turkey, stuffing, cranberry sauce, mashed potatoes, gravy, green beans, pumpkin and pecan pie",
		span: func(y int) (time.Time, time.Time) {
			t := Thanksgiving(y)
			return t.AddDate(0, 0, -14), t.AddDate(0, 0, 1)
		},
		words: safety.NewWords([]string{"turkey", "stuffing", "dressing", "cranberry", "pumpkin pie", "sweet potato", "green bean casserole",
			"mashed potato", "gravy", "pecan pie", "dinner roll", "thanksgiving"}, nil)},
	s("advent", "🕯️", "Advent & Christmas baking", "kitchen", "Advent and Christmas baking: cookies, gingerbread, stollen, shortbread, fudge, toffee",
		func(y int) (time.Time, time.Time) { return AdventSunday(y), day(y, time.December, 24) },
		[]string{"cookie", "gingerbread", "stollen", "fruitcake", "shortbread", "spritz", "linzer", "pfeffernusse", "candy cane", "peppermint bark",
			"fudge", "toffee", "panettone", "yule log", "snickerdoodle", "biscotti", "speculaas", "eggnog", "mulled"}),
	s("stnicholas", "🎅", "St. Nicholas", "kitchen", "St. Nicholas Day (December 6): speculaas, gingerbread, chocolate coins, mandarins",
		fixed(time.December, 1, time.December, 6), []string{"speculaas", "speculoos", "spekulatius", "gingerbread", "pfeffernusse", "st nicholas", "saint nicholas"}),
	s("stlucy", "👑", "St. Lucy", "kitchen", "St. Lucy's Day (December 13): saffron buns (lussekatter) and gingersnaps",
		fixed(time.December, 9, time.December, 13), []string{"saffron", "lussekatter", "lucia", "gingersnap"}),
	s("christmas", "🎄", "Christmas dinner", "kitchen", "Christmas dinner and breakfast: roast, ham, prime rib, stuffing, cranberry, pudding, cinnamon rolls",
		fixed(time.December, 18, time.January, 6), []string{"christmas", "yule", "ham", "prime rib", "roast", "stuffing", "cranberry", "pudding",
			"mince pie", "cinnamon roll", "eggnog", "wassail"}),
	s("winter", "🥣", "Winter comfort food", "kitchen", "warming winter food: soups, stews, chili, casseroles, pot pies, roasts and hot chocolate",
		fixed(time.December, 1, time.February, 28), []string{"soup", "stew", "chili", "casserole", "pot pie", "roast", "braise", "braised", "hot chocolate", "cocoa", "chowder"}),
}

// Find returns the season with key.
func Find(key string) (Season, bool) {
	for _, s := range All {
		if s.Key == key {
			return s, true
		}
	}
	return Season{}, false
}

// In reports whether now falls in the season (in any year's span, since some run over New Year).
func (s Season) In(now time.Time) bool {
	t := day(now.Year(), now.Month(), now.Day())
	for y := now.Year() - 1; y <= now.Year(); y++ {
		from, to := s.span(y)
		if !t.Before(from) && !t.After(to) {
			return true
		}
	}
	return false
}

// Current lists the seasons in season at now.
func Current(now time.Time) []Season {
	var out []Season
	for _, s := range All {
		if s.In(now) {
			out = append(out, s)
		}
	}
	return out
}

// Fits reports whether a recipe belongs on the shelf.
func (s Season) Fits(r *recipe.Recipe) bool {
	if r.Area != s.Area {
		return false
	}
	if s.Diet != "" {
		return r.Course != "dessert" && safety.Check(r, safety.Person{HeatMax: -1, Diets: []string{s.Diet}}).Status == safety.OK
	}
	return s.words.In(r.Title + " \n " + r.Summary + " \n " + r.Course + " \n " + r.Text())
}
