package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
)

func TestFireLayoutsKeepDistinctSeededSourcesInOneVolume(t *testing.T) {
	for _, layout := range []FireLayout{
		{Pattern: "cluster", Count: 7, Radius: measure.Metres(0.8), Variation: 0.3},
		{Pattern: "line", Count: 7, Length: measure.Metres(4), Variation: 0.3},
	} {
		spec := burnerFixture()
		spec.Layout = layout
		made, err := Burn(spec)
		if err != nil {
			t.Fatal(err)
		}
		volumes, _, err := made.Place(maths.Subspace{}, 100, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(volumes) != 2 {
			t.Fatal("source patches expanded into world volumes")
		}
		field, ok := volumes[0].Medium.(*flame.Field)
		if !ok {
			t.Fatal("layout did not produce a shared source field")
		}
		pieces := field.Pieces()
		if len(pieces) != 7 {
			t.Fatal("lost source patches")
		}
		seeds := map[uint64]bool{}
		for _, piece := range pieces {
			f, _, _, _ := piece.Body.Samples()
			if seeds[f.Seed] {
				t.Fatal("source phases duplicated")
			}
			seeds[f.Seed] = true
			if f.Height < 0.7*spec.Flame.Height || f.Height > 1.3*spec.Flame.Height {
				t.Fatal("height variation escaped its range")
			}
			if layout.Pattern == "line" && (piece.At.Z != 0 || math.Abs(piece.At.X) > 2) {
				t.Fatal("line placement escaped")
			}
			if layout.Pattern == "cluster" && math.Hypot(piece.At.X, piece.At.Z) > 0.8 {
				t.Fatal("cluster placement escaped")
			}
		}
		again, err := sourcePatches(spec)
		if err != nil || len(again) != len(pieces) {
			t.Fatal("layout did not repeat")
		}
		for i, piece := range pieces {
			f, _, _, _ := piece.Body.Samples()
			if again[i].At != piece.At || again[i].Fire != f {
				t.Fatal("same seed changed its source field")
			}
		}
	}
}

func TestFireLayoutRejectsImpossibleOrIgnoredParameters(t *testing.T) {
	for _, layout := range []FireLayout{
		{Pattern: "unknown", Count: 7},
		{Pattern: "cluster", Count: 1, Radius: measure.Metres(1)},
		{Pattern: "cluster", Count: 33, Radius: measure.Metres(1)},
		{Pattern: "cluster", Count: 7},
		{Pattern: "line", Count: 7},
		{Pattern: "line", Count: 7, Length: measure.Metres(1), Radius: measure.Metres(1)},
		{Pattern: "cluster", Count: 7, Radius: measure.Metres(1), Variation: math.NaN()},
		{Pattern: "cluster", Count: 7, Radius: measure.Metres(1), Variation: 0.9},
	} {
		spec := burnerFixture()
		spec.Layout = layout
		if _, err := Burn(spec); err == nil {
			t.Fatal("invalid source layout accepted")
		}
	}
}

func TestFireFuelRadiusAccountsForItsLayout(t *testing.T) {
	spec := burnerFixture()
	spec.Layout = FireLayout{Pattern: "line", Count: 3, Length: measure.Metres(4)}
	if math.Abs(spec.FuelRadius().InMetres()-2.6) > 1e-12 {
		t.Fatal("line fuel extent lost")
	}
	spec.Layout = FireLayout{Pattern: "cluster", Count: 3, Radius: measure.Metres(0.8)}
	if math.Abs(spec.FuelRadius().InMetres()-1.4) > 1e-12 {
		t.Fatal("cluster fuel extent lost")
	}
}
