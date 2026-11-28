//go:build draw && darwin

package gpu

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// limbSide is the side of the frame in pixels, odd so one pixel is the
// disc's centre. Chosen.
const limbSide = 41

// limbLimbRow is the row of a pixel just inside the disc's limb: the
// ball of radius one, three away, fills 76 per cent of the half frame of a
// 50 degree lens, so row 6 of 41 is inside it and row 5 outside.
const limbLimbRow = 6

// glowingBall is a ball of one metre, three metres in front of the eye,
// glowing at the centre of its disc with a limb coefficient, or evenly
// when the coefficient is nought.
func glowingBall(coefficient float64) *thing.World {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "star", Glow: maths.Grey(1),
			Limb: coefficient},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 0, -3)})})
	return world
}

// The kernel darkens a glowing ball toward its limb as slayer's tracer
// does: the centre of the disc is the glow, and a pixel near the limb is
// well under it, on the gpu and on the cpu alike.
func TestTheGPUDarkensAGlowingBallAtItsLimbAsSlayerDoes(t *testing.T) {
	drawn, truth := bothWaysAt(t, glowingBall(0.6), maths.Local(), limbSide,
		limbSide, 1)
	middle := limbSide / 2
	for name, got := range map[string][2]float64{
		"gpu": {drawn.At(middle, middle).Light().G,
			drawn.At(middle, limbLimbRow).Light().G},
		"cpu": {truth.At(middle, middle).Light().G,
			truth.At(middle, limbLimbRow).Light().G},
	} {
		if !(got[0] > 0.95 && got[1] < 0.75*got[0]) {
			t.Errorf("%s: centre %v, limb %v", name, got[0], got[1])
		}
	}
	gpuLimb := drawn.At(middle, limbLimbRow).Light().G
	cpuLimb := truth.At(middle, limbLimbRow).Light().G
	if math.Abs(gpuLimb-cpuLimb) > 0.03 {
		t.Errorf("the limb is %v on the gpu and %v on the cpu", gpuLimb,
			cpuLimb)
	}
}

// The anti-test: a ball with no limb coefficient is as bright at its limb
// as at its centre on the gpu, as before.
func TestTheGPUDrawsAGlowingBallWithNoLimbEvenly(t *testing.T) {
	drawn, _ := bothWaysAt(t, glowingBall(0), maths.Local(), limbSide,
		limbSide, 1)
	middle := limbSide / 2
	centre := drawn.At(middle, middle).Light().G
	limb := drawn.At(middle, limbLimbRow).Light().G
	if math.Abs(centre-limb) > 0.02 {
		t.Errorf("an even ball is %v at its centre and %v at its limb",
			centre, limb)
	}
}
