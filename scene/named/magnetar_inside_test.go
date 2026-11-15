package named

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/research"
)

// how near two lengths or fields must be to be the same: the
// arithmetic's own rounding, picked by the author of this scene,
// 2026-11-12
const slack = 1e-9

// alike is whether two numbers are the same to within slack of the
// larger.
func alike(one, other float64) bool {
	return math.Abs(one-other) <= slack*math.Max(math.Abs(one),
		math.Abs(other))
}

// the multiples the dipole's law is checked at, each defined: the
// field at a pole is twice the equator's, and twice as far out it is an
// eighth
const (
	poleTimes  = 2
	twiceOut   = 2
	eighthOut  = 8
	quarterOff = 4
)

// The dipole: at the surface's equator the field is the surface field,
// at its pole twice that, and twice as far out an eighth.
func TestTheDipoleFallsWithTheCube(t *testing.T) {
	body := realStar()
	for _, case_ := range []struct {
		out, colatitude, want float64
	}{
		{body.radius, math.Pi / 2, body.field},
		{body.radius, 0, poleTimes * body.field},
		{twiceOut * body.radius, math.Pi / 2, body.field / eighthOut},
		{twiceOut * body.radius, math.Pi, poleTimes * body.field /
			eighthOut},
	} {
		if got := body.fieldAt(case_.out, case_.colatitude); !alike(got,
			case_.want) {
			t.Errorf("at %g km, colatitude %g: %g G, want %g", case_.out,
				case_.colatitude, got, case_.want)
		}
	}
}

// A hoop's distance for a named field is the arithmetic's: an eighth
// of the surface field is twice the radius out, and the surface field
// is the surface.
func TestAHoopIsWhereTheFieldFalls(t *testing.T) {
	body := realStar()
	for field, want := range map[float64]float64{
		body.field / eighthOut: twiceOut * body.radius,
		body.field:             body.radius} {
		got, err := body.reach(field)
		if err != nil || !alike(got, want) {
			t.Errorf("%g G: %g km, %v; want %g km", field, got, err, want)
		}
	}
	for _, each := range dangers() {
		out, err := body.reach(each.field)
		if err != nil || !alike(body.fieldAt(out, math.Pi/2), each.field) {
			t.Errorf("%s: the field at its hoop is not its own", each.name)
		}
	}
}

// anti-tests: a field stronger than the surface, none, and not a number
// have no hoop.
func TestAFieldTheStarLacksHasNoHoop(t *testing.T) {
	body := realStar()
	for _, field := range []float64{twiceOut * body.field, 0,
		math.NaN(), -body.field} {
		if _, err := body.reach(field); !errors.Is(err, ErrField) {
			t.Errorf("%g G: %v; want ErrField", field, err)
		}
	}
}

