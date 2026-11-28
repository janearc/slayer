package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// the giant of the span tests: radius one, a ramp of this share of it,
// and the eye twenty out on the z axis.
const (
	spanRamp = 0.08
	spanEye  = 20
	spanTol  = 1e-9
	// spanGraze is how far from the axis a ray goes that passes over
	// the body under the ramp: past one less the ramp, within one.
	spanGraze = 0.97
)

// A ray at a giant stops being crossed where it meets the ball under the
// ramp: straight in, the span is the ramp's depth and no more.
func TestAGiantsSpanEndsWhereItsBodyBegins(t *testing.T) {
	giant := Giant{Radius: 1, Depth: spanRamp, Thick: 500}
	ray := maths.Ray{At: maths.V(0, 0, spanEye), Dir: maths.V(0, 0, -1)}
	enter, leave, crossed := giant.Span(ray, 0, math.Inf(1))
	if !crossed || math.Abs(enter-(spanEye-1)) > spanTol ||
		math.Abs(leave-(spanEye-1+spanRamp)) > spanTol {
		t.Errorf("straight in the span is %v to %v (%v), want %v to %v",
			enter, leave, crossed, float64(spanEye-1),
			spanEye-1+spanRamp)
	}
}

// Where the ray never reaches the body under the ramp, or begins under
// it, the whole of the crossing is the span, as it always was.
func TestAGiantsSpanKeepsACrossingThatNeverReachesItsBody(t *testing.T) {
	giant := Giant{Radius: 1, Depth: spanRamp, Thick: 500}
	over := maths.Ray{At: maths.V(spanGraze, 0, spanEye),
		Dir: maths.V(0, 0, -1)}
	whole := math.Sqrt(1 - spanGraze*spanGraze)
	enter, leave, _ := giant.Span(over, 0, math.Inf(1))
	if math.Abs(leave-enter-2*whole) > spanTol {
		t.Errorf("a ray grazing over the body has a span of %v, want %v",
			leave-enter, 2*whole)
	}
	within := maths.Ray{At: maths.V(0, 0, 0), Dir: maths.V(0, 0, -1)}
	enter, leave, crossed := giant.Span(within, 0, math.Inf(1))
	if !crossed || enter != 0 || math.Abs(leave-1) > spanTol {
		t.Errorf("a ray from inside has the span %v to %v (%v), want 0 "+
			"to 1", enter, leave, crossed)
	}
}
