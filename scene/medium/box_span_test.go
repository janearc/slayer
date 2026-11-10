package medium

import (
	"github.com/janearc/slayer/maths"
	"testing"
)

// boxOnlyMedium has no tighter Span method: the box is its entire support.
type boxOnlyMedium struct{}

func (boxOnlyMedium) Bounds() maths.Box {
	return maths.Box{Min: maths.V(-1, -1, -1), Max: maths.V(1, 1, 1)}
}
func (boxOnlyMedium) Density(maths.Vec) float64 { return 1 }

func TestWispyUsesTheBoxWhenItsMediumHasNoSpan(t *testing.T) {
	wispy := Wispy{Of: boxOnlyMedium{}}
	ray := maths.Ray{At: maths.V(0, 0, 2), Dir: maths.V(0, 0, -1)}
	enter, leave, hit := wispy.Span(ray, 0, 10)
	if !hit || enter != 1 || leave != 3 {
		t.Fatalf("box fallback: %g %g %v", enter, leave, hit)
	}
}
