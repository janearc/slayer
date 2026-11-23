package manufacture

import (
	"slices"
	"testing"
)

// TestEveryRungASaysIsMade is the contract of RungsOf: each rung it names
// for a plant is made, from a seed, without refusal.
func TestEveryRungASaysIsMade(t *testing.T) {
	for _, name := range mangroveKinds {
		row, err := KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, rung := range RungsOf(row) {
			if _, err := MakeWithComplexity(row, sow(1), rung); err != nil {
				t.Errorf("%s says %s and cannot make it: %v", name, rung,
					err)
			}
		}
	}
}

// TestAPlantSaysAllFourRungs checks the mangroves give the hull, clumps,
// quick and demo, coarse to fine.
func TestAPlantSaysAllFourRungs(t *testing.T) {
	row, err := KindNamed("avicennia-marina")
	if err != nil {
		t.Fatal(err)
	}
	if got := RungsOf(row); !slices.Equal(got, Complexities()) {
		t.Errorf("rungs %v, want %v", got, Complexities())
	}
}

// TestRungsAreOnlyKnownComplexities is the anti-test: no kind names a
// rung that Complexities does not, and every kind has quick.
func TestRungsAreOnlyKnownComplexities(t *testing.T) {
	for _, name := range KindNames() {
		row, err := KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		rungs := RungsOf(row)
		if !slices.Contains(rungs, "quick") {
			t.Errorf("%s says %v, with no quick", name, rungs)
		}
		for _, rung := range rungs {
			if !slices.Contains(Complexities(), rung) {
				t.Errorf("%s says %q, which is no complexity", name, rung)
			}
		}
	}
}

// TestAKindThatSaysNothingHasQuickAlone is the anti-test of a kind with a
// madeAt and no word on its rungs: it is not given a rung it may refuse.
func TestAKindThatSaysNothingHasQuickAlone(t *testing.T) {
	row, err := KindNamed("tower")
	if err != nil {
		t.Fatal(err)
	}
	if got := RungsOf(row); !slices.Equal(got, []string{"quick"}) {
		t.Errorf("rungs %v, want quick alone", got)
	}
}
