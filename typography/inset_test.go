package typography

import (
	"errors"
	"math"
	"testing"
)

// a square moved in by one is the square one smaller each side, every
// corner its own: each point moved to its own place in the inset.
func TestASquareMovedInIsASmallerSquare(t *testing.T) {
	square := ring{{0, 0}, {10, 0}, {10, 10}, {0, 10}}
	in, to, err := insetRing(square, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := ring{{1, 1}, {9, 1}, {9, 9}, {1, 9}}
	for i := range want {
		if in[i].sub(want[i]).length() > 1e-12 || to[i] != i {
			t.Errorf("corner %d moved to %v at %d, want %v", i,
				in[i], to[i], want[i])
		}
	}
}

// a corner cut off by a step shorter than the bevel turns it: the short
// step vanishes on the way in, its two ends move to one corner, and the
// inset is the square's own smaller square.
func TestAStepShorterThanTheBevelVanishes(t *testing.T) {
	cut := ring{{0, 0}, {9.5, 0}, {10, 0.5}, {10, 10}, {0, 10}}
	in, to, err := insetRing(cut, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(in) != 4 || to[1] != to[2] {
		t.Fatalf("inset %v, places %v", in, to)
	}
	at := in[to[1]]
	if at.sub(point{8, 2}).length() > 1e-9 {
		t.Errorf("the cut corner moved to %v, want 8, 2", at)
	}
	if math.Abs(abs(in.area())-36) > 1e-9 {
		t.Errorf("the inset's area is %v, want 36", in.area())
	}
}

// a stroke narrower than two bevels has nothing left once moved in, and
// a hole moved out across its outer crosses it: both are errBevel.
func TestAStrokeNarrowerThanTwoBevelsIsRefused(t *testing.T) {
	thin := ring{{0, 0}, {100, 0}, {100, 3}, {0, 3}}
	if _, _, err := insetRing(thin, 2); !errors.Is(err, errBevel) {
		t.Errorf("a stroke 3 wide moved in by 2: %v", err)
	}
	framed := solid{outer: ring{{0, 0}, {10, 0}, {10, 10}, {0, 10}},
		holes: []ring{{{2, 2}, {2, 8}, {8, 8}, {8, 2}}}}
	if _, _, err := inset(framed, 1.5); !errors.Is(err, errBevel) {
		t.Errorf("a frame 2 wide moved in by 1.5: %v", err)
	}
	if _, _, err := inset(framed, 0.5); err != nil {
		t.Errorf("a frame 2 wide moved in by 0.5: %v", err)
	}
}

// two lines that run the same way meet nowhere; two that cross meet
// where they cross.
func TestLinesMeetWhereTheyCross(t *testing.T) {
	across := edge{point{0, 1}, point{1, 0}}
	up := edge{point{3, 0}, point{0, 2}}
	if at, ok := meet(across, up); !ok || at != (point{3, 1}) {
		t.Errorf("across and up meet at %v, %v", at, ok)
	}
	if _, ok := meet(across, edge{point{0, 5}, point{-4, 0}}); ok {
		t.Error("two level lines meet")
	}
}

// each point is placed at its own step's corner when the step is alive,
// and at the next living step's corner when it is not, round the ring.
func TestPlacesFollowTheStepsAlive(t *testing.T) {
	got := places(6, []int{0, 2, 5})
	want := []int{0, 1, 1, 2, 2, 2}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("places %v, want %v", got, want)
		}
	}
	if got := places(4, []int{1, 2, 3}); got[0] != 0 || got[1] != 0 {
		t.Errorf("with step 0 gone, places %v", got)
	}
}
