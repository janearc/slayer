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

// an empty box is endlessly inside out, so a point joined to it is the
// box of that point alone, and a box joined to it is itself.
func TestAnEmptySpanTakesWhatIsJoinedToIt(t *testing.T) {
	empty := emptySpan()
	for axis := range 3 {
		if !math.IsInf(empty.low[axis], 1) ||
			!math.IsInf(empty.high[axis], -1) {
			t.Errorf("axis %d runs %v to %v", axis, empty.low[axis],
				empty.high[axis])
		}
	}
	point := partsOf(-3, 0, 4)
	if got := empty.with(point); got.low != point || got.high != point {
		t.Errorf("a point alone is %v", got)
	}
	box := span{low: partsOf(-1, 2, -3), high: partsOf(4, 5, 6)}
	if got := empty.joined(box); got != box {
		t.Errorf("joined to nothing it is %v", got)
	}
	if got := box.joined(empty); got != box {
		t.Errorf("nothing joined to it makes %v", got)
	}
}

// two boxes joined are the box round both: the lows' least and the
// highs' greatest, each axis alone.
func TestJoinedSpansTakeTheirExtremesByAxis(t *testing.T) {
	one := span{low: partsOf(0, 5, -2), high: partsOf(4, 6, 1)}
	other := span{low: partsOf(1, 3, -3), high: partsOf(2, 9, 0)}
	want := span{low: partsOf(0, 3, -3), high: partsOf(4, 9, 1)}
	if got := one.joined(other); got != want {
		t.Errorf("joined they are %v, not %v", got, want)
	}
	if got := one.with(partsOf(5, 4, -7)); got !=
		(span{low: partsOf(0, 4, -7), high: partsOf(5, 6, 1)}) {
		t.Errorf("grown to a point it is %v", got)
	}
}

// the middle is halfway between low and high on each axis.
func TestASpansMiddleIsHalfwayOnEachAxis(t *testing.T) {
	box := span{low: partsOf(-1, 2, -3), high: partsOf(3, 6, 5)}
	if got := box.middle(); got != partsOf(1, 4, 1) {
		t.Errorf("the middle is %v", got)
	}
}

// the area is half the surface: wide times tall, tall times deep, deep
// times wide, for a box two wide, five tall and nine deep.
func TestASpansAreaIsHalfItsSurface(t *testing.T) {
	box := span{low: partsOf(1, 2, 4), high: partsOf(3, 7, 13)}
	if got := box.area(); got != 2*5+5*9+9*2 {
		t.Errorf("the area is %v, not 73", got)
	}
}

// a box flat on one axis has the area of its face, and a box with a
// high below its low on any axis, or none at all, has none.
func TestASpansAreaIsNoughtOnlyWhereItIsInsideOut(t *testing.T) {
	for axis := range 3 {
		flat := span{low: partsOf(1, 1, 1), high: partsOf(3, 4, 6)}
		flat.high[axis] = 1
		want := map[int]float64{0: 3 * 5, 1: 2 * 5, 2: 2 * 3}[axis]
		if got := flat.area(); got != want {
			t.Errorf("flat on %d: %v, not %v", axis, got, want)
		}
		thin := span{low: partsOf(0, 0, 0), high: partsOf(2, 3, 4)}
		thin.high[axis] = 0.5
		if got := thin.area(); got == 0 {
			t.Errorf("half a unit thin on %d has no area", axis)
		}
		inside := span{low: partsOf(0, 0, 0), high: partsOf(2, 3, 4)}
		inside.high[axis] = -1
		if got := inside.area(); got != 0 {
			t.Errorf("inside out on %d: %v", axis, got)
		}
	}
	if got := emptySpan().area(); got != 0 {
		t.Errorf("an empty box has area %v", got)
	}
}

