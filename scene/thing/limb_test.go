package thing

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The linear limb law: the centre of the disc gives all of the glow, a
// look at sixty degrees to the normal gives one minus half the
// coefficient, and the edge gives one minus the coefficient.
func TestLimbShareFollowsTheLinearLaw(t *testing.T) {
	normal := maths.V(0, 0, 1)
	for name, test := range map[string]struct {
		toward maths.Vec
		want   float64
	}{
		"the centre": {maths.V(0, 0, 1), 1},
		"sixty degrees": {maths.V(math.Sin(math.Pi/3), 0,
			math.Cos(math.Pi/3)), 0.7},
		"the edge": {maths.V(1, 0, 0), 0.4},
	} {
		if got := LimbShare(0.6, normal, test.toward); math.Abs(got-test.want) > 1e-12 {
			t.Errorf("%s: %v, want %v", name, got, test.want)
		}
	}
}

// The anti-tests: no coefficient darkens nothing; a look from behind the
// surface is never brighter than the edge; and a coefficient past one
// never takes the edge below nought.
func TestLimbShareNeverBrightensNorGoesBelowNought(t *testing.T) {
	normal := maths.V(0, 0, 1)
	if got := LimbShare(0, normal, maths.V(1, 0, 0)); got != 1 {
		t.Errorf("no coefficient darkens to %v", got)
	}
	if got := LimbShare(0.6, normal, maths.V(0, 0, -1)); got != 0.4 {
		t.Errorf("a look from behind gives %v, want the edge's 0.4", got)
	}
	if got := LimbShare(7, normal, maths.V(1, 0, 0)); got != 0 {
		t.Errorf("a coefficient of seven gives %v at the edge", got)
	}
	if got := LimbShare(0.6, maths.Vec{}, maths.V(0, 0, 1)); got != 0.4 {
		t.Errorf("no normal gives %v, want the edge's 0.4", got)
	}
}
