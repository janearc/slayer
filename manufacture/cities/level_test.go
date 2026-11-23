package cities

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
)

// testAcross is the film of the tests' eye, in pixels across, and testDown
// down.
const (
	testAcross = 1000
	testDown   = 500
)

// testEye stands at a place looking at another, a quarter turn wide.
func testEye(from, to maths.Vec) strategies.Eye {
	return strategies.Eye{Place: maths.Subspace{Origin: from,
		Orientation: maths.LookAtUp(from, to, maths.Up)},
		Wide: math.Pi / 2, Across: testAcross, Down: testDown}
}

// testLadder is a ladder of three rungs, at nought, ten and a hundred.
var testLadder = []Rung{{Level: "hull"}, {Level: "quick", Pixels: 10},
	{Level: "demo", Pixels: 100}}

// ahead is an eye at the origin looking down minus z.
func ahead() strategies.Eye { return testEye(maths.Vec{}, maths.V(0, 0, -1)) }

func TestALevellerTakesTheFinestRungAThingReaches(t *testing.T) {
	leveller, err := NewLeveller(ahead(), testLadder)
	if err != nil {
		t.Fatal(err)
	}
	for distance, want := range map[float64]Detail{1000: "hull",
		10: "quick", 2: "demo"} {
		at := standingAt(maths.V(0, 0, -distance))
		if got := leveller(at, 1); got != want {
			t.Errorf("a metre at %v m is %s, not %s", distance, got, want)
		}
	}
}

func TestAThingOutOfViewIsDrawnAtTheFirstRung(t *testing.T) {
	eye := ahead()
	leveller, err := NewLeveller(eye, testLadder)
	if err != nil {
		t.Fatal(err)
	}
	behind, aside := maths.V(0, 0, 5), maths.V(50, 0, -5)
	for _, at := range []maths.Vec{behind, aside} {
		if got := leveller(standingAt(at), 1); got != "hull" {
			t.Errorf("a thing at %v out of view is %s", at, got)
		}
		corner := at.Add(maths.V(1, 1, 1))
		if eye.Sees(maths.Box{Min: at, Max: corner}) {
			t.Errorf("the eye sees %v", at)
		}
	}
	if !eye.Sees(maths.Box{Min: maths.V(-1, -1, -11),
		Max: maths.V(1, 1, -9)}) {
		t.Error("the eye does not see what is ahead of it")
	}
}

func TestAConstantLevellerIsOneLevelEverywhere(t *testing.T) {
	leveller := Constant("quick")
	for _, at := range []maths.Vec{{}, maths.V(0, 0, 1e6)} {
		if got := leveller(standingAt(at), 1); got != "quick" {
			t.Errorf("the constant leveller gives %s", got)
		}
	}
}

func TestALadderThatDoesNotStartAtNoughtOrRiseIsRefused(t *testing.T) {
	for name, ladder := range map[string][]Rung{"empty": nil,
		"from ten": {{Level: "hull", Pixels: 10}},
		"falling": {{Level: "hull"}, {Level: "quick", Pixels: 10},
			{Level: "demo", Pixels: 5}},
		"level": {{Level: "hull"}, {Level: "quick"}}} {
		if _, err := NewLeveller(ahead(), ladder); err == nil {
			t.Errorf("a ladder %s was taken", name)
		}
	}
}

func TestAnEyeThatSeesNothingIsRefused(t *testing.T) {
	for name, eye := range map[string]strategies.Eye{
		"no lens":     {Across: 1, Down: 1},
		"a half turn": {Wide: math.Pi, Across: 1, Down: 1},
		"no film":     {Wide: 1}} {
		if _, err := NewLeveller(eye, testLadder); err == nil {
			t.Errorf("an eye with %s was taken", name)
		}
	}
}

func TestTheLevelsFileHasItsLaddersAndBands(t *testing.T) {
	for _, name := range []string{"trees", "buildings", "manufactory"} {
		if _, err := Ladder(name); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := Ladder("nonesuch"); err == nil {
		t.Error("a ladder the file has not was given")
	}
	if storeys, err := band("tower-storeys"); err != nil || storeys < 1 {
		t.Errorf("a tower's band is %d storeys: %v", storeys, err)
	}
	if _, err := band("nonesuch"); err == nil {
		t.Error("a band the file has not was given")
	}
}
