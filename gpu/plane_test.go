//go:build draw && darwin

package gpu

import (
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"sync/atomic"
	"testing"
)

// The same infinite plane must catch primary, reflection and shadow rays.
func TestNativePlaneMatchesCPU(t *testing.T) {
	world, eye := hall(t)
	world.Things[0].Shape = shape.Plane{}
	againstAt(t, world, eye, 65, 43, 1)
	// A tilted, translated plane preserves local paint coordinates.
	world.Things[0].Motion = thing.Still(maths.Subspace{
		Origin: maths.V(0, -1, 0), Orientation: maths.LookAt(maths.V(0, 0, 0), maths.V(.1, .1, -1))})
	againstAt(t, world, eye, 65, 43, 1)
}

func TestNativePlaneIsVisibleWithoutOtherGeometry(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Colour: maths.Grey(.8)}})
	sun, err := emit.NewDistant(maths.Up, maths.Grey(1))
	if err != nil {
		t.Fatal(err)
	}
	world.Lamp(sun)
	eye := maths.Subspace{Origin: maths.V(0, 2, 3),
		Orientation: maths.LookAt(maths.V(0, 2, 3), maths.Vec{})}
	againstAt(t, world, eye, 32, 24, 1)
	encoded, err := encode(world, eye, film.Pinhole{}, 0, 32, 24)
	if err != nil {
		t.Fatal(err)
	}
	if encoded.counts.spheres != 1 || math.Float32bits(number(encoded.spheres, 6)) != 1 {
		t.Fatal("analytic plane missing from GPU buffer")
	}
}

func TestNativeVolumeGridBoundIsExplicit(t *testing.T) {
	for _, cells := range []int{0, 1, 193} {
		if _, err := NewRenderer(cells); err == nil {
			t.Fatal("invalid volume grid accepted", cells)
		}
	}
	renderer, err := NewRenderer(24)
	if err != nil {
		t.Fatal(err)
	}
	if renderer.volumeCells != 24 {
		t.Fatal("volume grid bound lost")
	}
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1},
		Material: &thing.Material{Glow: maths.Grey(1)}})
	encoded, err := encodeGrown(world, nil, maths.Subspace{}, film.Pinhole{}, 0, 16, 12, 24)
	if err != nil {
		t.Fatal(err)
	}
	grid := boxOf(medium.Puff{Radius: 1}, nil, nil, 24)
	if grid.cells != [3]int{24, 24, 24} || encoded.counts.volumes != 1 {
		t.Fatal("bounded volume missing", grid.cells)
	}
	if _, err = renderer.Sampled(world, maths.Subspace{}, film.Pinhole{}, 0, 16, 12, 1); err != nil {
		t.Fatal(err)
	}
}

type nativeCountedMedium struct {
	medium.Puff
	asks *atomic.Int64
}

func (m nativeCountedMedium) Density(p maths.Vec) float64 {
	m.asks.Add(1)
	return m.Puff.Density(p)
}
func (m nativeCountedMedium) Finest() float64 { return .001 }

func TestNativeRendererCarriesVolumeBoundToSampling(t *testing.T) {
	var asks atomic.Int64
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: nativeCountedMedium{
		Puff: medium.Puff{Radius: 1}, asks: &asks}})
	renderer, err := NewRenderer(24)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = renderer.Sampled(world, maths.Subspace{}, film.Pinhole{}, 0, 4, 3, 1); err != nil {
		t.Fatal(err)
	}
	if got := asks.Load(); got != 24*24*24 {
		t.Fatal("renderer dropped volume-grid bound", got)
	}
}
