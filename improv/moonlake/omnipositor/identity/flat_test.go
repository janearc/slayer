package identity

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readFlat(t *testing.T, dir string, k Key, want []byte) {
	t.Helper()
	path := filepath.Join(dir, fmt.Sprintf("%05d.png", k.frame))
	target, err := os.Readlink(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(target, ".generation-") || !strings.HasSuffix(target, "/frame.png") || strings.Count(target, "/") != 1 {
		t.Fatalf("not a direct relative PNG pointer: %q", target)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	im, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if im.Bounds().Dx() != k.width || im.Bounds().Dy() != k.height {
		t.Fatal("wrong flat dimensions")
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("flat pixels differ: %v", err)
	}
}

func TestFlatPNGPublication(t *testing.T) {
	for _, frame := range []int64{0, 17, 100000} {
		t.Run(fmt.Sprint(frame), func(t *testing.T) {
			dir := t.TempDir()
			store, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			spec := fixture(t)
			spec.Frame = frame
			k := build(t, spec)
			data := tiny(t, 1)
			if _, err := store.Publish(k, data, false); err != nil {
				t.Fatal(err)
			}
			readFlat(t, dir, k, data)
			replacement := tiny(t, 2)
			spec.Device = "redraw"
			other := build(t, spec)
			if _, err := store.Publish(other, replacement, true); err != nil {
				t.Fatal(err)
			}
			readFlat(t, dir, other, replacement)
		})
	}
}

func TestRefuseUnownedFlatSlots(t *testing.T) {
	for _, kind := range []string{"regular", "symlink", "generation-shaped", "copied-store-link", "directory"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			store, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			k := build(t, fixture(t))
			data := tiny(t, 1)
			path := filepath.Join(dir, fmt.Sprintf("%05d.png", k.frame))
			switch kind {
			case "regular":
				err = os.WriteFile(path, data, 0600)
			case "symlink":
				outside := filepath.Join(t.TempDir(), "pixels.png")
				if err := os.WriteFile(outside, data, 0600); err != nil {
					t.Fatal(err)
				}
				err = os.Symlink(outside, path)
			case "generation-shaped":
				err = os.Symlink(".generation-0123456789abcdef0123456789abcdef/frame.png", path)
			case "copied-store-link":
				if _, err := store.Publish(k, data, false); err != nil {
					t.Fatal(err)
				}
				target, e := os.Readlink(path)
				if e != nil {
					t.Fatal(e)
				}
				if e := os.Remove(path); e != nil {
					t.Fatal(e)
				}
				err = os.Symlink(target, path)
			case "directory":
				err = os.Mkdir(path, 0700)
			}
			if err != nil {
				t.Fatal(err)
			}
			before, err := os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, redraw := range []bool{false, true} {
				if reused, err := store.Publish(k, data, redraw); reused || !errors.Is(err, ErrCorrupt) {
					t.Fatalf("unowned slot accepted (redraw=%v): %v %v", redraw, reused, err)
				}
			}
			after, err := os.Lstat(path)
			if err != nil || !os.SameFile(before, after) {
				t.Fatalf("unowned slot replaced: %v", err)
			}
		})
	}
}

func TestLegacySlotLeftUntouched(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	k := build(t, fixture(t))
	legacy := filepath.Join(dir, fmt.Sprintf("frame-%012d", k.frame))
	target := ".generation-legacy"
	if err := os.Symlink(target, legacy); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Verify(k); !errors.Is(err, ErrMissing) {
		t.Fatalf("legacy slot was reused: %v", err)
	}
	data := tiny(t, 1)
	if _, err := store.Publish(k, data, false); err != nil {
		t.Fatal(err)
	}
	readFlat(t, dir, k, data)
	if got, err := os.Readlink(legacy); err != nil || got != target {
		t.Fatalf("legacy slot changed: %v", err)
	}
}

func TestOwnershipSurvivesReopenAndMissingGeneration(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	k := build(t, fixture(t))
	data := tiny(t, 1)
	if _, err := store.Publish(k, data, false); err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("%05d.png", k.frame)
	target, err := os.Readlink(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Dir(target)
	if err := os.RemoveAll(filepath.Join(dir, parent)); err != nil {
		t.Fatal(err)
	}
	store, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.Verify(k); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("missing generation: %v", err)
	}
	if _, err := store.Publish(k, data, false); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("missing generation reused: %v", err)
	}
	if _, err := store.Publish(k, data, true); err != nil {
		t.Fatal(err)
	}
	readFlat(t, dir, k, data)
}
