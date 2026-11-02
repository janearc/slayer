package typography

// tests written for what game mutants found the other tests could not
// tell apart: each holds a rule at the edge a changed line would move.

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"golang.org/x/image/font/gofont/goregular"
)

// a curve halved as deep as flattening goes is cut into 2^16 steps and
// no more, whatever the tolerance asks; a curve whose pull is exactly
// the tolerance is one step.
func TestFlatteningStopsAtItsDepthAndAtItsTolerance(t *testing.T) {
	tight := piece{{0, 0}, {500, 800}, {1000, 0}}
	if got := len(flatten(tight, 1e-300, deepest, nil)); got != 1<<16 {
		t.Errorf("at the deepest a curve is %d steps, not %d",
			got, 1<<16)
	}
	if got := len(flatten(piece{{0, 0}, {50, 80}, {100, 0}}, 80, deepest,
		nil)); got != 1 {
		t.Errorf("a curve pulled exactly the tolerance is %d steps",
			got)
	}
}

// apart is how far a point is from a step: from its nearer end when it
// is past either end, and from a step with no length, from its one
// point, never not a number; short steps, under a unit long, alike.
func TestApartIsTheDistanceToAStep(t *testing.T) {
	for _, each := range []struct {
		a, b, p point
		want    float64
	}{{point{0, 0}, point{0.5, 0}, point{0.25, 1}, 1},
		{point{0, 0}, point{1, 0}, point{1.5, 1}, math.Hypot(0.5, 1)},
		{point{0, 0}, point{1, 0}, point{-3, 4}, 5},
		{point{2, 2}, point{2, 2}, point{5, 6}, 5}} {
		got := apart(each.a, each.b, each.p)
		if !(math.Abs(got-each.want) <= 1e-12) {
			t.Errorf("%v from %v to %v: %v, want %v",
				each.p, each.a, each.b,
				got, each.want)
		}
	}
}

// within holds both ends and the middle of a step, and nothing past
// either end, along x and along y.
func TestWithinHoldsAStepsEnds(t *testing.T) {
	a, b := point{0, 0}, point{10, 5}
	for p, want := range map[point]bool{a: true, b: true, {5, 2.5}: true,
		{11, 5.5}: false, {-1, -0.5}: false, {10, 5.5}: false,
		{10.5, 5}: false} {
		if got := within(a, b, p); got != want {
			t.Errorf("%v within %v to %v: %v", p, a, b, got)
		}
	}
}

// side is one on a line's left however near, minus one on its right, and zero
// on it.
func TestSideIsWhichSideOfALine(t *testing.T) {
	a, b := point{0, 0}, point{1, 0}
	for p, want := range map[point]float64{{0.5, 0.25}: 1, {0.5, -0.25}: -1,
		{3, 0}: 0, {0.5, 1e-9}: 1} {
		if got := side(a, b, p); got != want {
			t.Errorf("%v of the line: %v, want %v", p, got, want)
		}
	}
}

// a step ending on the middle of another touches it, each way round;
// two steps end to end on one line touch at the end they share, along
// x and along y; two apart on one line do not.
func TestStepsThatMeetAtAPointTouch(t *testing.T) {
	for name, each := range map[string]struct {
		a, b, c, d point
		want       bool
	}{"t": {point{0, 0}, point{10, 0}, point{5, 0}, point{5, 5}, true},
		"t turned": {point{5, 5}, point{5, 0}, point{0, 0},
			point{10, 0}, true},
		"t at d": {point{0, 0}, point{10, 0}, point{5, 5},
			point{5, 0}, true},
		"t at b": {point{5, 5}, point{5, 0}, point{10, 0}, point{0, 0},
			true},
		"end to end": {point{0, 0}, point{10, 0}, point{10, 0},
			point{20, 0}, true},
		"end to end, up": {point{0, 0}, point{0, 10}, point{0, 10},
			point{0, 20}, true},
		"apart on a line": {point{0, 0}, point{10, 0}, point{11, 0},
			point{20, 0}, false},
	} {
		got := touch(each.a, each.b, each.c, each.d)
		if got != each.want {
			t.Errorf("%s: touch %v", name, got)
		}
	}
}

