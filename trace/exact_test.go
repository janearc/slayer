package trace

import (
	"fmt"
	"math"
	"sync"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// countedLens records the actual places the tracer asks the lens to see.
type countedLens struct {
	film.Pinhole
	mu     sync.Mutex
	places map[[2]float64]int
}

func (lens *countedLens) RayThrough(x, y int, across, down float64,
	w, h int, band maths.Band) maths.Ray {
	lens.mu.Lock()
	lens.places[[2]float64{across, down}]++
	lens.mu.Unlock()
	return lens.Pinhole.RayThrough(x, y, across, down, w, h, band)
}

func TestExactTracesTheRequestedPlaces(t *testing.T) {
	for _, count := range []int{1, 4, 7, 16, 25, 64} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			world := smoking()
			world.Filter = thing.TentFilter
			lens := &countedLens{places: make(map[[2]float64]int)}
			const width, height = 3, 2
			if _, err := Exact(world, overTheScene, lens, 0, width, height, count, nil); err != nil {
				t.Fatal(err)
			}
			if len(lens.places) != count {
				t.Fatalf("saw %d distinct ray places, want %d", len(lens.places), count)
			}
			for index := 0; index < count; index++ {
				across, down := film.Place(index, count, 1)
				if got := lens.places[[2]float64{across, down}]; got != width*height {
					t.Errorf("sample %d was traced %d times, want %d", index, got, width*height)
				}
			}
		})
	}
}

func TestExactOneMatchesRender(t *testing.T) {
	world := spreading()
	got, err := Exact(world, overTheScene, film.Pinhole{}, 0, spreadSide, spreadSide, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	samePixels(t, got, draw(world, overTheScene, film.Pinhole{}, 0, spreadSide, spreadSide), "one exact sample,")
}

// directionSky makes every sample's contribution independently calculable.
type directionSky struct{}

func (directionSky) Light(way maths.Vec) maths.Light {
	return maths.RGB(1+way.X, 1+way.Y, 1+way.Z)
}

func TestExactAveragesAllSevenSamples(t *testing.T) {
	world := &thing.World{Sky: directionSky{}, Filter: thing.BoxFilter}
	lens := film.Pinhole{}
	got, err := Exact(world, overTheScene, lens, 0, 1, 1, 7, nil)
	if err != nil {
		t.Fatal(err)
	}
	var expected maths.Light
	for index := 0; index < 7; index++ {
		across, down := film.Place(index, 7, 0)
		ray := film.LookThrough(overTheScene, lens, 0, 0, across, down, 1, 1, maths.Band{})
		expected = expected.Add(world.Sky.Light(ray.Dir).Scale(1.0 / 7))
	}
	pixel := got.At(0, 0)
	if math.Abs(float64(pixel.R)-expected.R) > 1e-6 || math.Abs(float64(pixel.G)-expected.G) > 1e-6 || math.Abs(float64(pixel.B)-expected.B) > 1e-6 {
		t.Fatalf("seven-sample mean = (%v, %v, %v), want %v", pixel.R, pixel.G, pixel.B, expected)
	}
}

func TestExactSquareCountsAgreeWithSpreadLegacySky(t *testing.T) {
	world := &thing.World{Sky: directionSky{}, Filter: thing.BoxFilter}
	for _, side := range []int{1, 2, 4, 5, 8} {
		got, err := Exact(world, overTheScene, film.Pinhole{}, 0, 3, 2, side*side, nil)
		if err != nil {
			t.Fatal(err)
		}
		old, err := Sampled(world, overTheScene, film.Pinhole{}, 0, 3, 2, side)
		if err != nil {
			t.Fatal(err)
		}
		for i, pixel := range got.Pix {
			prior := old.Pix[i]
			if math.Abs(float64(pixel.R-prior.R)) > 1e-6 || math.Abs(float64(pixel.G-prior.G)) > 1e-6 || math.Abs(float64(pixel.B-prior.B)) > 1e-6 {
				t.Fatalf("%d samples, pixel %d: exact %v, legacy %v", side*side, i, pixel, prior)
			}
		}
	}
}

func TestExactRejectsInvalidCounts(t *testing.T) {
	for _, count := range []int{0, -1, mostPixels + 1} {
		if _, err := Exact(smoking(), overTheScene, film.Pinhole{}, 0, 1, 1, count, nil); err == nil {
			t.Errorf("accepted %d samples", count)
		}
	}
}
