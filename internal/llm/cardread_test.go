package llm

import (
	"errors"
	"strings"
	"testing"
)

func TestParseTranscript(t *testing.T) {
	if _, err := ParseTranscript("NO RECIPE."); !errors.Is(err, ErrNoRecipe) {
		t.Fatalf("no recipe: %v", err)
	}
	if _, err := ParseTranscript("Soup"); err == nil {
		t.Fatal("a nearly empty copy was accepted")
	}
	got, err := ParseTranscript("```text\nSoup (2 serv.)\nIngredients:\n1 c. broth\n```")
	if err != nil || !strings.HasPrefix(got, "Soup (2 serv.)") || strings.Contains(got, "```") {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestTranscribePromptHints(t *testing.T) {
	if p := TranscribePrompt(nil); strings.Contains(p, "misread") || !strings.Contains(p, "c. = cup") {
		t.Fatalf("plain prompt: %s", p)
	}
	p := TranscribePrompt([]string{`"bell pepper flakes" was "red pepper flakes"`})
	if !strings.Contains(p, "red pepper flakes") {
		t.Fatalf("hints missing: %s", p)
	}
}
