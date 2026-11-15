package identity

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func fixture(t *testing.T) Spec {
	t.Helper()
	dir := t.TempDir()
	binary := filepath.Join(dir, "binary")
	input := filepath.Join(dir, "fire")
	if err := os.WriteFile(binary, []byte("binary"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("pixels"), 0600); err != nil {
		t.Fatal(err)
	}
	return Spec{BinaryPath: binary, Settings: []byte("render settings"), Inputs: map[string]Input{"fire.steady": {Path: input, Choice: "000017.pfm"}}, FPS: 24, Frame: 17, Device: "device-0", Backend: "cpu", Width: 2, Height: 3}
}

func build(t *testing.T, s Spec) Key {
	t.Helper()
	k, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	return k
}
func tiny(t *testing.T, c uint8) []byte {
	t.Helper()
	im := image.NewNRGBA(image.Rect(0, 0, 2, 3))
	im.SetNRGBA(0, 0, color.NRGBA{R: c, A: 255})
	var b bytes.Buffer
	if err := png.Encode(&b, im); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestIdentity(t *testing.T) {
	s := fixture(t)
	want := build(t, s)
	if build(t, s) != want {
		t.Fatal("not deterministic")
	}
	mutations := []func(*Spec){
		func(s *Spec) { s.Settings = []byte("other") }, func(s *Spec) { s.FPS = 25 }, func(s *Spec) { s.Frame++ }, func(s *Spec) { s.Device = "device-1" }, func(s *Spec) { s.Backend = "gpu" }, func(s *Spec) { s.Width++ },
		func(s *Spec) {
			s.Inputs = map[string]Input{"fire.steady": {Path: s.Inputs["fire.steady"].Path, Choice: "000018.pfm"}}
		},
		func(s *Spec) { s.Inputs = map[string]Input{"fire.burst": s.Inputs["fire.steady"]} },
	}
	for i, mutate := range mutations {
		copy := s
		mutate(&copy)
		if build(t, copy) == want {
			t.Fatalf("mutation %d ignored", i)
		}
	}
	if err := os.WriteFile(s.Inputs["fire.steady"].Path, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if build(t, s) == want {
		t.Fatal("input bytes ignored")
	}
	want = build(t, s)
	if err := os.WriteFile(s.BinaryPath, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if build(t, s) == want {
		t.Fatal("binary bytes ignored")
	}
}

func TestInputValidation(t *testing.T) {
	s := fixture(t)
	for _, mutate := range []func(*Spec){func(s *Spec) { s.FPS = 0 }, func(s *Spec) { s.Frame = -1 }, func(s *Spec) { s.Backend = "" }, func(s *Spec) { s.Width = 0 }, func(s *Spec) { s.Inputs = map[string]Input{"": {Path: s.BinaryPath}} }, func(s *Spec) { s.BinaryPath = filepath.Dir(s.BinaryPath) }, func(s *Spec) { s.BinaryPath += "missing" }} {
		copy := s
		mutate(&copy)
		if _, err := Build(copy); err == nil {
			t.Fatal("accepted invalid spec")
		}
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(s.BinaryPath, link); err != nil {
		t.Fatal(err)
	}
	s.BinaryPath = link
	if _, err := Build(s); err == nil {
		t.Fatal("accepted symlink")
	}
}

func TestPublication(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	s := fixture(t)
	k := build(t, s)
	data := tiny(t, 10)
	if _, err := store.Verify(k); !errors.Is(err, ErrMissing) {
		t.Fatalf("missing: %v", err)
	}
	reused, err := store.Publish(k, data, false)
	if err != nil || reused {
		t.Fatalf("publish: %v %v", reused, err)
	}
	got, err := store.Verify(k)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("verify: %v", err)
	}
	reused, err = store.Publish(k, tiny(t, 20), false)
	if err != nil || !reused {
		t.Fatalf("reuse: %v %v", reused, err)
	}
	s.Device = "other"
	other := build(t, s)
	if _, err := store.Verify(other); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale verify: %v", err)
	}
	if _, err := store.Publish(other, data, false); !errors.Is(err, ErrConflict) {
		t.Fatalf("clobbered: %v", err)
	}
	if _, err := store.Publish(other, tiny(t, 20), true); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Verify(k); !errors.Is(err, ErrConflict) {
		t.Fatalf("old identity survives redraw: %v", err)
	}
	if _, err := store.Verify(other); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentPublication(t *testing.T) {
	dir := t.TempDir()
	s := fixture(t)
	k := build(t, s)
	s.Device = "other"
	other := build(t, s)
	data := tiny(t, 10)
	for _, different := range []bool{false, true} {
		t.Run(map[bool]string{false: "same", true: "different"}[different], func(t *testing.T) {
			sub := filepath.Join(dir, t.Name()[len("TestConcurrentPublication/"):])
			if err := os.Mkdir(sub, 0700); err != nil {
				t.Fatal(err)
			}
			a, err := Open(sub)
			if err != nil {
				t.Fatal(err)
			}
			defer a.Close()
			b, err := Open(sub)
			if err != nil {
				t.Fatal(err)
			}
			defer b.Close()
			keys := []Key{k, k}
			if different {
				keys[1] = other
			}
			stores := []*Store{a, b}
			var wg sync.WaitGroup
			errs := make([]error, 2)
			reuse := make([]bool, 2)
			for i := range stores {
				wg.Add(1)
				go func(i int) { defer wg.Done(); reuse[i], errs[i] = stores[i].Publish(keys[i], data, false) }(i)
			}
			wg.Wait()
			if different {
				if (errs[0] == nil) == (errs[1] == nil) {
					t.Fatalf("want one winner: %v", errs)
				}
				for _, err := range errs {
					if err != nil && !errors.Is(err, ErrConflict) {
						t.Fatal(err)
					}
				}
			} else {
				if errs[0] != nil || errs[1] != nil || reuse[0] == reuse[1] {
					t.Fatalf("same identity: %v %v", errs, reuse)
				}
			}
			winner := 0
			if errs[0] != nil {
				winner = 1
			}
			readFlat(t, sub, keys[winner], data)
		})
	}
}

func TestRejectPNG(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	k := build(t, fixture(t))
	for _, data := range [][]byte{[]byte("not PNG"), tiny(t, 1)[:20], append(tiny(t, 1), 1, 2, 3)} {
		if _, err := store.Publish(k, data, false); err == nil {
			t.Fatal("accepted corrupt PNG")
		}
	}
	s := fixture(t)
	s.Width = 3
	k = build(t, s)
	if _, err := store.Publish(k, tiny(t, 1), false); err == nil {
		t.Fatal("accepted wrong dimensions")
	}
}
