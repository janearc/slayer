package session

import (
	"testing"

	"github.com/janearc/slayer/set"
)

// The runs the colour tests ask for: enough joints that a clash between
// the colours of neighbours would show if the stand let one through, and
// the seed of the scene the runs are drawn under.
const (
	testedRuns = 200
	sceneSeed  = 1
)

// seededStage is a stage that has only the scene's seed, which is all the
// pick of a run is drawn under.
func seededStage() *staging {
	stage := &staging{}
	stage.given.seed = sceneSeed
	return stage
}

// pickedRuns is the pick of each of the first testedRuns runs of a stand,
// asked for in order.
func pickedRuns(t *testing.T, stage *staging, town set.Stand) []uint64 {
	t.Helper()
	picks := make([]uint64, testedRuns)
	for number := range picks {
		pick, err := stage.runPick(town, number)
		if err != nil {
			t.Fatal(err)
		}
		picks[number] = pick
	}
	return picks
}

func TestNoRunOfAContourStartsInTheColourTheRunBeforeItEnds(t *testing.T) {
	town := townOfTheSteepCoast(t, "")
	climb, _ := town.Climb()
	stage := seededStage()
	picks := pickedRuns(t, stage, town)
	for number := 1; number < testedRuns; number++ {
		if number%climb.Runs == 0 {
			continue
		}
		before, err := stage.runFrom(town, picks[number-1])
		if err != nil {
			t.Fatal(err)
		}
		this, err := stage.runFrom(town, picks[number])
		if err != nil {
			t.Fatal(err)
		}
		if before.Last == this.First {
			t.Errorf("run %d starts in %s, the colour run %d ends in",
				number, this.First, number-1)
		}
	}
}

func TestTheFirstRunOfARowKeepsItsOwnPick(t *testing.T) {
	town := townOfTheSteepCoast(t, "")
	climb, _ := town.Climb()
	_, seed := town.Count()
	picks := pickedRuns(t, seededStage(), town)
	for number := 0; number < testedRuns; number += climb.Runs {
		if want := standPick(town, seed, number); picks[number] != want {
			t.Errorf("run %d, the first of its row, is drawn from %d, "+
				"not %d", number, picks[number], want)
		}
	}
}

func TestARunThatDidNotClashKeepsItsOwnPick(t *testing.T) {
	town := townOfTheSteepCoast(t, "")
	climb, _ := town.Climb()
	_, seed := town.Count()
	stage := seededStage()
	picks := pickedRuns(t, stage, town)
	kept := 0
	for number := 1; number < testedRuns; number++ {
		own := standPick(town, seed, number)
		run, err := stage.runFrom(town, own)
		if err != nil {
			t.Fatal(err)
		}
		before, err := stage.runFrom(town, picks[number-1])
		if err != nil {
			t.Fatal(err)
		}
		if number%climb.Runs == 0 || run.First == before.Last {
			continue
		}
		kept++
		if picks[number] != own {
			t.Errorf("run %d did not clash and is drawn from %d, not %d",
				number, picks[number], own)
		}
	}
	if kept == 0 {
		t.Error("no run in the test avoided a clash by itself")
	}
}

func TestARunAskedForOutOfOrderHasTheSamePickAsInOrder(t *testing.T) {
	town := townOfTheSteepCoast(t, "")
	inOrder := pickedRuns(t, seededStage(), town)
	stage := seededStage()
	for _, number := range []int{41, 7, 140, 3, 3} {
		pick, err := stage.runPick(town, number)
		if err != nil {
			t.Fatal(err)
		}
		if pick != inOrder[number] {
			t.Errorf("run %d is drawn from %d, and from %d in order",
				number, pick, inOrder[number])
		}
	}
}

func TestAStandThatIsNotAContourPicksAsItAlwaysDid(t *testing.T) {
	scatter := townOfTheSteepCoast(t, "lattice")
	_, seed := scatter.Count()
	for index := range 10 {
		got, err := seededStage().pickOf(scatter, seed, index, index+5)
		if err != nil {
			t.Fatal(err)
		}
		if want := standPick(scatter, seed, index); got != want {
			t.Errorf("site %d is drawn from %d, not %d", index, got, want)
		}
	}
}
