//go:build draw && darwin

package gpu

import (
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
	slayer "github.com/janearc/slayer/trace"
)

func TestNativeGiantDensityRemainsSmoothAtBoundedGridResolution(t *testing.T) {
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{
		Medium:   medium.Giant{Radius: 1, Depth: .08, Thick: 500},
		Material: &thing.Material{Glow: maths.Grey(1)},
	})
	eye := maths.Subspace{Origin: maths.V(0, 0, 4)}
	lens := film.Pinhole{Wide: 40 * maths.Degree}
	renderer, err := NewRenderer(32)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := renderer.Sampled(world, eye, lens, 0, 65, 49, 1)
	if err != nil {
		t.Fatal(err)
	}
	truth, err := slayer.Sampled(world, eye, lens, 0, 65, 49, 1)
	if err != nil {
		t.Fatal(err)
	}
	found := compare(truth, drawn, 65, 49)
	t.Logf("bounded giant density difference: %+v", found)
	if found.mean > .001 || found.largest > .01 {
		t.Fatal("bounded giant density developed voxel rings", found)
	}
}
