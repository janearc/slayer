package shape

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// The files of rows the placement tests read.
const (
	aGroupPlacementContractsPointsFile = "testdata/" +
		"a-group-placement-contracts-points.csv"
	aGroupPlacementContractsLengthsFile = "testdata/" +
		"a-group-placement-contracts-axis-lengths.csv"
	aGroupPlacementContractsPairsFile = "testdata/" +
		"a-group-placement-contracts-axis-pairs.csv"
	aGroupPlacementContractsDistancesFile = "testdata/" +
		"a-group-placement-contracts-face-distances.csv"
)

// The counts and the bits of a corner's number that the tests use.
const (
	aGroupPlacementContractsAxes    = 3 // defined: x, y and z
	aGroupPlacementContractsCorners = 8 // defined: the corners of a box
	aGroupPlacementContractsXBit    = 1 // defined: a corner's far side in x
	aGroupPlacementContractsYBit    = 2 // defined: a corner's far side in y
	aGroupPlacementContractsZBit    = 4 // defined: a corner's far side in z
)

// aGroupPlacementContractsAt is the frame at origin, not turned or scaled.
func aGroupPlacementContractsAt(origin maths.Vec) maths.Subspace {
	space := maths.Local()
	space.Origin = origin
	return space
}

// aGroupPlacementContractsScaled is the frame at the middle, scaled.
func aGroupPlacementContractsScaled(scale float64) maths.Subspace {
	space := maths.Local()
	space.Scale = scale
	return space
}

// aGroupPlacementContractsTurned is the frame at the middle, turned.
func aGroupPlacementContractsTurned(turn maths.Orientation) maths.Subspace {
	space := maths.Local()
	space.Orientation = turn
	return space
}

// aGroupPlacementContractsBoxOf is the box from low to high.
func aGroupPlacementContractsBoxOf(low, high maths.Vec) maths.Box {
	return maths.Box{Min: low, Max: high}
}

// aGroupPlacementContractsVecAxis is the part of a vector on an axis, as
// a place to set.
func aGroupPlacementContractsVecAxis(vec *maths.Vec, axis int) *float64 {
	switch axis {
	case nought:
		return &vec.X
	case unity:
		return &vec.Y
	default:
		return &vec.Z
	}
}

// aGroupPlacementContractsTurnAxis is an axis of a turn, as a place to
// set.
func aGroupPlacementContractsTurnAxis(turn *maths.Orientation,
	axis int) *maths.Vec {
	switch axis {
	case nought:
		return &turn.X
	case unity:
		return &turn.Y
	default:
		return &turn.Z
	}
}

// aGroupPlacementContractsPoint is the point in the row called name of
// the points file.
func aGroupPlacementContractsPoint(t *testing.T, name string) maths.Vec {
	t.Helper()
	row := triangleRow(t, aGroupPlacementContractsPointsFile, name)
	return trianglePoint(t, row)
}

// aGroupPlacementContractsValid says that the placements that are
// frames of the world are taken.
func aGroupPlacementContractsValid(t *testing.T) {
	t.Helper()
	axis := maths.V(unity, pair,
		picked(t, "a-group-placement-contracts-rotation-axis-z"))
	angle := picked(t, "a-group-placement-contracts-rotation-angle")
	turn := maths.Rotation(axis, angle)
	valid := []maths.Subspace{maths.Local(),
		aGroupPlacementContractsScaled(math.SmallestNonzeroFloat64),
		aGroupPlacementContractsScaled(pair),
		aGroupPlacementContractsTurned(turn)}
	for _, space := range valid {
		if !ValidPlacement(space) {
			t.Fatalf("valid placement refused: %+v", space)
		}
	}
}