// A field line leaves the star at its radius, crosses the equator at
// its apex, and every point on it is apex sin^2 out.
func TestAFieldLineIsTheDipoles(t *testing.T) {
	body := realStar()
	apex := quarterOff * body.radius
	points, err := fieldLine(body.radius, apex, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !alike(points[0].Len(), body.radius) {
		t.Errorf("the line leaves at %g km, want %g", points[0].Len(),
			body.radius)
	}
	middle := points[len(points)/2]
	if !alike(middle.Len(), apex) || math.Abs(middle.Y) > slack*apex {
		t.Errorf("the line crosses at %v, want %g out on the equator",
			middle, apex)
	}
	for _, at := range points {
		theta := math.Acos(at.Y / at.Len())
		if !alike(at.Len(), apex*math.Pow(math.Sin(theta), 2)) {
			t.Fatalf("%v is off the line", at)
		}
	}
}

// anti-tests: a line that would not leave the star, and a tube with one
// point or no thickness, are refused.
func TestALineThatCannotBeDrawnIsRefused(t *testing.T) {
	if _, err := fieldLine(1, 1, 0); !errors.Is(err, ErrLines) {
		t.Errorf("a line at the surface: %v", err)
	}
	if _, err := fieldLine(0, 1, 0); !errors.Is(err, ErrLines) {
		t.Errorf("a star of no size: %v", err)
	}
	var made tube
	if err := made.add([]maths.Vec{{}}, 1, false); !errors.Is(err,
		ErrLines) {
		t.Errorf("one point: %v", err)
	}
	if err := made.add(circle(1), 0, true); !errors.Is(err, ErrLines) {
		t.Errorf("no thickness: %v", err)
	}
}

// A closed hoop's tube is joined end to start: a ring of faces more
// than an open one of the same points.
func TestAClosedHoopIsJoined(t *testing.T) {
	var open, closed tube
	points := circle(1)
	if open.add(points, 0.1, false) != nil ||
		closed.add(points, 0.1, true) != nil {
		t.Fatal("a hoop would not make a tube")
	}
	if len(closed.faces)-len(open.faces) != 2*tubeSides {
		t.Errorf("closed %d faces, open %d", len(closed.faces),
			len(open.faces))
	}
	if _, err := closed.mesh(); err != nil {
		t.Error(err)
	}
}

// The flare: quiet before it, its peak at its rise, its tail pulsing at
// the spin period sped up, quiet again when the tail is over.
func TestTheFlaresLightCurve(t *testing.T) {
	lit, err := newFlare(time.Second, flareSpeed)
	if err != nil {
		t.Fatal(err)
	}
	if lit.Brightness(lit.start-time.Millisecond) != single {
		t.Error("the star brightens before the flare")
	}
	peak := lit.start + lit.rise
	if lit.Brightness(peak) != single+flarePeak {
		t.Errorf("at the peak %g, want %g", lit.Brightness(peak),
			single+flarePeak)
	}
	for at := lit.start; at < lit.Over(); at += time.Millisecond {
		if lit.Brightness(at) > lit.Brightness(peak) {
			t.Fatalf("at %v brighter than the peak", at)
		}
	}
	if lit.Brightness(lit.Over()) != single {
		t.Errorf("after the tail %g", lit.Brightness(lit.Over()))
	}
	pulse := time.Duration(float64(lit.period) / lit.speed)
	crest := lit.start + lit.spike + pulse
	if !(lit.Brightness(crest) > lit.Brightness(crest+pulse/2)) ||
		!(lit.Brightness(crest) > lit.Brightness(crest-pulse/2)) {
		t.Error("the tail does not crest at the spin period, sped up")
	}
}

// The flare's times are the measured ones.
func TestTheFlaresTimesAreMeasured(t *testing.T) {
	lit, _ := newFlare(0, single)
	for name, got := range map[string]time.Duration{
		"sgr1806-tail-period-seconds": lit.period,
		"sgr1806-tail-seconds":        lit.tail,
		"sgr1806-spike-seconds":       lit.spike} {
		if !alike(got.Seconds(), research.Value(name)) {
			t.Errorf("%s: %v", name, got)
		}
	}
	if lit.Over() != lit.spike+lit.tail {
		t.Errorf("at its true pace the tail is over at %v", lit.Over())
	}
}

// anti-test: a tail shown slower than it ran, or not shown, is refused.
func TestASlowTailIsRefused(t *testing.T) {
	for _, speed := range []float64{0.5, 0, math.NaN()} {
		if _, err := newFlare(0, speed); !errors.Is(err, ErrBounds) {
			t.Errorf("speed %g: %v", speed, err)
		}
	}
}

// The computed values are their arithmetic: zappy nine hundred times
// the real star, the light cylinder c P / 2 pi, a kiloparsec 3,261.56
// light years.
func TestTheComputedValuesAreTheirArithmetic(t *testing.T) {
	values := computed()
	body := realStar()
	for name, want := range map[string]float64{
		"zappy-times-larger": zappyKm() / body.radius,
		"light-cylinder-km":  lightKmS * body.period / (2 * math.Pi),
	} {
		if !alike(values[name], want) {
			t.Errorf("%s: %g, want %g", name, values[name], want)
		}
	}
	if math.Abs(1/lightYearKpc-3261.56) > 0.01 {
		t.Errorf("a kiloparsec is %g light years", 1/lightYearKpc)
	}
}

// A figure's significant digits: a fraction's past its leading zeros,
// a whole number's but its trailing zeros.
func TestAFiguresDigits(t *testing.T) {
	for written, want := range map[string]int{"16,400": 3, "0.2": 1,
		"900": 1, "7.56": 3, "2,000,000,000,000,000": 1, "0.94": 2} {
		if _, got := readFigure(written); got != want {
			t.Errorf("%s: %d digits, want %d", written, got, want)
		}
	}
	if !rounded(0, single, 0) || rounded(0, single, 1) {
		t.Error("nought is nought only")
	}
}
