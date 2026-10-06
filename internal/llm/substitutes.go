package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// SubstituteSystem asks for stand-ins for a missing ingredient. The AI sees
// only the recipe's title, the line and what's in the kitchen, never anyone's
// rules: every idea is checked against the people eating afterwards.
const SubstituteSystem = `You suggest substitutes for a missing recipe ingredient. Reply with JSON only:
{"ideas": [{"to": "plain food name", "note": "how much to use, and what changes"}]}`

// SubstitutePrompt describes the missing line; onHand are foods in the kitchen.
func SubstitutePrompt(title, line string, onHand []string) string {
	p := fmt.Sprintf("The recipe %q needs %q and the cook doesn't have it. Suggest up to 4 common substitutes, "+
		"simplest first. In \"to\" give plain food names only (join several with \"and\").", title, line)
	if len(onHand) > 40 {
		onHand = onHand[:40]
	}
	if len(onHand) > 0 {
		p += "\nPrefer what's in the kitchen: " + strings.Join(onHand, ", ") + "."
	}
	return p
}

// Substitute is one idea from the AI.
type Substitute struct {
	To   string `json:"to"`
	Note string `json:"note"`
}

// ParseSubstitutes reads the AI's ideas (at most 4, short ones).
func ParseSubstitutes(out string) ([]Substitute, error) {
	var a struct {
		Ideas []Substitute `json:"ideas"`
	}
	if err := json.Unmarshal([]byte(jsonObject(out)), &a); err != nil {
		return nil, errors.New("the AI's answer wasn't valid JSON")
	}
	var ideas []Substitute
	for _, i := range a.Ideas {
		i.To, i.Note = strings.TrimSpace(i.To), strings.TrimSpace(i.Note)
		if i.To == "" || len(i.To) > 60 {
			continue
		}
		if len(i.Note) > 140 {
			i.Note = i.Note[:140]
		}
		if ideas = append(ideas, i); len(ideas) == 4 {
			break
		}
	}
	return ideas, nil
}
