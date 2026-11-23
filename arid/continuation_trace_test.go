package arid_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

func flatContinuationWorld(t *testing.T, referencePlaneY float64) *thing.World {
	t.Helper()
	core, err := shape.NewHeightField(make([]float64, 81), 9, 8)
	if err != nil {
		t.Fatal(err)
	}
	water, err := arid.NewFlatMeanContinuation(core, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: water, Material: &thing.Material{
		Name: "synthetic-clear-interface", Gloss: 1, Through: 1,
		Index: 1.33, Rank: 0,
	}})
	planeSpace := maths.Subspace{Origin: maths.V(0, referencePlaneY, 0)}
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
		Glow: maths.Grey(1),
	}, Motion: thing.Still(planeSpace)})
	return world
}

func continuationPixel(t *testing.T, world *thing.World, eye maths.Subspace,
	initial []trace.InitialMedium,
) float64 {
	t.Helper()
	if len(initial) == 0 {
		picture, err := trace.Render(world, eye, film.Pinhole{}, 0, 1, 1)
		if err != nil {
			t.Fatal(err)
		}
		return float64(picture.At(0, 0).R)
	}
	picture, err := trace.RenderWithInitialMedia(world, eye,
		film.Pinhole{}, 0, 1, 1, initial)
	if err != nil {
		t.Fatal(err)
	}
	return float64(picture.At(0, 0).R)
}

func TestFlatMeanContinuationCrossesEachRegionOnceInBothDirections(t *testing.T) {
	world := flatContinuationWorld(t, -3)
	underwaterWorld := flatContinuationWorld(t, 3)
	want := 1 - maths.Fresnel(1, 1.33)
	for _, x := range []float64{0, 2, 2.001, 3, 3.999, -2, -2.001} {
		t.Run(fmt.Sprintf("x=%g", x), func(t *testing.T) {
			from := maths.V(x, 1, 0)
			eye := maths.Subspace{Origin: from,
				Orientation: maths.LookAt(from, maths.V(x, -3, 0))}
			got := continuationPixel(t, world, eye, nil)
			if math.Abs(got-want) > 2e-5 {
				t.Fatalf("air ray at x=%g transmitted %g, want one boundary %g",
					x, got, want)
			}
		})
		from := maths.V(x, -1, 0)
		eye := maths.Subspace{Origin: from,
			Orientation: maths.LookAt(from, maths.V(x, 3, 0))}
		got := continuationPixel(t, underwaterWorld, eye,
			[]trace.InitialMedium{{Index: 1.33, Rank: 0}})
		if math.Abs(got-want) > 2e-5 {
			t.Errorf("water ray at x=%g transmitted %g, want one boundary %g",
				x, got, want)
		}
	}

	// Past the explicit finite bound there is no second plane that could
	// double-cross the same clear medium. The reference plane is reached whole.
	from := maths.V(6.01, 1, 0)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(6.01, -3, 0))}
	if got := continuationPixel(t, world, eye, nil); got != 1 {
		t.Fatalf("ray beyond finite bound got %g, want the unbent reference", got)
	}
}

func TestFlatMeanContinuationGrazingParallelRayMissesWithinFiniteWork(t *testing.T) {
	world := flatContinuationWorld(t, -3)
	from := maths.V(-5.9, 1e-6, 0)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(5.9, 1e-6, 0))}
	picture, err := trace.Render(world, eye, film.Pinhole{}, 0, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if pixel := picture.At(0, 0); pixel.Cover != 0 || pixel.Depth != float32(math.Inf(1)) {
		t.Fatalf("grazing parallel ray fabricated a hit: cover=%g depth=%g",
			pixel.Cover, pixel.Depth)
	}
}

func TestInfiniteMeanPlaneCrossesBeyondCoreAndCollarInBothDirections(t *testing.T) {
	core, err := shape.NewHeightField(make([]float64, 81), 9, 8)
	if err != nil {
		t.Fatal(err)
	}
	water, err := arid.NewInfiniteMeanContinuation(core, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	makeWorld := func(referenceY float64) *thing.World {
		world := thing.NewWorld()
		boundary := world.Add(thing.Thing{Shape: water, Material: &thing.Material{
			Name: "infinite-seawater", Gloss: 1, Through: 1, Index: 1.33,
			Rank: 0,
		}})
		planeSpace := maths.Subspace{Origin: maths.V(0, referenceY, 0)}
		world.Add(thing.Thing{Shape: shape.Plane{}, Motion: thing.Still(planeSpace),
			Material: &thing.Material{Glow: maths.Grey(1)}})
		world.OceanSurface = &thing.OceanSurface{Boundary: boundary,
			InfiniteMeanPlane: true, Mean: maths.Vec{}, ExtentMetres: 8,
			MetresPerUnit: 1}
		return world
	}
	world, underwaterWorld := makeWorld(-3), makeWorld(3)
	want := 1 - maths.Fresnel(1, 1.33)
	for _, x := range []float64{-1000, -10, 10, 1000} {
		from := maths.V(x, 1, 0)
		eye := maths.Subspace{Origin: from,
			Orientation: maths.LookAt(from, maths.V(x, -3, 0))}
		image, err := trace.Render(world, eye, film.Pinhole{}, 0, 1, 1)
		if err != nil {
			t.Fatalf("air-to-water at x=%g: %v", x, err)
		}
		if got := float64(image.At(0, 0).R); math.Abs(got-want) > 2e-5 {
			t.Errorf("air ray at x=%g transmitted %g, want %g", x, got, want)
		}
		from = maths.V(x, -1, 0)
		eye = maths.Subspace{Origin: from,
			Orientation: maths.LookAt(from, maths.V(x, 3, 0))}
		image, err = trace.RenderWithInitialMedia(underwaterWorld, eye,
			film.Pinhole{}, 0, 1, 1,
			[]trace.InitialMedium{{Index: 1.33, Rank: 0}})
		if err != nil {
			t.Fatalf("water-to-air at x=%g: %v", x, err)
		}
		if got := float64(image.At(0, 0).R); math.Abs(got-want) > 2e-5 {
			t.Errorf("water ray at x=%g transmitted %g, want %g", x, got, want)
		}
	}
}
