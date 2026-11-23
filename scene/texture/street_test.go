package texture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// pavedFor is a road of grey binder and pale chippings, as wet as given.
func pavedFor(wet float64) Asphalt {
	return Asphalt{Dry: maths.RGB(0.1, 0.1, 0.1),
		Chipping: maths.RGB(0.2, 0.2, 0.2), Wet: wet, Tracks: 1.75, Seed: 7}
}

// a soaked road is darker than a dry one at the same place, and by about
// what asphaltDarkening says, since the binder's mottling is the same.
func TestASoakedRoadIsDarkerThanADryOne(t *testing.T) {
	at := maths.V(0.31, 0, 0.77)
	dry, wet := pavedFor(0).At(at), pavedFor(1).At(at)
	if wet.Luminance() >= dry.Luminance() {
		t.Fatalf("wet %v is not darker than dry %v", wet, dry)
	}
	want := dry.Luminance() * (1 - asphaltDarkening)
	if math.Abs(wet.Luminance()-want) > 1e-9 {
		t.Errorf("wet luminance %g, want %g", wet.Luminance(), want)
	}
}

// the road keeps from shimmering: a pixel wider than every feature sees
// the same road wherever it lies, and a lone ray does not.
func TestAWidePixelSeesTheSameRoadEverywhere(t *testing.T) {
	road := pavedFor(0.3)
	const wide = 4.0
	first := road.Within(maths.V(0.1, 0, 0.2), wide)
	for index := range 40 {
		at := maths.V(0.37*float64(index), 0, 0.91*float64(index))
		got := road.Within(at, wide)
		if math.Abs(got.Luminance()-first.Luminance()) > 1e-9 {
			t.Fatalf("at %v the road is %v, at the first %v", at, got, first)
		}
	}
	varied := false
	for index := range 40 {
		at := maths.V(0.037*float64(index), 0, 0.091*float64(index))
		if math.Abs(road.At(at).Luminance()-first.Luminance()) > 1e-6 {
			varied = true
		}
	}
	if !varied {
		t.Error("a lone ray saw an even road, which has mottling")
	}
}

// a dry road has no damp patch anywhere, and its footprint changes nothing
// about that.
func TestADryRoadHasNoDamp(t *testing.T) {
	road := pavedFor(0)
	for index := range 30 {
		at := maths.V(0.5*float64(index), 0, 0.3*float64(index))
		if got := road.damp(at, 0); got != 0 {
			t.Fatalf("damp %g at %v on a dry road", got, at)
		}
		if got := road.damp(at, 3); got != 0 {
			t.Fatalf("damp %g at %v on a dry road, seen wide", got, at)
		}
	}
}

// wheel tracks wet before the road between them.
func TestWheelTracksAreDampBeforeTheRoadBetween(t *testing.T) {
	road := pavedFor(0.2)
	if road.track(1.75) <= road.track(0) {
		t.Errorf("track %g in the track, %g between", road.track(1.75),
			road.track(0))
	}
	road.Tracks = 0
	if road.track(1.75) != 0 {
		t.Error("a road without tracks has a track")
	}
}

// slabs are darker for being wet, and their joints are found.
func TestPavingSlabsDarkenWhenWetAndHaveJoints(t *testing.T) {
	slabs := Slabs{Slab: maths.RGB(0.3, 0.3, 0.3), Size: 0.6, Joint: 0.02,
		Seed: 3}
	at := maths.V(0.3, 0, 0.3)
	dry := slabs.At(at)
	slabs.Wet = 1
	if slabs.At(at).Luminance() >= dry.Luminance() {
		t.Error("a wet slab is not darker than a dry one")
	}
	if !slabs.InJoint(maths.V(0.6, 0, 0.3)) {
		t.Error("the middle of a joint is not in it")
	}
	if slabs.InJoint(at) {
		t.Error("the middle of a slab is in a joint")
	}
	joint := slabs.Within(maths.V(0.6, 0, 0.3), 0)
	if joint.Luminance() >= slabs.Within(at, 0).Luminance() {
		t.Errorf("joint %v is not darker than the slab", joint)
	}
}

// a pixel wider than a slab sees the share of pavement the joints cover,
// the same everywhere, and not a joint or a slab.
func TestAWidePixelSeesPavingAsOneShare(t *testing.T) {
	slabs := Slabs{Slab: maths.RGB(0.3, 0.3, 0.3), Size: 0.6, Joint: 0.02,
		Seed: 3}
	joint, slab := maths.V(0.6, 0, 0.3), maths.V(0.3, 0, 0.3)
	if slabs.jointed(joint, 5) != slabs.jointed(slab, 5) {
		t.Errorf("wide pixels disagree: %g at a joint, %g on a slab",
			slabs.jointed(joint, 5), slabs.jointed(slab, 5))
	}
}

// a grating is iron where it has bars and the pit's black in its slots,
// and its mean once a pixel is as wide as the pitch.
func TestAGratingHasBarsAndBlackSlots(t *testing.T) {
	grate := Grate{Bar: maths.RGB(0.2, 0.2, 0.2), Pit: maths.RGB(0, 0, 0),
		Pitch: 0.06}
	bar, slot := grate.At(maths.V(0, 0, 0.01)), grate.At(maths.V(0, 0, 0.05))
	if bar.Luminance() <= slot.Luminance() {
		t.Errorf("bar %v is not lighter than slot %v", bar, slot)
	}
	if slot.Luminance() != 0 {
		t.Errorf("a slot shows %v, not the pit", slot)
	}
	mean := grate.Within(maths.V(0, 0, 0.05), 1)
	want := bar.Scale(1 - grateSlot)
	if math.Abs(mean.Luminance()-want.Luminance()) > 1e-9 {
		t.Errorf("seen wide, %v; want %v", mean, want)
	}
}

// resolved is whole for a lone ray and a narrow pixel, and nothing for a
// pixel as wide as the feature.
func TestAFeatureResolvesAsThePixelNarrows(t *testing.T) {
	if resolved(0.1, 0) != 1 || resolved(0.1, 0.01) != 1 {
		t.Error("a narrow pixel does not see a feature whole")
	}
	if resolved(0.1, 0.1) != 0 || resolved(0.1, 1) != 0 {
		t.Error("a wide pixel sees a feature")
	}
	if got := resolved(0, 1); got != 1 {
		t.Errorf("a feature of no width resolves as %g", got)
	}
}
