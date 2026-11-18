package thing

import (
	"github.com/janearc/slayer/maths"
	"math"
	"testing"
	"time"
)

func TestAxisSpinKeepsPivotAndComposesMovingScaledParent(t *testing.T) {
	parent := axisTestParent{}
	pivot := maths.V(2, 3, 4)
	spin, err := NewAxisSpin(parent, pivot, maths.V(0, 0, 2), 60, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := spin.At(0); got != parent.At(0) {
		t.Fatal("zero angle changed parent bits")
	}
	at := 250 * time.Millisecond
	pose := spin.At(at)
	base := parent.At(at)
	if got, want := pose.Out(pivot), base.Out(pivot); got.Sub(want).Len() > 1e-12 {
		t.Fatalf("pivot moved: %v want%v", got, want)
	}
	// Quarter-turn about local+Z sends a local+X offset to local+Y.
	if got, want := pose.Out(pivot.Add(maths.XAxis)), base.Out(pivot.Add(maths.YAxis)); got.Sub(want).Len() > 1e-12 {
		t.Fatalf("rotation/parent order wrong: %v want%v", got, want)
	}
	if pose.Scale != base.Scale {
		t.Fatal("parent scale changed")
	}
}

type axisTestParent struct{}

func (axisTestParent) At(at time.Duration) maths.Subspace {
	return maths.NewSubspace(maths.V(10+at.Seconds(), 20, 30), maths.Rotation(maths.YAxis, .4), 2)
}
func TestAxisSpinSignedRPMPhaseAndExactRest(t *testing.T) {
	parent := Still(maths.NewSubspace(maths.V(-0.0, 5, 0), maths.Identity, 0))
	spin, err := NewAxisSpin(parent, maths.Vec{}, maths.ZAxis, -60, math.Pi/2)
	if err != nil {
		t.Fatal(err)
	}
	if got := spin.At(0).Out(maths.XAxis); got.Sub(maths.V(0, 6, 0)).Len() > 1e-12 {
		t.Fatal(got)
	}
	if got := spin.At(250 * time.Millisecond); got != parent.At(250*time.Millisecond) {
		t.Fatal("zero resulting angle did not return exact parent")
	}
	stopped, err := NewAxisSpin(parent, maths.V(2, 3, 4), maths.ZAxis, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := stopped.At(123 * time.Second); got != parent.At(123*time.Second) {
		t.Fatal("stopped spin changed parent")
	}
}
func TestAxisSpinRefusesInvalidConfiguration(t *testing.T) {
	parent := Still(maths.Local())
	for _, row := range []struct {
		parent      Motion
		pivot, axis maths.Vec
		rpm, phase  float64
	}{
		{nil, maths.Vec{}, maths.ZAxis, 1, 0},
		{parent, maths.V(math.NaN(), 0, 0), maths.ZAxis, 1, 0},
		{parent, maths.Vec{}, maths.Vec{}, 1, 0},
		{parent, maths.Vec{}, maths.V(0, math.Inf(1), 0), 1, 0},
		{parent, maths.Vec{}, maths.ZAxis, math.NaN(), 0},
		{parent, maths.Vec{}, maths.ZAxis, MaximumAxisSpinRPM + 1, 0},
		{parent, maths.Vec{}, maths.ZAxis, 1, math.Inf(1)},
	} {
		if _, err := NewAxisSpin(row.parent, row.pivot, row.axis, row.rpm, row.phase); err == nil {
			t.Fatal("invalid spin accepted")
		}
	}
}
