package mesh

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The csv files of this file's fixtures, each row named in its first
// column.
const (
	boxSpansFile    = "testdata/box-spans.csv"
	boxPointsFile   = "testdata/box-points.csv"
	boxFlatFile     = "testdata/box-flat-areas.csv"
	boxBinsFile     = "testdata/box-bins.csv"
	boxNearerFile   = "testdata/box-nearer.csv"
	boxRaysFile     = "testdata/box-rays.csv"
	boxDescentsFile = "testdata/box-descents.csv"
	boxEntersFile   = "testdata/box-enters.csv"
	boxSightsFile   = "testdata/box-sights.csv"
)

// boxSlackName is the box whose low end on x is the slack past a far side.
const boxSlackName = "slack-edge"

// boxLoosenedWord is what the enter column of a row says when the way in
// is as far as the slack allows.
const boxLoosenedWord = "loosened"

// The places of a stack, from its bottom.
const (
	boxStackBottom = nought // defined: the first place of a stack
	boxStackSecond = unity  // defined: the place above the bottom
	boxStackThird  = pair   // defined: the place above the second
)

const (
	// boxHalf is half of one.
	boxHalf = 0.5 // defined: half is one over two
	// boxLastPlace is the step from one to the next float64 above it.
	boxLastPlace = 0x1p-52 // defined by float64: its fraction is 52 bits
)

// boxRowNamed is the row of a csv file with the given name. A name that
// no row has stops the test.
func boxRowNamed(test testing.TB, file, name string) map[string]string {
	test.Helper()
	for _, row := range read(test, file) {
		if row["name"] == name {
			return row
		}
	}
	test.Fatalf("%s: no row named %q", file, name)
	return nil
}

// boxPartsIn is the three parts of a point, read from the columns of a
// row that are named for the prefix and then x, y and z.
func boxPartsIn(test testing.TB, row map[string]string,
	prefix string) [axes]float64 {
	test.Helper()
	along := float(test, row, prefix+"x")
	up := float(test, row, prefix+"y")
	deep := float(test, row, prefix+"z")
	return partsOf(along, up, deep)
}

// boxVectorOf is a point's three parts as a vector.
func boxVectorOf(point [axes]float64) maths.Vec {
	return maths.V(point[alongX], point[alongY], point[alongZ])
}

// boxPointNamed is a point of the points file, by name.
func boxPointNamed(test testing.TB, name string) [axes]float64 {
	test.Helper()
	return boxPartsIn(test, boxRowNamed(test, boxPointsFile, name), "")
}

// boxSpanNamed is a box of the spans file, by name.
func boxSpanNamed(test testing.TB, name string) span {
	test.Helper()
	row := boxRowNamed(test, boxSpansFile, name)
	return span{
		low:  boxPartsIn(test, row, "low"),
		high: boxPartsIn(test, row, "high"),
	}
}

// boxNodeNamed is a node whose bounds are those of a box of the spans
// file, by name.
func boxNodeNamed(test testing.TB, name string) node {
	test.Helper()
	box := boxSpanNamed(test, name)
	return node{bounds: [boundEnds][axes]float64{low: box.low, high: box.high}}
}

// boxNodeAt is the node of a box of the spans file, by name, to be
// entered. The box named for the slack has the slack for its low end
// on x.
func boxNodeAt(test testing.TB, name string) *node {
	test.Helper()
	at := boxNodeNamed(test, name)
	if name == boxSlackName {
		at.bounds[low][alongX] = loosened
	}
	return &at
}

// boxRayIn is the ray of a row, from its at and dir columns.
func boxRayIn(test testing.TB, row map[string]string) maths.Ray {
	test.Helper()
	at := boxPartsIn(test, row, "at")
	dir := boxPartsIn(test, row, "dir")
	return maths.NewRay(boxVectorOf(at), boxVectorOf(dir))
}

