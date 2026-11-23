package arid

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// packedHit is the sea's search done with nothing but what Pack gave, as
// a device does it: the bands summed, stepped by the packed bound from
// the slab's top, met within the packed touch, and ended on the mean
// after the packed most steps. It is for a ray that falls from above.
func packedHit(packed SeaPacked, ray maths.Ray) (float64, bool) {
	rate := math.Abs(ray.Dir.Y) +
		packed.Steepest*math.Hypot(ray.Dir.X, ray.Dir.Z)
	along := math.Max((packed.Reach-ray.At.Y)/ray.Dir.Y, 0)
	for range packed.MostSteps {
		point := ray.Point(along)
		gap := point.Y - packedHeight(packed, point.X, point.Z)
		if math.Abs(gap) < packed.Touch {
			return along, true
		}
		along += math.Max(math.Abs(gap), packed.Touch) / rate
	}
	return -ray.At.Y / ray.Dir.Y, true
}

// packedHeight is the packed bands' summed height at an east and a south,
// longest band first, each band after the first read where the longer
// bands' water came from, by the push the packed bands carry, as the CPU
// sea does (Sea.waves). A device that does not read the push yet, as the
// GPU's, sums another surface than the one Hit searches.
func packedHeight(packed SeaPacked, east, south float64) float64 {
	var total float64
	last := len(packed.Bands) - 1
	for index, band := range packed.Bands {
		half := band.Extent / 2
		if index == last {
			return total + periodicSample(band, band.Height, half+east,
				half-south)
		}
		height, pushEast, pushNorth := carriedSample(band, half+east,
			half-south)
		total += height
		east, south = east-pushEast, south+pushNorth
	}
	return total
}

// A packed sea is the sea Hit searches: its own bands, its slab, its step
// bound and the search's tolerances, so a device that steps by them steps
// as Hit does.
func TestSeaPackIsTheSeaHitSearches(t *testing.T) {
	sea := fixtureSea(t)
	packed := sea.Pack()
	if len(packed.Bands) != len(sea.bands) {
		t.Fatalf("packed %d bands, the sea has %d", len(packed.Bands),
			len(sea.bands))
	}
	for index, band := range packed.Bands {
		if band != sea.bands[index] {
			t.Errorf("band %d is a copy, not the sea's own", index)
		}
	}
	if packed.Reach != sea.Bounds().Max.Y {
		t.Errorf("reach %g, the sea's slab tops at %g", packed.Reach,
			sea.Bounds().Max.Y)
	}
	// a ray stepped by the packed bound and tolerances meets the sea
	// where Hit does
	ray := maths.Ray{At: maths.V(0, 2, 0), Dir: maths.V(1, -0.05, 0.3)}
	want, met := sea.Hit(ray, 0, math.Inf(1))
	got, found := packedHit(packed, ray)
	if met != found || math.Abs(got-want) > packed.Touch {
		t.Errorf("packed search met %t at %g, Hit %t at %g", found, got,
			met, want)
	}
}

// Anti-test: the packed numbers carry the sea's surface. A height summed
// from the packed bands as the sea sums them is the sea's height; with one
// band left out it is not.
func TestSeaPackWithABandLeftOutIsAnotherSurface(t *testing.T) {
	sea := fixtureSea(t)
	packed := sea.Pack()
	east, south := 13.7, -41.2
	whole := (&Sea{bands: packed.Bands}).height(east, south)
	if whole != sea.height(east, south) {
		t.Fatalf("the packed bands sum to %g, the sea to %g", whole,
			sea.height(east, south))
	}
	short := (&Sea{bands: packed.Bands[1:]}).height(east, south)
	if short == whole {
		t.Errorf("leaving out the longest band left the height at %g", whole)
	}
}
