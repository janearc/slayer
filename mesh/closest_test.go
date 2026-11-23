package mesh

import (
	"fmt"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// this file's name sorts before any that builds a mesh. game mutants
// runs the suite failfast, in file order, so a change that makes a
// cut, a root or a walk never end is killed here, by name, before a
// mesh's build hangs. renamed to sort later, those changes time out.

// The csv files of this file's fixtures, each row named in its first
// column.
const (
	closestRegionsFile    = "testdata/closest-regions.csv"
	closestCornersFile    = "testdata/closest-corners.csv"
	closestSumCornersFile = "testdata/closest-sum-corners.csv"
	closestSumsFile       = "testdata/closest-sums.csv"
	closestFlatsFile      = "testdata/closest-flats.csv"
)

// The names of the numbers this file picks, in numbers-closest.yaml.
const (
	closestAgreement = "closest-agreement"
	closestGridStart = "closest-grid-start"
	closestGridEnd   = "closest-grid-end"
	closestGridStep  = "closest-grid-step"
	closestGridBelow = "closest-grid-below"
	closestSmall     = "closest-small-scale"
)

// closestSpin is how many places the parts of a point are turned round:
// zero leaves x, y and z, one makes them y, z and x.
type closestSpin int

// of is a point made of the parts, turned round.
func (spin closestSpin) of(place [axes]float64) maths.Vec {
	return maths.V(place[int(spin)%axes], place[(int(spin)+unity)%axes],
		place[(int(spin)+pair)%axes])
}

// point is the point with its parts turned round.
func (spin closestSpin) point(point maths.Vec) maths.Vec {
	return spin.of(parts(point))
}

// xyz is the point of the three parts, turned round.
func (spin closestSpin) xyz(x, y, z float64) maths.Vec {
	return spin.of(partsOf(x, y, z))
}

// closestRowPoint is the point in the x, y and z columns of a row.
func closestRowPoint(test testing.TB, row map[string]string) maths.Vec {
	return maths.V(float(test, row, "x"), float(test, row, "y"),
		float(test, row, "z"))
}

// closestRowWeights is the weights in the first, second and third
// columns of a row.
func closestRowWeights(test testing.TB, row map[string]string) weights {
	return weights{float(test, row, "first"), float(test, row, "second"),
		float(test, row, "third")}
}

// the right triangle with corners (0,0,0), (4,0,0) and (0,3,0), whose
// legs are 4 and 3 and hypotenuse 5, is seen from points over each of
// its seven regions and on the lines between: the point nearest on it,
// as its corners' weights, and how far that is, squared, each worked
// out by hand. the line 3x+4y=12 is the hypotenuse. each point's
// nearest on the triangle is where the hand says, whichever corner is
// called first and whichever way the axes run: the corners turned round
// three ways and the parts of every point turned round three, which
// changes which line of the code each point reaches.
func TestTheNearestPointOfAFaceIsInTheRegionItIsOver(t *testing.T) {
	corners := vectorsIn(t, closestCornersFile)
	for turn := range triangleCorners {
		for spin := range axes {
			var turned []maths.Vec
			for place := range corners {
				corner := corners[(place+turn)%triangleCorners]
				turned = append(turned, closestSpin(spin).point(corner))
			}
			made := mustMesh(t, turned, theOneFace(), nil, false)
			for _, row := range read(t, closestRegionsFile) {
				checkRegion(t, made, row, turn, spin)
			}
		}
	}
}

// checkRegion holds one row's point to the weights and the squared
// distance the row gives, on the mesh whose corners are turned.
func checkRegion(t *testing.T, made *Mesh, row map[string]string, turn,
	spin int) {
	given := closestRowWeights(t, row)
	var want weights
	for place := range want {
		want[(place+triangleCorners-turn)%triangleCorners] = given[place]
	}
	over := closestRowPoint(t, row)
	point := closestSpin(spin).point(over)
	got := made.closest(nought, point)
	if !sameWeights(t, got, want) {
		t.Errorf("turn %d spin %d: over %v the nearest is %v, not %v",
			turn, spin, over, got, want)
	}
	_, _, squared := made.nearest(point)
	if !(math.Abs(squared-float(t, row, "squared")) <=
		picked(t, closestAgreement)) {
		t.Errorf("turn %d spin %d: over %v it is %v away, squared, "+
			"not %v", turn, spin, over, squared, float(t, row, "squared"))
	}
}

// sameWeights is whether two sets of weights are one to a part in a
// million million.
func sameWeights(test testing.TB, one, other weights) bool {
	agreement := picked(test, closestAgreement)
	for place := range one {
		if !(math.Abs(one[place]-other[place]) <= agreement) {
			return false
		}
	}
	return true
}

// the point on a face its weights give is the weighted sum of its
// corners: halfway along the side from the second corner to the third,
// and a third of the way round from the first.
func TestThePointAtAFacesWeightsIsTheirSumOfCorners(t *testing.T) {
	corners := vectorsIn(t, closestSumCornersFile)
	made := mustMesh(t, corners, theOneFace(), nil, false)
	for _, row := range read(t, closestSumsFile) {
		got := made.pointAt(nought, closestRowWeights(t, row))
		if want := closestRowPoint(t, row); got != want {
			t.Errorf("%s it is %v, not %v", row["name"], got, want)
		}
	}
}

// flat is a triangle in the plane z=0, its corners' x and y.
type flat [triangleCorners][pair]float64

// flatRow is the triangle in the columns x0, y0, x1, y1, x2 and y2 of a
// row.
func flatRow(test testing.TB, row map[string]string) flat {
	var corners flat
	for place := range corners {
		corners[place] = [pair]float64{
			float(test, row, fmt.Sprintf("x%d", place)),
			float(test, row, fmt.Sprintf("y%d", place))}
	}
	return corners
}

// flatSide is the way from one corner of a triangle to another.
func flatSide(from, to [pair]float64) [pair]float64 {
	return [pair]float64{to[alongX] - from[alongX],
		to[alongY] - from[alongY]}
}

// over is whether the point (x, y) is over the triangle, on its edge
// or inside: it is on the same side of each side, or on it, by the
// signs of the sides' turns.
func (corners flat) over(x, y float64) bool {
	var below, above bool
	for i := range corners {
		from, to := corners[i], corners[(i+unity)%triangleCorners]
		turn := (to[alongX]-from[alongX])*(y-from[alongY]) -
			(to[alongY]-from[alongY])*(x-from[alongX])
		below, above = below || turn < nought, above || turn > nought
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
	var nearX, nearY float64
	least := math.Inf(plusInfinity)
	for i := range corners {
		from, to := corners[i], corners[(i+unity)%triangleCorners]
		side := flatSide(from, to)
		share := ((x-from[alongX])*side[alongX] +
			(y-from[alongY])*side[alongY]) /
			(side[alongX]*side[alongX] + side[alongY]*side[alongY])
		share = math.Min(unity, math.Max(nought, share))
		atX, atY := from[alongX]+share*side[alongX],
			from[alongY]+share*side[alongY]
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
	small := picked(t, closestSmall)
	for _, row := range read(t, closestFlatsFile) {
		corners := flatRow(t, row)
		for turn := range triangleCorners {
			for spin := range axes {
				checkFlat(t, row["name"], corners, turn, spin, unity)
				checkFlat(t, row["name"], corners, turn, spin, small)
			}
		}
	}
}

// flatGrid is one triangle, turned and spun, on a mesh, to be held to
// the plane's nearest at each point of a grid.
type flatGrid struct {
	test      *testing.T
	name      string
	made      *Mesh
	small     flat
	spin      closestSpin
	turn      int
	scale     float64
	agreement float64
	heights   []float64
}

// checkFlat holds one triangle, turned and spun, to the plane's nearest
// at each point of the grid.
func checkFlat(t *testing.T, name string, corners flat, turn, spin int,
	scale float64) {
	grid := flatGrid{test: t, name: name, small: corners,
		spin: closestSpin(spin), turn: turn, scale: scale,
		agreement: picked(t, closestAgreement)}
	grid.heights = []float64{nought, scale,
		picked(t, closestGridBelow) * scale}
	var turned []maths.Vec
	for i := range corners {
		corner := corners[(i+turn)%triangleCorners]
		grid.small[i] = [pair]float64{corner[alongX] * scale,
			corner[alongY] * scale}
		turned = append(turned, grid.spin.xyz(corner[alongX]*scale,
			corner[alongY]*scale, nought))
	}
	grid.made = mustMesh(t, turned, theOneFace(), nil, false)
	grid.run()
}

// run holds the mesh to the plane's nearest at each point of the grid.
func (grid flatGrid) run() {
	start := picked(grid.test, closestGridStart) * grid.scale
	end := picked(grid.test, closestGridEnd) * grid.scale
	step := picked(grid.test, closestGridStep) * grid.scale
	for x := start; x <= end; x += step {
		for y := start; y <= end; y += step {
			wantX, wantY := grid.small.nearest(x, y)
			for _, z := range grid.heights {
				grid.check(grid.spin.xyz(x, y, z),
					grid.spin.xyz(wantX, wantY, nought))
			}
		}
	}
}

// check holds the mesh's nearest point to a point to the one wanted,
// and its weights to a sum of one.
func (grid flatGrid) check(point, want maths.Vec) {
	on := grid.made.closest(nought, point)
	got := grid.made.pointAt(nought, on)
	if got.Sub(want).Len() > grid.agreement*grid.scale ||
		!(math.Abs(on[first]+on[second]+on[third]-unity) <=
			grid.agreement) {
		grid.test.Fatalf("%s turn %d spin %d scale %v: over %v the "+
			"nearest is %v at weights %v, not %v", grid.name, grid.turn,
			grid.spin, grid.scale, point, got, on, want)
	}
}
