package trace

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// the numbers of the opaque giant's tests, each said once: sunBright is
// how many times white the ball behind is, a sun's order; offsets is how
// many rays are sent across the giant's disc; discReach is how far out
// from its centre, as a share of its radius, they go (past it the rays
// graze the thin top of the cloud, which lets light through); the
// giant's ramp is the share of the radius it thickens over; and
// thickBody and thinBody are its density at depth, optical depths per
// radius, for the opaque body and for the one that is not.
const (
	sunBright = 1e4
	offsets   = 90
	discReach = 0.9
	ramp      = 0.08
	thickBody = 500
	thinBody  = 3
	// leastSeen is the least luminance of the ball that has to get through the
	// thin body: its march ends with about a share of 2.5e-3 left, so a
	// tenth of that, of a ball sunBright bright, is far under it.
	leastSeen = 0.1 * 2.5e-3 * sunBright
)

// aGiantOver is a giant of a density at depth, radius one, in front of a
// ball of a great light, radius ten, far behind it.
func aGiantOver(density float64) *thing.World {
	made := thing.NewWorld()
	made.AddVolume(thing.Volume{Medium: medium.Giant{Radius: 1, Depth: ramp,
		Thick: density}, Material: unlit})
	made.Add(thing.Thing{Shape: shape.Sphere{Radius: 10},
		Material: glows(maths.Grey(sunBright)),
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 0, -50),
			Orientation: maths.Rotation(maths.XAxis, 0)})})
	return made
}

// The brightest the ball behind shows anywhere over the giant's disc, to
// the offset, from rays sent straight at it.
func brightestOver(world *thing.World) float64 {
	var brightest float64
	for step := range offsets {
		reach := discReach * float64(step) / offsets
		ray := maths.Ray{At: maths.V(reach, 0, 20), Dir: maths.V(0, 0, -1)}
		brightest = max(brightest, lone(world, ray, 0).Light().Luminance())
	}
	return brightest
}

// A giant is a body, not a haze: a ball a sun's brightness behind it does
// not show through its disc anywhere. The march stopped once under a
// thousandth was left and kept that thousandth, which a ball ten thousand
// times white turned into light across the whole planet.
func TestAGiantHidesAVeryBrightBallBehindIt(t *testing.T) {
	brightest := brightestOver(aGiantOver(thickBody))
	if brightest != 0 {
		t.Errorf("the ball behind the giant shows through its disc, up "+
			"to a luminance of %v of %v", brightest, float64(sunBright))
	}
}

// A body that is not opaque still shows what is behind it: it is the
// stop for want of light that hides the ball, and a march that ends
// with light left, as through a thin body, keeps it.
func TestAThinGiantStillShowsTheBallBehindIt(t *testing.T) {
	brightest := brightestOver(aGiantOver(thinBody))
	if brightest < leastSeen {
		t.Errorf("the thin giant hides the ball: the brightest it shows "+
			"is %v, and %v is the least that stays", brightest, leastSeen)
	}
}
