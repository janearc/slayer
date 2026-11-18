//go:build draw && darwin

package gpu

import (
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	seaocean "github.com/janearc/slayer/scene/ocean"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
	"math"
	"testing"
)

func TestMetalExactCalmInfinitePlaneMatchesCPUAboveBelowAndGrazing(t *testing.T) {
	water, err := thing.NewClearMediumWithScattering("shifted infinite water", maths.RGB(.025, .015, .008), maths.RGB(.012, .018, .02), .8, 2, 32, 12)
	if err != nil {
		t.Fatal(err)
	}
	core, err := shape.NewHeightField([]float64{2.5, 2.5, 2.5, 2.5}, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	field, err := seaocean.NewInfiniteMeanContinuation(core, 2.5, 4)
	if err != nil {
		t.Fatal(err)
	}
	origin := maths.V(100, 7, -50)
	world := thing.NewWorld()
	world.Sky = sky.Gradient{Horizon: maths.Grey(1), Zenith: maths.Grey(1), Ground: maths.Grey(1)}
	boundary := world.Add(thing.Thing{Shape: field, Material: &thing.Material{Through: 1, Gloss: 1, Index: 1.333, ClearMedium: water}, Motion: thing.Still(maths.NewSubspace(origin, maths.Identity, .5))})
	moon, err := emit.NewDistant(maths.V(.6, .8, 0), maths.RGB(.03, .04, .05))
	if err != nil {
		t.Fatal(err)
	}
	world.Lamps = []thing.Lamp{moon}
	world.OceanSurface = &thing.OceanSurface{Boundary: boundary, Mean: origin, ExtentMetres: 20, MetresPerUnit: 2, DirectMoonMeanPlane: true, InfiniteMeanPlane: true, SunRoute: thing.OceanSunRouteMoon, SunLampIndex: 0, FillLampIndex: -1}
	const level = 8.25
	for _, row := range []struct {
		name      string
		y         float64
		direction maths.Vec
		under     bool
	}{
		{"below_down", level - 2, maths.V(0, -1, 0), true},
		{"below_up", level - 2, maths.Up, true},
		{"below_grazing", level - 2, maths.V(1, .0001, 0), true},
		{"above_down", level + 2, maths.V(0, -1, 0), false},
		{"above_grazing", level + 2, maths.V(1, -.0001, 0), false},
	} {
		t.Run(row.name, func(t *testing.T) {
			eye := opticalEye(maths.V(1100, row.y, -50), row.direction)
			var initial []trace.InitialMedium
			if row.under {
				initial = []trace.InitialMedium{{Index: 1.333, ClearMedium: water}}
			}
			cpu, err := trace.SampledWithInitialMedia(world, eye, film.Pinhole{}, 0, 1, 1, 1, initial)
			if err != nil {
				t.Fatal(err)
			}
			got, profile, err := SampledWithInitialMedia(Software, world, eye, film.Pinhole{}, 0, 1, 1, 1, initial)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%s %q CPU%v Metal%v", row.name, profile.Device, cpu.At(0, 0), got.At(0, 0))
			a, b := cpu.At(0, 0), got.At(0, 0)
			for channel, pair := range [][2]float32{{a.R, b.R}, {a.G, b.G}, {a.B, b.B}} {
				if math.Abs(float64(pair[0]-pair[1])) > math.Abs(float64(pair[0]))*3e-4+1e-8 {
					t.Fatalf("channel%d CPU%g Metal%g", channel, pair[0], pair[1])
				}
			}
			encoded, err := encodeKeptWithInitialMedia(world, nil, eye, film.Pinhole{}, 0, 1, 1, nil, initial)
			if err != nil {
				t.Fatal(err)
			}
			if encoded.counts.fields != 0 || encoded.counts.spheres != 1 || !encoded.oceanMoon.Unbounded || encoded.oceanMoon.Centre.Y != level {
				t.Fatalf("incorrect exact plane lowering: %+v", encoded.oceanMoon)
			}
		})
	}
}
