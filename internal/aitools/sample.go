package aitools

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// The speed test's made-up recipe card. It's drawn here, so there's no
// photo of anyone's real card in RecipeBank.
var sampleLines = []string{
	"Oat Cookies",
	"",
	"1 cup butter, softened",
	"1 cup brown sugar",
	"2 eggs",
	"1 tsp vanilla",
	"1 1/2 cups flour",
	"1 tsp baking soda",
	"3 cups rolled oats",
	"",
	"Cream the butter and sugar. Beat in the",
	"eggs and vanilla. Stir in the flour, soda",
	"and oats. Bake at 350F for 10 minutes.",
}

// sampleWord is a word only a model that read the card can answer with.
const sampleWord = "oats"

// SampleText is the card as text, for timing the text work.
func SampleText() string {
	var b bytes.Buffer
	for _, l := range sampleLines {
		b.WriteString(l + "\n")
	}
	return b.String()
}

// SampleCard draws the card as a JPEG, like a photo of a printed card.
func SampleCard() []byte {
	const scale, pad, lineH = 3, 14, 15 // basicfont is 7×13 pixels; drawn small, then enlarged
	w, h := 300, pad*2+lineH*len(sampleLines)
	small := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(small, small.Bounds(), &image.Uniform{color.RGBA{250, 246, 232, 255}}, image.Point{}, draw.Src)
	d := font.Drawer{Dst: small, Src: image.NewUniform(color.RGBA{30, 30, 60, 255}), Face: basicfont.Face7x13}
	for i, l := range sampleLines {
		d.Dot = fixed.P(pad, pad+lineH*(i+1)-3)
		d.DrawString(l)
	}
	big := image.NewRGBA(image.Rect(0, 0, w*scale, h*scale))
	for y := 0; y < h*scale; y++ {
		for x := 0; x < w*scale; x++ {
			big.Set(x, y, small.At(x/scale, y/scale))
		}
	}
	var out bytes.Buffer
	_ = jpeg.Encode(&out, big, &jpeg.Options{Quality: 85})
	return out.Bytes()
}
