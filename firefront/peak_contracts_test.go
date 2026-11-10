package firefront

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// directPeak counts every event at each birth. Occupancy can only increase
// at births, so this needs no sorted edges or sweep-state arithmetic.
func directPeak(groups [][]Event, tail time.Duration) int {
	if tail < 0 {
		tail = 0
	}
	peak := 0
	for _, group := range groups {
		for _, born := range group {
			occupied := 0
			for _, others := range groups {
				for _, event := range others {
					age := born.Start - event.Start
					if age >= 0 && (age < event.Duration || (tail > 0 && age-event.Duration < tail)) {
						occupied++
					}
				}
			}
			peak = max(peak, occupied)
		}
	}
	return peak
}

func TestParticlePeakMatchesDirectOccupancyAcrossSeparatedBurns(t *testing.T) {
	for _, xs := range [][]float64{{0, 2}, {0, 1, 5, 5.25, 5.5}, {-2, -1, 0, 8, 8.25, 8.5}} {
		spec := Spec{Direction: maths.XAxis, Speed: 1, Cell: 10, Duration: 2 * time.Second}
		var targets []Target
		for i, x := range xs {
			targets = append(targets, Target{At: maths.V(x, 0, 0), Seed: uint64(i)})
		}
		schedule, err := New(spec, targets)
		if err != nil {
			t.Fatal(err)
		}
		groups := schedule.Groups()
		count := 0
		for _, group := range groups {
			count += len(group)
		}
		if count != len(targets) {
			t.Fatal("occupancy fixture lost targets")
		}
		for _, tail := range []time.Duration{-1, 0, time.Nanosecond, time.Second, 10 * time.Second, time.Duration(math.MaxInt64)} {
			expected := directPeak(groups, tail)
			if got := schedule.PeakTargets(tail); got != expected {
				t.Fatalf("positions %v tail %s: peak %d, direct occupancy %d", xs, tail, got, expected)
			}
		}
	}
	empty := &plan{}
	if empty.PeakTargets(0) != 0 || empty.PeakTargets(time.Second) != 0 {
		t.Fatal("empty plan reserved living particles")
	}
}
