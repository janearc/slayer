//go:build draw

package gpu

import (
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// gem is an octahedron a unit from its middle to each corner, as a
// shape.Mesh: eight faces turning counterclockwise seen from outside,
// and at each corner a normal straight out from the middle. Shaded
// smooth it is a ball drawn with eight faces; flat it is its facets.
func gem(t *testing.T, smooth bool) *shape.Mesh {
	t.Helper()
	corners := []maths.Vec{maths.V(1, 0, 0), maths.V(-1, 0, 0),
		maths.V(0, 1, 0), maths.V(0, -1, 0), maths.V(0, 0, 1),
		maths.V(0, 0, -1)}
	var faces [][3]int
	for _, x := range []int{0, 1} {
		for _, y := range []int{2, 3} {
			for _, z := range []int{4, 5} {
				// an even count of corners on the far side keeps the
				// turn of x, y, z; an odd one wants it turned over
				if (x+y+z)%2 == 0 {
					faces = append(faces, [3]int{x, y, z})
				} else {
					faces = append(faces, [3]int{x, z, y})
				}
			}
		}
	}
	made, err := shape.NewMesh(corners, faces, corners, smooth)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// gemLit is a gem of clay over a floor, lit low from one side so the
// line where its light ends crosses the faces toward the eye.
func gemLit(t *testing.T, smooth bool) (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	clay := &thing.Material{Name: "clay", Colour: maths.Grey(0.7)}
	world.Add(thing.Thing{Shape: gem(t, smooth), Material: clay})
	floor := &thing.Material{Name: "floor", Colour: maths.Grey(0.4)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 200}, Material: floor,
		Motion: at(0, -201.2, 0)})
	world.Lamp(emit.Lit{At: maths.V(6, 1.5, 2), Strength: maths.Grey(40)})
	from := maths.V(0.6, 0.8, 4)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
}

// the gpu draws a shape.Mesh shaded smooth as slayer's tracer does: its
// corners' normals blended across each face, and the point left where
// the ray met it. slayer's own mesh is lifted toward the surface its
// normals draw and a shape.Mesh is not, so a lift taken here by mistake
// would move the line where the gem's light ends and show in this test
func TestTheGPUDrawsAPlainMeshShadedSmoothAsSlayerDoes(t *testing.T) {
	world, eye := gemLit(t, true)
	against(t, world, eye)
}

// the anti-test of the one above: the gem smooth and the gem flat are
// not one picture on the gpu, so the kernel is blending the normals it
// was given and not drawing the facets
func TestASmoothGemIsNotItsFacetsOnTheGPU(t *testing.T) {
	smooth, eye := gemLit(t, true)
	flat, _ := gemLit(t, false)
	found := compare(drawn(t, smooth, eye), drawn(t, flat, eye), 96, 64)
	if found.mean < 1e-2 {
		t.Errorf("the smooth gem and the flat one are %.5f apart: the "+
			"blend is not drawn", found.mean)
	}
}
