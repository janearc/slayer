package smoke_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/smoke"
)

// Every smoke is in exactly one group, and its group is one the list of
// groups names.
func TestEverySmokeHasAKnownGroup(t *testing.T) {
	known := map[string]bool{}
	for _, g := range smoke.Groups() {
		known[g] = true
	}
	for _, s := range smoke.All() {
		if s.Group() == "" {
			t.Errorf("smoke %d, %s, is in no group", s.Number, s.Name)
		}
		if !known[s.Group()] {
			t.Errorf("smoke %d is in group %q, which Groups does not "+
				"list", s.Number, s.Group())
		}
	}
}

// Every group the list names has a smoke, and the groups between them
// hold every smoke once.
func TestEveryGroupHasSmokesAndTogetherTheyHoldAll(t *testing.T) {
	const most = 8 // the limit on groups
	if n := len(smoke.Groups()); n > most {
		t.Errorf("%d groups, at most %d", n, most)
	}
	seen := map[int]int{}
	for _, g := range smoke.Groups() {
		in, err := smoke.InGroup(g)
		if err != nil || len(in) == 0 {
			t.Errorf("group %q has no smokes: %v", g, err)
		}
		for _, s := range in {
			seen[s.Number]++
		}
	}
	for _, s := range smoke.All() {
		if seen[s.Number] != 1 {
			t.Errorf("smoke %d is in %d groups, want 1", s.Number,
				seen[s.Number])
		}
	}
}

// A group no smoke is in is refused, and the error names the input and
// lists the groups there are.
func TestAnUnknownGroupIsRefusedAndTheKnownOnesListed(t *testing.T) {
	in, err := smoke.InGroup("fire")
	if err == nil || in != nil {
		t.Fatalf("fire gave %v, %v", in, err)
	}
	if !strings.Contains(err.Error(), `"fire"`) {
		t.Errorf("the error does not name the input: %v", err)
	}
	for _, g := range smoke.Groups() {
		if !strings.Contains(err.Error(), g) {
			t.Errorf("the error does not list %q: %v", g, err)
		}
	}
}

// The four smokes that take minutes to draw are marked heavy, and nothing
// else is.
func TestTheHeavySmokesAreTheFourThatTakeMinutes(t *testing.T) {
	want := map[int]bool{29: true, 35: true, 36: true, 37: true}
	for _, s := range smoke.All() {
		if s.Heavy() != want[s.Number] {
			t.Errorf("smoke %d heavy is %v, want %v", s.Number, s.Heavy(),
				want[s.Number])
		}
	}
}