// parts and vectors go to and from three parts, each part in its place.
func TestPartsAndVectorsKeepEachPartInItsPlace(t *testing.T) {
	if got := vectorOf(partsOf(1, 2, 3)); got != maths.V(1, 2, 3) {
		t.Errorf("vectorOf gave %v", got)
	}
	if got := parts(maths.V(4, 5, 6)); got != partsOf(4, 5, 6) {
		t.Errorf("parts gave %v", got)
	}
	at := node{bounds: [2][3]float64{{1, 2, 3}, {4, 5, 6}}}
	if got := at.box(); got.Min != maths.V(1, 2, 3) ||
		got.Max != maths.V(4, 5, 6) {
		t.Errorf("the node's box is %v", got)
	}
}

// a middle falls in one of sixteen bins along a box's width, the first
// where the box has none, and the last at its far side.
func TestAMiddleFallsInABinAlongTheBox(t *testing.T) {
	box := span{low: partsOf(0, 2, 0), high: partsOf(0, 10, 0)}
	for _, each := range []struct {
		middle float64
		want   int
	}{{2, 0}, {2.4, 0}, {3, 2}, {6, 8}, {9, 14}, {9.9, 15}, {10, 15}} {
		if got := binOf(each.middle, box, 1); got != each.want {
			t.Errorf("%v falls in bin %d, not %d", each.middle, got,
				each.want)
		}
	}
	for _, middle := range []float64{0, 5, -5} {
		if got := binOf(middle, box, 0); got != 0 {
			t.Errorf("%v with no width falls in bin %d", middle, got)
		}
	}
	narrow := span{low: partsOf(0, 0, 0), high: partsOf(0, 0.5, 0)}
	if got := binOf(0.4, narrow, 1); got != 12 {
		t.Errorf("in half a unit it falls in bin %d, not 12", got)
	}
}

// closer puts the nearer child on top of the stack: both in the stack
// for a point nearer the left, nearer the right, and the same from
// both, which puts the left on top.
func TestTheNearerChildIsOnTopOfTheStack(t *testing.T) {
	nodes := []node{
		{bounds: [2][3]float64{{0, 0, 0}, {1, 1, 1}}},
		{bounds: [2][3]float64{{3, 0, 0}, {4, 1, 1}}}}
	for _, each := range []struct {
		point [3]float64
		top   int
	}{{[3]float64{-1, 0.5, 0.5}, 0}, {[3]float64{6, 0.5, 0.5}, 1},
		{[3]float64{2, 0.5, 0.5}, 0}} {
		var stack [deepest + 2]waiting
		stack[0] = waiting{node: 7}
		if got := closer(nodes, 0, each.point, &stack, 1); got != 3 {
			t.Fatalf("the stack is %d high", got)
		}
		other := 1 - each.top
		if stack[0].node != 7 || stack[2].node != each.top ||
			stack[1].node != other {
			t.Errorf("from %v the stack is %v", each.point, stack[:3])
		}
		if stack[2].enter != awayFrom(&nodes[each.top], each.point) ||
			stack[1].enter != awayFrom(&nodes[other], each.point) {
			t.Errorf("from %v the stack's distances are %v",
				each.point, stack[:3])
		}
	}
}

// the slack let to a box's far side is three units in the last place:
// one and two gammas of three come to a little over 6.66e-16, which
// rounds to three steps of 2^-52 above one.
func TestTheSlackIsThreeUnitsInTheLastPlace(t *testing.T) {
	if want := 1 + 3*0x1p-52; loosened != want {
		t.Errorf("the slack is %v, not %v", loosened, want)
	}
}

