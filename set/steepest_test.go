package set_test

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// standNamed is the stand of a name in a set, or a failure.
func standNamed(t *testing.T, made set.Set, name string) set.Stand {
	t.Helper()
	for _, each := range made.Stands() {
		if each.Name() == name {
			return each
		}
	}
	t.Fatalf("the set has no stand %s", name)
	return set.Stand{}
}

func TestAStandSaysHowSteepTheGroundItGrowsOnMayBe(t *testing.T) {
	made, err := readMangroveSet(t, "portofino.yaml", same)
	if err != nil {
		t.Fatal(err)
	}
	if got := standNamed(t, made, "holm oaks a").Steepest(); got != 0.8 {
		t.Errorf("the holm oaks grow on ground as steep as %v, not 0.8", got)
	}
	if got := standNamed(t, made, "tree heath a").Steepest(); got != 1.2 {
		t.Errorf("the tree heath grows on ground as steep as %v, not 1.2", got)
	}
}

func TestAStandThatSaysNoSteepestGrowsOnAnySlope(t *testing.T) {
	made, err := readMangroveSet(t, "mangrove-shore.yaml", same)
	if err != nil {
		t.Fatal(err)
	}
	for _, each := range made.Stands() {
		if !math.IsInf(each.Steepest(), 1) {
			t.Errorf("%s says no steepest and reads %v", each.Name(),
				each.Steepest())
		}
	}
}

func TestASteepestBelowLevelOrPastACliffIsRefused(t *testing.T) {
	for _, steepness := range []string{"-0.1", "11"} {
		_, err := readMangroveSet(t, "portofino.yaml", func(text string) string {
			return strings.Replace(text, "steepest: 0.8", "steepest: "+
				steepness, 1)
		})
		if err == nil {
			t.Errorf("a steepest of %s was taken", steepness)
		}
	}
}

func TestASteepestOnAStandLaidByAHabitIsRefused(t *testing.T) {
	_, err := readMangroveSet(t, "portofino.yaml", func(text string) string {
		return strings.Replace(text, "        habit: contour\n        count: 6\n"+
			"        seed: 11\n", "        habit: contour\n        count: 6\n"+
			"        seed: 11\n        steepest: 0.5\n", 1)
	})
	if err == nil || !strings.Contains(err.Error(), "steepest") {
		t.Errorf("a contour stand saying steepest was not refused for it: %v",
			err)
	}
}
