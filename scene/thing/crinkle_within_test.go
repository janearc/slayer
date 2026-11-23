package thing

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The csv table of pixel widths the fading test tries, narrowest first.
const crinkleWithinPixelsFile = "testdata/crinkle-within-pixels.csv"

// crinkleWithinNearest is the largest tilt, in radians, that wrinkles
// show to a pixel a millimetre across anywhere on the first grid. It
// holds that a pixel a metre across sees a flat face and that a lone
// ray sees what Crinkled does.
func crinkleWithinNearest(t *testing.T) float64 {
	t.Helper()
	up, look := crinkleUp(), crinkleLook(t)
	wrinkles := crinkleWrinkles(t)
	columns := pickedCount(t, "crinkle-columns")
	farPixel := picked(t, "crinkle-within-far-pixel")
	nearPixel := picked(t, "crinkle-within-near-pixel")
	var near float64
	for i := range pickedCount(t, "crinkle-places") {
		point := crinklePoint(t, i, columns)
		if far := wrinkles.CrinkledWithin(point, up, look, farPixel); far != up {
			t.Fatalf("a pixel of a metre sees wrinkles of 5 cm: %v", far)
		}
		seen := wrinkles.CrinkledWithin(point, up, look, nearPixel)
		near = math.Max(near, crinkleTilt(seen, up))
	}
	return near
}

// A pixel a metre across sees a flat face where the wrinkles are a few
// centimetres, a lone ray sees what Crinkled does, and a near pixel
// still sees the wrinkles.
func TestAFarWallIsFlatAndANearOneStillWrinkled(t *testing.T) {
	near := crinkleWithinNearest(t)
	if !(near > picked(t, "crinkle-within-near-tilt")) {
		t.Errorf("near, the wrinkles tilt a face by only %v radians", near)
	}
}

// As a pixel widens the wrinkles fade, so a wider pixel never tilts a
// face more than a narrower one, and the fade never turns it the other
// way.
func TestAPixelBetweenFadesTheTiltAndNeverReversesIt(t *testing.T) {
	up, look := crinkleUp(), crinkleLook(t)
	wrinkles := crinkleWrinkles(t)
	point := maths.V(picked(t, "crinkle-within-point-x"), nought,
		picked(t, "crinkle-within-point-z"))
	tilts := []float64{}
	for _, pixel := range crinkleColumn(t, crinkleWithinPixelsFile, "pixel") {
		faded := wrinkles.CrinkledWithin(point, up, look, pixel)
		tilts = append(tilts, crinkleTilt(faded, up))
	}
	tolerance := picked(t, "crinkle-within-fade-tolerance")
	for index := unity; index < len(tilts); index++ {
		if tilts[index] > tilts[index-unity]+tolerance {
			t.Errorf("a wider pixel tilts more: %v", tilts)
		}
	}
}