// aimed makes a ray ready: where it starts, one over each part of its way,
// which bound of a box it crosses first, which axis it runs along most, the
// other two in turn, and the shear that straightens it. a part that is zero has
// an endless inverse, either sign.
func TestARayIsMadeReadyAxisByAxis(t *testing.T) {
	inf := math.Inf(1)
	for name, each := range map[string]struct {
		ray  maths.Ray
		want sight
	}{
		"along z": {maths.Ray{At: maths.V(1, 2, 3), Dir: maths.V(0, 0, 1)},
			sight{from: [3]float64{1, 2, 3}, inverse: [3]float64{inf, inf, 1},
				kx: 0, ky: 1, kz: 2, shearZ: 1}},
		"back along z": {maths.Ray{At: maths.V(1, 2, 3),
			Dir: maths.V(0, 0, -1)},
			sight{from: [3]float64{1, 2, 3},
				inverse: [3]float64{inf, inf, -1}, entry: [3]int{0, 0, 1},
				kx: 1, ky: 0, kz: 2, shearZ: -1}},
		"negative zero": {maths.Ray{At: maths.V(0, 0, 0),
			Dir: maths.V(math.Copysign(0, -1), 1, 0)},
			sight{inverse: [3]float64{inf, 1, inf}, kx: 2, ky: 0, kz: 1,
				shearZ: 1}},
		"skew": {maths.Ray{At: maths.V(0, 0, 0), Dir: maths.V(2, -4, 1)},
			sight{inverse: [3]float64{0.5, -0.25, 1}, entry: [3]int{0, 1, 0},
				kx: 0, ky: 2, kz: 1, shearX: -0.5, shearY: -0.25,
				shearZ: -0.25}},
		"slow": {maths.Ray{At: maths.V(0, 0, 0), Dir: maths.V(0, 0, 0.5)},
			sight{inverse: [3]float64{inf, inf, 2}, kx: 0, ky: 1, kz: 2,
				shearZ: 2}},
		"slow back": {maths.Ray{At: maths.V(0, 0, 0),
			Dir: maths.V(0, 0, -0.5)},
			sight{inverse: [3]float64{inf, inf, -2},
				entry: [3]int{0, 0, 1}, kx: 1, ky: 0, kz: 2, shearZ: -2}},
		"long": {maths.Ray{At: maths.V(0, 0, 0), Dir: maths.V(2, 0, 0)},
			sight{inverse: [3]float64{0.5, inf, inf}, kx: 1, ky: 2, kz: 0,
				shearZ: 0.5}},
		"two as long": {maths.Ray{At: maths.V(0, 0, 0),
			Dir: maths.V(1, 1, 0)},
			sight{inverse: [3]float64{1, 1, inf}, kx: 1, ky: 2, kz: 0,
				shearX: 1, shearY: 0, shearZ: 1}},
		"three as long": {maths.Ray{At: maths.V(0, 0, 0),
			Dir: maths.V(0, 2, 2)},
			sight{inverse: [3]float64{inf, 0.5, 0.5}, kx: 2, ky: 0, kz: 1,
				shearX: 1, shearY: 0, shearZ: 0.5}},
	} {
		if got := aimed(each.ray); got != each.want {
			t.Errorf("%s: %+v, not %+v", name, got, each.want)
		}
	}
}

