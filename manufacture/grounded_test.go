package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/manufacture/cities"
)

// risingGround is a ground that climbs a tenth of a metre for each metre
// along x, and is level along z.
func risingGround(x, _ float64) float64 { return 0.1 * x }

func TestOnlyTheBuildingsOfPackageCitiesThatHavePlinthsKnowSlopes(t *testing.T) {
	for name, want := range map[string]bool{"terrace-groningen": true,
		"terrace-portofino": true, "terrace-midwest": true,
		"condominium": true, "yurt": false, "winnebago": false,
		"no-such-kind": false} {
		if got := SlopeAware(name); got != want {
			t.Errorf("SlopeAware(%q) is %v, want %v", name, got, want)
		}
	}
}

func TestAKindThatCannotStandOnASlopeOrAnEmptyGroundIsRefused(t *testing.T) {
	yurt, err := KindNamed("yurt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Grounded(yurt, risingGround); err == nil {
		t.Error("a yurt was given a slope to stand on")
	}
	terrace, err := KindNamed("terrace-groningen")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Grounded(terrace, nil); err == nil {
		t.Error("a terrace was grounded on no ground")
	}
}

func TestAGroundedTerraceStandsHigherUpASlopeAndLevelIsAsItWas(t *testing.T) {
	for _, name := range []string{"terrace-groningen", "condominium"} {
		row, err := KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		bare, err := MakeWithComplexity(row, sow(1), "hull")
		if err != nil {
			t.Fatal(err)
		}
		level, err := Grounded(row, func(float64, float64) float64 { return 0 })
		if err != nil {
			t.Fatal(err)
		}
		flat, err := MakeWithComplexity(level, sow(1), "hull")
		if err != nil {
			t.Fatal(err)
		}
		sloped, err := Grounded(row, cities.Terrain(risingGround))
		if err != nil {
			t.Fatal(err)
		}
		hill, err := MakeWithComplexity(sloped, sow(1), "hull")
		if err != nil {
			t.Fatal(err)
		}
		if flat.Bounds() != bare.Bounds() || flat.faces() != bare.faces() {
			t.Errorf("%s on level ground is not as it was", name)
		}
		if hill.Bounds().Max.Y <= bare.Bounds().Max.Y+1 {
			t.Errorf("%s: %v high on a slope and %v on the flat", name,
				hill.Bounds().Max.Y, bare.Bounds().Max.Y)
		}
		if math.Abs(hill.Bounds().Min.Y) > 5 {
			t.Errorf("%s: its foot is %v from the origin's ground", name,
				hill.Bounds().Min.Y)
		}
	}
}
