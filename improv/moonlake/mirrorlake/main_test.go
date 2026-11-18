package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// TestClampHoldsAValueInsideTheRange checks both ends and the inside.
func TestClampHoldsAValueInsideTheRange(t *testing.T) {
	cases := []struct{ v, n, want int }{
		{-3, 10, 0}, {0, 10, 0}, {4, 10, 4}, {9, 10, 9}, {10, 10, 9}, {99, 10, 9},
	}
	for _, c := range cases {
		if got := clamp(c.v, c.n); got != c.want {
			t.Errorf("clamp(%d, %d) = %d, want %d", c.v, c.n, got, c.want)
		}
	}
}

// TestOrderFollowsTheSignOfTheScale checks that a positive scale is big
// endian and a negative one little endian.
func TestOrderFollowsTheSignOfTheScale(t *testing.T) {
	if order(1) != binary.BigEndian {
		t.Error("a positive scale is not big endian")
	}
	if order(-1) != binary.LittleEndian {
		t.Error("a negative scale is not little endian")
	}
}

// TestWriteThenReadKeepsTheLight checks a pfm of two by one pixels comes
// back with its size, scale and values, in both byte orders.
func TestWriteThenReadKeepsTheLight(t *testing.T) {
	light := []float32{0.25, 0.5, 1, 2, 4, 8}
	for _, scale := range []float64{1, -1} {
		path := filepath.Join(t.TempDir(), "a.pfm")
		if err := write(path, 2, 1, scale, light); err != nil {
			t.Fatal(err)
		}
		width, height, gotScale, pixels, err := read(path)
		if err != nil {
			t.Fatal(err)
		}
		if width != 2 || height != 1 || gotScale != scale || len(pixels) != 6 {
			t.Fatalf("read %dx%d scale %v with %d values", width, height, gotScale, len(pixels))
		}
		for i := range light {
			if pixels[i] != light[i] {
				t.Errorf("scale %v value %d = %v, want %v", scale, i, pixels[i], light[i])
			}
		}
	}
}

// TestReadRefusesWhatIsNotAColourPfm checks a missing file, a grey pfm,
// a header that is not numbers and a file cut short.
func TestReadRefusesWhatIsNotAColourPfm(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"grey":      "Pf\n1 1\n-1\n\x00\x00\x00\x00",
		"header":    "PF\nwide tall\n-1\n",
		"truncated": "PF\n2 2\n-1\n\x00\x00\x00\x00",
	}
	for name, text := range cases {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, _, _, _, err := read(path); err == nil {
			t.Errorf("%s: read accepted it", name)
		}
	}
	if _, _, _, _, err := read(filepath.Join(dir, "missing")); err == nil {
		t.Error("read accepted a missing file")
	}
}

// TestWriteRefusesAMissingDirectory checks the create error.
func TestWriteRefusesAMissingDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no", "such", "a.pfm")
	if err := write(path, 1, 1, 1, []float32{1, 2, 3}); err == nil {
		t.Fatal("write made a file in a directory that is not there")
	}
}
