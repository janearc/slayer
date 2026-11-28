//go:build draw && darwin

package gpu

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// evenShine is a texture that gives one light at every point: a ball
// that shines by it is as a glowing ball, but by a texture's own road.
type evenShine struct{ light maths.Light }

// At is the one light, wherever it is asked.
func (shine evenShine) At(maths.Vec) maths.Light { return shine.light }

// shiningBall is the glowing ball of limb_draw_test.go, shining by a
// texture and not by a glow, with a limb coefficient.
func shiningBall(coefficient float64) *thing.World {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "drop",
			Shine: evenShine{maths.Grey(1)}, Limb: coefficient},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 0, -3)})})
	return world
}

// A ball that shines by a texture is darkened toward its limb as a glowing
// one is, on the gpu and on the cpu alike: its crust is no brighter at the
// edge of the disc than the magma at its centre. The anti-test: with no
// coefficient the same ball is even.
func TestAShiningBallIsDarkenedAtItsLimbOnBothRenderers(t *testing.T) {
	middle := limbSide / 2
	drawn, truth := bothWaysAt(t, shiningBall(0.6), maths.Local(), limbSide,
		limbSide, 1)
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
	even, _ := bothWaysAt(t, shiningBall(0), maths.Local(), limbSide,
		limbSide, 1)
	if math.Abs(even.At(middle, middle).Light().G-
		even.At(middle, limbLimbRow).Light().G) > 0.02 {
		t.Errorf("a shining ball with no limb is not even")
	}
}
