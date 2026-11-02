package mesh

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// this file's name sorts before any that builds a mesh. game mutants
// runs the suite failfast, in file order, so a change that makes a
// cut, a root or a walk never end is killed here, by name, before a
// mesh's build hangs. renamed to sort later, those changes time out.

// the right triangle with corners (0,0,0), (4,0,0) and (0,3,0), whose
// legs are 4 and 3 and hypotenuse 5, is seen from points over each of
// its seven regions and on the lines between: the point nearest on it,
// as its corners' weights, and how far that is, squared, each worked
// out by hand. the line 3x+4y=12 is the hypotenuse.
var closestOnTriangle = []struct {
	point   maths.Vec
	weights weights
	squared float64
}{
	// over a corner, and in the plane's lines through it
	{maths.V(-1, -1, 2), weights{1, 0, 0}, 6},
	{maths.V(0, 0, 5), weights{1, 0, 0}, 25},
	{maths.V(6, -1, 1), weights{0, 1, 0}, 6},
	{maths.V(4, 0, -2), weights{0, 1, 0}, 4},
	{maths.V(5, 1, 0), weights{0, 1, 0}, 2},
	{maths.V(-1, 5, -1), weights{0, 0, 1}, 6},
	{maths.V(0, 3, 1), weights{0, 0, 1}, 1},
	{maths.V(-1, 4, 0), weights{0, 0, 1}, 2},
	// over a side: the line along AB, AC, then BC
	{maths.V(1, -2, 3), weights{0.75, 0.25, 0}, 13},
	{maths.V(2, 0, 1), weights{0.5, 0.5, 0}, 1},
	{maths.V(3, -0.5, 0), weights{0.25, 0.75, 0}, 0.25},
	{maths.V(-2, 1.5, 1), weights{0.5, 0, 0.5}, 5},
	{maths.V(0, 1.5, 2), weights{0.5, 0, 0.5}, 4},
	{maths.V(-0.5, 2, 0), weights{1.0 / 3, 0, 2.0 / 3}, 0.25},
	{maths.V(3, 3, 1), weights{0, 0.48, 0.52}, 4.24},
	{maths.V(4.5, 2, 0), weights{0, 0.84, 0.16}, 3.61},
	{maths.V(2, 1.5, 0), weights{0, 0.5, 0.5}, 0},
	// over the face, which is its own nearest point
	{maths.V(1, 1, 2), weights{5.0 / 12, 0.25, 1.0 / 3}, 4},
	{maths.V(1, 1, -3), weights{5.0 / 12, 0.25, 1.0 / 3}, 9},
	{maths.V(3, 0.5, 1), weights{1.0 / 12, 0.75, 1.0 / 6}, 1},
	{maths.V(0.5, 0.5, 0), weights{17.0 / 24, 0.125, 1.0 / 6}, 0},
}

// each point's nearest on the triangle is where the hand says, whichever
// corner is called first and whichever way the axes run: the corners
// turned round three ways and the parts of every point turned round
// three, which changes which line of the code each point reaches.
func TestTheNearestPointOfAFaceIsInTheRegionItIsOver(t *testing.T) {
	corners := []maths.Vec{maths.V(0, 0, 0), maths.V(4, 0, 0),
		maths.V(0, 3, 0)}
	for turn := range 3 {
		for spin := range 3 {
			at := func(point maths.Vec) maths.Vec {
				p := parts(point)
				return maths.V(p[spin%3], p[(spin+1)%3], p[(spin+2)%3])
			}
			var turned []maths.Vec
			for i := range corners {
				turned = append(turned, at(corners[(i+turn)%3]))
			}
			made := mustMesh(t, turned, [][3]int{{0, 1, 2}}, nil, false)
			for _, each := range closestOnTriangle {
				want := weights{}
				for i := range want {
					want[(i+3-turn)%3] = each.weights[i]
				}
				point := at(each.point)
				got := made.closest(0, point)
				if !sameWeights(got, want) {
					t.Errorf("turn %d spin %d: over %v the nearest is "+
						"%v, not %v", turn, spin, each.point, got, want)
				}
				_, _, squared := made.nearest(point)
				if math.Abs(squared-each.squared) > 1e-12 {
					t.Errorf("turn %d spin %d: over %v it is %v away, "+
						"squared, not %v", turn, spin, each.point,
						squared, each.squared)
				}
			}
		}
	}
}

// sameWeights is whether two sets of weights are one to a part in a
// million million.
func sameWeights(one, other weights) bool {
	for i := range one {
		if math.Abs(one[i]-other[i]) > 1e-12 {
			return false
		}
	}
	return true
}

