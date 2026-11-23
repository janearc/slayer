package medium

import (
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// fizzTested is the fizz the tests below hold, read from the chosen numbers.
func fizzTested(t *testing.T) Fizz {
	t.Helper()
	return Fizz{Radius: picked(t, "fizz-radius"),
		Height: picked(t, "fizz-height"),
		Thick:  picked(t, "fizz-thick")}
}

// fizzChurned is the tested fizz with its rise, and with its churn and seed
// when churning is true.
func fizzChurned(t *testing.T, churning bool) Fizz {
	t.Helper()
	fizz := fizzTested(t)
	fizz.Rise = picked(t, "fizz-rise")
	if churning {
		fizz.Churn = picked(t, "fizz-churn")
		fizz.Seed = uint64(pickedCount(t, "fizz-seed"))
	}
	return fizz
}

// fizzAt is the point at a distance out along x and a height up.
func fizzAt(out, height float64) maths.Vec {
	return maths.V(out, height, nought)
}

// A fizz is thick at the middle of its foot, thinner further up, and
// nothing outside its column.
func TestAFizzThinsAsItClimbsAndEndsAtItsEdges(t *testing.T) {
	fizz := fizzTested(t)
	footHeight := picked(t, "fizz-foot-height")
	foot := fizz.Density(fizzAt(nought, footHeight))
	if foot <= nought {
		t.Fatal("the foot of the fizz has no bubbles")
	}
	high := picked(t, "fizz-high-height")
	if up := fizz.Density(fizzAt(nought, high)); up <= nought || up >= foot {
		t.Fatalf("high in the column the fizz is %v against %v at the "+
			"foot", up, foot)
	}
	for _, outside := range []maths.Vec{
		fizzAt(nought, -picked(t, "fizz-below-depth")),
		fizzAt(nought, picked(t, "fizz-above-height")),
		fizzAt(picked(t, "fizz-outside-out"), picked(t, "fizz-outside-height"))} {
		if density := fizz.Density(outside); density != nought {
			t.Errorf("outside the column at %v the fizz is %v", outside,
				density)
		}
	}
	fizzWidens(t, fizz)
}

// fizzWidens checks the column widens: a point outside the foot's radius
// holds bubbles further up.
func fizzWidens(t *testing.T, fizz Fizz) {
	t.Helper()
	flank := maths.V(picked(t, "fizz-flank-out"), nought, nought)
	lowHeight := picked(t, "fizz-flank-low")
	highHeight := picked(t, "fizz-flank-high")
	low := fizz.Density(fizzAt(flank.X, lowHeight))
	high := fizz.Density(fizzAt(flank.X, highHeight))
	if low != nought || high <= nought {
		t.Errorf("the column does not widen: %v low, %v high", low, high)
	}
}

// fizzMoment is the span of time the churn is given, in whole seconds read
// from the chosen numbers.
func fizzMoment(t *testing.T) time.Duration {
	t.Helper()
	return time.Duration(pickedCount(t, "fizz-churn-seconds")) * time.Second
}

// The churn rises with the bubbles: the same point thins and thickens
// as time passes, and without churn it never changes.
func TestAFizzsChurnRisesWithTime(t *testing.T) {
	churned := fizzChurned(t, true)
	at := maths.V(picked(t, "fizz-point-x"), picked(t, "fizz-point-y"),
		picked(t, "fizz-point-z"))
	moment := fizzMoment(t)
	first := churned.Tick(nought).Density(at)
	later := churned.Tick(moment).Density(at)
	if first == later {
		t.Errorf("three seconds of churn left %v unchanged", first)
	}
	still := fizzChurned(t, false)
	before := still.Tick(nought).Density(at)
	after := still.Tick(moment).Density(at)
	if before != after {
		t.Errorf("a churnless fizz moved: %v then %v", before, after)
	}
	// the same seed is the same fizz for ever
	again := fizzChurned(t, true)
	if again.Tick(moment).Density(at) != later {
		t.Error("the same seed and moment differ")
	}
}
