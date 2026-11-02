package emit

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// spot is a point emitter kept to a cone straight down, twenty degrees
// wide with ten more of soft edge, a unit above the ground.
func spot() Lit {
	return Point{At: maths.V(0, 1, 0), Light: sunlike(),
		Toward: maths.V(0, -3, 0), Width: 20 * maths.Degree,
		Soft: 10 * maths.Degree}.When(0)
}

// lightAt is the green light a lamp one unit up gives the ground so
// far off its beam, with the fall by distance taken back out.
func lightAt(lit Lit, off float64) float64 {
	ground := maths.V(math.Tan(off), 0, 0)
	_, _, light := lit.Light(ground, maths.Vec{})
	far := ground.Sub(lit.At).Len()
	return light.G * far * far
}

// a point emitter kept to a cone lights what is inside its width in
// full, nothing past its soft edge, and half way through the edge at
// its middle. the cone's way need not be a unit long.
func TestAConeLightsInsideItsWidthAndFadesAcrossItsEdge(t *testing.T) {
	lit := spot()
	full := Point{At: maths.V(0, 1, 0), Light: sunlike()}.When(0)
	whole := lightAt(full, 0)
	fades := map[float64]float64{0: 1, 19: 1, 25: 0.5, 30: 0, 60: 0}
	for off, want := range fades {
		got := lightAt(lit, off*maths.Degree) / whole
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("%v degrees off the beam is %v of it, want %v",
				off, got, want)
		}
	}
}

// with no width a point emitter is as it always was: alike every way,
// whatever its way says.
func TestNoWidthIsEveryWay(t *testing.T) {
	plain := Point{At: maths.V(0, 1, 0), Light: sunlike()}.When(0)
	aimed := Point{At: maths.V(0, 1, 0), Light: sunlike(),
		Toward: maths.V(1, 0, 0)}.When(0)
	for _, off := range []float64{0, 40, 80} {
		if got, want := lightAt(aimed, off*maths.Degree), lightAt(plain,
			off*maths.Degree); got != want {
			t.Errorf("%v degrees off: %v, want %v", off, got, want)
		}
	}
	behind := maths.V(0, 5, 0)
	if _, _, light := aimed.Light(behind, maths.Vec{}); light.G <= 0 {
		t.Error("an emitter with no width lights nothing behind it")
	}
}

// described is a light somebody else made, as emit reads one.
type described struct {
	light      bands.Light
	strength   float64
	half, soft float64
	dim        float64
}

func (d described) Bands() bands.Light { return d.light }
func (d described) Strength() float64  { return d.strength }
func (d described) Angles() (measure.Angle, measure.Angle) {
	return measure.Radians(d.half), measure.Radians(d.soft)
}
func (d described) Share(time.Duration) float64 { return d.dim }

// total is all the power of a spectrum, band by band.
func total(light bands.Light) float64 {
	first, last := light.Span()
	sum := 0.0
	for band := first; band <= last; band++ {
		sum += light.Power(band)
	}
	return sum
}

// an emitter made of a description carries its power and nothing
// else's: whatever scale the description's spectrum came at, the
// emitter's, added up over every band, is the description's strength
// times the exposure, which is what a watt is in a picture's light.
func TestAnEmitterOfADescriptionCarriesItsStrength(t *testing.T) {
	for _, scale := range []float64{1, 1e-6, 4000} {
		lamp := described{light: sunlike().Scale(scale), strength: 80,
			half: 0.3, soft: 0.1, dim: 1}
		point, err := Of(lamp, maths.V(1, 2, 3), maths.V(0, 0, -2), 0.5)
		if err != nil {
			t.Fatal(err)
		}
		if got := total(point.Light); !near(got, 40) {
			t.Errorf("a spectrum scaled %v adds up to %v, want 40",
				scale, got)
		}
		if point.At != maths.V(1, 2, 3) ||
			!point.Toward.Near(maths.V(0, 0, -1), 1e-12) {
			t.Errorf("it is at %v shining %v", point.At,
				point.Toward)
		}
		if point.Width != 0.3 || point.Soft != 0.1 {
			t.Errorf("a cone of %v and %v", point.Width, point.Soft)
		}
	}
}

// a description that shines every way is an emitter with no width,
// and its flicker is the emitter's pulse.
func TestAnEveryWayDescriptionHasNoWidthAndPulses(t *testing.T) {
	fire := described{light: sunlike(), strength: 10, half: math.Pi,
		dim: 0.25}
	point, err := Of(fire, maths.Vec{}, maths.Up, 1)
	if err != nil {
		t.Fatal(err)
	}
	if point.Width != 0 {
		t.Errorf("every way came out %v wide", point.Width)
	}
	steady := Point{Light: point.Light}.When(0).Strength
	if got := point.When(0).Strength; !near(got.G, 0.25*steady.G) {
		t.Errorf("at a quarter of itself it is %v of %v", got, steady)
	}
}

func TestADescriptionThatCannotBeAnEmitterIsRefused(t *testing.T) {
	good := described{light: sunlike(), strength: 1, half: 0.2, dim: 1}
	dark := good
	dark.light = bands.Light{}
	weak := good
	weak.strength = 0
	cases := map[string]struct {
		lamp     Described
		toward   maths.Vec
		exposure float64
	}{
		"nothing":            {nil, maths.Up, 1},
		"no light":           {dark, maths.Up, 1},
		"no strength":        {weak, maths.Up, 1},
		"no way to shine":    {good, maths.Vec{}, 1},
		"no exposure":        {good, maths.Up, 0},
		"an exposure of nan": {good, maths.Up, math.NaN()},
	}
	for name, with := range cases {
		_, err := Of(with.lamp, maths.Vec{}, with.toward, with.exposure)
		if !errors.Is(err, errDescribed) {
			t.Errorf("%s: %v, want %v", name, err, errDescribed)
		}
	}
}

// a way a hair longer than a unit, as rounding makes them, is still
// straight down the beam: the cosine is held to one before its angle
// is taken, or the angle would be no number and the light with it.
func TestAWayAHairPastAUnitIsStillInTheBeam(t *testing.T) {
	lit := spot()
	for _, long := range []float64{1 + 1e-9, 1 + 1e-15} {
		if got := lit.Along(lit.Toward.Scale(long)); got != 1 {
			t.Errorf("a way %v long down the beam is %v of it",
				long, got)
		}
		if got := lit.Along(lit.Toward.Scale(-long)); got != 0 {
			t.Errorf("a way %v long back up the beam is %v of it",
				long, got)
		}
	}
}

// a description's spectrum and strength may be as small as they like and still
// be something: only zero and below are refused.
func TestAFaintDescriptionIsStillAnEmitter(t *testing.T) {
	// a black body's spectrum from libtheme adds up to some 1e32 in
	// its own units, so this is how far down it has to go to be faint
	faint := described{light: sunlike().Scale(1e-36), strength: 1e-9,
		half: 0.2, dim: 1}
	if got := total(faint.light); got <= 0 || got >= 1 {
		t.Fatalf("the test's faint spectrum adds up to %v", got)
	}
	point, err := Of(faint, maths.Vec{}, maths.Up, 1e-3)
	if err != nil {
		t.Fatalf("a faint light was refused: %v", err)
	}
	if got := total(point.Light); !near(got*1e12, 1) {
		t.Errorf("it adds up to %v, want 1e-12", got)
	}
}