// the point on a face its weights give is the weighted sum of its
// corners: halfway along the side from the second corner to the third,
// and a third of the way round from the first.
func TestThePointAtAFacesWeightsIsTheirSumOfCorners(t *testing.T) {
	made := mustMesh(t, []maths.Vec{maths.V(1, 2, 3), maths.V(5, 2, 3),
		maths.V(1, 8, 9)}, [][3]int{{0, 1, 2}}, nil, false)
	if got := made.pointAt(0, weights{0, 0.5, 0.5}); got !=
		maths.V(3, 5, 6) {
		t.Errorf("halfway along the far side it is %v", got)
	}
	if got := made.pointAt(0, weights{0.5, 0.25, 0.25}); got !=
		maths.V(2, 3.5, 4.5) {
		t.Errorf("weighted it is %v", got)
	}
}

// flat is a triangle in the plane z=0, its corners' x and y.
type flat [3][2]float64

// over is whether the point (x, y) is over the triangle, on its edge
// or inside: it is on the same side of each side, or on it, by the
// signs of the sides' turns.
func (corners flat) over(x, y float64) bool {
	var below, above bool
	for i := range corners {
		from, to := corners[i], corners[(i+1)%3]
		turn := (to[0]-from[0])*(y-from[1]) - (to[1]-from[1])*(x-from[0])
		below, above = below || turn < 0, above || turn > 0
	}
	return !(below && above)
}

// nearest is the point of the triangle nearest (x, y): the point itself
// when it is over the triangle, or else the nearest of the three sides'
// points, each side's the point projected on it and held to the side.
func (corners flat) nearest(x, y float64) (float64, float64) {
	if corners.over(x, y) {
		return x, y
	}
	nearX, nearY, least := 0.0, 0.0, math.Inf(1)
	for i := range corners {
		from, to := corners[i], corners[(i+1)%3]
		side := [2]float64{to[0] - from[0], to[1] - from[1]}
		share := ((x-from[0])*side[0] + (y-from[1])*side[1]) /
			(side[0]*side[0] + side[1]*side[1])
		share = math.Min(1, math.Max(0, share))
		atX, atY := from[0]+share*side[0], from[1]+share*side[1]
		if gap := math.Hypot(x-atX, y-atY); gap < least {
			nearX, nearY, least = atX, atY, gap
		}
	}
	return nearX, nearY
}

// whatever the triangle, right, acute, obtuse at either of two corners or a
// sliver, whichever corner is first, and whichever way the axes run, a point's
// nearest on a face is the nearest the plane's own geometry gives, for every
// half a unit over and round it, and at three heights above and below; and the
// same at an eighth the size, where the dots of the corners' sides are all
// under one.
func TestTheNearestPointOfAFaceIsTheNearestInItsPlane(t *testing.T) {
	for name, corners := range map[string]flat{
		"right":         {{0, 0}, {4, 0}, {0, 3}},
		"acute":         {{0, 0}, {5, 0}, {2, 3}},
		"obtuse at c":   {{0, 0}, {6, 0}, {3, 1}},
		"obtuse at a":   {{0, 0}, {4, 0}, {-2, 2}},
		"sliver":        {{0, 0}, {8, 0}, {7, 1}},
		"turned inward": {{1, 1}, {1, 5}, {-3, 2}},
	} {
		for turn := range 3 {
			for spin := range 3 {
				checkFlat(t, name, corners, turn, spin, 1)
				checkFlat(t, name, corners, turn, spin, 0.125)
			}
		}
	}
}

// checkFlat holds one triangle, turned and spun, to the plane's nearest
// at each point of the grid.
func checkFlat(t *testing.T, name string, corners flat, turn, spin int,
	scale float64) {
	at := func(x, y, z float64) maths.Vec {
		p := [3]float64{x, y, z}
		return maths.V(p[spin%3], p[(spin+1)%3], p[(spin+2)%3])
	}
	var turned []maths.Vec
	small := corners
	for i := range corners {
		c := corners[(i+turn)%3]
		small[i] = [2]float64{c[0] * scale, c[1] * scale}
		turned = append(turned, at(c[0]*scale, c[1]*scale, 0))
	}
	made := mustMesh(t, turned, [][3]int{{0, 1, 2}}, nil, false)
	for x := -6.0 * scale; x <= 10*scale; x += 0.5 * scale {
		for y := -6.0 * scale; y <= 10*scale; y += 0.5 * scale {
			wantX, wantY := small.nearest(x, y)
			for _, z := range []float64{0, scale, -2.5 * scale} {
				point := at(x, y, z)
				on := made.closest(0, point)
				got := made.pointAt(0, on)
				want := at(wantX, wantY, 0)
				if got.Sub(want).Len() > 1e-12*scale ||
					math.Abs(on[0]+on[1]+on[2]-1) > 1e-12 {
					t.Fatalf("%s turn %d spin %d scale %v: over %v the "+
						"nearest is %v at weights %v, not %v", name, turn,
						spin, scale, point, got, on, want)
				}
			}
		}
	}
}
