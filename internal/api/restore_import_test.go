package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// When the AI shortens pasted lines (dropping "half milk chocolate"), the
// draft gets the person's own lines back, so a milk allergy is still caught.
func TestPastedLinesKeptWhole(t *testing.T) {
	c, srv := setup(t)
	ai := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		answer := `{"title":"Cookies","ingredients":[{"line":"1 cup butter"},{"line":"1 1/2 cups chocolate chips"},{"line":"340g"},{"line":"2 cups flour"}],
			"steps":[{"text":"Mix and bake 10 minutes."}]}`
		resp, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": answer}}}})
		w.Write(resp)
	}))
	defer ai.Close()
	useAI(srv.Store, ai.URL)
	c.do("POST", "/api/people", map[string]any{"name": "Kid", "heat_max": -1,
		"rules": []map[string]string{{"kind": "allergy", "key": "milk", "severity": "allergic"}}}, nil)

	paste := "Cookies (24):\n1 cup butter\n1 1/2 cups chocolate chips (I use half milk chocolate) 340g\n2 cups flour (more as needed)\n\nMix and bake 10 minutes."
	var draft struct {
		Recipe struct {
			Ingredients []struct{ Line, Note string }
		}
		Verdicts []struct{ Status string }
	}
	if code := c.do("POST", "/api/import/text", map[string]string{"text": paste}, &draft); code != 200 {
		t.Fatalf("import: %d", code)
	}
	ings := draft.Recipe.Ingredients
	if len(ings) != 3 || ings[1].Line != "1 1/2 cups chocolate chips (I use half milk chocolate) 340g" || ings[2].Line != "2 cups flour (more as needed)" {
		t.Fatalf("lines: %+v", ings)
	}
	if len(draft.Verdicts) != 1 || draft.Verdicts[0].Status != "no" {
		t.Fatalf("milk chocolate must be a no for a milk allergy: %+v", draft.Verdicts)
	}
}
