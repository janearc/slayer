package flame_test

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"math"
	"testing"
	"time"
)

func TestFieldStrengthChangesActualDensityAndRetainsPalette(t *testing.T) {
	f := flame.Fire{Radius: 1, Height: 2, Absorption: 1, Hottest: measure.Kelvin(1700), Edge: measure.Kelvin(900), Seed: 7}
	field, e := flame.NewField([]flame.Ignition{{Fire: f}, {At: maths.V(3, 0, 0), Fire: f}}, 100)
	if e != nil {
		t.Fatal(e)
	}
	weighted, e := field.WithStrengths([]float64{.125, 0})
	if e != nil {
		t.Fatal(e)
	}
	pieces := weighted.Pieces()
	if len(pieces) != 1 {
		t.Fatal("extinguished source retained", len(pieces))
	}
	got, _, _, glow := pieces[0].Body.Samples()
	_, _, _, original := field.Pieces()[0].Body.Samples()
	if got.Absorption != .125 || math.Abs(got.Height-1) > 1e-12 || glow[200] != original[200] {
		t.Fatal("strength not carried to GPU description")
	}
	if len(field.Pieces()) != 2 {
		t.Fatal("mutated original")
	}
	late := weighted.Tick(time.Second).(*flame.Field)
	if len(late.Pieces()) != 1 || late.Bounds() != field.Bounds() {
		t.Fatal("lost weights or bounds")
	}
	for _, weights := range [][]float64{{1}, {1, -1}, {1, math.NaN()}, {1, 2}} {
		if _, e := field.WithStrengths(weights); e == nil {
			t.Fatal("invalid strength accepted")
		}
	}
}
