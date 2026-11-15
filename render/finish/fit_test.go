package finish

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/slayer/render/frame"
	"pgregory.net/rapid"
)

// near is whether two numbers are within a hair.
func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// fitFor is the fit of a job that takes the frames of a directory at a size.
func fitFor(t *testing.T, film string, w, h int) (Fit, error) {
	t.Helper()
	directory := t.TempDir()
	pfmIn(t, directory, "a.pfm", w, h, 1)
	plates, err := OpenPlates(directory)
	if err != nil {
		t.Fatal(err)
	}
	made := jobIn(t, t.TempDir(), "$schema: https://github.com/janearc/"+
		"slayer/schemas/job-v1.json\nscene: {frames: "+directory+"}\n"+
		film)
	return NewFit(made, plates)
}

// a landscape plate for a portrait film is scaled to cover it and its
// sides are cut: 1920x1080 to 1080x1920 takes the height to 1920, the
// scale 16/9, and cuts the width down to 607.5 pixels of the plate.
func TestALandscapePlateIsCutToPortrait(t *testing.T) {
	fit, err := fitFor(t, "film: {width: 1080, height: 1920}\n", 1920,
		1080)
	if err != nil {
		t.Fatal(err)
	}
	cut := fit.Cut()
	if !near(cut.Scale(), 16.0/9) || !near(cut.Across(), (1920-607.5)/2) ||
		!near(cut.Down(), 0) || !cut.Cuts() {
		t.Errorf("cut %v", cut)
	}
	if w, h := fit.Size(); w != 1080 || h != 1920 {
		t.Errorf("the film is %dx%d", w, h)
	}
	if !strings.Contains(cut.String(), "cut 656.2 pixels from the left") {
		t.Errorf("the cut says %q", cut)
	}
}

// and back: a portrait plate for a landscape film cuts its top and
// bottom.
func TestAPortraitPlateIsCutToLandscape(t *testing.T) {
	fit, err := fitFor(t, "film: {width: 1920, height: 1080}\n", 1080,
		1920)
	if err != nil {
		t.Fatal(err)
	}
	cut := fit.Cut()
	if !near(cut.Scale(), 16.0/9) || !near(cut.Across(), 0) ||
		!near(cut.Down(), (1920-607.5)/2) {
		t.Errorf("cut %v", cut)
	}
}

// a plate of the film's size is untouched, and a plate of the film's
// shape at another size is scaled and nothing is cut.
func TestAPlateOfTheFilmsShapeIsNotCut(t *testing.T) {
	same, err := fitFor(t, "film: {width: 8, height: 6}\n", 8, 6)
	if err != nil {
		t.Fatal(err)
	}
	if same.Cut().Cuts() || !near(same.Cut().Scale(), 1) {
		t.Errorf("a plate of the size was cut: %v", same.Cut())
	}
	plate := frame.New(8, 6)
	if got, _ := same.Apply(plate); got != plate {
		t.Error("a plate of the size was copied")
	}
	smaller, err := fitFor(t, "film: {width: 4, height: 3}\n", 8, 6)
	if err != nil {
		t.Fatal(err)
	}
	if smaller.Cut().Cuts() || !near(smaller.Cut().Scale(), 0.5) {
		t.Errorf("same shape, other size: %v", smaller.Cut())
	}
	if !strings.Contains(smaller.Cut().String(), "nothing cut") {
		t.Errorf("the words say %q", smaller.Cut())
	}
}

// a job that names no size keeps the plates' size; one that names half
// a size is refused.
func TestAJobWithNoSizeKeepsThePlatesAndHalfASizeIsRefused(t *testing.T) {
	keep, err := fitFor(t, "", 8, 6)
	if err != nil {
		t.Fatal(err)
	}
	if w, h := keep.Size(); w != 8 || h != 6 || keep.Cut().Cuts() {
		t.Errorf("kept %dx%d", w, h)
	}
	_, err = fitFor(t, "film: {width: 4}\n", 8, 6)
	if err == nil || !strings.Contains(err.Error(), "width or a height") {
		t.Errorf("half a size gave %v", err)
	}
}

