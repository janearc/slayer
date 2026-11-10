package flame

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

func TestFieldRejectsEachNonfiniteDirectionAndNegativeDuration(t *testing.T) {
	for _, point := range []maths.Vec{
		maths.V(math.Inf(-1), 0, 0), maths.V(0, math.Inf(-1), 0), maths.V(0, 0, math.Inf(-1)),
		maths.V(math.Inf(1), 0, 0), maths.V(0, math.NaN(), 0),
	} {
		if _, err := NewField([]Ignition{{At: point, Fire: campfire}}, 1); err == nil {
			t.Fatalf("nonfinite source %v accepted", point)
		}
	}
	// At this endpoint the overflow check alone cannot stand in for a
	// duration check: MaxInt64-(-1) wraps to MinInt64.
	if _, err := NewField([]Ignition{{Fire: campfire, Start: time.Duration(math.MinInt64), Duration: -time.Nanosecond}}, 1); err == nil {
		t.Fatal("negative duration accepted at the minimum start")
	}
}

func TestFieldAcceptsHalfDurationFadeAndTheLastRepresentableEnd(t *testing.T) {
	start := time.Duration(math.MaxInt64) - 2*time.Second
	field, err := NewField([]Ignition{{Fire: campfire, Start: start, Duration: 2 * time.Second, Fade: time.Second}}, 1)
	if err != nil {
		t.Fatalf("valid fade/end boundary refused: %v", err)
	}
	peak := field.Tick(start + time.Second).(*Field)
	if len(peak.Pieces()) != 1 || !(peak.Density(maths.V(0, .05, 0)) > 0) {
		t.Fatal("half-duration fades lost their shared peak")
	}
	if got := field.Tick(time.Duration(math.MaxInt64)).(*Field); len(got.Pieces()) != 0 || len(got.Sources(6)) != 0 {
		t.Fatal("last representable end is not exclusive")
	}
}
