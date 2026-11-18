package trace

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"testing"
)

func TestClearCloudWaterRaySpacingBudget(t *testing.T) {
	water, err := thing.NewClearMediumWithScattering("spacing", maths.Grey(.1), maths.Grey(.01), 0, 1, 64, 100)
	if err != nil {
		t.Fatal(err)
	}
	ray := maths.Ray{Dir: maths.V(0, 0, 2)}
	previous := 0
	for _, tc := range []struct {
		gap  float64
		want int
	}{{.01, 1}, {1, 2}, {5, 7}, {25, 32}, {50, 64}} {
		got, ok := clearCloudWaterSteps(ray, 50, tc.gap, water)
		if !ok || got != tc.want || got < previous {
			t.Fatalf("gap=%g steps=%d valid=%t want=%d", tc.gap, got, ok, tc.want)
		}
		previous = got
	}
	for _, bad := range []float64{0, -1, math.NaN()} {
		if _, ok := clearCloudWaterSteps(ray, bad, 1, water); ok {
			t.Fatalf("accepted full=%g", bad)
		}
	}
	if got, ok := clearCloudWaterSteps(ray, math.Inf(1), math.Inf(1), water); !ok || got != 64 {
		t.Fatalf("unbounded capped ray: %d %t", got, ok)
	}
}

func TestClearCloudWaterStepWrapperPreservesDefaultAndBeer(t *testing.T) {
	world, _, seen, _ := diffuseSkyOceanFixture(t, 1.333, 8)
	water, err := thing.NewClearMediumWithScattering("step-water", maths.Grey(.1), maths.Grey(.01), 0, 1, 64, 20)
	if err != nil {
		t.Fatal(err)
	}
	water, err = water.WithDiffuseSkyInscatter(8)
	if err != nil {
		t.Fatal(err)
	}
	world.OceanSurface.Boundary.Material.ClearMedium = water
	seen.inside[0].clearMedium = water
	scene := stage(world, 0)
	ray := maths.Ray{At: maths.V(0, -3, 0), Dir: maths.V(1, 0, 0)}
	seen.ray = ray
	wrapped := oceanClearInscatter(scene, seen, ray, 2, water)
	if wrapped.R <= 0 || wrapped.G <= 0 || wrapped.B <= 0 {
		t.Fatalf("constant-source fixture produced no scattered light: %v", wrapped)
	}
	explicit := oceanClearInscatterSteps(scene, seen, ray, 2, water, 64)
	if wrapped != explicit {
		t.Fatalf("default wrapper changed: %v %v", wrapped, explicit)
	}
	// The horizontal path has constant depth/source. Exact cell integrals make
	// one-cell and sixty-four-cell source integration agree analytically.
	coarse := oceanClearInscatterSteps(scene, seen, ray, 2, water, 1)
	for _, pair := range [][2]float64{{coarse.R, wrapped.R}, {coarse.G, wrapped.G}, {coarse.B, wrapped.B}} {
		if math.Abs(pair[0]-pair[1]) > 1e-12 {
			t.Fatalf("constant source depends on cell count: %v %v", coarse, wrapped)
		}
	}
	for _, steps := range []int{0, -1, 65} {
		if got := oceanClearInscatterSteps(scene, seen, ray, 2, water, steps); got != (maths.Light{}) {
			t.Fatalf("invalid budget produced light %v", got)
		}
	}
}
