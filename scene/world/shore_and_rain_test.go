package world

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// A shore far from the sea's tiles has no strip at all, whatever its
// height.
func TestNoShoreFarFromTheSea(t *testing.T) {
	ground := &scrooged{w: New(terran), look: rampLook(), mix: 0.35}
	at := here{smooth: Ground{Wet: 5}, shoreMix: 1}
	if width := ground.stripWidth(at); width != 0 {
		t.Errorf("a shore far from the sea has a strip %v high", width)
	}
}

// A steep shore is rock and a flat one beach, both thin: nothing past
// the widest strip, whatever the noise says.
func TestSteepShoresAreRockAndFlatOnesBeach(t *testing.T) {
	ground := &scrooged{w: New(terran), look: rampLook(), mix: 0.35}
	flat := here{smooth: Ground{Wet: 0}, shoreMix: 1, steep: 0}
	steep := here{smooth: Ground{Wet: 0}, shoreMix: 1, steep: 1}
	if class := ground.shoreClass(flat); class != Beach {
		t.Errorf("a flat shore is %v", class)
	}
	if class := ground.shoreClass(steep); class != Rock {
		t.Errorf("a steep shore is %v", class)
	}
	if width := ground.stripWidth(flat); width <= 0 || width > shoreMost {
		t.Errorf("a flat shore's strip is %v high, not up to %v", width,
			shoreMost)
	}
}

// Anti-test: the strip is not a ribbon of one width. At a shore mix of
// nought there is none, and the tree's shore detail takes it from none
// to some along a shore.
func TestAShoreStripComesAndGoes(t *testing.T) {
	world := New(terran)
	ground := &scrooged{w: world, look: rampLook(), mix: 0.35}
	none := here{smooth: Ground{Wet: 0}, shoreMix: 0}
	if width := ground.stripWidth(none); width != 0 {
		t.Errorf("the strip is %v high where the noise says none", width)
	}
	least, most := 1.0, 0.0
	for place := range pointsAsked {
		detail := world.tree().gather(scatterOnSphere(place),
			nearFootprint).detail[shoreAttribute]
		mix := maths.Smooth(-shoreSoft, shoreSoft, detail)
		least, most = min(least, mix), max(most, mix)
	}
	if least > 0.1 || most < 0.9 {
		t.Errorf("the shore's mix goes only from %v to %v", least, most)
	}
}

// rainStep is the most the rain may change between two points a distant
// pixel apart.
const rainStep = 0.5

// Rain does not flip across a desert's edge between neighbouring pixels
// of a distant eye: at a footprint of a tile's tenth, two points a
// pixel apart have rains within a small share.
func TestRainIsSteadyForADistantPixel(t *testing.T) {
	world := New(terran)
	rain := func(p maths.Vec) float64 {
		return world.rainAt(p, world.Smoothed(p)) + world.tree().gather(p,
			distantFootprint).detail[rainAttribute]
	}
	for place := range pointsAsked {
		a := scatterOnSphere(place)
		b := a.Add(maths.V(distantFootprint, 0, 0)).Unit()
		if d := rain(a) - rain(b); d > rainStep || d < -rainStep {
			t.Fatalf("rain steps %v across one pixel at %v", d, a)
		}
	}
}
