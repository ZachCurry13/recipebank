package files

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func TestDir(t *testing.T) {
	data, mount := t.TempDir(), t.TempDir()
	t.Setenv("RB_TEST_DIR", "")
	if got := Dir("RB_TEST_DIR", filepath.Join(mount, "missing"), data, "photos"); got != filepath.Join(data, "photos") {
		t.Errorf("no mount: %s", got)
	}
	if got := Dir("RB_TEST_DIR", mount, data, "photos"); got != mount {
		t.Errorf("mounted: %s", got)
	}
	t.Setenv("RB_TEST_DIR", "/elsewhere")
	if got := Dir("RB_TEST_DIR", mount, data, "photos"); got != "/elsewhere" {
		t.Errorf("env: %s", got)
	}
}

func TestMoveAll(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	os.WriteFile(filepath.Join(from, "a.jpg"), []byte("photo a"), 0o640)
	os.WriteFile(filepath.Join(from, "b.jpg"), []byte("photo b"), 0o640)
	os.WriteFile(filepath.Join(to, "b.jpg"), []byte("photo b"), 0o640) // copied before
	os.WriteFile(filepath.Join(from, "c.jpg"), []byte("mine"), 0o640)
	os.WriteFile(filepath.Join(to, "c.jpg"), []byte("someone else's"), 0o640)
	if n := MoveAll(from, to); n != 2 {
		t.Fatalf("moved %d, want 2", n)
	}
	if b, _ := os.ReadFile(filepath.Join(to, "a.jpg")); string(b) != "photo a" {
		t.Errorf("a.jpg: %q", b)
	}
	if _, err := os.Stat(filepath.Join(from, "a.jpg")); !os.IsNotExist(err) {
		t.Error("a.jpg still in the old folder")
	}
	if b, _ := os.ReadFile(filepath.Join(from, "c.jpg")); string(b) != "mine" {
		t.Error("c.jpg (a clash) should stay in the old folder untouched")
	}
}

func TestThumb(t *testing.T) {
	dir, cache := t.TempDir(), t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 1600, 900))
	for x := 0; x < 1600; x++ {
		img.Set(x, 450, color.RGBA{200, 50, 50, 255})
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, nil)
	src := filepath.Join(dir, "abc.jpg")
	os.WriteFile(src, buf.Bytes(), 0o640)

	out, err := Thumb(cache, src, 480)
	if err != nil {
		t.Fatal(err)
	}
	f, _ := os.Open(out)
	cfg, _, err := image.DecodeConfig(f)
	f.Close()
	if err != nil || cfg.Width != 480 || cfg.Height != 270 {
		t.Fatalf("preview %dx%d, %v", cfg.Width, cfg.Height, err)
	}
	if _, err := Thumb(cache, src, 333); err == nil {
		t.Error("made a preview of a size that isn't offered")
	}
	PruneThumbs(cache, func(string) bool { return false })
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Error("preview of a removed photo was kept")
	}
}

func TestThumbOfSmallPhotoIsTheOriginal(t *testing.T) {
	dir, cache := t.TempDir(), t.TempDir()
	var buf bytes.Buffer
	jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 300, 200)), nil)
	src := filepath.Join(dir, "small.jpg")
	os.WriteFile(src, buf.Bytes(), 0o640)
	if out, err := Thumb(cache, src, 480); err != nil || out != src {
		t.Fatalf("got %s, %v; want the original", out, err)
	}
}

func TestFit(t *testing.T) {
	var buf bytes.Buffer
	jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1000, 2000)), nil)
	out, err := Fit(buf.Bytes(), 640)
	if err != nil {
		t.Fatal(err)
	}
	cfg, _, _ := image.DecodeConfig(bytes.NewReader(out))
	if cfg.Height != 640 || cfg.Width != 320 {
		t.Fatalf("fitted to %dx%d, want 320x640", cfg.Width, cfg.Height)
	}
	if same, _ := Fit(out, 1280); !bytes.Equal(same, out) {
		t.Error("a small photo was re-encoded")
	}
}

func TestDataDir(t *testing.T) {
	legacy, mount := t.TempDir(), t.TempDir()
	if got := DataDir("", legacy, filepath.Join(mount, "missing")); got != legacy {
		t.Errorf("nothing mounted: %s", got)
	}
	if got := DataDir("", legacy, mount); got != mount {
		t.Errorf("/config mounted: %s", got)
	}
	os.WriteFile(filepath.Join(legacy, "recipebank.db"), []byte("db"), 0o640)
	if got := DataDir("", legacy, mount); got != legacy {
		t.Errorf("an existing /data database must win: %s", got)
	}
	if got := DataDir("/x", legacy, mount); got != "/x" {
		t.Errorf("env: %s", got)
	}
}
