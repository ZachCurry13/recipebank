package llm

import (
	"errors"
	"strings"
)

// Reading a recipe from photos happens in two steps: the vision model copies
// the card as plain text (what it does best, and a small answer that fits a
// home AI), then a text step organizes the copy. The copy is also read
// line by line without the AI, so amounts can't get lost on the way.

const transcribeBase = `These photos show one recipe: a handwritten card, a cookbook page or a clipping
(several photos may be the front and back, or pages in order). Copy it as plain text, exactly as written:
- First the title line, as written (keep anything like "(4 serv.)").
- Then a line "Ingredients:" and every ingredient on its own line, keeping its amount and abbreviations
  exactly as written, for example "1 1/2 c. diced onion" or "3 tbsp. olive oil".
- If the ingredients are in two columns, copy the whole left column top to bottom first, then the whole
  right column. Every dash or line on the card is one ingredient: don't leave any out.
- Then a line "Instructions:" and the instructions.
- Write [?] right after any word or amount you can't read clearly.
- Don't add headings, words or ingredients that aren't on the card.
If the photos don't show a recipe, reply NO RECIPE.
Reply with the copied text only.`

// Abbreviations on handwritten cards, always given to the reader.
const abbreviations = `Abbreviations on cards: c. = cup, T. or tbsp. = tablespoon, t. or tsp. = teaspoon, lb. = pound,
oz. = ounce, pkg. = package, serv. = servings, qt. = quart, pt. = pint.`

// TranscribePrompt is the vision step's instruction, with hints learned from
// the family's earlier corrections ("bell pepper flakes" was "red pepper flakes").
func TranscribePrompt(hints []string) string {
	p := transcribeBase + "\n" + abbreviations
	if len(hints) > 0 {
		p += "\nIn this family's handwriting these were misread before, so look twice at similar words, " +
			"but always copy what is really written: " + strings.Join(hints, "; ") + "."
	}
	return p
}

// CardText is the text step's message: organize the copy, keeping amounts.
func CardText(transcript string) string {
	return "This text was copied from a photo of a recipe card or page. Keep every ingredient line with its amount " +
		"exactly as copied. A [?] marks something hard to read: keep that line, leave out the [?], and set \"unsure\": true.\n\n" + transcript
}

// ParseTranscript checks the vision step's answer.
func ParseTranscript(out string) (string, error) {
	t := strings.TrimSpace(out)
	t = strings.TrimPrefix(strings.TrimPrefix(t, "```text"), "```")
	t = strings.TrimSpace(strings.TrimSuffix(t, "```"))
	if t == "" || (len(t) < 40 && strings.Contains(strings.ToUpper(t), "NO RECIPE")) {
		return "", ErrNoRecipe
	}
	if len(t) < 20 {
		return "", errors.New("the AI copied almost nothing from the photo")
	}
	return t, nil
}
