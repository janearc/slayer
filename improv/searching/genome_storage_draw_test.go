//go:build draw && darwin

package searching

import (
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"testing"
	"time"
)

// Compare the old public-mesh representation with immutable geometry on the
// actual native path. This is a small correctness control, not a throughput
// benchmark or a reduced-detail production render.
func TestGenomeStorageNativeKeepsMovingLightAndDepth(t *testing.T) {
	stage := &heardStage{world: thing.NewWorld(), crowd: &multitude{}}
	stage.world.Sky = sky.Stars{Seed: 29, Share: .1, Size: .12, Bright: .2}
	stage.world.Lamp(thing.Sun(maths.V(-1, 2, 3), maths.RGB(.8, .6, .4)))
	for i, id := range []int{0, 41, 5, 10} {
		being := crowdBeing{identity: identityOf(id), scale: 1, at: maths.V(float64(i%2)*7-3.5, float64(i/2)*6-3, 0), members: 1}
		if err := stage.buildIndividual(&being, true); err != nil {
			t.Fatal(err)
		}
		stage.crowd.beings = append(stage.crowd.beings, being)
	}
	old := *stage.world
	old.Things = nil
	for _, part := range stage.world.Things {
		copy := *part
		if solid, ok := part.Shape.(*mesh.Mesh); ok {
			var err error
			copy.Shape, err = shape.NewMesh(solid.Points(), solid.Faces(), nil, false)
			if err != nil {
				t.Fatal(err)
			}
		}
		old.Add(copy)
	}
	current, err := gpu.NewFrames(gpu.FasterLighting)
	if err != nil {
		t.Fatal(err)
	}
	defer current.Close()
	reference, err := gpu.NewFrames(gpu.FasterLighting)
	if err != nil {
		t.Fatal(err)
	}
	defer reference.Close()
	eye := maths.NewSubspace(maths.V(0, 0, 23), maths.LookAt(maths.V(0, 0, 23), maths.Vec{}), 1)
	lens := film.Pinhole{Wide: 60 * maths.Degree}
	for index := range 2 {
		at := 277*time.Second + time.Duration(index)*time.Second/6
		stage.updateMultitude(at.Seconds())
		for i, part := range stage.world.Things {
			old.Things[i].Motion = part.Motion
		}
		got, profile, err := current.Draw(stage.world, uint64(index+1), eye, lens, at, 192, 108, 1)
		if err != nil {
			t.Fatal(err)
		}
		want, _, err := reference.Draw(&old, uint64(index+1), eye, lens, at, 192, 108, 1)
		if err != nil {
			t.Fatal(err)
		}
		if profile.Device == "" || profile.Native.Candidates == 0 {
			t.Fatal("native geometry was not exercised", profile.Device, profile.Native)
		}
		for pixel, actual := range got.Pix {
			expected := want.Pix[pixel]
			a := [5]float32{actual.R, actual.G, actual.B, actual.Cover, actual.Depth}
			b := [5]float32{expected.R, expected.G, expected.B, expected.Cover, expected.Depth}
			for lane := range a {
				if math.Float32bits(a[lane]) != math.Float32bits(b[lane]) {
					t.Fatalf("frame %d pixel %d lane %d changed: %g vs %g", index, pixel, lane, a[lane], b[lane])
				}
			}
		}
	}
}
