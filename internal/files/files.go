// Package files decides where RecipeBank keeps its files and looks after
// them: the database in /data, photos in /photos and re-creatable previews
// in /cache, each its own TrueNAS dataset when mounted, else inside /data.
package files

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Dir picks a folder: the environment variable when set, else mount when
// that folder exists (a dataset mounted into the container), else sub
// inside the data folder.
func Dir(env, mount, dataDir, sub string) string {
	if v := strings.TrimSpace(os.Getenv(env)); v != "" {
		return v
	}
	if st, err := os.Stat(mount); err == nil && st.IsDir() {
		return mount
	}
	return filepath.Join(dataDir, sub)
}

// Writable reports whether files can be created in dir (making it if needed).
func Writable(dir string) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".write-test-*")
	if err != nil {
		return err
	}
	f.Close()
	return os.Remove(f.Name())
}

// MoveAll moves every file from one folder to another (photos, when a
// /photos dataset is added to an install that kept them in /data). Each
// file is copied, checked and only then removed from the old folder; a file
// that fails stays where it was (the app still finds it there).
func MoveAll(from, to string) (moved int) {
	entries, err := os.ReadDir(from)
	if err != nil || filepath.Clean(from) == filepath.Clean(to) {
		return 0
	}
	if err := os.MkdirAll(to, 0o750); err != nil {
		log.Printf("can't create %s: %v", to, err)
		return 0
	}
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		src, dst := filepath.Join(from, e.Name()), filepath.Join(to, e.Name())
		if err := moveOne(src, dst); err != nil {
			log.Printf("couldn't move %s to %s (it stays where it is): %v", src, to, err)
			continue
		}
		moved++
	}
	return moved
}

func moveOne(src, dst string) error {
	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if di, err := os.Stat(dst); err == nil {
		if di.Size() != si.Size() {
			return fmt.Errorf("a different file with that name is already there")
		}
		return os.Remove(src) // copied before, but the old one wasn't removed yet
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".moving-*")
	if err != nil {
		return err
	}
	n, err := io.Copy(tmp, in)
	if err == nil {
		err = tmp.Sync()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil && n != si.Size() {
		err = fmt.Errorf("copied %d of %d bytes", n, si.Size())
	}
	if err == nil {
		err = os.Rename(tmp.Name(), dst)
	}
	if err != nil {
		os.Remove(tmp.Name())
		return err
	}
	in.Close()
	return os.Remove(src)
}
