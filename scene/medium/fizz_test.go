package medium

import (
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// A fizz is thick at the middle of its foot, thinner further up, and
// nothing outside its column.
func TestAFizzThinsAsItClimbsAndEndsAtItsEdges(t *testing.T) {
	fizz := Fizz{Radius: 0.2, Height: 1, Thick: 2}
	foot := fizz.Density(maths.V(0, 0.01, 0))
	if foot <= 0 {
		t.Fatal("the foot of the fizz has no bubbles")
	}
	if up := fizz.Density(maths.V(0, 0.8, 0)); up <= 0 || up >= foot {
		t.Fatalf("high in the column the fizz is %v against %v at the "+
			"foot", up, foot)
	}
	for _, outside := range []maths.Vec{maths.V(0, -0.01, 0),
		maths.V(0, 1.01, 0), maths.V(0.5, 0.1, 0)} {
		if d := fizz.Density(outside); d != 0 {
			t.Errorf("outside the column at %v the fizz is %v", outside, d)
		}
	}
	// the column widens: a point outside the foot's radius holds
	// bubbles further up
	flank := maths.V(0.25, 0, 0.0)
	low := fizz.Density(maths.V(flank.X, 0.05, 0))
	high := fizz.Density(maths.V(flank.X, 0.5, 0))
	if low != 0 || high <= 0 {
		t.Errorf("the column does not widen: %v low, %v high", low, high)
	}
}

// The churn rises with the bubbles: the same point thins and thickens
// as time passes, and without churn it never changes.
func TestAFizzsChurnRisesWithTime(t *testing.T) {
	churned := Fizz{Radius: 0.2, Height: 1, Thick: 2, Rise: 0.1,
		Churn: 0.8, Seed: 7}
	at := maths.V(0.05, 0.3, 0.02)
	first := churned.Tick(0).Density(at)
	later := churned.Tick(3 * time.Second).Density(at)
	if first == later {
		t.Errorf("three seconds of churn left %v unchanged", first)
	}
	still := Fizz{Radius: 0.2, Height: 1, Thick: 2, Rise: 0.1}
	if a, b := still.Tick(0).Density(at), still.Tick(3*time.Second).Density(at); a != b {
		t.Errorf("a churnless fizz moved: %v then %v", a, b)
	}
	// the same seed is the same fizz for ever
	again := Fizz{Radius: 0.2, Height: 1, Thick: 2, Rise: 0.1,
		Churn: 0.8, Seed: 7}
	if again.Tick(3*time.Second).Density(at) != later {
		t.Error("the same seed and moment differ")
	}
}
