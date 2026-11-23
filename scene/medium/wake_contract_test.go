package medium_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"pgregory.net/rapid"
)

// wakeContractShape is the wake the properties are drawn from: its reach and
// clear, the radius of its one path, and the ranges of the grain's scale and
// of the gas's thick. A grain of one octave is the noise that can pass one,
// so it is the one that could take the gas below nought. beyond is how far
// past the reach a point may be drawn.
type wakeContractShape struct {
	reach, clear, path, scale, least, thick, beyond float64
	octaves                                         int
}

// wakeContractShapeOf reads the numbers the properties chose.
func wakeContractShapeOf(t *testing.T) wakeContractShape {
	t.Helper()
	return wakeContractShape{
		reach:   picked(t, "wake-contract-reach"),
		clear:   picked(t, "wake-contract-clear"),
		path:    picked(t, "wake-contract-path"),
		scale:   picked(t, "wake-contract-scale"),
		least:   picked(t, "wake-contract-least"),
		thick:   picked(t, "wake-contract-thick"),
		beyond:  picked(t, "wake-contract-beyond"),
		octaves: pickedCount(t, "wake-contract-octaves"),
	}
}

// wakeContractProperty is a property of wakes drawn within a shape.
type wakeContractProperty struct{ shape wakeContractShape }

// drawWake is a wake of one path, a grain of one octave and gas of a thick
// and a scale drawn within the property's range.
func (property wakeContractProperty) drawWake(draw *rapid.T) *medium.Wake {
	shape := property.shape
	grain, err := medium.NewGrain(rapid.Uint64().Draw(draw, "seed"),
		rapid.Float64Range(shape.least, shape.scale).Draw(draw,
			"scale"), shape.octaves)
	if err != nil {
		draw.Fatal(err)
	}
	path, err := medium.NewPath(maths.Local(), shape.path)
	if err != nil {
		draw.Fatal(err)
	}
	thick := rapid.Float64Range(nought, shape.thick).Draw(draw, "thick")
	wake, err := medium.NewWake(shape.reach, shape.clear, thick, grain,
		[]*medium.Path{path})
	if err != nil {
		draw.Fatal(err)
	}
	return wake
}

// coordinate is one coordinate of a drawn point, within the reach the point
// may be drawn to.
func (property wakeContractProperty) coordinate(
	draw *rapid.T, label string,
) float64 {
	reach := property.shape.reach * property.shape.beyond
	return rapid.Float64Range(-reach, reach).Draw(draw, label)
}

// drawWakePoint is a point within beyond times the wake's reach of the
// middle, so some are outside the ball.
func (property wakeContractProperty) drawWakePoint(draw *rapid.T) maths.Vec {
	return maths.V(property.coordinate(draw, "x"),
		property.coordinate(draw, "y"), property.coordinate(draw, "z"))
}

// densityIsFiniteAndNeverNegative holds that a drawn wake's density at a
// drawn point is a finite number of at least nought.
func (property wakeContractProperty) densityIsFiniteAndNeverNegative(
	draw *rapid.T,
) {
	wake := property.drawWake(draw)
	point := property.drawWakePoint(draw)
	density := wake.Density(point)
	if math.IsNaN(density) || math.IsInf(density, nought) || density < nought {
		draw.Fatalf("the wake is %v at %v", density, point)
	}
}

// a wake's density is a finite number of at least nought at every point,
// whatever the noise does: the cube of the raggedness cannot cross zero.
func TestWakeDensityIsFiniteAndNeverNegative(t *testing.T) {
	property := wakeContractProperty{wakeContractShapeOf(t)}
	rapid.Check(t, property.densityIsFiniteAndNeverNegative)
}