// a ring whose first point is the one on a line is cleaned of it too.
func TestCleaningAsksTheFirstPoint(t *testing.T) {
	r := ring{{5, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}
	if got := cleaned(r); len(got) != 4 {
		t.Errorf("cleaned to %v", got)
	}
}

// idle holds a point exactly at its cutoff, a part in a million million
// of the steps' lengths, and not one a little past it, for steps of
// lengths other than one.
func TestIdleIsAPartInAMillionMillion(t *testing.T) {
	if !idle(point{0, 0}, point{1, 0}, point{2, 1e-12}) {
		t.Error("a turn of exactly the cutoff is not idle")
	}
	if !idle(point{0, 0}, point{4, 0}, point{6, 1e-12}) {
		t.Error("a turn under the cutoff, steps four and two long, " +
			"is not idle")
	}
	if idle(point{0, 0}, point{4, 0}, point{6, 1e-9}) {
		t.Error("a turn past the cutoff is idle")
	}
}

// square is a ring round a square, counterclockwise.
func square(x, y, side float64) ring {
	return ring{{x, y}, {x + side, y}, {x + side, y + side}, {x, y + side}}
}

// four squares each inside the next are two solids, each with one hole:
// the outer with the second, the third with the fourth; and that holds
// with the third written first, so the fourth's smallest holder is the
// first ring of all.
func TestRingsNestFourDeep(t *testing.T) {
	a, b, c, d := square(0, 0, 40), square(5, 5, 30), square(10, 10, 20),
		square(15, 15, 10)
	solids := nested([]ring{c, a, b, d})
	if len(solids) != 2 {
		t.Fatalf("%d solids", len(solids))
	}
	for _, one := range solids {
		holes := map[float64]float64{1600: -900, 400: -100}
		want := holes[one.outer.area()]
		if len(one.holes) != 1 || one.holes[0].area() != want {
			t.Errorf("the solid of %v has holes %v",
				one.outer.area(),
				one.holes)
		}
	}
}

// a ring of less than a unit's area is turned as its sign says, not
// its size.
func TestATinyRingIsTurnedByItsSign(t *testing.T) {
	tiny := ring{{0, 0}, {1, 0}, {0, 1}}
	if got := turned(tiny, 1); got.area() != 0.5 {
		t.Errorf("a tiny counterclockwise ring turned to %v", got)
	}
}

// meets is where a step reaches a height: between its ends, at an end
// that is there, and the nearer end toward minus x of a level step.
func TestMeetsIsWhereAStepReachesAHeight(t *testing.T) {
	for _, each := range []struct {
		a, b point
		want float64
	}{{point{0, -1}, point{2, 1}, 1}, {point{5, 0}, point{1, 0}, 1},
		{point{3, 0}, point{7, 4}, 3}, {point{7, 4}, point{3, 0}, 3}} {
		if got := meets(each.a, each.b, 0); got != each.want {
			t.Errorf("%v to %v at nought: %v, want %v",
				each.a, each.b, got,
				each.want)
		}
	}
}

// a hole sees the corner of the outer its ray's step starts or ends at,
// whichever is further toward plus x, the step written either way.
func TestAHoleSeesTheFurtherEndOfTheStepItsRayMeets(t *testing.T) {
	up := ring{{10, -10}, {30, 10}, {0, 10}, {0, -10}}
	if got := seen(up, point{12, 0}); got != 1 {
		t.Errorf("a step ending further right: corner %d, not 1", got)
	}
	down := ring{{30, 10}, {0, 10}, {0, -10}, {10, -10}}
	if got := seen(down, point{12, 0}); got != 0 {
		t.Errorf("a step starting further right: corner %d, not 0", got)
	}
}

// notched is a box 30 by 20 with spikes hanging from its top to tips,
// each tip turning inward, the box's corners turning outward.
func notched(tips ...point) ring {
	r := ring{{0, 0}, {30, 0}, {30, 20}}
	for _, tip := range tips {
		r = append(r, point{tip.x + 0.5, 20}, tip,
			point{tip.x - 0.5, 20})
	}
	return append(r, point{0, 20})
}

// of the inward corners in the bridge's triangle, the one nearest the
// ray's way is taken, though another is nearer m; of two on one line
// from m, the nearer, whichever comes first; and with none, the step's
// end.
func TestTheBridgeGoesToTheCornerNearestTheRay(t *testing.T) {
	m, hit := point{2, 10}, point{30, 10}
	r := notched(point{20, 9}, point{15, 7})
	if got := r[inward(r, m, hit, 1)]; got != (point{20, 9}) {
		t.Errorf("the nearer in angle: %v", got)
	}
	for _, r := range []ring{notched(point{22, 6}, point{12, 8}),
		notched(point{12, 8}, point{22, 6})} {
		if got := r[inward(r, m, hit, 1)]; got != (point{12, 8}) {
			t.Errorf("of two on one line: %v", got)
		}
	}
	r = notched()
	if got := inward(r, m, hit, 1); got != 1 {
		t.Errorf("with none in the way: %d", got)
	}
}

// reflex is whether a ring turns clockwise at a point, small turns too:
// not at a corner that turns counterclockwise, nor one that goes
// straight on.
func TestReflexIsAClockwiseTurn(t *testing.T) {
	r := ring{{0, 0}, {1, 0}, {1, 1}, {0.5, 0.25}, {0, 1}}
	for i, want := range []bool{false, false, false, true, false} {
		if got := reflex(r, i); got != want {
			t.Errorf("corner %d: reflex %v", i, got)
		}
	}
	if reflex(ring{{0, 0}, {1, 0}, {2, 0}, {1, 1}}, 1) {
		t.Error("a point on a straight run is reflex")
	}
}

// a corner opens toward a point strictly inside its turn: a corner that
// turns counterclockwise, however little, opens only on the left of
// both its steps, and one that turns clockwise on the left of either; a
// point on a step's line is not inside.
func TestACornerOpensBetweenItsSteps(t *testing.T) {
	a, p, c := point{0, 0}, point{1, 0}, point{1, 1}
	for m, want := range map[point]bool{{0.5, 0.5}: true, {2, 0.5}: false,
		{0.5, -0.5}: false, {0.5, 0}: false, {1, 0.5}: false} {
		if got := opens(a, p, c, m); got != want {
			t.Errorf("the left corner opens toward %v: %v", m, got)
		}
	}
	ca, cp, cc := point{0, 0}, point{0.1, 0}, point{0.1, 0.1}
	if opens(ca, cp, cc, point{0.2, 0.05}) {
		t.Error("a small left corner opens outside its steps")
	}
	for m, want := range map[point]bool{{0.5, 0.5}: false, {2, 0.5}: true,
		{0.5, -0.5}: true, {2, -0.5}: true} {
		if got := opens(c, p, a, m); got != want {
			t.Errorf("the right corner opens toward %v: %v", m, got)
		}
	}
}

// a point on any edge of a triangle is inside it.
func TestATrianglesEdgesAreInsideIt(t *testing.T) {
	a, b, c := point{0, 0}, point{4, 0}, point{0, 4}
	for _, p := range []point{{2, 0}, {2, 2}, {0, 2}, {1, 1}} {
		if !inside(a, b, c, p) {
			t.Errorf("%v is not inside", p)
		}
	}
	if inside(a, b, c, point{3, 3}) {
		t.Error("3, 3 is inside")
	}
}

// a ring a unit across is cut: a triangle of half a unit's area and a
// square of one are faces, not refused as having none.
func TestASmallRingIsCut(t *testing.T) {
	for _, r := range []ring{{{0, 0}, {1, 0}, {0, 1}}, square(0, 0, 1)} {
		tris, err := clipped(r)
		if err != nil {
			t.Fatalf("%v: %v", r, err)
		}
		if got, _ := covered(tris); got != r.area() {
			t.Errorf("%v cut to %v of %v", r, got, r.area())
		}
	}
}

// no ear at all is minus one, and a backward ring has none.
func TestABackwardRingHasNoEarAnywhere(t *testing.T) {
	backward := square(0, 0, 10).reversed()
	if got := earAt(backward, []int{0, 1, 2, 3}, 0); got != -1 {
		t.Errorf("an ear at %d", got)
	}
}

// a hole of more points than its outer is spliced in whole.
func TestAHoleOfManyPointsIsBridged(t *testing.T) {
	var hole ring
	for k := range 12 {
		turn := -2 * math.Pi * float64(k) / 12
		hole = append(hole, point{5 + 2*math.Cos(turn),
			5 + 2*math.Sin(turn)})
	}
	joined := bridge(square(0, 0, 10), hole)
	if len(joined) != 4+12+2 {
		t.Errorf("the bridged ring has %d points", len(joined))
	}
}

// where two copies of a bridge's end lie in a ring, the one whose
// corner opens toward m is the one taken.
func TestTheCopyThatOpensTowardMIsTaken(t *testing.T) {
	hole := ring{{3, 0.5}, {3, 4.5}, {7, 4.5}, {7, 0.5}}
	joined := bridge(square(0, 0, 10), hole)
	copies := []int{}
	for i, p := range joined {
		if p == (point{10, 0}) {
			copies = append(copies, i)
		}
	}
	if len(copies) != 2 {
		t.Fatalf("the corner is in the ring %d times: %v",
			len(copies), joined)
	}
	if got := opening(joined, copies[0], point{9, 8}); got != copies[1] {
		t.Errorf("the copy at %d is taken, not %d", got, copies[1])
	}
}

// a letter's front and back faces have the front's and the back's
// normal exactly, at every corner.
func TestFrontAndBackFacesFaceStraightOut(t *testing.T) {
	face := mustFont(t, goregular.TTF)
	solid := made(t, face, 'o', plain)
	for i := range solid.FaceCount() {
		a, b, c := solid.Corners(i)
		turn := b.Sub(a).Cross(c.Sub(a)).Unit()
		if math.Abs(turn.Z) < 1-1e-12 {
			continue
		}
		normals, _ := solid.CornerNormals(i)
		for _, n := range normals {
			if n != maths.V(0, 0, math.Copysign(1, turn.Z)) {
				t.Fatalf("face %d faces %v, a corner %v",
					i, turn, n)
			}
		}
	}
}

// two long lines a hair off level meet nowhere: the cutoff scales with
// both lines' lengths.
func TestLongLinesNearlyLevelMeetNowhere(t *testing.T) {
	one := edge{point{0, 0}, point{10, 0}}
	two := edge{point{0, 1}, point{10, 5e-9}}
	if _, ok := meet(one, two); ok {
		t.Error("lines ten long, five billionths apart in turn, meet")
	}
}

// a solid is kept by its turns, at any size: a tiny outer and a tiny
// hole are kept, an outer with no area is not, and a hole turning the
// outer's way or with no area is not.
func TestASolidIsKeptAtAnySize(t *testing.T) {
	tiny := ring{{0, 0}, {1, 0}, {0, 1}}
	small := ring{{0.1, 0.1}, {0.1, 0.2}, {0.2, 0.1}}
	flat := ring{{0, 0}, {1, 0}, {2, 0}}
	for name, each := range map[string]struct {
		one  solid
		want bool
	}{"tiny": {solid{outer: tiny, holes: []ring{small}}, true},
		"flat outer": {solid{outer: flat}, false},
		"hole turned": {solid{outer: tiny,
			holes: []ring{small.reversed()}}, false},
		"flat hole": {solid{outer: tiny, holes: []ring{{{0.1, 0.1},
			{0.2, 0.1}, {0.3, 0.1}}}}, false},
	} {
		if got := kept(each.one); got != each.want {
			t.Errorf("%s: kept %v", name, got)
		}
	}
}

// the step that vanishes may be the ring's first.
func TestTheFirstStepCanVanish(t *testing.T) {
	cut := ring{{9.5, 0}, {10, 0.5}, {10, 10}, {0, 10}, {0, 0}}
	in, to, err := insetRing(cut, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(in) != 4 || to[0] != to[1] ||
		in[to[0]].sub(point{8, 2}).length() > 1e-9 {
		t.Errorf("inset %v, places %v", in, to)
	}
}

// two steps alive side by side that run the same way have no corner:
// errBevel.
func TestStepsAliveThatRunTheSameWayHaveNoCorner(t *testing.T) {
	edges := []edge{{point{0, 0}, point{1, 0}}, {point{5, 0}, point{1, 0}},
		{point{9, 9}, point{0, 1}}}
	if _, err := cornersOf(edges, []int{0, 1, 2}, 1); !errors.Is(err,
		errBevel) {
		t.Errorf("level steps side by side: %v", err)
	}
}
