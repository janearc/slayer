package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"slices"
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
