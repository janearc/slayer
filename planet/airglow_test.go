package planet

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// sloped is weather whose air came from so many times its latitude, everywhere
// still and fresh: one is unstirred air, zero air mixed flat, more air drawn
// steeply together.
type sloped float64

func (slope sloped) At(latitude, longitude float64) (origin, spin,
	age float64) {
	return float64(slope) * latitude, 0, 0
}

// glowing is a face glowing two watts over a band ten degrees either
// side of its equator, its pole up y and its eye along z, its steepest
// fifth its fronts and its calm read from its weather.
func glowing(flow Flow) *Face {
	face := &Face{airglow: 2, airband: 10, fronts: 0.2, flow: flow,
		ways: frameOf(maths.V(0, 1, 0), maths.V(0, 0, 1))}
	face.calm = face.calmness()
	return face
}

// fronted is weather unstirred but for one front across the equator, a
// degree either side of it, where its air's origin changes three
// degrees a degree.
type fronted struct{}

func (fronted) At(latitude, longitude float64) (origin, spin, age float64) {
	held := math.Max(-maths.Degree, math.Min(maths.Degree, latitude))
	return latitude + 2*held, 0, 0
}

// the glow holds both d lines when the band does, half each; as full
// just inside the band's edge as at its middle, looking straight down;
// none at or past its edge.
func TestTheAirglowIsSodiumsTwoLines(t *testing.T) {
	face := glowing(nil)
	if got := face.glowedAt(0, 1, 580, 600); math.Abs(got-2) > 1e-12 {
		t.Errorf("both lines: %g, want 2", got)
	}
	if got := face.glowedAt(-9.9, 1, 580, 600); math.Abs(got-2) > 1e-12 {
		t.Errorf("just inside the edge: %g, want 2", got)
	}
	if got := face.glowedAt(0, 1, 588, 589.2); math.Abs(got-1) > 1e-12 {
		t.Errorf("d2 alone: %g, want 1", got)
	}
	if got := face.glowedAt(0, 1, 600, 620); got != 0 {
		t.Errorf("no line: %g", got)
	}
	if got := face.glowedAt(10, 1, 580, 600); got != 0 {
		t.Errorf("at the band's edge: %g", got)
	}
	if got := face.glowedAt(-12, 1, 580, 600); got != 0 {
		t.Errorf("past it: %g", got)
	}
}

// a thin layer brightens toward the limb as the eye's way through it
// lengthens, and stays finite where the eye grazes it.
func TestTheAirglowBrightensToTheLimb(t *testing.T) {
	face := glowing(nil)
	down := face.glowedAt(0, 1, 580, 600)
	slant := face.glowedAt(0, 0.5, 580, 600)
	if math.Abs(slant-2*down) > 1e-12 {
		t.Errorf("at half height %g, want twice %g", slant, down)
	}
	edge := face.glowedAt(0, 0, 580, 600)
	if math.IsInf(edge, 0) || math.Abs(edge-down/grazing) > 1e-9 {
		t.Errorf("grazing: %g, want %g", edge, down/grazing)
	}
}

// against a calm of one degree a degree, a front glows by how much more
// steeply its air's origin changes: air as calm not at all, air mixed
// flat not at all, air two and a half times as steep one and a half
// times, the steepest held to mostFront. with no weather, once.
func TestAFrontGlowsByItsSteepnessOverTheCalm(t *testing.T) {
	up := frameOf(maths.V(0, 1, 0), maths.V(0, 0, 1)).toward(4, 30)
	for _, x := range []struct{ slope, want float64 }{
		{1, 0}, {2.5, 1.5}, {0, 0}, {40, mostFront},
	} {
		face := glowing(sloped(x.slope))
		face.calm = 1
		if got := face.front(up, 0); math.Abs(got-x.want) > 1e-6 {
			t.Errorf("slope %g: %g, want %g", x.slope, got, x.want)
		}
	}
	if got := glowing(nil).front(up, 0); got != 1 {
		t.Errorf("with no weather, %g", got)
	}
}

// a band stirred evenly has no fronts: its calm is its own steepness,
// and nowhere in it glows.
func TestAnEvenBandHasNoFronts(t *testing.T) {
	face := glowing(sloped(2))
	if math.Abs(face.calm-2) > 1e-6 {
		t.Errorf("calm %g, want 2", face.calm)
	}
	up := face.ways.toward(-6, 200)
	if got := face.front(up, 0); math.Abs(got) > 1e-6 {
		t.Errorf("glows %g", got)
	}
}

// one front across the equator is a tenth of a band twenty degrees
// wide, so the band's calm is the unstirred rest, one degree a degree;
// the front glows twice, three degrees a degree over one, and the rest
// not at all.
func TestTheBandsInsideIsDarkAndItsFrontsGlow(t *testing.T) {
	face := glowing(fronted{})
	if math.Abs(face.calm-1) > 1e-6 {
		t.Errorf("calm %g, want 1", face.calm)
	}
	if got := face.front(face.ways.toward(0, 40), 0); math.Abs(got-2) >
		1e-6 {
		t.Errorf("on the front, %g, want 2", got)
	}
	if got := face.front(face.ways.toward(5, 40), 0); math.Abs(got) >
		1e-6 {
		t.Errorf("inside the band, %g, want nought", got)
	}
}

// a band's fronts are its steepest share, more than zero and under one; any
// other is refused.
func TestAnAirglowsFrontsAreAShare(t *testing.T) {
	for _, fronts := range []float64{0, 1, -0.1, 1.5, math.NaN()} {
		if _, err := (Body{}).Airglowing(2, 10, fronts); err == nil {
			t.Errorf("fronts the steepest %g", fronts)
		}
	}
	body, err := (Body{}).Airglowing(2, 10, 0.05)
	if err != nil || body.airfronts != 0.05 {
		t.Errorf("fronts %g, %v", body.airfronts, err)
	}
}

// a pixel wider than the front's step reads it across the pixel: on a
// steady slope, the same.
func TestAWidePixelReadsTheFrontAcrossIt(t *testing.T) {
	up := frameOf(maths.V(0, 1, 0), maths.V(0, 0, 1)).toward(-3, 100)
	face := glowing(sloped(2.7))
	face.calm = 1
	if got := face.front(up, 3*maths.Degree); math.Abs(got-1.7) > 1e-6 {
		t.Errorf("%g, want 1.7", got)
	}
}
