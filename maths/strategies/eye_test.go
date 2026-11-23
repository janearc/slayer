package strategies

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// testAcross is the film of the tests' eye, in pixels across, and testDown
// down; with a lens a quarter turn wide a metre spans testFocal pixels at a
// metre.
const (
	testAcross = 1000
	testDown   = 500
	testFocal  = testAcross / 2
)

// testEye stands at a place looking at another, a quarter turn wide.
func testEye(from, to maths.Vec) Eye {
	return Eye{Place: maths.Subspace{Origin: from,
		Orientation: maths.LookAtUp(from, to, maths.Up)},
		Wide: math.Pi / 2, Across: testAcross, Down: testDown}
}

// ahead is an eye at the origin looking down minus z.
func ahead() Eye { return testEye(maths.Vec{}, maths.V(0, 0, -1)) }

func TestASpanIsTheExtentOverTheDistanceToItsNearSide(t *testing.T) {
	got := ahead().Span(maths.V(0, 0, -100), 2)
	want := 2.0 * testFocal / 99
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("a span of %v pixels, not %v", got, want)
	}
}

func TestTheEyeSeesWhatIsAheadAndNotWhatIsBehindOrAside(t *testing.T) {
	eye := ahead()
	behind, aside := maths.V(0, 0, 5), maths.V(50, 0, -5)
	for _, at := range []maths.Vec{behind, aside} {
		unit := maths.V(1, 1, 1)
		if eye.Sees(maths.Box{Min: at, Max: at.Add(unit)}) {
			t.Errorf("the eye sees %v", at)
		}
	}
	if !eye.Sees(maths.Box{Min: maths.V(-1, -1, -11),
		Max: maths.V(1, 1, -9)}) {
		t.Error("the eye does not see what is ahead of it")
	}
}

func TestAnEyeThatSeesNothingIsRefused(t *testing.T) {
	for name, eye := range map[string]Eye{"no lens": {Across: 1, Down: 1},
		"a half turn": {Wide: math.Pi, Across: 1, Down: 1},
		"no film":     {Wide: 1}} {
		if eye.Check() == nil {
			t.Errorf("an eye with %s was taken", name)
		}
	}
	if err := ahead().Check(); err != nil {
		t.Errorf("an eye that sees was refused: %v", err)
	}
}

func TestACoverIsWhetherTheBoxIsInViewAndHowWideItSpans(t *testing.T) {
	eye := ahead()
	box := maths.Box{Min: maths.V(-1, -1, -101), Max: maths.V(1, 1, -99)}
	got := eye.Cover(box)
	reach := box.Max.Sub(box.Min).Len()
	want := eye.Span(maths.V(0, 0, -100), reach)
	if !got.Visible || math.Abs(got.Pixels-want) > 1e-9 {
		t.Errorf("a cover of %+v, not %v pixels in view", got, want)
	}
	behind := maths.Box{Min: maths.V(-1, -1, 9), Max: maths.V(1, 1, 11)}
	if eye.Cover(behind).Visible {
		t.Error("a box behind the eye was in view")
	}
}

func TestABoxWithNoEdgeIsSeenAndFillsTheFilm(t *testing.T) {
	got := ahead().Cover(maths.Everywhere())
	if !got.Visible || got.Pixels != testAcross {
		t.Errorf("a box with no edge covers %+v", got)
	}
}
