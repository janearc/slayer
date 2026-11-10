package main

import (
	"bytes"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// A supplied linear frame isolates the command's selected display curve.
// Check the actual written PNG against the independent authored selection,
// including highlights that distinguish the two observed curve faults.
func TestSavedPNGUsesTheChosenLegacyToneCurve(t *testing.T) {
	var lights []maths.Light
	for _, value := range []float64{0, .1, .5, .7, .8, 1, 2, 4, 6, 7, 8, 9, 12, 16} {
		lights = append(lights, maths.Grey(value))
	}
	lights = append(lights, maths.RGB(.1, 1, 9), maths.RGB(9, 9, .25), maths.RGB(0, 8, 0))
	linear := frame.New(len(lights), 1)
	for x, light := range lights {
		linear.Set(x, 0, frame.Pixel{R: float32(light.R), G: float32(light.G), B: float32(light.B), Depth: float32(math.Inf(1))})
	}
	reference := (colour.Tone{Knee: .7, Top: 8}).Develop(linear)
	for _, alternate := range []colour.Tone{{Knee: 2.4, Top: 8}, {Knee: .7, Top: 9}} {
		if bytes.Equal(reference.Pix, alternate.Develop(linear).Pix) {
			t.Fatal("linear frame does not expose an observed display-curve fault")
		}
	}
	given := mustParse(t)
	given.out = filepath.Join(t.TempDir(), "chosen-tone.png")
	if err := save(linear, given, thing.NewWorld(), 0); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(given.out)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	actual, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	if actual.Bounds() != reference.Bounds() {
		t.Fatal("saved frame changed its actual image bounds")
	}
	for x := range lights {
		ar, ag, ab, aa := actual.At(x, 0).RGBA()
		wr, wg, wb, wa := reference.At(x, 0).RGBA()
		if ar != wr || ag != wg || ab != wb || aa != wa {
			t.Fatalf("saved pixel %d differs from the chosen display curve", x)
		}
	}
}
