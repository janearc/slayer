//go:build draw && darwin

package gpu

import (
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	slayer "github.com/janearc/slayer/trace"
)

// The actual patterned species are visible, and both paths paint the same
// static bodies in local coordinates after independent rigid placements.
func TestGPUAndCPUPaintStaticSnakeProfiles(t *testing.T) {
	world := thing.NewWorld()
	for i, name := range []string{"boa-constrictor-occidentalis", "bitis-gabonica"} {
		row := gpuSnake(t, name)
		parent := maths.NewSubspace(maths.V(0, 0, float64(i)*1.1-.55), maths.Identity, 1)
		for _, part := range row.Parts {
			part.Motion = thing.Still(part.At(0).Within(parent))
			world.Add(part)
		}
	}
	world.Lamp(emit.Lit{At: maths.V(0, 5, 3), Strength: maths.RGB(90, 90, 90)})
	from := maths.V(0, 4, 5)
	eye := maths.NewSubspace(from, maths.LookAt(from, maths.V(0, 0, 0)), 1)
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	truth, err := slayer.Sampled(world, eye, lens, 0, 96, 64, 2)
	if err != nil {
		t.Fatal(err)
	}
	visible := 0
	for _, pixel := range truth.Pix {
		if pixel.Cover > 0 && pixel.R+pixel.G+pixel.B > 0 {
			visible++
		}
	}
	if visible < 20 {
		t.Fatalf("only %d lit snake pixels; comparison would be vacuous", visible)
	}
	within(t, world, eye, 96, 64, 2, tolerance{mean: 1e-5, largest: 5e-3})
}