// the sight enters a box where the slabs of its three axes say, and
// does so only if the last it enters is before the first it leaves: a
// box from (1, 2, 3) to (4, 6, 9), met along and against each axis,
// clipped by near and far, missed, and grazed so that the way in is as
// far as the slack allows past the way out.
func TestASightEntersABoxWhereItsSlabsSay(t *testing.T) {
	box := &node{bounds: [2][3]float64{{1, 2, 3}, {4, 6, 9}}}
	slack := &node{bounds: [2][3]float64{{loosened, 0, 0}, {5, 1, 1}}}
	point, inf := maths.V, math.Inf(1)
	for _, each := range []struct {
		name      string
		at        *node
		from, dir maths.Vec
		near, far float64
		enter     float64
		ok        bool
	}{
		{"along x", box, point(0, 3, 4), point(2, 0, 0), 0, inf, 0.5, true},
		{"against x", box, point(10, 3, 4), point(-2, 0, 0), 0, inf, 3,
			true},
		{"along y", box, point(2, 0, 4), point(0, 4, 0), 0, inf, 0.5, true},
		{"against y", box, point(2, 9, 4), point(0, -4, 0), 0, inf, 0.75,
			true},
		{"along z", box, point(2, 3, 0), point(0, 0, 2), 0, inf, 1.5, true},
		{"against z", box, point(2, 3, 12), point(0, 0, -2), 0, inf, 1.5,
			true},
		{"near past", box, point(0, 3, 4), point(1, 0, 0), 2, 3, 2, true},
		{"far before", box, point(0, 3, 4), point(1, 0, 0), 0, 0.5, 1,
			false},
		{"far inside", box, point(0, 3, 4), point(1, 0, 0), 0, 1.5, 1,
			true},
		{"missed", box, point(0, 10, 4), point(1, 0, 0), 0, inf, 1, false},
		{"missed on x", box, point(0, 3, 4), point(0, 1, 0), 0, inf, inf,
			false},
		{"no way", box, point(0, 3, 4), point(0, 0, 0), 0, inf, inf, false},
		{"corner", box, point(0, 0, 0), point(1, 2, 1), 0, inf, 3, true},
		{"diagonal", box, point(0, 0, 0), point(1, 1, 1), 0, inf, 3, true},
		{"just missed", box, point(0, 0, 0), point(2, 1, 1), 0, inf, 3,
			false},
		{"at the slack", slack, point(0, 0, 0.5), point(1, 1, 0), 0, inf,
			loosened, true},
	} {
		look := aimed(maths.Ray{At: each.from, Dir: each.dir})
		enter, ok := look.enters(each.at, each.near, each.far)
		if ok != each.ok || enter != each.enter {
			t.Errorf("%s: %v %v, not %v %v", each.name, enter, ok,
				each.enter, each.ok)
		}
	}
}

// descend puts the children a sight enters on the stack, the nearer on
// top, what was on it left alone: both, in either order, and the left
// first when they are as near; one; neither; and not a child it does
// not enter, whichever of the two is nearer. a ray along x from
// (-5, 0.5, 0.5) meets the unit box at 5 and the box at 3 to 4 at 8.
func TestTheNearerChildTheSightEntersIsOnTop(t *testing.T) {
	unit := [2][3]float64{{0, 0, 0}, {1, 1, 1}}
	far := [2][3]float64{{3, 0, 0}, {4, 1, 1}}
	aside := [2][3]float64{{3, 5, 5}, {4, 6, 6}}
	forward := maths.Ray{At: maths.V(-5, 0.5, 0.5), Dir: maths.V(1, 0, 0)}
	back := maths.Ray{At: maths.V(6, 0.5, 0.5), Dir: maths.V(-1, 0, 0)}
	for _, each := range []struct {
		name        string
		left, right [2][3]float64
		ray         maths.Ray
		near, far   float64
		top         []int
	}{
		{"left first", unit, far, forward, 0, 99, []int{0, 1}},
		{"right first", unit, far, back, 0, 99, []int{1, 0}},
		{"as near", unit, unit, forward, 0, 99, []int{0, 1}},
		{"left only", unit, far, forward, 0, 7, []int{0}},
		{"right only", unit, far, forward, 7, 99, []int{1}},
		{"neither", unit, far, forward, 20, 99, nil},
		{"left, not a nearer right", unit, aside, forward, 0, 99, []int{0}},
		{"right, not a nearer left", aside, unit, forward, 0, 99, []int{1}},
	} {
		nodes := []node{{bounds: each.left}, {bounds: each.right}}
		var stack [deepest + 2]waiting
		stack[0], stack[1] = waiting{node: 7}, waiting{node: 8}
		look := aimed(each.ray)
		top := look.descend(nodes, 0, each.near, each.far, &stack, 2)
		if top != 2+len(each.top) || stack[0].node != 7 ||
			stack[1].node != 8 {
			t.Errorf("%s: the stack is %d high: %v", each.name, top,
				stack[:top])
			continue
		}
		for i, want := range each.top {
			if got := stack[top-1-i].node; got != want {
				t.Errorf("%s: %d from the top is node %d, not %d",
					each.name, i, got, want)
			}
		}
	}
}
