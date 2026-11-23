package arid

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// marchStep is how far apart, in metres, a march along a ray looks for the
// surface: picked, a centimetre, well under the shortest wave a band holds.
const marchStep = 0.01

// marchFar is how far along a ray, in metres, a march looks before it
// says the ray stays under the water: picked, long enough for a ray that
// climbs at all to leave the largest wave.
const marchFar = 400.0

// seamRun is how far before the core's edge, in metres along its ray, a
// seam ray begins: picked, a millimetre.
const seamRun = 1e-3

// seamDepth is how far under the surface, in metres, a seam ray begins:
// picked, five millimetres, above the most the surface can rise in a seam
// ray's run since it can rise at most a metre a metre.
const seamDepth = 5e-3

// seamStep is how far apart, in metres, the starts a seam ray is looked
// for are: picked, a nanometre, so that each ray's arithmetic rounds on
// its own.
const seamStep = 1e-9

// seamTries is how many starts a seam ray is looked for among: picked, a
// thousand, many times the chance one rounds as wanted.
const seamTries = 1000

// seamEast is where along the core's edge a seam ray crosses it: picked.
const seamEast = -131.886821540482

// underneath is a ray that begins depth below the surface over an east and
// a south, as Distance measures it, and goes along dir. It is made from
// the surface it is met on, so it holds for any wave field.
func underneath(surface *InfiniteMeanContinuation, east, south, depth float64,
	dir maths.Vec) maths.Ray {
	height, _, _ := surface.finite.sample(east, south)
	return maths.Ray{
		At:  maths.V(east, height-depth/surface.finite.slant, south),
		Dir: dir.Unit(),
	}
}

// firstAbove is the first distance along the ray, from from by steps of
// step up to far, at which it is over the surface, found by marching and
// not by Hit, or false if it is under the surface all the way.
func firstAbove(surface *InfiniteMeanContinuation, ray maths.Ray, from, step,
	far float64) (float64, bool) {
	for along := from; along < far; along += step {
		if surface.Distance(ray.Point(along)) > 0 {
			return along, true
		}
	}
	return 0, false
}

// coreEdgeBox is the infinite column over the surface's core, whose sides
// are the core's edges.
func coreEdgeBox(surface *InfiniteMeanContinuation) maths.Box {
	half := surface.finite.half
	return maths.Box{Min: maths.V(-half, math.Inf(-1), -half),
		Max: maths.V(half, math.Inf(1), half)}
}

// seamRay is a ray that begins just under the surface inside the core, a
// millimetre before its south edge, and goes out across it along dir. Its
// start is looked for so that the point one representable step past where
// it leaves the core's box is still inside the core by rounding: the seam
// a march must not lose. It also returns where the ray leaves the box.
func seamRay(t *testing.T, surface *InfiniteMeanContinuation,
	dir maths.Vec) (maths.Ray, float64) {
	t.Helper()
	half := surface.finite.half
	box := coreEdgeBox(surface)
	for index := range seamTries {
		south := half - seamRun*dir.Unit().Z - float64(index)*seamStep
		ray := underneath(surface, seamEast, south, seamDepth, dir)
		_, leaves, crosses := box.Hit(ray, 0, marchFar)
		if !crosses {
			continue
		}
		past := math.Nextafter(leaves, math.Inf(1))
		probe := ray.Point(past)
		if math.Abs(probe.Z) <= half {
			return ray, leaves
		}
	}
	t.Fatal("no start among the tries rounds a ray's core seam inside")
	return maths.Ray{}, 0
}
