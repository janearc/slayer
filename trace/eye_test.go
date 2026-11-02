package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// A look off a mirror has come no way that is not accounted for, and
// has one bounce more; a look along a segment keeps its bounces and
// its pixel, and has come as far as it is told. Both are later legs,
// no longer the eye's own ray.
func TestWhatALookCarries(t *testing.T) {
	seen := look{pixel: 0.01, bounce: 1, come: 5}
	off := seen.mirrored(alongZ)
	if off.come != 0 || off.bounce != 2 || off.pixel != 0.01 ||
		off.ray != alongZ || !off.onward || seen.onward {
		t.Errorf("off a mirror the look is %+v", off)
	}
	on := seen.along(alongZ, 7)
	if on.come != 7 || on.bounce != 1 || on.pixel != 0.01 ||
		on.ray != alongZ || !on.onward {
		t.Errorf("along a segment the look is %+v", on)
	}
}

// A moving eye inside a field sees the sky crowd toward the way it is
// going, which is aberration: to an eye at half the speed of light
// along x, the light it sees square across its way, down z, came from
// behind that, half way toward minus x. celestial's hole's own Arriving
// says the same of such an eye.
//
// A weak field bends nothing, so that is all that happens, and an eye
// held still sees straight down z.
func TestAMovingEyeInAFieldSeesAberration(t *testing.T) {
	const speed = 0.5
	stretch := 1 / math.Sqrt(1-speed*speed)
	across := maths.Ray{At: maths.V(0, 5, 0), Dir: maths.V(0, 0, 1)}
	seenBy := func(field thing.Field) maths.Vec {
		world := thing.NewWorld()
		world.Sky = compass{}
		world.AddField(field)
		light := lone(world, across, 0).Light()
		return maths.V(light.R, light.G, light.B)
	}
	weak := thing.Field{Mass: 1e-6, Reach: 40}
	if got := seenBy(weak); !within(got.X, 0, 1e-4) ||
		!within(got.Z, 1, 1e-4) {
		t.Errorf("a still eye looking down z sees the sky toward %v",
			got)
	}
	moving := weak
	moving.EyeTime, moving.Eye = stretch, maths.V(stretch*speed, 0, 0)
	want := maths.V(-speed, 0, math.Sqrt(1-speed*speed))
	if got := seenBy(moving); !within(got.X, want.X, 1e-3) ||
		!within(got.Z, want.Z, 1e-3) {
		t.Errorf("a moving eye looking down z sees the sky toward %v, "+
			"want %v", got, want)
	}
}
