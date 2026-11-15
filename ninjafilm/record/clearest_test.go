package record_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/record"
	"pgregory.net/rapid"
)

// tenfold is the base of a density. It is defined: a density of one
// passes a tenth.
const tenfold = 10

// A record at its clearest holds its curve's least density in each lamp
// its image blocks, and none in the others.
func TestARecordsClearestIsItsFloorInItsImage(t *testing.T) {
	rapid.Check(t, clearestIsTheFloor)
}

func clearestIsTheFloor(t *rapid.T) {
	window := drawnLamps().Draw(t, "window")
	image := drawnLamps().Draw(t, "image")
	least := rapid.Float64Range(floorLeast, floorMost).Draw(t, "least")
	tone := toneOf(t, least, least+roomMost, slopeMost)
	made := recordOf(t, window, image, acrossLeast, tone)
	if got, want := made.Clearest(), image.Scale(least); got != want {
		t.Fatalf("an image of %v with a floor of %v holds %v", image, least,
			got)
	}
}

// A density passes ten to its minus in each lamp: none passes all, one
// passes a tenth.
func TestADensityPassesTenToItsMinus(t *testing.T) {
	rapid.Check(t, densityPasses)
}

func densityPasses(t *rapid.T) {
	density := lampsUpTo(t, "density", roomMost)
	got := record.Passed(density)
	want := maths.RGB(math.Pow(tenfold, -density.R),
		math.Pow(tenfold, -density.G), math.Pow(tenfold, -density.B))
	if !within(got.R, want.R, toleranceLoose) ||
		!within(got.G, want.G, toleranceLoose) ||
		!within(got.B, want.B, toleranceLoose) {
		t.Fatalf("a density of %v passes %v, want %v", density, got, want)
	}
	if record.Passed(maths.Light{}) != maths.Grey(unity) {
		t.Fatal("no density does not pass all the light")
	}
}

// Anti: a density in one lamp dims only that lamp. The others pass
// whole.
func TestADensityInOneLampDimsNoOther(t *testing.T) {
	rapid.Check(t, densityStaysInItsLamp)
}

func densityStaysInItsLamp(t *rapid.T) {
	dense := rapid.Float64Range(lampLeast, roomMost).Draw(t, "dense")
	got := record.Passed(maths.RGB(none, dense, none))
	if got.R != unity || got.B != unity || !(got.G < unity) {
		t.Fatalf("a density of %v in green passes %v", dense, got)
	}
}

// Anti: a record's clearest does not come from its window. Two records
// of one image and floor, seeing different lamps, are as clear.
func TestARecordsClearestIsNotItsWindow(t *testing.T) {
	rapid.Check(t, clearestIgnoresTheWindow)
}

func clearestIgnoresTheWindow(t *rapid.T) {
	image := drawnLamps().Draw(t, "image")
	least := rapid.Float64Range(floorLeast, floorMost).Draw(t, "least")
	tone := toneOf(t, least, least+roomMost, slopeMost)
	one := recordOf(t, drawnLamps().Draw(t, "one"), image, acrossLeast,
		tone)
	other := recordOf(t, drawnLamps().Draw(t, "other"), image,
		acrossLeast, tone)
	if one.Clearest() != other.Clearest() {
		t.Fatalf("two windows over one image clear to %v and %v",
			one.Clearest(), other.Clearest())
	}
}
