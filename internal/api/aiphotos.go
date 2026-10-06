package api

import (
	"context"
	"net/http"

	"github.com/zachcurry13/recipebank/internal/files"
	"github.com/zachcurry13/recipebank/internal/llm"
)

// aiPhotoSides: the AI gets a copy of each photo this long on its long side,
// smaller again each time the AI answers that it doesn't fit (a home AI
// server often has room for only about 4,000 tokens, photos included).
var aiPhotoSides = []int{1280, 896, 640}

// askPhotos sends photos to the vision model, made smaller while they don't
// fit, and returns the model that answered.
func (s *Server) askPhotos(ctx context.Context, prompt string, originals []llm.Image, parse func(string) error) (string, error) {
	for _, side := range aiPhotoSides {
		images := make([]llm.Image, len(originals))
		for i, im := range originals {
			images[i] = im
			if small, err := files.Fit(im.Data, side); err == nil && len(small) != len(im.Data) {
				images[i] = llm.Image{Data: small, MediaType: "image/jpeg"}
			}
		}
		model, err := llm.AskImages(ctx, s.Store, prompt, images, parse)
		if err == nil || !llm.TooLong(err) {
			return model, err
		}
	}
	return "", errPhotoTooBig
}

// readImages decodes 1 to max photos sent as data: URLs; false means an
// error was already written.
func readImages(w http.ResponseWriter, list []string, max int) ([]llm.Image, bool) {
	if len(list) == 0 || len(list) > max {
		writeErr(w, http.StatusBadRequest, "add 1 to "+itoa(int64(max))+" photos")
		return nil, false
	}
	var out []llm.Image
	for _, d := range list {
		data, mt, err := decodeDataURL(d)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return nil, false
		}
		out = append(out, llm.Image{Data: data, MediaType: mt})
	}
	return out, true
}
