package files

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif" // photos may be GIFs
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

// Preview widths, in pixels: Library cards and the recipe page.
var Widths = map[int]bool{480: true, 1200: true}

// Thumb returns a JPEG copy of the photo at src, at most width pixels wide,
// made once and kept in cacheDir/thumbs/<width>/. Photo names never change,
// so a preview is never stale; deleting the cache only costs the time to
// make them again.
func Thumb(cacheDir, src string, width int) (string, error) {
	if !Widths[width] {
		return "", fmt.Errorf("no preview size %d", width)
	}
	name := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src)) + ".jpg"
	dir := filepath.Join(cacheDir, "thumbs", fmt.Sprint(width))
	out := filepath.Join(dir, name)
	if _, err := os.Stat(out); err == nil {
		return out, nil
	}
	f, err := os.Open(src)
	if err != nil {
		return "", err
	}
	// A photo already this small is served as it is.
	if cfg, _, err := image.DecodeConfig(f); err == nil && cfg.Width <= width {
		f.Close()
		return src, nil
	}
	if _, err := f.Seek(0, 0); err != nil {
		f.Close()
		return "", err
	}
	img, _, err := image.Decode(f)
	f.Close()
	if err != nil {
		return "", fmt.Errorf("read photo: %w", err) // e.g. WebP: the original is served instead
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(dir, ".thumb-*")
	if err != nil {
		return "", err
	}
	err = jpeg.Encode(tmp, shrink(img, width), &jpeg.Options{Quality: 82})
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), out)
	}
	if err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return out, nil
}

// PruneThumbs removes previews whose photo is gone (keep reports whether a
// photo name, without its extension, is still used).
func PruneThumbs(cacheDir string, keep func(base string) bool) {
	sizes, _ := os.ReadDir(filepath.Join(cacheDir, "thumbs"))
	for _, s := range sizes {
		dir := filepath.Join(cacheDir, "thumbs", s.Name())
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if base := strings.TrimSuffix(e.Name(), ".jpg"); !keep(base) {
				_ = os.Remove(filepath.Join(dir, e.Name()))
			}
		}
	}
}

// shrink scales img down to at most w pixels wide, averaging the source
// pixels under each new one (from NovelCheck's covers).
func shrink(img image.Image, w int) image.Image {
	b := img.Bounds()
	if b.Dx() <= w {
		return img
	}
	h := max(1, b.Dy()*w/b.Dx())
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		y0, y1 := b.Min.Y+y*b.Dy()/h, b.Min.Y+(y+1)*b.Dy()/h
		for x := 0; x < w; x++ {
			x0, x1 := b.Min.X+x*b.Dx()/w, b.Min.X+(x+1)*b.Dx()/w
			var r, g, bl, n uint64
			for sy := y0; sy < max(y1, y0+1); sy++ {
				for sx := x0; sx < max(x1, x0+1); sx++ {
					cr, cg, cb, _ := img.At(sx, sy).RGBA()
					r, g, bl, n = r+uint64(cr), g+uint64(cg), bl+uint64(cb), n+1
				}
			}
			i := dst.PixOffset(x, y)
			dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2], dst.Pix[i+3] = uint8(r/n>>8), uint8(g/n>>8), uint8(bl/n>>8), 255
		}
	}
	return dst
}

// Fit returns the photo as a JPEG no longer than maxSide pixels on its long
// side (the AI's copy: smaller photos fit small AI models). A photo already
// small enough is returned unchanged.
func Fit(data []byte, maxSide int) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("read photo: %w", err)
	}
	b := img.Bounds()
	long := max(b.Dx(), b.Dy())
	if long <= maxSide {
		return data, nil
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, shrink(img, max(1, b.Dx()*maxSide/long)), &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
