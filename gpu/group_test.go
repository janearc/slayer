//go:build darwin

package gpu

import (
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func groupFixture(t *testing.T) (*shape.MeshGroup, *mesh.Mesh) {
	t.Helper()
	triangle, err := mesh.New([]maths.Vec{maths.V(-1, -1, 0), maths.V(1, -1, 0), maths.V(0, 1, 0)}, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	var members []shape.MeshMember
	for i := range 8 {
		members = append(members, shape.MeshMember{
			Mesh: triangle,
			Space: maths.Subspace{
				Origin:      maths.V(float64(i%4)*2, 0, float64(i/4)*-2),
				Orientation: maths.About(maths.YAxis, float64(i)*0.2), Scale: 0.5 + float64(i%2)*0.3,
			},
		})
	}
	group, err := shape.NewMeshGroup(members)
	if err != nil {
		t.Fatal(err)
	}
	return group, triangle
}

// Emission removes lighting and shadow choices from this silhouette check.
// A passing trunk render alone must not certify the foliage hierarchy.
func TestWholeCrownFoliageReachesGPU(t *testing.T) {
	for _, name := range []string{"coast-redwood", "giant-sequoia"} {
		t.Run(name, func(t *testing.T) {
			row, err := manufacture.KindNamed(name)
			if err != nil {
				t.Fatal(err)
			}
			made, err := manufacture.MakeWithComplexity(row, manufacture.Sow(5), "quick")
			if err != nil {
				t.Fatal(err)
			}
			things, err := made.Things(maths.Subspace{})
			if err != nil {
				t.Fatal(err)
			}
			world := thing.NewWorld()
			for _, one := range things {
				if one.Material.Name == "redwood foliage" || one.Material.Name == "sequoia foliage" {
					one.Material = &thing.Material{Glow: maths.RGB(0, 1, 0)}
					world.Add(one)
				}
			}
			height := made.Bounds().Max.Y
			from := maths.V(0, height*0.6, height*1.3)
			eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.V(0, height*0.6, 0))}
			picture := drawn(t, world, eye)
			green := 0
			for _, pixel := range picture.Pix {
				if pixel.G > 0 {
					green++
				}
			}
			if green < 100 {
				t.Fatalf("only %d foliage pixels reached the GPU", green)
			}
			apart := drawnBoth(t, world, eye)
			// Uniform green emission makes relative light difference a coverage
			// measurement: neither bark nor shadows can hide missing foliage.
			if apart.mean > 2e-3 {
				t.Fatalf("foliage CPU/GPU disagreement: %+v", apart)
			}
		})
	}
}

func TestRepeatedGroupsKeepOneLocalHierarchyAndOneMesh(t *testing.T) {
	group, _ := groupFixture(t)
	world := thing.NewWorld()
	for i := range 100 {
		world.Add(thing.Thing{Shape: group, Motion: at(float64(i)*10, 0, 0)})
	}
	encoded := newScene()
	if err := encoded.things(world, 0); err != nil {
		t.Fatal(err)
	}
	if len(encoded.groups) != 1 || encoded.counts.instances != 108 || encoded.counts.triangles != 1 {
		t.Fatalf("%d groups, %d instances, %d faces", len(encoded.groups), encoded.counts.instances, encoded.counts.triangles)
	}
}

func TestGPURendersGroupTransformsAndNormalsLikeCPU(t *testing.T) {
	group, _ := groupFixture(t)
	world := thing.NewWorld()
	space := maths.Subspace{Origin: maths.V(-3, 1, 0), Orientation: maths.About(maths.YAxis, 0.2), Scale: 1.5}
	world.Add(thing.Thing{Shape: group, Motion: thing.Still(space), Material: &thing.Material{Colour: maths.RGB(0.4, 0.7, 0.3)}})
	lamp, err := emit.NewDistant(maths.V(-1, 1, 3), maths.Grey(1))
	if err != nil {
		t.Fatal(err)
	}
	world.Lamp(lamp)
	from := maths.V(2, 4, 14)
	eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.V(0, 1, 0))}
	picture := drawn(t, world, eye)
	covered := 0
	for _, p := range picture.Pix {
		if p.Cover > 0 {
			covered++
		}
	}
	if covered < 100 {
		t.Fatal("group did not reach the image")
	}
	apart := drawnBoth(t, world, eye)
	if apart.cover != 0 || apart.mean > 1e-5 {
		t.Fatalf("group CPU/GPU difference: %+v", apart)
	}
}
