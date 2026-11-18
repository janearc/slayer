package trace

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestWakeFoamDirectWorldRejectsUnsampledSourceBeforeEncoding(t *testing.T) {
	foam := &thing.WakeFoamResponse{HalfWidth: 5, Length: 24,
		Wavelength: 3, Coverage: 1, Band: .2,
		Reflectance: maths.Grey(.9), Satin: .8}
	for _, ocean := range []*thing.OceanSurface{
		{DirectSunMeanPlane: true}, {DirectMoonMeanPlane: true}, {SpotInscatter: true},
		{ClearCloudSkySamples: 4},
	} {
		world := &thing.World{OceanSurface: ocean}
		world.Add(thing.Thing{Material: &thing.Material{Through: 1, WakeFoam: foam}})
		if _, err := PrepareClearState(world, nil); err == nil ||
			!strings.Contains(err.Error(), "participating source approximations") {
			t.Fatalf("unsupported direct world reached CPU/GPU encoding: %v", err)
		}
	}
}

func TestWakeFoamCoverageIsBoundedAndFollowsSceneClock(t *testing.T) {
	foam := thing.WakeFoamResponse{OriginEast: 5, Heading: math.Pi / 2,
		Speed: 6, HalfWidth: 5, Length: 48, Wavelength: 16,
		Coverage: 1, Band: 1, Reflectance: maths.RGB(.7, .8, .9), Satin: .8}
	point := maths.V(0, 0, 0) // five metres astern at the authored instant
	initial := foam.At(point, 0)
	if !(initial > 0 && initial <= 1) {
		t.Fatalf("coverage at the wake centre = %g, want (0,1]", initial)
	}
	if moved := foam.At(point, 1); moved == initial {
		t.Fatalf("coverage did not change as the wake translated: %g", moved)
	}
	if ahead := foam.At(maths.V(10, 0, 0), 0); ahead != 0 {
		t.Fatalf("coverage ahead of the craft = %g, want zero", ahead)
	}
	if outside := foam.At(maths.V(0, 0, 11), 0); outside != 0 {
		t.Fatalf("coverage outside the faded support = %g, want zero", outside)
	}
}

func TestWakeFoamAddsOpaqueDiffuseShareToClearWaterBoundary(t *testing.T) {
	medium, err := thing.NewClearMedium("foam test water", maths.Light{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	field, err := shape.NewHeightField([]float64{0, 0, 0, 0}, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	base := &thing.Material{Name: "clear water", Colour: maths.Grey(.1),
		Through: 1, Gloss: 1, Index: 1.333, ClearMedium: medium}
	makeWorld := func(foam bool) *thing.World {
		material := *base
		if foam {
			material.WakeFoam = &thing.WakeFoamResponse{
				OriginEast: 5, Heading: math.Pi / 2, HalfWidth: 5,
				Length: 48, Wavelength: 16, Coverage: 1, Band: 1,
				Reflectance: maths.RGB(.7, .8, .9), Satin: .8,
			}
		}
		world := thing.NewWorld()
		world.Add(thing.Thing{Shape: field, Material: &material})
		world.Add(thing.Thing{Shape: shape.Plane{},
			Material: &thing.Material{Colour: maths.Grey(.3)},
			Motion:   thing.Still(maths.Subspace{Origin: maths.V(0, -10, 0)})})
		world.Lamp(thing.Sun(maths.Up, maths.Grey(5)))
		return world
	}
	eye := mediumEye(maths.V(0, 2, 0), maths.Up.Neg())
	control, err := Sampled(makeWorld(false), eye,
		film.Pinhole{}, 0, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	foam, err := Sampled(makeWorld(true), eye,
		film.Pinhole{}, 0, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := foam.At(0, 0).R, control.At(0, 0).R; !(got > want+.05) {
		t.Fatalf("foam red = %g, clear control %g; opaque diffuse coverage did not contribute", got, want)
	}
}
