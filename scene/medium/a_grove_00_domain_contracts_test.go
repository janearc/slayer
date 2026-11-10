package medium

import (
	"math"
	"sync/atomic"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

func TestGroveRequiresEachWeatherAndCanopyDomainIndependently(t *testing.T) {
	weather := GroveWeather{Extinction: .008, Depth: 8, Coverage: 1}
	tree := Canopy{Radius: .5, Height: .25}
	ground := func(x, z float64) float64 { return 0 }
	check := func(name string, w GroveWeather, trees []Canopy, floor func(float64, float64) float64, want bool) {
		t.Helper()
		type result struct {
			grove *Grove
			err   error
		}
		done := make(chan result, 1)
		go func() { g, e := NewGrove(w, trees, floor); done <- result{g, e} }()
		select {
		case got := <-done:
			if want && got.grove != nil && len(got.grove.canopies) > 9*len(trees) {
				t.Fatalf("%s: %d canopy buckets for %d trees", name, len(got.grove.canopies), len(trees))
			}
			if (got.err == nil) != want || (got.grove != nil) != want {
				t.Fatalf("%s: grove %v, error %v, valid %t", name, got.grove, got.err, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("%s: canopy indexing did not finish within one second", name)
		}
	}
	check("positive", weather, []Canopy{tree}, ground, true)
	thin := weather
	thin.Depth = .1
	check("thin layer with broad canopy", thin, []Canopy{{Radius: 1000, Height: 20}}, ground, true)
	w := weather
	w.Extinction = 0
	check("zero extinction", w, []Canopy{tree}, ground, true)
	check("nil ground", weather, []Canopy{tree}, nil, false)
	check("no canopies", weather, nil, ground, false)
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		for _, field := range []string{"extinction", "depth", "coverage", "floor", "wind x", "wind y", "wind z"} {
			w := weather
			switch field {
			case "extinction":
				w.Extinction = v
			case "depth":
				w.Depth = v
			case "coverage":
				w.Coverage = v
			case "floor":
				w.GroundFloor = v
			case "wind x":
				w.Wind.X = v
			case "wind y":
				w.Wind.Y = v
			case "wind z":
				w.Wind.Z = v
			}
			check(field, w, []Canopy{tree}, ground, false)
		}
		for _, field := range []string{"radius", "height", "x", "y", "z"} {
			canopy := tree
			switch field {
			case "radius":
				canopy.Radius = v
			case "height":
				canopy.Height = v
			case "x":
				canopy.At.X = v
			case "y":
				canopy.At.Y = v
			case "z":
				canopy.At.Z = v
			}
			check("canopy "+field, weather, []Canopy{canopy}, ground, false)
		}
	}
	for _, field := range []string{"extinction", "depth", "coverage", "coverage above one", "floor above tree", "zero radius", "zero height"} {
		w, canopy := weather, tree
		switch field {
		case "extinction":
			w.Extinction = -math.SmallestNonzeroFloat64
		case "depth":
			w.Depth = 0
		case "coverage":
			w.Coverage = -math.SmallestNonzeroFloat64
		case "coverage above one":
			w.Coverage = math.Nextafter(1, 2)
		case "floor above tree":
			w.GroundFloor = math.SmallestNonzeroFloat64
		case "zero radius":
			canopy.Radius = 0
		case "zero height":
			canopy.Height = 0
		}
		check(field, w, []Canopy{canopy}, ground, false)
	}
}

func TestGroveBoundsFinestAndTranslatedCanopyEdges(t *testing.T) {
	weather := GroveWeather{Extinction: .008, Depth: 8, Coverage: 1}
	trees := []Canopy{
		{At: maths.V(3, 2, -5), Radius: 2, Height: 1},
		{At: maths.V(-8, 0, 7), Radius: 3, Height: 20},
	}
	grove, err := NewGrove(weather, trees, func(x, z float64) float64 { return .5 })
	if err != nil {
		t.Fatal(err)
	}
	want := maths.Box{Min: maths.V(-11, 0, -7), Max: maths.V(5, 20, 10)}
	if got := grove.Bounds(); got != want {
		t.Fatalf("bounds %v, want %v", got, want)
	}
	if got := grove.Finest(); got != 2 {
		t.Fatalf("finest %g, want 2 m", got)
	}
	// At 2 m above the ground, smoothstep(1/4) = 5/32.
	const expected = .008 * (1 - 5.0/32)
	if got := grove.Density(maths.V(3, 2.5, -5)); !(math.Abs(got-expected) <= 1e-15) {
		t.Fatalf("translated canopy density %g, want %g", got, expected)
	}
	// The short canopy ends at 3 m, inside the tall canopy's bounding box.
	atTop := maths.V(3, 3, -5)
	if !(grove.Density(atTop) > 0) {
		t.Fatal("short canopy lost its top boundary")
	}
	if got := grove.Density(maths.V(3, math.Nextafter(3, 4), -5)); got != 0 {
		t.Fatalf("past short canopy: %g", got)
	}
	for _, p := range []maths.Vec{maths.V(5, 2, -5), maths.V(3, 2, -7), maths.V(3, 2, -3), maths.V(1, 2, -5)} {
		if d := grove.Density(p); d != 0 {
			t.Fatalf("radial boundary %v has density %g", p, d)
		}
	}
	for _, p := range []maths.Vec{maths.V(3, .5, -5), maths.V(3, .4, -5), maths.V(-8, 8.5, 7)} {
		if d := grove.Density(p); d != 0 {
			t.Fatalf("vertical boundary %v has density %g", p, d)
		}
	}
}

func TestGroveTaperUsesTheOuterThirtyPercentOnly(t *testing.T) {
	weather := GroveWeather{Extinction: .008, Depth: 8, Coverage: 1, Fade: true}
	grove, err := NewGrove(weather, []Canopy{{At: maths.V(3, 0, -5), Radius: 10, Height: 20}},
		func(x, z float64) float64 { return 0 })
	if err != nil {
		t.Fatal(err)
	}
	// Halfway up has half the extinction. At radius 85%, the radial
	// smoothstep is halfway through its 70%-100% transition, hence 1/2.
	for _, row := range []struct{ radius, want float64 }{{0, .004}, {.5, .004}, {.7, .004}, {.85, .002}, {1, 0}} {
		got := grove.Density(maths.V(3+10*row.radius, 4, -5))
		if !(math.Abs(got-row.want) <= 1e-15) {
			t.Fatalf("radius %g: density %g, want %g", row.radius, got, row.want)
		}
	}
}

func TestGroveDoesNotSampleTerrainOutsideItsBounds(t *testing.T) {
	var calls atomic.Int64
	grove, err := NewGrove(GroveWeather{Extinction: .01, Depth: 10, Coverage: 1},
		[]Canopy{{Radius: 2, Height: 3}}, func(x, z float64) float64 { calls.Add(1); return 0 })
	if err != nil {
		t.Fatal(err)
	}
	// Bounds are an evaluation boundary too: a sampled terrain callback
	// may only cover the prescribed footprint.
	for _, p := range []maths.Vec{maths.V(-3, 1, 0), maths.V(3, 1, 0), maths.V(0, -1, 0),
		maths.V(0, 4, 0), maths.V(0, 1, -3), maths.V(0, 1, 3)} {
		if got := grove.Density(p); got != 0 {
			t.Fatalf("outside field %v: %g", p, got)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("sampled terrain %d times outside field", calls.Load())
	}
}
