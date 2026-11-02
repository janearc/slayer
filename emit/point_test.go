package emit

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/slayer/maths"
)

// a sun-like star's light, the spectrum of a black body at its surface
func sunlike() bands.Light { return bands.Split(swatch.Blackbody(5772)) }

// near is got within a part in a billion of want
func near(got, want float64) bool {
	return math.Abs(got-want) <= 1e-9*math.Max(1, math.Abs(want))
}

func TestTheLightFallsWithTheSquareOfTheDistance(t *testing.T) {
	lit := Point{Light: sunlike()}.When(0)
	_, _, one := lit.Light(maths.V(1, 0, 0), maths.Vec{})
	_, _, two := lit.Light(maths.V(2, 0, 0), maths.Vec{})
	if !near(two.G, one.G/4) {
		t.Errorf("at twice the distance %g, want a quarter of %g",
			two.G, one.G)
	}
}

func TestTheLightComesFromThePoint(t *testing.T) {
	lit := Point{At: maths.V(0, 3, 0), Light: sunlike(), Beyond: 1}.When(0)
	toward, distance, _ := lit.Light(maths.V(0, 0, 0), maths.Vec{})
	if !near(toward.Y, 1) || !near(toward.Len(), 1) {
		t.Errorf("toward the emitter is %v, want straight up", toward)
	}
	if !near(distance, 2) {
		t.Errorf("the light travels %g past the surface, want 2",
			distance)
	}
}

func TestNothingArrivesInsideTheSurface(t *testing.T) {
	lit := Point{Light: sunlike(), Beyond: 1}.When(0)
	_, _, light := lit.Light(maths.V(0.5, 0, 0), maths.Vec{})
	if light != (maths.Light{}) {
		t.Errorf("inside the star the light is %v, want none", light)
	}
}

// the spectrum is carried as it is: twice the power in every band is
// twice the light, so nothing on the way normalises it
func TestTheSpectrumIsCarriedLinearly(t *testing.T) {
	once := Point{Light: sunlike()}.When(0).Strength
	twice := Point{Light: sunlike().Scale(2)}.When(0).Strength
	if !near(twice.R, 2*once.R) || !near(twice.B, 2*once.B) {
		t.Errorf("twice the spectrum gave %v from %v", twice, once)
	}
}

func TestASunlikeStarIsNearlyWhite(t *testing.T) {
	strength := Point{Light: sunlike()}.When(0).Strength
	most := math.Max(strength.R, math.Max(strength.G, strength.B))
	least := math.Min(strength.R, math.Min(strength.G, strength.B))
	if least <= 0 || least/most < 0.7 {
		t.Errorf("a 5772 kelvin star reduces to %v, want near white",
			strength)
	}
}

func TestThePulseIsWorkedOutAtTheTime(t *testing.T) {
	flare := func(at time.Duration) float64 {
		if at >= time.Second {
			return 10
		}
		return 1
	}
	point := Point{Light: sunlike(), Pulse: flare}
	before := point.When(0).Strength
	during := point.When(time.Second).Strength
	if !near(during.G, 10*before.G) {
		t.Errorf("in the flare %g, want ten times %g", during.G,
			before.G)
	}
}

// an emitter that shines only in x-ray, a line at 1.24 nanometres, a
// kiloelectronvolt: nothing in the visible, all of it in its band
func TestAnXRayEmitterLightsInItsBandAlone(t *testing.T) {
	xray := bands.Lines(map[float64]float64{1.24: 5})
	lit := Point{Light: xray}.When(0)
	if got := lit.In(maths.Band{Nanometres: 1.24}); !near(got, 5) {
		t.Errorf("in its own band %g, want 5", got)
	}
	if got := lit.In(maths.Band{Nanometres: 550}); got != 0 {
		t.Errorf("in green %g, want nothing", got)
	}
	if lit.Strength != (maths.Light{}) {
		t.Errorf("an x-ray line is %v in the visible, want dark",
			lit.Strength)
	}
}

// what lands of a band falls as the light does: by the inverse square,
// none inside the surface, and only within the cone
func TestABandLandsAsTheLightFalls(t *testing.T) {
	xray := bands.Lines(map[float64]float64{1.24: 5})
	band := maths.Band{Nanometres: 1.24}
	lit := Point{Light: xray, Beyond: 1}.When(0)
	if got := lit.Lands(maths.V(2, 0, 0), band); !near(got, 5.0/4) {
		t.Errorf("two away %g, want a quarter of 5", got)
	}
	if got := lit.Lands(maths.V(0.5, 0, 0), band); got != 0 {
		t.Errorf("inside the surface %g, want none", got)
	}
	if got := lit.Lands(maths.V(2, 0, 0),
		maths.Band{Nanometres: 550}); got != 0 {
		t.Errorf("in green %g, want none", got)
	}
	cone := Point{Light: xray, Toward: maths.V(0, 0, 1),
		Width: 10 * maths.Degree, Soft: 5 * maths.Degree}.When(0)
	if got := cone.Lands(maths.V(0, 0, 2), band); !near(got, 5.0/4) {
		t.Errorf("down the cone %g, want a quarter of 5", got)
	}
	if got := cone.Lands(maths.V(2, 0, 0), band); got != 0 {
		t.Errorf("out of the cone %g, want none", got)
	}
}

// a lamp lights in a band by the same falling as in the visible, so a
// band picture and the ordinary one agree on where the light is
func TestABandLandsAsTheVisibleDoes(t *testing.T) {
	green := maths.Band{Nanometres: 550}
	lit := Point{At: maths.V(1, 2, 3), Light: sunlike(), Toward: maths.V(0,
		-1, 0), Width: 30 * maths.Degree, Soft: 20 * maths.Degree}.When(0)
	for _, at := range []maths.Vec{maths.V(1, 0, 3), maths.V(2, -1, 3.5),
		maths.V(1.5, 0, 2)} {
		_, _, light := lit.Light(at, maths.Vec{})
		share := light.G / lit.Strength.G
		if got := lit.Lands(at, green); !near(got, lit.In(green)*share) {
			t.Errorf("at %v %g lands, want %g", at, got,
				lit.In(green)*share)
		}
	}
}

func TestTheBandFollowsThePulse(t *testing.T) {
	xray := bands.Lines(map[float64]float64{1.24: 5})
	twice := func(time.Duration) float64 { return 2 }
	lit := Point{Light: xray, Pulse: twice}.When(0)
	if got := lit.In(maths.Band{Nanometres: 1.24}); !near(got, 10) {
		t.Errorf("pulsed to twice, %g in its band, want 10", got)
	}
}