// boxSightIn is the sight a row says a ray is made ready as.
func boxSightIn(test testing.TB, row map[string]string) sight {
	test.Helper()
	return sight{
		from:    boxPartsIn(test, row, "at"),
		inverse: boxPartsIn(test, row, "inverse"),
		entry: [axes]int{whole(test, row, "entryx"),
			whole(test, row, "entryy"), whole(test, row, "entryz")},
		kx:     whole(test, row, "kx"),
		ky:     whole(test, row, "ky"),
		kz:     whole(test, row, "kz"),
		shearX: float(test, row, "shearx"),
		shearY: float(test, row, "sheary"),
		shearZ: float(test, row, "shearz"),
	}
}

// boxEnterIn is the distance a row says a sight enters a box at.
func boxEnterIn(test testing.TB, row map[string]string) float64 {
	test.Helper()
	if row["enter"] == boxLoosenedWord {
		return loosened
	}
	return float(test, row, "enter")
}

// boxTopNodes is the nodes a row says are on top of a stack, from the
// top down, as whole numbers with spaces between.
func boxTopNodes(test testing.TB, text string) []int {
	test.Helper()
	var nodes []int
	for _, word := range strings.Fields(text) {
		number, err := strconv.Atoi(word)
		if err != nil {
			test.Fatal(err)
		}
		nodes = append(nodes, number)
	}
	return nodes
}

// this file's name sorts before any that builds a mesh. game mutants
// runs the suite failfast, in file order, so a change that makes a
// cut, a root or a walk never end is killed here, by name, before a
// mesh's build hangs. renamed to sort later, those changes time out.

// an empty box is endlessly inside out, so a point joined to it is the
// box of that point alone, and a box joined to it is itself.
func TestAnEmptySpanTakesWhatIsJoinedToIt(t *testing.T) {
	empty := emptySpan()
	for axis := range axes {
		if !math.IsInf(empty.low[axis], plusInfinity) ||
			!math.IsInf(empty.high[axis], -plusInfinity) {
			t.Errorf("axis %d runs %v to %v", axis, empty.low[axis],
				empty.high[axis])
		}
	}
	point := boxPointNamed(t, "join-point")
	if got := empty.with(point); got.low != point || got.high != point {
		t.Errorf("a point alone is %v", got)
	}
	box := boxSpanNamed(t, "join-box")
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
	one := boxSpanNamed(t, "join-one")
	other := boxSpanNamed(t, "join-other")
	want := boxSpanNamed(t, "join-want")
	if got := one.joined(other); got != want {
		t.Errorf("joined they are %v, not %v", got, want)
	}
	grown := boxSpanNamed(t, "join-grown-want")
	if got := one.with(boxPointNamed(t, "grown-point")); got != grown {
		t.Errorf("grown to a point it is %v", got)
	}
}

// the middle is halfway between low and high on each axis.
func TestASpansMiddleIsHalfwayOnEachAxis(t *testing.T) {
	box := boxSpanNamed(t, "middle-box")
	if got := box.middle(); got != boxPointNamed(t, "middle-want") {
		t.Errorf("the middle is %v", got)
	}
}

// the area is half the surface: wide times tall, tall times deep, deep
// times wide, for a box two wide, five tall and nine deep.
func TestASpansAreaIsHalfItsSurface(t *testing.T) {
	box := boxSpanNamed(t, "area-box")
	wide := picked(t, "box-area-wide")
	tall := picked(t, "box-area-tall")
	deep := picked(t, "box-area-deep")
	want := wide*tall + tall*deep + deep*wide
	if got := box.area(); got != want {
		t.Errorf("the area is %v, not %v", got, want)
	}
}

// boxCheckFlatArea checks that a box flat on one axis has the area of
// its face.
func boxCheckFlatArea(test testing.TB, axis int) {
	test.Helper()
	flat := boxSpanNamed(test, "flat-base")
	flat.high[axis] = unity
	want := float(test, read(test, boxFlatFile)[axis], "area")
	if got := flat.area(); got != want {
		test.Errorf("flat on %d: %v, not %v", axis, got, want)
	}
}

