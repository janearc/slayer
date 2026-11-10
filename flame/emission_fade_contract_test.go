package flame

import (
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

func TestUnitDensityStillEmitsItsSourceFunction(t *testing.T) {
	point := maths.V(0, .05, 0)
	for seed := range uint64(32) {
		spec := campfire
		spec.Seed, spec.Absorption = seed, 1
		reference, err := NewBody(spec, 1)
		if err != nil {
			t.Fatal(err)
		}
		density := reference.Density(point)
		if !(density > 0) {
			continue
		}
		spec.Absorption = 1 / density
		field, err := NewField([]Ignition{{Fire: spec}}, 1)
		if err != nil {
			t.Fatal(err)
		}
		if field.Density(point) != 1 {
			continue
		}
		light := field.Shine().At(point)
		if !(light.R+light.G+light.B > 0) {
			t.Fatal("unit density was mistaken for empty space")
		}
		return
	}
	t.Fatal("could not construct an exact unit-density control")
}

func TestFadePlateauPreservesUnitOpacityAndEmission(t *testing.T) {
	field, err := NewField([]Ignition{{Fire: campfire, Duration: 4 * time.Second, Fade: 500 * time.Millisecond}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	at := 2 * time.Second
	reference, err := NewBody(campfire, 1)
	if err != nil {
		t.Fatal(err)
	}
	reference = reference.Tick(at).(*Body)
	point := maths.V(0, .05, 0)
	expected := reference.Density(point)
	if !(expected > 0) {
		t.Fatal("fade reference misses its source")
	}
	peak := field.Tick(at).(*Field)
	if got := peak.Density(point); got != expected {
		t.Fatalf("fade plateau changed absorption: got %g want %g", got, expected)
	}
	if got, want := field.Shine().Tick(at).At(point), reference.Shine().At(point); got != want {
		t.Fatalf("fade plateau changed source radiance: got %v want %v", got, want)
	}
	if len(peak.Sources(6)) == 0 {
		t.Fatal("fade plateau lost emitted power")
	}
}
