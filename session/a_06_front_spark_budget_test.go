package session

import (
	"testing"
)

// An ordinary fire without a spark source reserves no particle slots.
func TestAbsentSparkSourceReservesNoParticleSlots(t *testing.T) {
	var source *retainedSparks
	if got := source.budget(); got != 0 {
		t.Fatalf("absent source reserves %d particle slots, want zero", got)
	}
}
