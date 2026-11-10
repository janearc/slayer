package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// Full coverage hides wind mistakes because it bypasses the patch mask.
// Use partial coverage and compare actual timed mist with explicit cardinal
// downwind vectors. The reference is the public medium, not skyward.
func TestGroveMistUsesHorizontalSceneWindAtPartialCoverage(t *testing.T) {
	for _, sample := range []struct {
		from  int
		drift maths.Vec
	}{
		{0, maths.V(0, 0, 5)}, {90, maths.V(-5, 0, 0)},
		{180, maths.V(0, 0, -5)}, {270, maths.V(5, 0, 0)},
	} {
		t.Run(fmt.Sprintf("from%d", sample.from), func(t *testing.T) {
			given := mustParse(t)
			given.set = onFixture(t, "", "")
			raw, err := os.ReadFile(given.set)
			if err != nil {
				t.Fatal(err)
			}
			weather := fmt.Sprintf(`  weather:
    wind:
      from: {value: %d, unit: deg}
      speed: {value: 5, unit: m/s}
    mist:
      kind: prescribed-mist
      extinction: {value: 0.006, unit: m^-1}
      depth: {value: 20, unit: m}
      coverage: 0.6
      seed: 23
  terrain:`, sample.from)
			text := strings.Replace(string(raw), "  terrain:", weather, 1)
			text = strings.Replace(text, "        count: 12", "        count: 12\n        atmosphere: {weather: mist, extent: canopy-and-ground, fade-at-boundary: true}", 1)
			if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			read := readBoundarySet(t, given.set)
			stands := read.Stands()
			if len(stands) != 1 {
				t.Fatal("fixture lost its single checked stand")
			}
			name, fade := stands[0].Atmosphere()
			if name != "mist" || !fade {
				t.Fatal("fixture did not retain its atmosphere")
			}
			stage := staging{given: given, recipes: read, world: thing.NewWorld()}
			if err := stage.terrain(read); err != nil {
				t.Fatal(err)
			}
			base := stage.ground.At(0, 0)
			w, h, step := stage.ground.Size()
			sx := (stage.ground.At(1, 0) - base) / step
			sz := (stage.ground.At(0, 1) - base) / step
			ground := func(x, z float64) float64 { return base + sx*x + sz*z - stage.lowest }
			if !(math.Abs(stage.ground.At(w-1, h-1)-stage.lowest-ground(float64(w-1)*step, float64(h-1)*step)) <= 1e-9) {
				t.Fatal("height fixture lost its affine ground reference")
			}
			centre := maths.V(10, ground(10, 10), 10)
			trees := []groveTree{{at: centre,
				bounds: maths.Box{Min: maths.V(-2, 0, -1), Max: maths.V(3, 80, 4)}}}
			if err := stage.grove(stands[0], trees); err != nil {
				t.Fatal(err)
			}
			if len(stage.world.Volumes) != 1 {
				t.Fatal("stand did not retain one shared mist medium")
			}
			volume := stage.world.Volumes[0]
			actual, ok := volume.Medium.(*medium.Grove)
			if !ok || volume.Material == nil || volume.Material.Colour != maths.Grey(.95) {
				t.Fatal("stand lost its actual grove medium or authored mist reflectance")
			}
			// The outermost x=3,z=4 corner gives a five-metre footprint.
			// Unequal opposite extents expose choosing the smaller side.
			canopy := []medium.Canopy{{At: centre, Radius: 5, Height: 80}}
			spec := medium.GroveWeather{Extinction: .006, Depth: 20, Coverage: .6, Seed: 23, Wind: sample.drift, Fade: true}
			reference, err := medium.NewGrove(spec, canopy, ground)
			if err != nil {
				t.Fatal(err)
			}
			if actual.Bounds() != reference.Bounds() {
				t.Fatal("actual mist footprint does not enclose its asymmetric 3-4-5 canopy")
			}
			spec.Wind = maths.Vec{}
			still, err := medium.NewGrove(spec, canopy, ground)
			if err != nil {
				t.Fatal(err)
			}
			peak, moving := 0.0, 0.0
			for _, at := range []time.Duration{2 * time.Second, 7 * time.Second, 13 * time.Second, 29 * time.Second} {
				got, want, fixed := actual.Tick(at), reference.Tick(at), still.Tick(at)
				for x := -2; x <= 2; x++ {
					for z := -2; z <= 2; z++ {
						for _, height := range []float64{1, 5, 10} {
							px, pz := 10+float64(x), 10+float64(z)
							point := maths.V(px, ground(px, pz)+height, pz)
							expected := want.Density(point)
							peak = math.Max(peak, expected)
							moving = math.Max(moving, math.Abs(expected-fixed.Density(point)))
							if gap := math.Abs(got.Density(point) - expected); !(gap <= .006*2e-12) {
								t.Fatalf("at %s/%v: actual mist differs from horizontal advection by %g", at, point, gap)
							}
						}
					}
				}
			}
			if !(peak > 0 && moving > peak*1e-6) {
				t.Fatal("partial-coverage controls did not expose actual wind advection")
			}
		})
	}
}
