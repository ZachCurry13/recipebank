package llm

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// BothFailed is what the photo AI and then the main AI said when neither
// could read the photos.
type BothFailed struct{ Photo, Main error }

func (e *BothFailed) Error() string {
	return fmt.Sprintf("the photo AI couldn't (%v), and neither could the main AI (%v)", e.Photo, e.Main)
}

func (e *BothFailed) Unwrap() error { return e.Main }

var (
	busyRe  = regexp.MustCompile(`\b(503|529)\b|overloaded|high demand|temporarily unavailable`)
	limitRe = regexp.MustCompile(`\b429\b`)
)

// Busy reports whether an online AI said it has too many requests for the
// moment (Gemini's 503 "high demand", Claude's 529 "overloaded"): worth
// asking again after a short wait.
func Busy(err error) bool {
	return err != nil && busyRe.MatchString(strings.ToLower(err.Error()))
}

// Limited reports whether an online AI said this key has used up its
// allowance for now (429: free plans limit requests per minute and per day).
func Limited(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return limitRe.MatchString(msg) || strings.Contains(msg, "quota") || strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "resource_exhausted") || strings.Contains(msg, "resource has been exhausted")
}

// NoPictures reports whether a photo model's picture reader (the "CLIP
// model" or projector Ollama loads next to it) wouldn't load: the model can't
// start, not even for text.
func NoPictures(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "clip model") || strings.Contains(msg, "mmproj")
}

// Explain says in plain words what went wrong with the AI and what to do,
// ending with what it said. ok is false when there's nothing plainer to say.
func Explain(err error) (msg string, ok bool) {
	var both *BothFailed
	if errors.As(err, &both) {
		photo, _ := explain(both.Photo)
		main, _ := explain(both.Main)
		return "Neither AI could do it. The AI for photos " + photo + " The main AI " + main, true
	}
	if what, ok := explain(err); ok {
		return "The AI " + what, true
	}
	return "", false
}

// explain finishes a sentence about one AI: "The AI …".
func explain(err error) (string, bool) {
	said := " (It said: " + truncate(err.Error(), 140) + ")"
	switch { // Ollama's own problems first: their messages carry long numbers
	case NoPictures(err):
		return "couldn't load the photo model's picture reader (the part that sees images) on Ollama. Update the Ollama app " +
			"and restart it. If that doesn't help, download the model again (Admin → AI → 🧹 Installed models: delete it, " +
			"then download it), or try another photo model such as gemma3:4b." + said, true
	case WontLoad(err):
		return "couldn't load its model (on Ollama, usually its graphics card couldn't start or is out of memory). " +
			"Restart the Ollama app (TrueNAS: Apps → ollama → Restart) and try again. If it keeps happening, another model may be " +
			"filling the card's memory, or the card isn't shared with the app: check the app's GPU setting, or use a smaller model." + said, true
	case Busy(err):
		return "is busy right now: the service has too many requests. Try again in a minute or two." + said, true
	case Limited(err):
		return "says this key has used up its allowance for now (free plans limit requests per minute and per day). " +
			"Wait a while, or check the plan on the service's website." + said, true
	}
	return "couldn't do it." + said, false
}
