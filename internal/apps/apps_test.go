package apps

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var tinyJPEG = []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00")

func gz(t *testing.T, v any) []byte {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	json.NewEncoder(w).Encode(v)
	w.Close()
	return b.Bytes()
}

func zipOf(t *testing.T, files map[string][]byte) []byte {
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for name, data := range files {
		f, _ := w.Create(name)
		f.Write(data)
	}
	w.Close()
	return b.Bytes()
}

func readBytes(t *testing.T, data []byte) (string, []Found, error) {
	p := filepath.Join(t.TempDir(), "export")
	os.WriteFile(p, data, 0o600)
	return Read(p)
}

func TestPaprika(t *testing.T) {
	pie := map[string]any{"name": "Apple pie", "ingredients": "For the crust:\n2 cups flour\n1 cup butter\n\nFor the filling:\n6 apples",
		"directions": "1. Make the crust.\n2. Fill it.\n\nBake 45 minutes.", "servings": "Serves 6-8", "prep_time": "30 mins",
		"cook_time": "1 hr 15 min", "categories": []string{"Desserts", "Fall"}, "source": "Grandma", "rating": 5,
		"photo_data": base64.StdEncoding.EncodeToString(tinyJPEG)}
	soup := map[string]any{"name": "Soup", "ingredients": "1 onion", "directions": "Simmer.", "source_url": "https://example.com/soup"}
	app, found, err := readBytes(t, zipOf(t, map[string][]byte{"Apple pie.paprikarecipe": gz(t, pie), "Soup.paprikarecipe": gz(t, soup)}))
	if err != nil || app != "Paprika" || len(found) != 2 {
		t.Fatalf("%s %v %d", app, err, len(found))
	}
	var r Found
	for _, f := range found {
		if f.Recipe.Title == "Apple pie" {
			r = f
		}
	}
	rc := r.Recipe
	if len(rc.Ingredients) != 3 || rc.Ingredients[0].Section != "For the crust" || rc.Ingredients[2].Section != "For the filling" ||
		len(rc.Steps) != 3 || rc.Steps[0].Text != "Make the crust." || rc.Servings != 6 || rc.YieldText != "Serves 6-8" ||
		rc.PrepMin != 30 || rc.CookMin != 75 || rc.Course != "dessert" || rc.Rating != 5 || rc.SourceNote != "From Paprika (Grandma)" ||
		!bytes.Equal(r.Photo, tinyJPEG) {
		t.Fatalf("pie: %+v", rc)
	}
	// One recipe on its own is a gzip file.
	if app, found, err := readBytes(t, gz(t, soup)); err != nil || app != "Paprika" || found[0].Recipe.SourceURL != "https://example.com/soup" {
		t.Fatalf("one: %s %v %+v", app, err, found)
	}
}

func TestTandoor(t *testing.T) {
	rec := map[string]any{"name": "Chili", "servings": 6, "servings_text": "bowls", "working_time": 20, "waiting_time": 60,
		"keywords": []map[string]string{{"name": "Dinner"}},
		"steps": []map[string]any{{"name": "", "instruction": "Brown the beef.\nAdd everything and simmer.", "ingredients": []map[string]any{
			{"is_header": true, "note": "Chili"},
			{"food": map[string]string{"name": "ground beef"}, "unit": map[string]string{"name": "lb"}, "amount": 2, "note": "lean"},
			{"original_text": "2 cans kidney beans, drained", "food": map[string]string{"name": "kidney beans"}},
			{"food": map[string]string{"name": "salt"}, "no_amount": true}}}}}
	recJSON, _ := json.Marshal(rec)
	inner := zipOf(t, map[string][]byte{"recipe.json": recJSON, "image.jpg": tinyJPEG})
	app, found, err := readBytes(t, zipOf(t, map[string][]byte{"1.zip": inner}))
	if err != nil || app != "Tandoor" || len(found) != 1 {
		t.Fatalf("%s %v %d", app, err, len(found))
	}
	rc := found[0].Recipe
	lines := []string{}
	for _, in := range rc.Ingredients {
		lines = append(lines, in.Section+"|"+in.Line)
	}
	if strings.Join(lines, ";") != "Chili|2 lb ground beef, lean;Chili|2 cans kidney beans, drained;Chili|salt" ||
		len(rc.Steps) != 2 || rc.Servings != 6 || rc.YieldText != "6 bowls" || rc.TotalMin != 80 || rc.Course != "main" ||
		!bytes.Equal(found[0].Photo, tinyJPEG) {
		t.Fatalf("chili: %v %+v", lines, rc)
	}
}

