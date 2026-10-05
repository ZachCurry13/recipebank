package api

import (
	"bytes"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func jpegBytes(w, h int) []byte {
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h)), nil)
	return buf.Bytes()
}

// A photo not yet moved out of the data folder is still served, and ?w=480
// serves a smaller preview from the cache.
func TestPhotoFoldersAndPreviews(t *testing.T) {
	c, srv := setup(t)
	srv.OldPhotos, srv.CacheDir = t.TempDir(), t.TempDir()
	name := "0123456789abcdef0123456789abcdef.jpg"
	if err := os.WriteFile(filepath.Join(srv.OldPhotos, name), jpegBytes(1600, 1200), 0o640); err != nil {
		t.Fatal(err)
	}
	get := func(path string) (int, []byte) {
		resp, err := c.http.Get(c.base + path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, b
	}
	code, full := get("/api/photos/" + name)
	if code != http.StatusOK {
		t.Fatalf("old-folder photo: %d", code)
	}
	code, small := get("/api/photos/" + name + "?w=480")
	cfg, _, err := image.DecodeConfig(bytes.NewReader(small))
	if code != http.StatusOK || err != nil || cfg.Width != 480 || len(small) >= len(full) {
		t.Fatalf("preview: %d %dx%d %v", code, cfg.Width, cfg.Height, err)
	}
	if code, _ := get("/api/photos/../recipebank.db"); code == http.StatusOK {
		t.Fatal("served a file outside the photos")
	}
}

func TestInside(t *testing.T) {
	for _, c := range []struct {
		dir, path string
		want      bool
	}{{"/data", "/data/photos", true}, {"/data", "/data2/photos", false}, {"/data", "/photos", false}} {
		if got := inside(filepath.FromSlash(c.dir), filepath.FromSlash(c.path)); got != c.want {
			t.Errorf("inside(%s, %s) = %v", c.dir, c.path, got)
		}
	}
}
