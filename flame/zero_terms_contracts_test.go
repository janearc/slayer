package flame

import (
	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"math"
	"testing"
)

func TestLineOnlyCompositionDoesNotAdvertiseThermalContinuumBands(t *testing.T) {
	spec := lineSpec()
	spec.Continuum = 0
	profile, err := NewComposition(spec)
	if err != nil {
		t.Fatal(err)
	}
	expected := bands.Lines(map[float64]float64{588.995: 2, 589.5924: 1})
	first, last := expected.Span()
	gotFirst, gotLast := profile.Spectrum(spec.Reference).Span()
	if gotFirst != first || gotLast != last {
		t.Fatalf("line-only span %v..%v, want %v..%v", gotFirst, gotLast, first, last)
	}
}

func TestEmptyLocalDensityCannotPoisonItsNeighbourWithOverflowingRadiance(t *testing.T) {
	bright := campfire
	bright.Hottest = measure.Kelvin(math.MaxFloat64)
	bright.Edge = measure.Kelvin(math.MaxFloat64 / 2)
	point := maths.V(0, .05, 0)
	place := maths.V(2, 0, 0)
	transparent, err := NewBody(bright, 1)
	if err != nil {
		t.Fatal(err)
	}
	if transparent.Density(point.Sub(place)) != 0 {
		t.Fatal("empty-space control intersects its displaced flame")
	}
	light := transparent.Shine().At(point.Sub(place))
	if !(math.IsNaN(light.R+light.G+light.B) || math.IsInf(light.R+light.G+light.B, 0)) {
		t.Fatal("transparent control does not challenge radiance overflow")
	}
	alone, err := NewField([]Ignition{{Fire: campfire}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	expected := alone.Shine().At(point)
	if !(expected.R+expected.G+expected.B > 0) {
		t.Fatal("active source control misses its flame")
	}
	for _, cells := range [][]Ignition{{{Fire: campfire}, {Fire: bright, At: place}}, {{Fire: bright, At: place}, {Fire: campfire}}} {
		field, err := NewField(cells, 1)
		if err != nil {
			t.Fatal(err)
		}
		if got := field.Shine().At(point); got != expected {
			t.Fatalf("zero local absorption poisoned active radiance: %v want %v", got, expected)
		}
	}
}