// boxCheckThinArea checks that a box half a unit thin on one axis has
// an area.
func boxCheckThinArea(test testing.TB, axis int) {
	test.Helper()
	thin := boxSpanNamed(test, "origin-base")
	thin.high[axis] = boxHalf
	if got := thin.area(); got == nought {
		test.Errorf("half a unit thin on %d has no area", axis)
	}
}

// boxCheckInsideOutArea checks that a box with its high below its low on
// one axis has no area.
func boxCheckInsideOutArea(test testing.TB, axis int) {
	test.Helper()
	inside := boxSpanNamed(test, "origin-base")
	inside.high[axis] = -unity
	if got := inside.area(); got != nought {
		test.Errorf("inside out on %d: %v", axis, got)
	}
}

// a box flat on one axis has the area of its face, and a box with a
// high below its low on any axis, or none at all, has none.
func TestASpansAreaIsNoughtOnlyWhereItIsInsideOut(t *testing.T) {
	for axis := range axes {
		boxCheckFlatArea(t, axis)
		boxCheckThinArea(t, axis)
		boxCheckInsideOutArea(t, axis)
	}
	if got := emptySpan().area(); got != nought {
		t.Errorf("an empty box has area %v", got)
	}
}

// parts and vectors go to and from three parts, each part in its place.
func TestPartsAndVectorsKeepEachPartInItsPlace(t *testing.T) {
	ascending := boxSpanNamed(t, "ascending")
	if got := vectorOf(ascending.low); got != boxVectorOf(ascending.low) {
		t.Errorf("vectorOf gave %v", got)
	}
	if got := parts(boxVectorOf(ascending.high)); got != ascending.high {
		t.Errorf("parts gave %v", got)
	}
	at := boxNodeNamed(t, "ascending")
	if got := at.box(); got.Min != boxVectorOf(ascending.low) ||
		got.Max != boxVectorOf(ascending.high) {
		t.Errorf("the node's box is %v", got)
	}
}

// a middle falls in one of sixteen bins along a box's width, the first
// where the box has none, and the last at its far side.
func TestAMiddleFallsInABinAlongTheBox(t *testing.T) {
	for _, row := range read(t, boxBinsFile) {
		box := boxSpanNamed(t, row["span"])
		axis := whole(t, row, "axis")
		middle := float(t, row, "middle")
		want := whole(t, row, "bin")
		if got := binOf(middle, box, axis); got != want {
			t.Errorf("%s: %v falls in bin %d, not %d", row["name"],
				middle, got, want)
		}
	}
}

// boxCheckNearerOnTop checks that closer puts the child a point is nearer
// to on top of the stack, and the other below it.
func boxCheckNearerOnTop(test testing.TB, nodes []node,
	row map[string]string) {
	test.Helper()
	point := boxPartsIn(test, row, "")
	top := whole(test, row, "top")
	other := unity - top
	under := pickedCount(test, "box-stack-bottom")
	var stack [deepest + pushed]waiting
	stack[boxStackBottom] = waiting{node: under}
	height := closer(nodes, nought, point, &stack, unity)
	if height != unity+pushed {
		test.Fatalf("the stack is %d high", height)
	}
	shown := stack[:boxStackThird+unity]
	if stack[boxStackBottom].node != under ||
		stack[boxStackThird].node != top ||
		stack[boxStackSecond].node != other {
		test.Errorf("from %v the stack is %v", point, shown)
	}
	if stack[boxStackThird].enter != awayFrom(&nodes[top], point) ||
		stack[boxStackSecond].enter != awayFrom(&nodes[other], point) {
		test.Errorf("from %v the stack's distances are %v", point, shown)
	}
}

// closer puts the nearer child on top of the stack: both in the stack
// for a point nearer the left, nearer the right, and the same from
// both, which puts the left on top.
func TestTheNearerChildIsOnTopOfTheStack(t *testing.T) {
	nodes := []node{boxNodeNamed(t, "unit"), boxNodeNamed(t, "far")}
	for _, row := range read(t, boxNearerFile) {
		boxCheckNearerOnTop(t, nodes, row)
	}
}

