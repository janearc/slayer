package manufacture

import (
	"github.com/janearc/celestial/measure"
	"math"
	"testing"
)

func TestFireLayoutAcceptsItsClosedCountAndVariationLimits(t *testing.T) {
	for _, pattern := range []string{"cluster", "line"} {
		for _, count := range []int{2, 32} {
			for _, variation := range []float64{0, .5} {
				spec := burnerFixture()
				spec.Layout = FireLayout{Pattern: pattern, Count: count, Variation: variation}
				if pattern == "cluster" {
					spec.Layout.Radius = measure.Metres(.5)
				} else {
					spec.Layout.Length = measure.Metres(.5)
				}
				if _, err := Burn(spec); err != nil {
					t.Fatalf("%s count%d variation%g: %v", pattern, count, variation, err)
				}
				patches, err := sourcePatches(spec)
				if err != nil || len(patches) != count {
					t.Fatalf("layout lost patches: %v", err)
				}
				for _, patch := range patches {
					if !(patch.Fire.Height >= spec.Flame.Height*.5 && patch.Fire.Height <= spec.Flame.Height*1.5) {
						t.Fatal("height left its authored variation interval")
					}
				}
			}
		}
	}
}

func TestSingleLayoutRefusesIgnoredRadiusLengthAndVariation(t *testing.T) {
	for _, layout := range []FireLayout{
		{Pattern: "single", Radius: measure.Metres(1)},
		{Pattern: "single", Length: measure.Metres(1)},
		{Pattern: "single", Variation: .1},
	} {
		spec := burnerFixture()
		spec.Layout = layout
		if _, err := Burn(spec); err == nil {
			t.Fatal("single source accepted an ignored layout parameter")
		}
	}
	spec := burnerFixture()
	spec.Layout = FireLayout{Pattern: "cluster", Count: 2, Radius: measure.Metres(.5), Variation: -math.SmallestNonzeroFloat64}
	if _, err := Burn(spec); err == nil {
		t.Fatal("negative variation accepted")
	}
}

func TestFireLayoutsKeepTheirFullLineSpanAndAreaBalancedDisk(t *testing.T) {
	for _, count := range []int{2, 3, 32} {
		for _, pattern := range []string{"line", "cluster"} {
			spec := burnerFixture()
			spec.Layout = FireLayout{Pattern: pattern, Count: count}
			if pattern == "line" {
				spec.Layout.Length = measure.Metres(.5)
			} else {
				spec.Layout.Radius = measure.Metres(.5)
			}
			patches, err := sourcePatches(spec)
			if err != nil {
				t.Fatal(err)
			}
			squareSum := 0.0
			for _, patch := range patches {
				if patch.At.Y != 0 {
					t.Fatal("source left its ground plane")
				}
				if patch.Fire.Height != spec.Flame.Height {
					t.Fatal("zero variation changed source height")
				}
				squareSum += patch.At.X*patch.At.X + patch.At.Z*patch.At.Z
			}
			if pattern == "line" {
				if patches[0].At.X != -.25 || patches[count-1].At.X != .25 {
					t.Fatal("line lost its full authored span")
				}
				for i := 1; i < count; i++ {
					if patches[i].At.Z != 0 || !(math.Abs(patches[i].At.X-patches[i-1].At.X-.5/float64(count-1)) <= 1e-12) {
						t.Fatal("line sources are not uniformly spaced")
					}
				}
			} else {
				// An area-balanced disk has mean squared radius R^2/2. This holds
				// the chosen placement rule, not a combustion or fuel distribution.
				if !(math.Abs(squareSum/float64(count)-.125) <= 1e-12) {
					t.Fatal("disk source placement lost area balance")
				}
			}
		}
	}
}