// aGroupPlacementContractsNonfinite says that a nonfinite part of the
// origin, on any axis, or of the scale, is refused, and so is a negative
// scale.
func aGroupPlacementContractsNonfinite(t *testing.T) {
	t.Helper()
	for _, bad := range []float64{math.NaN(), math.Inf(unity),
		math.Inf(-unity)} {
		for axis := range aGroupPlacementContractsAxes {
			var origin maths.Vec
			*aGroupPlacementContractsVecAxis(&origin, axis) = bad
			space := aGroupPlacementContractsAt(origin)
			if ValidPlacement(space) {
				t.Fatalf("accepted nonfinite origin axis %d", axis)
			}
		}
		if ValidPlacement(aGroupPlacementContractsScaled(bad)) {
			t.Fatal("accepted nonfinite scale")
		}
	}
	negative := aGroupPlacementContractsScaled(
		-math.SmallestNonzeroFloat64)
	if ValidPlacement(negative) {
		t.Fatal("accepted negative scale")
	}
}

// aGroupPlacementContractsLengths says that an axis of a turn that is
// not one long is refused, on any axis.
func aGroupPlacementContractsLengths(t *testing.T) {
	t.Helper()
	lengths := readSheet(t, aGroupPlacementContractsLengthsFile)
	for axis := range aGroupPlacementContractsAxes {
		for _, row := range lengths {
			size := cell(t, row, "length")
			turn := maths.Identity
			stretched := aGroupPlacementContractsTurnAxis(&turn, axis)
			*stretched = stretched.Scale(size)
			if ValidPlacement(aGroupPlacementContractsTurned(turn)) {
				t.Fatalf("accepted axis %d length %g", axis, size)
			}
		}
	}
}

// aGroupPlacementContractsSkews says that axes that are one long and
// skewed from square by the tolerance, or more, are refused, for each
// pair of axes, and that less is taken.
func aGroupPlacementContractsSkews(t *testing.T) {
	t.Helper()
	tolerance := picked(t, "a-group-placement-contracts-skew-tolerance")
	twice := picked(t, "a-group-placement-contracts-skew-twice")
	large := picked(t, "a-group-placement-contracts-skew-large")
	pairs := readSheet(t, aGroupPlacementContractsPairsFile)
	// Unit-length skew axes isolate orthogonality from length validation.
	for _, skew := range []float64{tolerance,
		math.Nextafter(tolerance, nought), twice, large} {
		for _, row := range pairs {
			turn := maths.Identity
			first := cellCount(t, row, "first")
			second := cellCount(t, row, "second")
			held := aGroupPlacementContractsTurnAxis(&turn, first)
			moved := aGroupPlacementContractsTurnAxis(&turn, second)
			*moved = moved.Scale(math.Sqrt(unity - skew*skew)).
				Add(held.Scale(skew))
			got := ValidPlacement(aGroupPlacementContractsTurned(turn))
			if got != (skew < tolerance) {
				t.Fatalf("axis pair %v skew %.17g: accepted %t",
					row["name"], skew, got)
			}
		}
	}
}

// TestGroupPlacementChecksEachIndependentAxis says that a placement is
// taken or refused for each part of it alone: its origin, its scale and
// each axis of its turn.
func TestGroupPlacementChecksEachIndependentAxis(t *testing.T) {
	aGroupPlacementContractsValid(t)
	aGroupPlacementContractsNonfinite(t)
	aGroupPlacementContractsLengths(t)
	aGroupPlacementContractsSkews(t)
	turn := maths.Identity
	turn.Z = turn.Z.Neg()
	if ValidPlacement(aGroupPlacementContractsTurned(turn)) {
		t.Fatal("accepted left-handed placement")
	}
}