// the slack let to a box's far side is three units in the last place:
// one and two gammas of three come to a little over 6.66e-16, which
// rounds to three steps of 2^-52 above one.
func TestTheSlackIsThreeUnitsInTheLastPlace(t *testing.T) {
	steps := picked(t, "box-slack-steps")
	if want := unity + steps*boxLastPlace; loosened != want {
		t.Errorf("the slack is %v, not %v", loosened, want)
	}
}

// aimed makes a ray ready: where it starts, one over each part of its way,
// which bound of a box it crosses first, which axis it runs along most, the
// other two in turn, and the shear that straightens it. a part that is zero has
// an endless inverse, either sign.
func TestARayIsMadeReadyAxisByAxis(t *testing.T) {
	for _, row := range read(t, boxSightsFile) {
		ray := boxRayIn(t, row)
		want := boxSightIn(t, row)
		if got := aimed(ray); got != want {
			t.Errorf("%s: %+v, not %+v", row["name"], got, want)
		}
	}
}

// the sight enters a box where the slabs of its three axes say, and
// does so only if the last it enters is before the first it leaves: a
// box from (1, 2, 3) to (4, 6, 9), met along and against each axis,
// clipped by near and far, missed, and grazed so that the way in is as
// far as the slack allows past the way out.
func TestASightEntersABoxWhereItsSlabsSay(t *testing.T) {
	for _, row := range read(t, boxEntersFile) {
		look := aimed(boxRayIn(t, row))
		at := boxNodeAt(t, row["node"])
		near, far := float(t, row, "near"), float(t, row, "far")
		enter, ok := look.enters(at, near, far)
		wantEnter := boxEnterIn(t, row)
		wantOk := whole(t, row, "ok") == unity
		if ok != wantOk || enter != wantEnter {
			t.Errorf("%s: %v %v, not %v %v", row["name"], enter, ok,
				wantEnter, wantOk)
		}
	}
}

// boxCheckDescent checks one row of the descents: the stack's height
// after descend, the places under it left alone, and the nodes on top.
func boxCheckDescent(test testing.TB, row map[string]string) {
	test.Helper()
	nodes := []node{boxNodeNamed(test, row["left"])}
	nodes = append(nodes, boxNodeNamed(test, row["right"]))
	ray := boxRayIn(test, boxRowNamed(test, boxRaysFile, row["ray"]))
	under := pickedCount(test, "box-stack-bottom")
	above := pickedCount(test, "box-stack-second")
	var stack [deepest + pushed]waiting
	stack[boxStackBottom] = waiting{node: under}
	stack[boxStackSecond] = waiting{node: above}
	look := aimed(ray)
	near, far := float(test, row, "near"), float(test, row, "far")
	top := look.descend(nodes, nought, near, far, &stack, pair)
	want := boxTopNodes(test, row["top"])
	if top != pair+len(want) || stack[boxStackBottom].node != under ||
		stack[boxStackSecond].node != above {
		test.Errorf("%s: the stack is %d high: %v", row["name"], top,
			stack[:top])
		return
	}
	for place, wanted := range want {
		if got := stack[top-unity-place].node; got != wanted {
			test.Errorf("%s: %d from the top is node %d, not %d",
				row["name"], place, got, wanted)
		}
	}
}

// descend puts the children a sight enters on the stack, the nearer on
// top, what was on it left alone: both, in either order, and the left
// first when they are as near; one; neither; and not a child it does
// not enter, whichever of the two is nearer. a ray along x from
// (-5, 0.5, 0.5) meets the unit box at 5 and the box at 3 to 4 at 8.
func TestTheNearerChildTheSightEntersIsOnTop(t *testing.T) {
	for _, row := range read(t, boxDescentsFile) {
		boxCheckDescent(t, row)
	}
}
