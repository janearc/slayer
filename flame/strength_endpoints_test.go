package flame

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestUnitFuelStrengthPreservesTheOriginalField(t *testing.T) {
	original, err := NewField(patchFixture(), 1)
	if err != nil {
		t.Fatal(err)
	}
	strengths := make([]float64, len(patchFixture()))
	for i := range strengths {
		strengths[i] = 1
	}
	full, err := original.WithStrengths(strengths)
	if err != nil {
		t.Fatalf("unit fuel strength refused: %v", err)
	}
	if full.Bounds() != original.Bounds() {
		t.Fatal("unit strength changed physical extent")
	}
	for _, p := range []maths.Vec{maths.Vec{}, maths.V(.1, .1, 0), maths.V(1, .2, .1), maths.V(3, .3, -.1)} {
		if full.Density(p) != original.Density(p) || full.Shine().At(p) != original.Shine().At(p) {
			t.Fatalf("unit fuel changed transport at %v", p)
		}
	}
	if len(full.Sources(6)) == 0 {
		t.Fatal("unit-strength source control emitted no light")
	}
}

func TestSingleCellFieldSamplingKeepsAnOccupiedSource(t *testing.T) {
	spec := campfire
	spec.Intermittency = .95
	single, err := NewBody(spec, 1)
	if err != nil {
		t.Fatal(err)
	}
	expected := single.Sources(1)
	if len(expected) != 1 {
		t.Fatal("one-cell control misses its flame")
	}
	field, err := NewField([]Ignition{{Fire: spec}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	got := field.Sources(1)
	if len(got) != 1 || got[0] != expected[0] {
		t.Fatalf("one-cell field source %v expected %v", got, expected)
	}
}
