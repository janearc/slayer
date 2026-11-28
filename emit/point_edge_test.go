package emit

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// A point lamp's light is none inside Beyond less its edge, whole from
// Beyond out, and between them a smooth rise: no step for a float's
// rounding to toss.
func TestAPointLampsEdgeRisesSmoothlyFromNoneToWhole(t *testing.T) {
	const beyond = 2.0
	lit := Lit{Beyond: beyond, Strength: maths.Grey(1)}
	light := func(distance float64) float64 {
		_, _, got := lit.Light(maths.V(distance, 0, 0), maths.Vec{})
		return float64(got.R) * distance * distance
	}
	if got := light(beyond * (1 - 2*EdgeShare)); got != 0 {
		t.Errorf("well inside the edge, light %v, want none", got)
	}
	if got := light(beyond); got < 1-1e-12 || got > 1+1e-12 {
		t.Errorf("at Beyond, light %v of the whole, want all", got)
	}
	last := 0.0
	for step := range 11 {
		got := light(beyond * (1 - EdgeShare + EdgeShare*float64(step)/10))
		if got < last {
			t.Errorf("light fell from %v to %v going out through the edge",
				last, got)
		}
		last = got
	}
	if half := light(beyond * (1 - EdgeShare/2)); half < 0.4 || half > 0.6 {
		t.Errorf("half way through the edge, light %v, want about a half",
			half)
	}
}

// Anti: past Beyond the light is the inverse square, as it was when the
// edge was a step.
func TestAPointLampsLightPastBeyondIsTheInverseSquare(t *testing.T) {
	lit := Lit{Beyond: 2, Strength: maths.Grey(3)}
	for _, distance := range []float64{2, 2.002, 5} {
		_, far, got := lit.Light(maths.V(distance, 0, 0), maths.Vec{})
		want := 3 / (distance * distance)
		if d := float64(got.R) - want; d < -1e-12 || d > 1e-12 {
			t.Errorf("light at %v is %v, want %v", distance, got.R, want)
		}
		if d := far - (distance - 2); d < -1e-12 || d > 1e-12 {
			t.Errorf("far at %v is %v, want %v", distance, far, distance-2)
		}
	}
}