// the property is not met by a wake with no gas: outside its ball the density
// is exactly nought, and inside there is gas to be found, so the property
// above is about gas and not about nothing.
func TestWakeIsClearOutsideItsReachAndHasGasInside(t *testing.T) {
	each := orbitalWake(t)
	made, _ := built(t, each)
	beyond := picked(t, "wake-contract-beyond")
	outside := maths.V(each.reach*(unity+beyond), nought, nought)
	if density := made.Density(outside); density != nought {
		t.Errorf("gas %v outside the wake's reach", density)
	}
	found := false
	count := pickedCount(t, "wake-contract-points")
	for _, point := range wakePoints(t, each, count) {
		if made.Density(point) > nought {
			found = true
			break
		}
	}
	if !found {
		t.Error("no gas in the wake")
	}
}

// a wake's paths are its own: what Paths gives is a copy, and a reader who
// overwrites one does not change the wake.
func TestWakePathsAreCopies(t *testing.T) {
	each := orbitalWake(t)
	made, _ := built(t, each)
	paths := made.Paths()
	*paths[wakeFirstPath] = medium.Path{}
	again := made.Paths()
	if again[wakeFirstPath].Radius() != each.radii[wakeFirstPath] {
		t.Errorf("a reader's overwrite changed the wake's path to %v",
			again[wakeFirstPath].Radius())
	}
}

// two readings of a wake's paths do not share a path: each is its own copy.
func TestTwoReadingsOfAWakesPathsAreApart(t *testing.T) {
	made, _ := built(t, orbitalWake(t))
	first, second := made.Paths(), made.Paths()
	if len(first) != len(second) ||
		first[wakeFirstPath] == second[wakeFirstPath] {
		t.Error("two readings of the paths share a path")
	}
}

// wakeContractRays are the rays a wake's span is held to a ball's by: one
// through the middle, one across it and one that starts inside and slants.
func wakeContractRays(t *testing.T, each wakeCase) []maths.Ray {
	t.Helper()
	behind := picked(t, "wake-contract-behind")
	third := picked(t, "wake-contract-third")
	fifth := picked(t, "wake-contract-fifth")
	slant := maths.V(unity, unity, unity).Unit()
	return []maths.Ray{
		{At: maths.V(-behind*each.reach, nought, nought), Dir: maths.XAxis},
		{At: maths.V(-behind*each.reach, each.reach/pair, nought),
			Dir: maths.XAxis},
		{At: maths.V(each.reach/third, each.reach/fifth, nought),
			Dir: slant},
	}
}

// a wake's span is its ball's, found as every other ball's is: the very bits
// of a puff of the wake's reach, for a ray through the middle, one across it
// and one cut by near and far.
func TestWakeSpanIsTheSharedSphere(t *testing.T) {
	each := orbitalWake(t)
	made, _ := built(t, each)
	ball := medium.Puff{Radius: each.reach,
		Thick: picked(t, "wake-contract-ball-thick")}
	far := picked(t, "wake-contract-far") * each.reach
	for _, ray := range wakeContractRays(t, each) {
		for _, near := range []float64{nought, each.reach} {
			gotEnter, gotLeave, gotHit := made.Span(ray, near, far)
			wantEnter, wantLeave, wantHit := ball.Span(ray, near, far)
			if math.Float64bits(gotEnter) != math.Float64bits(wantEnter) ||
				math.Float64bits(gotLeave) != math.Float64bits(wantLeave) ||
				gotHit != wantHit {
				t.Errorf("span of %v from %v is %v %v %v, a ball's is %v %v %v",
					ray, near, gotEnter, gotLeave, gotHit, wantEnter,
					wantLeave, wantHit)
			}
		}
	}
	missHeight := picked(t, "wake-contract-miss-height")
	behind := picked(t, "wake-contract-behind")
	miss := maths.Ray{
		At:  maths.V(-behind*each.reach, missHeight*each.reach, nought),
		Dir: maths.XAxis}
	if _, _, hit := made.Span(miss, nought, far); hit {
		t.Error("a ray past the ball has a span")
	}
}
