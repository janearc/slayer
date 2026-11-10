package flame

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestPlacedSourceKeepsCellPowerPerEmittingVolume(t *testing.T) {
	cell := Source{At: maths.V(1, 2, 3), Across: 2, Strength: maths.RGB(3, 4, 5)}
	if got := cell.Placed(maths.Subspace{}); got != cell {
		t.Fatalf("zero-scale identity: %+v", got)
	}
	got := cell.Placed(maths.Subspace{Origin: maths.V(10, 20, 30), Scale: 2})
	// A doubled cube contains eight original cells of emitting material.
	power := maths.Light{}
	for range 8 {
		power = power.Add(cell.Strength)
	}
	if got.At != maths.V(12, 24, 36) || got.Across != 4 || got.Strength != power {
		t.Fatalf("placed source: %+v, eight cells %v", got, power)
	}
	if cell.At != maths.V(1, 2, 3) || cell.Across != 2 {
		t.Fatal("placement mutated its source")
	}
}
