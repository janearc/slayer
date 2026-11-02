package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// A jet is two cones, dense at the heart of the base, the same up and down,
// thinning to zero at the end and soft across; its span pulls in to the gas.
func TestJetIsTwoConesThinningOut(t *testing.T) {
	j := Jet{Length: 10, Base: 1, Spread: 0.1, Thick: 1}
	if d := j.Density(maths.V(0, 0.1, 0)); d < 0.98 {
		t.Errorf("the base is %v", d)
	}
	up, down := j.Density(maths.V(0, 5, 0)), j.Density(maths.V(0, -5, 0))
	if up != down || up < 0.45 || up > 0.55 {
		t.Errorf("halfway up %v, halfway down %v", up, down)
	}
	if d := j.Density(maths.V(0, 9.9, 0)); d > 0.02 {
		t.Errorf("the end is %v", d)
	}
	if j.Density(maths.V(0, 11, 0)) != 0 || j.Density(maths.V(5, 5, 0)) > 1e-6 {
		t.Error("gas beyond the jet")
	}
	// side-on through the middle of the upper cone, the span is about
	// the cone's width there, not the box's
	r := maths.Ray{At: maths.V(-20, 5, 0), Dir: maths.XAxis}
	enter, leave, ok := j.Span(r, 0, 40)
	if !ok || leave-enter > 6 || enter > 20 || leave < 20 {
		t.Errorf("the span is %v to %v, %v", enter, leave, ok)
	}
	if _, _, ok := j.Span(maths.Ray{At: maths.V(-20, 12, 0), Dir: maths.XAxis}, 0, 40); ok {
		t.Error("a ray over the jet has a span")
	}
}

// A jet that starts out from the origin is empty before its start.
func TestJetStartsWhereAsked(t *testing.T) {
	j := Jet{Length: 10, Base: 1, Spread: 0.1, Thick: 1, From: 2}
	if j.Density(maths.V(0, 1, 0)) != 0 || j.Density(maths.V(0, -1, 0)) != 0 {
		t.Error("gas before the start")
	}
	if j.Density(maths.V(0, 2.5, 0)) <= 0 {
		t.Error("no gas after the start")
	}
}

// Knots bunch a jet along its length: with them the density varies up
// the axis, without them it falls smoothly.
func TestKnotsBunchAJet(t *testing.T) {
	smooth := Jet{Length: 10, Base: 1, Spread: 0.1, Thick: 1}
	knotted := smooth
	knotted.Knots = 0.8
	// the knots' own factor, the knotted density over the smooth one,
	// wanders up the axis; without knots it is one all the way
	var wander float64
	last := -1.0
	for y := 1.0; y < 9; y += 0.25 {
		k := knotted.Density(maths.V(0, y, 0)) / smooth.Density(maths.V(0, y, 0))
		if last >= 0 {
			wander += math.Abs(k - last)
		}
		last = k
	}
	if !(wander > 1.5) {
		t.Errorf("the knots' factor wanders only %.3f up the jet", wander)
	}
}
