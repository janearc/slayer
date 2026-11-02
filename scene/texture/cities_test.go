package texture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/maths/noise"
)

// scatter is the i-th of a spread of directions over the sphere.
func scatter(i int) maths.Vec {
	k := int64(i)
	return maths.V(noise.Hash(1, k)-0.5, noise.Hash(2, k)-0.5, noise.Hash(3, k)-0.5).Unit()
}

// Cities light some of the land and none of the sea, more in the
// middle latitudes than at the equator, more on a coast than inland,
// clumped round metros, and are dark with no land to stand on.
func TestCitiesLightTheLandAsTheMarbleDoes(t *testing.T) {
	land := func(p maths.Vec) bool { return p.X > 0 }
	coast := func(p maths.Vec) bool { return p.Z > 0 }
	c := Cities{Tiling: hexes.Sphere{N: 6}, Land: land, Coast: coast, Rungs: 3,
		Share: 0.2, Metro: 0.08, Glow: maths.RGB(1, 0.8, 0.5), Seed: 9}
	var lit, dark, mid, midN, eq, eqN, shore, shoreN, inland, inlandN int
	for i := range 6000 {
		p := scatter(i)
		if p.X <= 0.05 {
			continue
		}
		on := c.At(p).Luminance() > 0
		if on {
			lit++
		} else {
			dark++
		}
		lat := math.Abs(math.Asin(p.Y)) / maths.Degree
		switch {
		case lat > 25 && lat < 45:
			midN++
			if on {
				mid++
			}
		case lat < 10:
			eqN++
			if on {
				eq++
			}
		}
		if p.Z > 0 {
			shoreN++
			if on {
				shore++
			}
		} else {
			inlandN++
			if on {
				inland++
			}
		}
	}
	if lit < 100 || dark < 1000 {
		t.Errorf("%d lit and %d dark points on the land", lit, dark)
	}
	if !(float64(mid)/float64(midN) > 2*float64(eq)/float64(eqN)) {
		t.Errorf("the middle latitudes light %d of %d, the equator %d of %d", mid, midN, eq, eqN)
	}
	if !(float64(shore)/float64(shoreN) > 1.2*float64(inland)/float64(inlandN)) {
		t.Errorf("the coast lights %d of %d, inland %d of %d", shore, shoreN, inland, inlandN)
	}
	for i := range 2000 {
		p := scatter(i)
		if p.X >= -0.05 {
			continue
		}
		if c.At(p) != (maths.Light{}) {
			t.Fatalf("a city at sea, at %v", p)
		}
	}
	if (Cities{}).At(maths.XAxis) != (maths.Light{}) {
		t.Error("cities with nothing to stand on")
	}
}
