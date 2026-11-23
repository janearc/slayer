package identity

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCorruptionNeverReused(t *testing.T) {
	for _, what := range []string{"pixels", "manifest", "digest", "decode", "dimensions", "missing-pixels", "missing-manifest", "symlink-pixels", "trailing-manifest"} {
		t.Run(what, func(t *testing.T) {
			dir := t.TempDir()
			store, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			k := build(t, fixture(t))
			data := tiny(t, 1)
			if _, err := store.Publish(k, data, false); err != nil {
				t.Fatal(err)
			}
			slotPath := filepath.Join(dir, slot(k))
			target, err := os.Readlink(slotPath)
			if err != nil {
				t.Fatal(err)
			}
			pixels := filepath.Join(dir, target)
			metadata := filepath.Join(dir, filepath.Dir(target), "manifest.json")
			raw, err := os.ReadFile(metadata)
			if err != nil {
				t.Fatal(err)
			}
			var m manifest
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatal(err)
			}
			write := func(path string, b []byte) {
				t.Helper()
				if err := os.WriteFile(path, b, 0600); err != nil {
					t.Fatal(err)
				}
			}
			switch what {
			case "pixels":
				write(pixels, tiny(t, 2))
			case "manifest":
				write(metadata, []byte("{"))
			case "digest":
				m.SHA256 = strings.Repeat("0", 64)
			case "decode":
				broken := []byte("invalid PNG")
				write(pixels, broken)
				m.SHA256 = digest(broken)
				m.Bytes = len(broken)
			case "dimensions":
				m.Width++
			case "missing-pixels":
				if err := os.Remove(pixels); err != nil {
					t.Fatal(err)
				}
			case "missing-manifest":
				if err := os.Remove(metadata); err != nil {
					t.Fatal(err)
				}
			case "symlink-pixels":
				outside := filepath.Join(t.TempDir(), "pixels")
				write(outside, data)
				if err := os.Remove(pixels); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, pixels); err != nil {
					t.Fatal(err)
				}
			case "trailing-manifest":
				trailer := append(raw, []byte(" {}")...)
				write(metadata, trailer)
			}
			if what == "digest" || what == "decode" || what == "dimensions" {
				raw, err = json.Marshal(m)
				if err != nil {
					t.Fatal(err)
				}
				write(metadata, raw)
			}
			if _, err := store.Verify(k); !errors.Is(err, ErrCorrupt) {
				t.Fatalf("corrupt verify: %v", err)
			}
			if reuse, err := store.Publish(k, data, false); reuse || !errors.Is(err, ErrCorrupt) {
				t.Fatalf("corrupt reuse: %v %v", reuse, err)
			}
			if _, err := store.Publish(k, data, true); err != nil {
				t.Fatal(err)
			}
			if got, err := store.Verify(k); err != nil || !bytes.Equal(got, data) {
				t.Fatalf("recovery: %v", err)
			}
		})
	}
}

func TestInterruptedGeneration(t *testing.T) {
	for _, stage := range []string{"empty", "pixels", "complete", "dangling-slot", "incomplete-slot"} {
		t.Run(stage, func(t *testing.T) {
			dir := t.TempDir()
			store, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			k := build(t, fixture(t))
			data := tiny(t, 1)
			generation := ".generation-interrupted"
			if err := os.Mkdir(filepath.Join(dir, generation), 0700); err != nil {
				t.Fatal(err)
			}
			if stage != "empty" {
				if err := os.WriteFile(filepath.Join(dir, generation, "frame.png"), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if stage == "complete" {
				raw, _ := json.Marshal(manifest{Version: 1, Key: k.digest, Frame: k.frame, Width: k.width, Height: k.height, Bytes: len(data), SHA256: digest(data)})
				if err := os.WriteFile(filepath.Join(dir, generation, "manifest.json"), raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			occupied := stage == "dangling-slot" || stage == "incomplete-slot"
			if occupied {
				if _, err := store.Publish(k, data, false); err != nil {
					t.Fatal(err)
				}
				slotPath := filepath.Join(dir, slot(k))
				target, err := os.Readlink(slotPath)
				if err != nil {
					t.Fatal(err)
				}
				committed := filepath.Join(dir, filepath.Dir(target))
				if stage == "dangling-slot" {
					if err := os.RemoveAll(committed); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := os.Remove(filepath.Join(committed, "manifest.json")); err != nil {
						t.Fatal(err)
					}
				}
			}
			_, err = store.Verify(k)
			expected := ErrMissing
			if occupied {
				expected = ErrCorrupt
			}
			if !errors.Is(err, expected) {
				t.Fatalf("interrupted state: %v", err)
			}
			reuse, err := store.Publish(k, data, occupied)
			if err != nil || reuse {
				t.Fatalf("recover: %v %v", reuse, err)
			}
			if _, err := store.Verify(k); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDirectorySafetyAndPrivacy(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(dir, "output")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if store, err := Open(link); err == nil {
		store.Close()
		t.Fatal("symlink output accepted")
	}
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	spec := fixture(t)
	spec.Settings = []byte("secret access token")
	k := build(t, spec)
	data := tiny(t, 1)
	sentinel := filepath.Join(outside, "sentinel")
	if err := os.WriteFile(sentinel, []byte("untouched"), 0600); err != nil {
		t.Fatal(err)
	}
	slotPath := filepath.Join(dir, slot(k))
	if err := os.Symlink(outside, slotPath); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(k, data, false); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("unsafe slot: %v", err)
	}
	if _, err := store.Publish(k, data, true); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("unsafe redraw: %v", err)
	}
	if err := os.Remove(slotPath); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(k, data, false); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(sentinel); err != nil || string(got) != "untouched" {
		t.Fatalf("outside changed: %v", err)
	}
	target, err := os.Readlink(slotPath)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, filepath.Dir(target), "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{string(spec.Settings), spec.BinaryPath, spec.Inputs["fire.steady"].Path, spec.Device} {
		if bytes.Contains(raw, []byte(secret)) {
			t.Fatal("manifest leaks specification")
		}
	}
	generationPath := filepath.Join(dir, filepath.Dir(target))
	info, err := os.Stat(generationPath)
	if err != nil || info.Mode().Perm() != 0700 {
		t.Fatalf("generation not private: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Verify(k); err == nil {
		t.Fatal("closed store accepted")
	}
}

func TestInputOrderAndLocationIndependence(t *testing.T) {
	s := fixture(t)
	p := s.Inputs["fire.steady"].Path
	s.Inputs = map[string]Input{"fire.steady": {Path: p, Choice: "17"}, "score": {Path: s.BinaryPath}}
	k := build(t, s)
	s.Inputs = map[string]Input{"score": {Path: s.BinaryPath}, "fire.steady": {Path: p, Choice: "17"}}
	if build(t, s) != k {
		t.Fatal("map order changes key")
	}
	moved := filepath.Join(t.TempDir(), "different-path")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(moved, b, 0600); err != nil {
		t.Fatal(err)
	}
	s.Inputs["fire.steady"] = Input{Path: moved, Choice: "17"}
	if build(t, s) != k {
		t.Fatal("farm asset location changes identity")
	}
}
