package fixture_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/burning-trousers/fake"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// the slab the plane's lamp stands in: one of medium below the lamp
// and three above it, wide enough that every way from the lamp to the
// ground points the fixture looks at leaves through its floor, and too
// high for any ray from the eye to cross.
const (
	slabDensity = 0.3
	slabFloor   = 1.0
	slabTop     = 5.0
	slabHalf    = 2.5
)

// inMedium is the plane's ground drawn with its lamp inside the slab:
// one red value a ground point, as the plane's own drawn gives.
func inMedium(t *testing.T) []float64 {
	t.Helper()
	world := plane.world()
	world.Volumes = []*thing.Volume{{Medium: fake.NewSlab(
		maths.V(-slabHalf, slabFloor, -slabHalf),
		maths.V(slabHalf, slabTop, slabHalf), slabDensity)}}
	picture, err := trace.Render(world, maths.Subspace{}, plane.sensor(),
		0, len(plane.xs), 1)
	if err != nil {
		t.Fatal(err)
	}
	var red []float64
	for x := range plane.xs {
		red = append(red, float64(picture.At(x, 0).R))
	}
	return red
}

// A lamp inside a medium is dimmed on its way to a point only by the
// medium between them: what lies past the lamp is behind the light and
// takes none. the ground at x is dimmed by exp(-k l), where l is the
// part of the way from it to the lamp that is above the slab's floor,
// and never by the three of medium over the lamp. each point is drawn
// with the slab and without, so whatever the tracer does to bare light
// cancels.
func TestALampInAMediumIsDimmedOnlyByWhatIsBetween(t *testing.T) {
	bare, dimmed := plane.drawn(t), inMedium(t)
	for at, x := range plane.xs {
		way := math.Hypot(x, plane.height)
		inside := way * (plane.height - slabFloor) / plane.height
		want := math.Exp(-slabDensity * inside)
		got := dimmed[at] / bare[at]
		if math.Abs(got/want-1) > 1e-3 {
			t.Errorf("at x %.1f the ground keeps %.5f of its light; "+
				"only the medium between says %.5f", x, got, want)
		}
	}
}
