package flame

import (
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

func TestZeroAbsorptionIsAValidTransparentNonemittingFlame(t *testing.T) {
	spec := campfire
	spec.Absorption = 0
	made, err := NewBody(spec, 1)
	if err != nil {
		t.Fatalf("a transparent medium was refused: %v", err)
	}
	for _, at := range []time.Duration{0, time.Second, 3 * time.Second} {
		field := made.Tick(at).(*Body)
		for _, point := range []maths.Vec{maths.Vec{}, low(spec), maths.V(.1, .3, 0)} {
			if density := field.Density(point); density != 0 {
				t.Fatalf("zero absorption has density %g", density)
			}
		}
		if samples := field.Sources(6); len(samples) != 0 {
			t.Fatalf("zero absorption emitted %d light samples", len(samples))
		}
	}
}

func TestEqualFlameEdgeAndPeakTemperaturesAreValid(t *testing.T) {
	spec := campfire
	spec.Edge = spec.Hottest
	made, err := NewBody(spec, 1)
	if err != nil {
		t.Fatal(err)
	}
	if heat := made.Heat(low(spec)); heat != spec.Hottest {
		t.Fatalf("isothermal body heat %v, expected %v", heat, spec.Hottest)
	}
}
