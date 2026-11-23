package arid

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
)

// seenNear is a pixel narrower than any band's samples, in metres:
// picked, a millimetre.
const seenNear = 1e-3

// seenFar is a pixel wider than twice any band's samples, in metres:
// picked, a kilometre.
const seenFar = 1e3

// seenClose is how near two unit normals are taken to agree, picked.
const seenClose = 1e-12

// down is a ray's way falling onto the sea at a slant.
var down = maths.V(1, -0.5, 0).Unit()

// above is a point on the surface's own height at east and south, lifted
// by a hair so it is on the side above.
func above(sea *Sea, east, south float64) maths.Vec {
	return maths.V(east, sea.height(east, south)+seaTouch/2, south)
}

// A pixel finer than every band shows every band whole: nothing is lost,
// and on a place that is a sample of every band, where the smooth read
// and the bilinear one both pass through the samples, the normal is
// Normal's.
func TestSeaSeenByAFinePixelIsTheSpectralNormal(t *testing.T) {
	sea := fixtureSea(t)
	for _, place := range [][2]float64{{0, 0}, {32, -48}, {160, 64}} {
		point := above(sea, place[0], place[1])
		seen, lost := sea.Seen(point, point.Sub(down), seenNear)
		want := sea.Normal(point)
		if seen.Sub(want).Len() > seenClose || lost != 0 {
			t.Errorf("at %v: seen %v lost %g, want %v and none", place,
				seen, lost, want)
		}
	}
}

// A pixel wider than every band shows none: the normal is straight up
// and every band's slope variance is lost.
func TestSeaSeenByAWidePixelLosesEveryBand(t *testing.T) {
	sea := fixtureSea(t)
	point := above(sea, 3.3, -7.1)
	seen, lost := sea.Seen(point, point.Sub(down), seenFar)
	var total float64
	for _, slope := range sea.slopes {
		total += slope
	}
	if seen.Sub(maths.V(0, 1, 0)).Len() > seenClose ||
		math.Abs(lost-total) > seenClose*total {
		t.Errorf("seen %v lost %g, want up and %g", seen, lost, total)
	}
}

// Cubic convolution's weights share one whole at every fraction, and at
// a sample take that sample alone; anti: half way they are not the
// bilinear read's halves.
func TestCubicWeightsPassThroughTheSamples(t *testing.T) {
	for _, fraction := range []float64{0, 0.25, 0.5, 0.9} {
		weights := cubicWeights(fraction)
		sum := weights[0] + weights[1] + weights[2] + weights[3]
		if math.Abs(sum-1) > seenClose {
			t.Errorf("weights at %g sum to %g", fraction, sum)
		}
	}
	if cubicWeights(0) != [cubicTaps]float64{0, 1, 0, 0} {
		t.Errorf("weights at a sample %v", cubicWeights(0))
	}
	if half := cubicWeights(0.5); half[0] == 0 || half[1] == 0.5 {
		t.Errorf("weights half way %v are the bilinear read's", half)
	}
}

// A look leaving the sea downward from where a falling ray met it starts
// below the surface, by the clearance, and does not meet the surface it
// just crossed within a millimetre.
func TestSeaClearanceStartsALeavingLookOnItsSide(t *testing.T) {
	sea := fixtureSea(t)
	ray := maths.Ray{At: maths.V(0, 2, 0), Dir: down}
	along, met := sea.Hit(ray, 0, math.Inf(1))
	if !met {
		t.Fatal("the falling ray missed the sea")
	}
	point := ray.Point(along)
	drop := maths.V(0, -sea.Clearance(), 0)
	start := point.Add(drop)
	if gap := start.Y - sea.height(start.X, start.Z); !(gap < 0) {
		t.Errorf("a leaving look starts %g above the surface", gap)
	}
	bent := maths.Ray{At: start, Dir: maths.V(0.2, -1, 0).Unit()}
	if again, hit := sea.Hit(bent, 0, seenNear); hit {
		t.Errorf("the leaving look met the surface again at %g", again)
	}
}

// Anti: a ray that met the sea from below it sees the surface face down,
// so its mirror stays in the water and its bend leaves it.
func TestSeaSeenFromBelowFacesDown(t *testing.T) {
	sea := fixtureSea(t)
	point := maths.V(3.3, sea.height(3.3, -7.1)-seaTouch/2, -7.1)
	seen, _ := sea.Seen(point, point.Add(down), seenNear)
	if !(seen.Y < 0) {
		t.Errorf("seen from below %v, want it facing down", seen)
	}
}

// Anti: the share of a band a middling pixel shows is between none and
// all, so the variance lost is between nought and the whole.
func TestSeaSeenByAMiddlingPixelLosesPart(t *testing.T) {
	sea := fixtureSea(t)
	finest := sea.bands[len(sea.bands)-1]
	texel := finest.Extent / float64(finest.N)
	// the finest band fades from a pixel across keep samples to one across
	// drop samples: the pixel is taken half way between
	keep, drop := oceantex.PixelShareSpan()
	point := above(sea, 0, 0)
	_, lost := sea.Seen(point, point.Sub(down), (keep+drop)/2*texel)
	var total float64
	for _, slope := range sea.slopes {
		total += slope
	}
	if !(lost > 0) || !(lost < total) {
		t.Errorf("lost %g of %g, want some and not all", lost, total)
	}
}