// TestGroupBoxDistanceIsTheEuclideanDistanceToTheBox says that a point in
// the box is nought from it, and that a point outside is as far as the
// straight line to the nearest point of the box.
func TestGroupBoxDistanceIsTheEuclideanDistanceToTheBox(t *testing.T) {
	low := aGroupPlacementContractsPoint(t, "box-low")
	high := aGroupPlacementContractsPoint(t, "box-high")
	box := aGroupPlacementContractsBoxOf(low, high)
	origin := maths.V(nought, nought, nought)
	for _, point := range []maths.Vec{origin, box.Min, box.Max} {
		if distance := groupBoxDistance(box, point); distance != nought {
			t.Fatalf("point %v in box has distance %g", point, distance)
		}
	}
	far := aGroupPlacementContractsPoint(t, "far-corner")
	near := aGroupPlacementContractsPoint(t, "near-corner")
	want := picked(t, "a-group-placement-contracts-corner-distance")
	for mask := range aGroupPlacementContractsCorners {
		point := far
		if mask&aGroupPlacementContractsXBit != nought {
			point.X = near.X
		}
		if mask&aGroupPlacementContractsYBit != nought {
			point.Y = near.Y
		}
		if mask&aGroupPlacementContractsZBit != nought {
			point.Z = near.Z
		}
		// Orthogonal offsets of 3, 4 and 12 give distance 13.
		if distance := groupBoxDistance(box, point); distance != want {
			t.Fatalf("corner %d distance %g, want %g", mask, distance, want)
		}
	}
	for _, row := range readSheet(t, aGroupPlacementContractsDistancesFile) {
		point := maths.V(cell(t, row, "x"), cell(t, row, "y"),
			cell(t, row, "z"))
		if distance := groupBoxDistance(box, point); distance !=
			cell(t, row, "distance") {
			t.Fatalf("point %v distance %g, want %g", point, distance,
				cell(t, row, "distance"))
		}
	}
}

// aGroupPlacementContractsRow is a group of a triangle placed again and
// again along x, a hierarchy deep enough to walk for a while.
func aGroupPlacementContractsRow(t *testing.T) *MeshGroup {
	t.Helper()
	points := []maths.Vec{maths.V(nought, nought, nought),
		maths.V(unity, nought, nought), maths.V(nought, unity, nought)}
	faces := triangleFaces(t, triangleFacesFile, "lone")
	solid, err := mesh.New(points, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	spacing := picked(t, "a-group-placement-contracts-spacing")
	var members []MeshMember
	for i := range pickedCount(t, "a-group-placement-contracts-members") {
		origin := maths.V(float64(i)*spacing, nought, nought)
		members = append(members, MeshMember{Mesh: solid,
			Space: aGroupPlacementContractsAt(origin)})
	}
	group, err := NewMeshGroup(members)
	if err != nil {
		t.Fatal(err)
	}
	return group
}

// aGroupPlacementContractsHit puts the ray to the group, and says on done
// whether it met a face a metre along.
func aGroupPlacementContractsHit(group *MeshGroup, ray maths.Ray,
	far, tolerance float64, done chan<- bool) {
	distance, hit := group.Hit(ray, nought, far)
	done <- hit && math.Abs(distance-unity) <= tolerance
}

// aGroupPlacementContractsNearest asks the group how far the point is,
// and says the answer on done.
func aGroupPlacementContractsNearest(group *MeshGroup, point maths.Vec,
	done chan<- float64) {
	done <- group.Distance(point)
}

// TestGroupNearestTraversalFinishesForADeepHierarchy says that a ray's
// hit, and a point's nearest distance, are answered within a second by a
// group of many members, and are the right answers.
func TestGroupNearestTraversalFinishesForADeepHierarchy(t *testing.T) {
	group := aGroupPlacementContractsRow(t)
	offset := picked(t, "a-group-placement-contracts-ray-offset")
	far := picked(t, "a-group-placement-contracts-ray-far")
	tolerance := picked(t, "a-group-placement-contracts-tolerance")
	point := maths.V(offset, offset, unity)
	ray := triangleRay(point, maths.ZAxis.Neg())
	hitDone := make(chan bool, unity)
	go aGroupPlacementContractsHit(group, ray, far, tolerance, hitDone)
	select {
	case ok := <-hitDone:
		if !ok {
			t.Fatal("deep hierarchy lost its nearest ray hit")
		}
	case <-time.After(time.Second):
		t.Fatal("hit traversal did not finish within one second")
	}

	done := make(chan float64, unity)
	go aGroupPlacementContractsNearest(group, point, done)
	select {
	case distance := <-done:
		if !(math.Abs(distance-unity) <= tolerance) {
			t.Fatalf("nearest distance %g, want 1", distance)
		}
	case <-time.After(time.Second):
		t.Fatal("nearest traversal did not finish within one second")
	}
}
