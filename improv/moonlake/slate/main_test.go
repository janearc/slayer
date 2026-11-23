package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Bar 97.1 is 218.53 seconds into the song: at 24 frames a second that
// is frame 5245 of the film, 3:38.53, and a fire lit there and struck
// there says so.
func TestTheSlateWritesTheFramesPlaceInTheFilm(t *testing.T) {
	cues := map[string]string{"bar": "97.1", "fire_lit": "0.0400",
		"fire_since": "0.0400,1.7100"}
	got, err := wording(cues, 218.53, 24)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"frame 5245", "3:38.53", "bar 97.1",
		"fire lit +0.04 s  struck +0.04 s"}
	if !slices.Equal(got, want) {
		t.Errorf("the slate says %q, want %q", got, want)
	}
}

// Before the fire is lit the slate says so; the strikes are not read.
func TestTheSlateSaysTheFireIsNotLitBeforeItsCue(t *testing.T) {
	cues := map[string]string{"bar": "intro", "fire_lit": "-1.0000",
		"fire_since": ""}
	got, err := wording(cues, 5, 24)
	if err != nil {
		t.Fatal(err)
	}
	if want := "fire not lit"; got[3] != want || got[0] != "frame 120" ||
		got[1] != "0:05.00" {
		t.Errorf("the slate says %q", got)
	}
}

// A cue sheet with no fire_lit, or a strike that is not a number, is
// refused by name, not drawn as nonsense.
func TestTheSlateRefusesCuesItCannotRead(t *testing.T) {
	for _, cues := range []map[string]string{
		{"bar": "97.1"},
		{"bar": "97.1", "fire_lit": "NaN"},
		{"bar": "97.1", "fire_lit": "0.5", "fire_since": "soon"},
		{"bar": "97.1", "fire_lit": "0.5", "fire_since": "NaN"},
	} {
		if _, err := wording(cues, 218.53, 24); err == nil {
			t.Errorf("cues %v were read", cues)
		}
	}
}

// stamp draws over the top left of the png and leaves the rest of it as
// it was.
func TestTheSlateDrawsOnlyItsCorner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "frame.png")
	grey := color.RGBA{90, 90, 90, 255}
	img := image.NewRGBA(image.Rect(0, 0, 640, 360))
	for i := 0; i < len(img.Pix); i += 4 {
		copy(img.Pix[i:], []uint8{grey.R, grey.G, grey.B, grey.A})
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
	file.Close()
	if err := stamp(path, []string{"frame 5245"}, 218.53); err != nil {
		t.Fatal(err)
	}
	file, err = os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	drawn, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	if got := color.RGBAModel.Convert(drawn.At(600, 340)); got != grey {
		t.Errorf("the far corner is %v, was %v", got, grey)
	}
	changed := false
	for y := 0; y < 60 && !changed; y++ {
		for x := 0; x < 200; x++ {
			if color.RGBAModel.Convert(drawn.At(x, y)) != grey {
				changed = true
				break
			}
		}
	}
	if !changed {
		t.Error("nothing was drawn in the top left")
	}
}

// writeGrey writes a grey png of the given size and returns its path.
func writeGrey(t *testing.T, dir string, width, height int) string {
	t.Helper()
	path := filepath.Join(dir, "frame.png")
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestReadCuesKeepsKeyValueLines checks that readCues keeps what follows
// the first equals sign and skips lines without one.
func TestReadCuesKeepsKeyValueLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cue.env")
	text := "bar=97.1\nfire_lit=0.04\nnoise without a sign\nnote=a=b\nempty=\n"
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readCues(path)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"bar": "97.1", "fire_lit": "0.04", "note": "a=b", "empty": ""}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for key, value := range want {
		if have, ok := got[key]; !ok || have != value {
			t.Errorf("%s = %q (%v), want %q", key, have, ok, value)
		}
	}
}

// TestReadCuesRefusesAMissingFile checks that a file that is not there is
// an error.
func TestReadCuesRefusesAMissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "none.env")
	if _, err := readCues(missing); err == nil {
		t.Error("a cue file that is not there was read")
	}
}

// TestReadCuesRefusesALineTooLongToScan checks that a scanner failure is
// returned, not swallowed.
func TestReadCuesRefusesALineTooLongToScan(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cue.env")
	long := make([]byte, 200000)
	for i := range long {
		long[i] = 'x'
	}
	body := append([]byte("bar="), long...)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readCues(path); err == nil {
		t.Error("a line over the scanner's limit was read")
	}
}

// TestStampRefusals checks each way stamp turns a frame away and that the
// frame is left as it was.
func TestStampRefusals(t *testing.T) {
	dir := t.TempDir()
	if err := stamp(filepath.Join(dir, "none.png"), []string{"x"}, 1); err == nil {
		t.Error("a frame that is not there was stamped")
	}

	notPNG := filepath.Join(dir, "text.png")
	if err := os.WriteFile(notPNG, []byte("not a picture"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := stamp(notPNG, []string{"x"}, 1); err == nil || !strings.Contains(err.Error(), notPNG) {
		t.Errorf("text taken for a png, or unnamed: %v", err)
	}

	small := writeGrey(t, t.TempDir(), 64, 40)
	before, _ := os.ReadFile(small)
	if err := stamp(small, []string{"x"}, 1); err == nil || !strings.Contains(err.Error(), "rows") {
		t.Errorf("a frame too short for the rows: %v", err)
	}
	after, _ := os.ReadFile(small)
	if !bytes.Equal(before, after) {
		t.Error("a refused frame was changed")
	}
}

// TestStampRefusesAFolderItCannotWriteIn checks that a frame in a folder
// the slate cannot create a file in is an error, with the frame as it was.
func TestStampRefusesAFolderItCannotWriteIn(t *testing.T) {
	dir := t.TempDir()
	path := writeGrey(t, dir, 640, 360)
	before, _ := os.ReadFile(path)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0o700)
	if f, err := os.CreateTemp(dir, ".probe-*"); err == nil {
		f.Close()
		t.Skip("known: the folder is writable despite its mode, as for a superuser")
	}
	if err := stamp(path, []string{"x"}, 1); err == nil {
		t.Error("stamped in a folder that cannot be written")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Error("the frame changed")
	}
}

// TestStampLeavesNoTemporaryFile checks that a good stamp renames over the
// frame and leaves no other file in the folder.
func TestStampLeavesNoTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	path := writeGrey(t, dir, 640, 360)
	if err := stamp(path, []string{"frame 1"}, 1); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "frame.png" {
		t.Errorf("folder holds %v, want only frame.png", entries)
	}
}
