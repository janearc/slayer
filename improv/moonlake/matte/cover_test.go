package matte

import (
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
)

// TestApplyBlendsOnlyBelowTheShore checks rows well above the shore keep the raw, rows well below take the sea, and the row at the shore is half and half.
func TestApplyBlendsOnlyBelowTheShore(t *testing.T) {
	const height = 20
	raw := Image{Width: 1, Height: height, Scale: -1, Pixels: make([]float32, height*3)}
	sea := Image{Width: 1, Height: height, Scale: -1, Pixels: make([]float32, height*3)}
	for i := range raw.Pixels {
		raw.Pixels[i] = 1
		sea.Pixels[i] = 2
	}
	// One bright pixel makes the moon (10, 10, 10); the dim pixels, summing to 3, are under half of 30.
	for c := range 3 {
		raw.Pixels[5*3+c] = 10
	}
	cfg := Config{Top: 10, Bottom: -10, Shore: 0, MoonRadius: 0.27}
	out, err := Apply(raw, sea, cfg)
	if err != nil {
		t.Fatal(err)
	}
	// The shore row is 10 * 20 / 20 = 10, with a feather of 4 rows either side.
	disc := math.Pi * math.Pow(0.27*math.Pi/180, 2)
	water := 2 * 10 * disc
	at := func(r int) float32 { return out.Pixels[(height-1-r)*3] }
	if at(0) != 1 || at(6) != 1 {
		t.Errorf("rows above the shore changed: %v %v", at(0), at(6))
	}
	if want := float32(0.5*1 + 0.5*water); at(10) != want {
		t.Errorf("shore row is %v, want half raw and half sea %v", at(10), want)
	}
	if want := float32(water); at(19) != want {
		t.Errorf("bottom row is %v, want the sea %v", at(19), want)
	}
}

// TestReadWriteRoundTrip checks an image written in either byte order reads back the same, with its scale.
func TestReadWriteRoundTrip(t *testing.T) {
	for _, scale := range []float64{-1, 1} {
		image := Image{Width: 2, Height: 2, Scale: scale, Pixels: []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12.5}}
		path := filepath.Join(t.TempDir(), "round.pfm")
		if err := Write(path, image); err != nil {
			t.Fatal(err)
		}
		got, err := Read(path)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, image) {
			t.Errorf("scale %v: read %+v, wrote %+v", scale, got, image)
		}
	}
}

// TestWriteLittleEndianBytes checks the bytes of a one-pixel file against the PFM layout worked out by hand.
func TestWriteLittleEndianBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "one.pfm")
	if err := Write(path, Image{Width: 1, Height: 1, Scale: -1, Pixels: []float32{1, 2, 4}}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// 1.0, 2.0 and 4.0 as little-endian float32 are 00 00 80 3f, 00 00 00 40 and 00 00 80 40.
	want := "PF\n1 1\n-1\n" + "\x00\x00\x80\x3f" + "\x00\x00\x00\x40" + "\x00\x00\x80\x40"
	if string(raw) != want {
		t.Fatalf("file is %q, want %q", raw, want)
	}
}

// TestWriteRefusesAMissingDirectory checks a path that cannot be created is reported.
func TestWriteRefusesAMissingDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "none", "out.pfm")
	if err := Write(path, Image{Width: 1, Height: 1, Pixels: []float32{1, 2, 3}}); err == nil {
		t.Fatal("write into a missing directory succeeded")
	}
}

// TestReadRefusals feeds Read a missing file, an empty one, a header cut short and a grey one.
func TestReadRefusals(t *testing.T) {
	dir := t.TempDir()
	if _, err := Read(filepath.Join(dir, "absent.pfm")); err == nil {
		t.Error("missing file accepted")
	}
	for name, data := range map[string]string{
		"empty":         "",
		"no newline":    "PF\n1 1\n-1",
		"grey":          "Pf\n1 1\n-1\nabcd",
		"not a pfm":     "P5\n1 1\n-1\nabcd",
		"words for dim": "PF\nwide tall\n-1\n",
	} {
		path := filepath.Join(dir, strings.ReplaceAll(name, " ", "_"))
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Read(path); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	grey := filepath.Join(dir, "grey")
	if _, err := Read(grey); err == nil || !strings.Contains(err.Error(), "not a colour pfm") {
		t.Errorf("grey: %v", err)
	}
}

// TestReadRefusesANamedPipe checks a file that is not a regular file is refused after its header is read.
func TestReadRefusesANamedPipe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pipe.pfm")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Skipf("no named pipes here: %v", err)
	}
	go func() {
		w, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			return
		}
		defer w.Close()
		io.WriteString(w, "PF\n1 1\n-1\n")
	}()
	if _, err := Read(path); err == nil || !strings.Contains(err.Error(), "not a regular pfm file") {
		t.Fatalf("err=%v", err)
	}
}

// TestBrightestSkipsDimPixels checks the moon is the mean of the pixels at least half as bright as the brightest.
func TestBrightestSkipsDimPixels(t *testing.T) {
	got := brightest([]float32{1, 1, 1, 4, 4, 4, 6, 6, 6, 0, 0, 0})
	if got != [3]float64{5, 5, 5} {
		t.Fatalf("brightest=%v, want (5, 5, 5)", got)
	}
}

// TestOrderFollowsTheSignOfTheScale checks positive scales are big-endian and others little-endian.
func TestOrderFollowsTheSignOfTheScale(t *testing.T) {
	if order(1) == order(-1) || order(0) != order(-1) {
		t.Fatal("byte order does not follow the scale's sign")
	}
}
