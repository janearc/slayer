package gpu

import (
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/typography"
	"golang.org/x/image/font/gofont/goregular"
)

// lettered is carrie in go's regular, bevelled and shaded smooth across
// her curves, turned a little on her line, on a floor under a lamp low
// to her left, seen from in front and above: each letter its own mesh,
// placed by a motion nested in the line's.
func lettered(t *testing.T) (*thing.World, maths.Subspace) {
	t.Helper()
	face, err := typography.Parse(goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	letters, err := typography.Make(face, typography.Text{Words: "carrie",
		Size: 2, Depth: 0.5, Bevel: 0.02, Smooth: 30 * maths.Degree})
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	clay := &thing.Material{Name: "clay", Colour: maths.RGB(0.7, 0.4, 0.3)}
	turned := thing.Spin{In: maths.Subspace{Origin: maths.V(0, -0.8, 0)},
		Axis: maths.Up, Phase: 15 * maths.Degree}
	for _, each := range typography.Things(letters, clay, turned) {
		world.Add(each)
	}
	world.Add(thing.Thing{Shape: hills(t, 10, false),
		Material: &thing.Material{Name: "floor",
			Colour: maths.Grey(0.5)},
		Motion: at(0, -1.6, 0)})
	world.Lamp(emit.Lit{At: maths.V(-8, 4, 6),
		Strength: maths.RGB(400, 380, 350)})
	from := maths.V(0, 2, 7)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 0, 0))}
}

// the gpu draws typography's letters as slayer's tracer does: meshes
// with bevels and walls shaded smooth, each standing where its line
// puts it
func TestTheGPUDrawsLettersAsSlayerDoes(t *testing.T) {
	world, eye := lettered(t)
	against(t, world, eye)
}

// the letters are in the picture, so the test above is asking the gpu
// to draw them: without them it is a different picture
func TestTheLettersAreInTheGPUsPicture(t *testing.T) {
	world, eye := lettered(t)
	bare := thing.NewWorld()
	bare.Add(*world.Things[len(world.Things)-1])
	bare.Lamps = world.Lamps
	found := compare(drawn(t, world, eye), drawn(t, bare, eye), 96, 64)
	if found.mean < 1e-2 {
		t.Errorf("with and without the letters %.5f apart", found.mean)
	}
}
