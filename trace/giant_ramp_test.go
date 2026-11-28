package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// the numbers of the giant's ramp test: how many rays go across the lit
// side, from the nearest to the farthest of the disc's share, how many
// steps the fine march takes against the usual strides, and the most
// the usual march may differ from the fine one, as a share.
const (
	rampRays     = 40
	rampNearest  = 0.1
	rampFarthest = 0.85
	rampFine     = 2000
	rampLeast    = 0.03
)

// aLitGiant is a giant of grey gas under a sun from plus x, with no
// ball behind it.
func aLitGiant() *thing.World {
	made := thing.NewWorld()
	made.AddVolume(thing.Volume{Medium: medium.Giant{Radius: 1,
		Depth: 0.08, Thick: 500},
		Material: &thing.Material{Name: "gas", Colour: maths.Grey(0.5)}})
	made.Lamp(thing.Sun(maths.XAxis, maths.Grey(1)))
	return made
}

// A giant's light is the same however finely the march is cut. The
// steps used to be spread over the whole chord, mostly in the dark below
// the ramp, so how many fell in the ramp changed with the ray, and the
// disc was banded in rings where that count changed. Cut into the usual
// strides it is now within three hundredths of the same march cut forty
// times as fine, at every ray across the lit side; it was nine tenths out.
func TestAGiantsLightDoesNotDependOnHowTheMarchIsCut(t *testing.T) {
	scene := stage(aLitGiant(), 0)
	var worst float64
	for step := range rampRays {
		reach := rampNearest + (rampFarthest-rampNearest)*
			float64(step)/rampRays
		ray := maths.Ray{At: maths.V(reach, 0, 20), Dir: maths.V(0, 0, -1)}
		usual := throughGain(scene, ray, math.Inf(1), nil, strides).added
		fine := throughGain(scene, ray, math.Inf(1), nil, rampFine).added
		worst = max(worst, math.Abs(usual.Luminance()-fine.Luminance())/
			fine.Luminance())
	}
	t.Logf("the usual march is within %v of the fine one at worst", worst)
	if worst > rampLeast {
		t.Errorf("the usual march differs from the fine one by %v, "+
			"more than %v", worst, rampLeast)
	}
}