// a plate of the wrong size for the fit is refused, with both sizes.
func TestAPlateOfAnotherSizeIsRefusedByTheFit(t *testing.T) {
	fit, err := fitFor(t, "film: {width: 4, height: 3}\n", 8, 6)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fit.Apply(frame.New(9, 6)); err == nil ||
		!strings.Contains(err.Error(), "9x6") {
		t.Errorf("a plate of another size gave %v", err)
	}
}

// a flat plate stays flat through any fit, down or up, and the fit is
// at the film's size: the light is carried, not invented.
func TestAFlatPlateStaysFlat(t *testing.T) {
	for _, size := range [][2]int{{4, 3}, {7, 11}, {16, 12}, {5, 20}} {
		film := "film: {width: " + strconv.Itoa(size[0]) + ", height: " +
			strconv.Itoa(size[1]) + "}\n"
		fit, err := fitFor(t, film, 8, 6)
		if err != nil {
			t.Fatal(err)
		}
		plate := frame.New(8, 6)
		for i := range plate.Pix {
			plate.Pix[i].R, plate.Pix[i].G, plate.Pix[i].B = 2, 3, 5
		}
		got, err := fit.Apply(plate)
		if err != nil {
			t.Fatal(err)
		}
		if got.W != size[0] || got.H != size[1] {
			t.Fatalf("fitted to %dx%d, want %v", got.W, got.H, size)
		}
		for _, pixel := range got.Pix {
			if math.Abs(float64(pixel.R)-2) > 1e-5 ||
				math.Abs(float64(pixel.G)-3) > 1e-5 ||
				math.Abs(float64(pixel.B)-5) > 1e-5 {
				t.Fatalf("a flat plate has %v at %v", pixel, size)
			}
		}
	}
}

// the kept part is the middle: a plate that is bright only in its
// middle third across is, cut to a narrower film, bright everywhere; one
// bright only at its left edge has none of it left.
func TestTheMiddleIsKept(t *testing.T) {
	plate := frame.New(9, 3)
	for y := range 3 {
		for x := range 9 {
			pixel := plate.At(x, y)
			if x >= 3 && x < 6 {
				pixel.G = 1
			}
			if x == 0 {
				pixel.R = 1
			}
			plate.Set(x, y, pixel)
		}
	}
	fit, err := fitFor(t, "film: {width: 3, height: 3}\n", 9, 3)
	if err != nil {
		t.Fatal(err)
	}
	got, err := fit.Apply(plate)
	if err != nil {
		t.Fatal(err)
	}
	for _, pixel := range got.Pix {
		if math.Abs(float64(pixel.G)-1) > 1e-5 || pixel.R != 0 {
			t.Fatalf("the middle was not kept: %v", pixel)
		}
	}
}

// the crop is centred and inside the plate, whatever the shapes.
func TestTheCropIsCentredAndInsideThePlate(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		width := rapid.IntRange(1, 4000).Draw(t, "plate width")
		height := rapid.IntRange(1, 4000).Draw(t, "plate height")
		wide := rapid.IntRange(1, 4000).Draw(t, "film width")
		high := rapid.IntRange(1, 4000).Draw(t, "film height")
		cut := covering(width, height, wide, high)
		kept := float64(wide) / cut.Scale()
		keptHigh := float64(high) / cut.Scale()
		if cut.Across() < -1e-9 || cut.Down() < -1e-9 {
			t.Fatalf("a negative cut: %v", cut)
		}
		if math.Abs(cut.Across()*2+kept-float64(width)) > 1e-6 ||
			math.Abs(cut.Down()*2+keptHigh-float64(height)) > 1e-6 {
			t.Fatalf("the window is not centred: %v", cut)
		}
		if cut.Across() > 1e-6 && cut.Down() > 1e-6 {
			t.Fatalf("both axes cut: %v", cut)
		}
	})
}
