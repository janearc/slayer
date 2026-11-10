package embers

import (
	"math"
	"testing"
	"time"
)

// Snapshot work is bounded by living particles, independent of elapsed time.
// This check runs before the ordinary particle fixtures so a reversed birth
// loop fails its explicit deadline instead of consuming the suite timeout.
func TestSnapshotsRemainBoundedAtDistantTimes(t *testing.T) {
	spec := fixture()
	spec.Rate = 2
	source, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan []Particle, 1)
	go func() { done <- source.At(time.Duration(math.MaxInt64 / 4)) }()
	select {
	case particles := <-done:
		if len(particles) < 5 || len(particles) > 7 {
			t.Fatalf("distant snapshot lost live births: %d", len(particles))
		}
		for _, particle := range particles {
			if particle.Age() < 0 || particle.Age() >= spec.Life {
				t.Fatal("distant snapshot returned an expired birth")
			}
		}
	case <-time.After(time.Second):
		t.Fatal("snapshot work grew with elapsed time or moved away from its birth bound")
	}
}

func TestZeroTimeAndStaggeredBirthIntervalsKeepTheirExactIDs(t *testing.T) {
	spec := fixture()
	spec.Seed, spec.Rate = 0, 2
	source, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	zero := source.At(0)
	if len(zero) != 1 || zero[0].ID() != 0 || zero[0].Age() != 0 {
		t.Fatal("seed-zero birth at the clock origin was lost")
	}
	// The SplitMix oracle's birth phases put exactly IDs 2..7 in the last
	// three seconds at 4.125 s; ID 2 was born after the 1.125 s cutoff.
	at := 4125 * time.Millisecond
	particles := source.At(at)
	if len(particles) != 6 {
		t.Fatalf("staggered births: got %d, want six IDs 2..7", len(particles))
	}
	for i, particle := range particles {
		if particle.ID() != int64(i+2) {
			t.Fatalf("birth %d has ID %d", i, particle.ID())
		}
		birth := at - particle.Age()
		start := time.Duration(particle.ID()) * 500 * time.Millisecond
		if birth < start || birth >= start+500*time.Millisecond {
			t.Fatalf("birth %d at %s lies outside its own interval", particle.ID(), birth)
		}
	}
	// This nonzero seed is the first state in the published zero-state
	// SplitMix sequence, so its first birth has a strictly positive phase.
	spec.Seed = 0x9e3779b97f4a7c15
	later, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	if len(later.At(0)) != 0 {
		t.Fatal("a positive-phase birth happened before release")
	}
}
