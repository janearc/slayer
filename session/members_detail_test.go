package session

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies/scrooge"
)

// allRungs is every complexity, as a kind that makes them all has them.
func allRungs() []string { return manufacture.Complexities() }

// TestTheLadderIsPickedByPixels checks each rung of the ladder under the
// set's own, at the edges of its spans.
func TestTheLadderIsPickedByPixels(t *testing.T) {
	cases := []struct {
		pixels float64
		want   string
	}{
		{0, "hull"}, {hullBelow - 0.01, "hull"},
		{hullBelow, "clumps"}, {clumpsBelow - 0.01, "clumps"},
		{clumpsBelow, "quick"}, {quickBelow - 0.01, "quick"},
		{quickBelow, "demo"}, {1e6, "demo"},
	}
	for _, each := range cases {
		if got := rungAt(each.pixels, allRungs(), "demo"); got != each.want {
			t.Errorf("%v pixels: %s, want %s", each.pixels, got, each.want)
		}
	}
}

// TestNoRungIsFinerThanTheSets is the anti-test of the limit: whatever
// the span, a thing is never made finer than the set says.
func TestNoRungIsFinerThanTheSets(t *testing.T) {
	order := allRungs()
	for limit := range order {
		for _, pixels := range []float64{0, 3, 10, 100, 1000, 1e9} {
			got := rungAt(pixels, order, order[limit])
			if indexOf(order, got) > limit {
				t.Errorf("%v pixels under %s gave %s", pixels,
					order[limit], got)
			}
		}
	}
}

// indexOf is where a complexity stands among the others.
func indexOf(order []string, level string) int {
	for index, each := range order {
		if each == level {
			return index
		}
	}
	return -1
}

// TestAKindWithoutARungTakesTheNextFinerOne checks a kind that has only
// quick is given quick for a hull's span, and the set's if it is coarser.
func TestAKindWithoutARungTakesTheNextFinerOne(t *testing.T) {
	if got := rungAt(1, []string{"quick"}, "demo"); got != "quick" {
		t.Errorf("a quick-only kind at a hull's span got %s", got)
	}
	if got := rungAt(1, []string{"quick", "demo"}, "clumps"); got != "clumps" {
		t.Errorf("a kind with none up to the set's got %s, want the set's",
			got)
	}
}

// TestOutOfViewIsTheCoarsestRungThere checks the coarsest rung a kind has,
// held to the set's.
func TestOutOfViewIsTheCoarsestRungThere(t *testing.T) {
	if got := coarsestRung(allRungs(), "quick"); got != "hull" {
		t.Errorf("coarsest of all, got %s", got)
	}
	if got := coarsestRung([]string{"quick", "demo"}, "demo"); got != "quick" {
		t.Errorf("coarsest of quick and demo, got %s", got)
	}
	if got := coarsestRung([]string{"demo"}, "clumps"); got != "clumps" {
		t.Errorf("a kind with none under the set's got %s", got)
	}
}

// TestNothingChangesWhereScroogeIsNotAsked is the anti-test of the
// switch: unset, the set's complexity comes back and the stage, which
// here is empty, is never touched.
func TestNothingChangesWhereScroogeIsNotAsked(t *testing.T) {
	t.Setenv(scrooge.Variable, "")
	stage := &staging{}
	got, err := stage.memberDetail("avicennia-marina", "", "clumps", 1, "",
		maths.Subspace{}, false)
	if err != nil || got != "clumps" {
		t.Errorf("got %q, %v; want clumps", got, err)
	}
	if len(stage.rungs.made) != 0 || len(stage.facts) != 0 {
		t.Error("an unasked stage counted or learned something")
	}
}

// TestLevelledAndBurningAreLeftAlone is the anti-test of what is not this
// rule's: a thing the set levels its own way, and one that burns, keep
// their complexity, with scrooge asked for.
func TestLevelledAndBurningAreLeftAlone(t *testing.T) {
	t.Setenv(scrooge.Variable, "1")
	stage := &staging{}
	got, err := stage.memberDetail("tower", "", manufacture.Levelled, 1, "",
		maths.Subspace{}, false)
	if err != nil || got != manufacture.Levelled {
		t.Errorf("levelled: got %q, %v", got, err)
	}
	got, err = stage.memberDetail("avicennia-marina", "", "quick", 1, "",
		maths.Subspace{}, true)
	if err != nil || got != "quick" {
		t.Errorf("burning: got %q, %v", got, err)
	}
}

// TestTheNoteSaysEachRung checks the session's note names every rung and
// the unseen, and that nothing counted says nothing.
func TestTheNoteSaysEachRung(t *testing.T) {
	stage := &staging{}
	stage.rungNote()
	if len(stage.given.cached) != 0 {
		t.Fatal("a note was made with nothing counted")
	}
	stage.counted("hull", true)
	stage.counted("hull", false)
	stage.counted("quick", false)
	stage.rungNote()
	if len(stage.given.cached) != 1 {
		t.Fatalf("%d notes, want one", len(stage.given.cached))
	}
	said := stage.given.cached[0].String()
	for _, want := range []string{"hull 2", "clumps 0", "quick 1", "demo 0",
		"1 out of view"} {
		if !strings.Contains(said, want) {
			t.Errorf("note %q lacks %q", said, want)
		}
	}
}