func TestMealie(t *testing.T) {
	rec := map[string]any{"name": "Pancakes", "recipeYield": "4 servings", "totalTime": "1 hour 15 minutes", "prepTime": "PT10M",
		"orgURL": "https://example.com/pancakes", "recipeCategory": []map[string]string{{"name": "Breakfast"}},
		"recipeIngredient": []map[string]any{{"originalText": "1 cup flour", "title": "Batter"},
			{"quantity": 2, "unit": map[string]string{"name": "tbsp"}, "food": map[string]string{"name": "sugar"}, "note": ""},
			{"display": "1 egg"}},
		"recipeInstructions": []map[string]string{{"text": "Mix."}, {"text": "Cook on a griddle."}},
		"notes":              []map[string]string{{"title": "Tip", "text": "Rest the batter."}}}
	recJSON, _ := json.Marshal(rec)
	app, found, err := readBytes(t, zipOf(t, map[string][]byte{"recipes/pancakes/pancakes.json": recJSON,
		"recipes/pancakes/images/original.webp": []byte("RIFF....WEBP")}))
	if err != nil || app != "Mealie" || len(found) != 1 {
		t.Fatalf("%s %v %d", app, err, len(found))
	}
	rc := found[0].Recipe
	if len(rc.Ingredients) != 3 || rc.Ingredients[1].Line != "2 tbsp sugar" || rc.Ingredients[0].Section != "Batter" ||
		len(rc.Steps) != 2 || rc.Servings != 4 || rc.TotalMin != 75 || rc.PrepMin != 10 || rc.Course != "breakfast" ||
		rc.Notes != "Tip: Rest the batter." || found[0].Photo == nil {
		t.Fatalf("pancakes: %+v", rc)
	}

	// A Mealie backup: the database's tables and the photos beside them.
	db := map[string]any{
		"recipes":             []map[string]any{{"id": "r1", "name": "Tacos", "recipe_yield": "8", "total_time": "30 minutes"}},
		"recipes_ingredients": []map[string]any{{"recipe_id": "r1", "position": 1, "note": "", "quantity": 8, "food_id": "f2"}, {"recipe_id": "r1", "position": 0, "quantity": 1, "unit_id": "u1", "food_id": "f1"}},
		"recipe_instructions": []map[string]any{{"recipe_id": "r1", "position": 1, "text": "Fill the tortillas."}, {"recipe_id": "r1", "position": 0, "text": "Brown the turkey."}},
		"ingredient_foods":    []map[string]string{{"id": "f1", "name": "ground turkey"}, {"id": "f2", "name": "tortillas"}},
		"ingredient_units":    []map[string]string{{"id": "u1", "name": "lb"}},
	}
	dbJSON, _ := json.Marshal(db)
	app, found, err = readBytes(t, zipOf(t, map[string][]byte{"database.json": dbJSON, "data/recipes/r1/images/original.webp": []byte("RIFF....WEBP")}))
	if err != nil || app != "Mealie" || len(found) != 1 {
		t.Fatalf("backup: %s %v %d", app, err, len(found))
	}
	rc = found[0].Recipe
	if len(rc.Ingredients) != 2 || rc.Ingredients[0].Line != "1 lb ground turkey" || rc.Steps[0].Text != "Brown the turkey." ||
		rc.Servings != 8 || rc.TotalMin != 30 || found[0].Photo == nil || rc.SourceNote != "From Mealie" {
		t.Fatalf("tacos: %+v", rc)
	}
}

func TestRecipeJSONAndUnknown(t *testing.T) {
	ld := `{"@context":"https://schema.org","@type":"Recipe","name":"Bread","recipeIngredient":["3 cups flour","1 tsp yeast"],
		"recipeInstructions":[{"@type":"HowToSection","name":"Dough","itemListElement":[{"@type":"HowToStep","text":"Mix."},{"text":"Knead."}]},"Bake."],
		"recipeYield":["1 loaf"],"cookTime":"PT35M"}`
	app, found, err := readBytes(t, []byte(ld))
	if err != nil || app != "Recipe JSON" || len(found) != 1 {
		t.Fatalf("%s %v", app, err)
	}
	rc := found[0].Recipe
	if len(rc.Steps) != 3 || rc.Steps[0].Section != "Dough" || rc.Steps[2].Text != "Bake." || rc.CookMin != 35 || rc.Servings != 1 {
		t.Fatalf("bread: %+v", rc)
	}
	for _, junk := range [][]byte{[]byte("hello"), zipOf(t, map[string][]byte{"notes.txt": []byte("hi")}), []byte(`{"name":"no ingredients"}`)} {
		if _, _, err := readBytes(t, junk); !errors.Is(err, ErrUnknown) {
			t.Fatalf("%q: %v", junk[:2], err)
		}
	}
}

func TestMinutesAndServings(t *testing.T) {
	for in, want := range map[string]int{"15 mins": 15, "1 hr 30 min": 90, "PT1H5M": 65, "45": 45, "2 hours": 120, "": 0, "overnight": 0} {
		if got := minutesOf(in); got != want {
			t.Errorf("%q: %d, want %d", in, got, want)
		}
	}
	if n, y := servings("12 cookies"); n != 12 || y != "12 cookies" {
		t.Errorf("cookies: %v %q", n, y)
	}
	if n, y := servings("4"); n != 4 || y != "" {
		t.Errorf("4: %v %q", n, y)
	}
}
