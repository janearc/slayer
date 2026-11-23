package medium

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// axes are the six directions along the axes: a point on one of them has
// a length that is exactly its coordinate, and a direction that is exactly
// the axis, so a patch of cloud is the same at every height along it.
var axes = []maths.Vec{
	maths.V(unity, nought, nought), maths.V(-unity, nought, nought),
	maths.V(nought, unity, nought), maths.V(nought, -unity, nought),
	maths.V(nought, nought, unity), maths.V(nought, nought, -unity),
}

// mutantsCloudsLayer is a layer of cloud over the body the tests chose, from
// low to high above it, with the cover and the seed it is asked for.
func mutantsCloudsLayer(
	t *testing.T, low, high, cover float64, seed uint64,
) Clouds {
	t.Helper()
	return Clouds{
		Radius: picked(t, "mutants-clouds-radius"),
		Low:    low,
		High:   high,
		Cover:  cover,
		Thick:  picked(t, "mutants-clouds-thick"),
		Seed:   seed,
	}
}

// a layer of cloud is whole to a third of the way up, from its very bottom
// at Low, and thins to 0.15 of that at its very top, High, where it stops.
//
// along an axis the patch is one patch at every height, so for a patch
// that is there at 0.2 up, the bottom is the same and the top is 0.15 of
// it. the layer is from 0.5 to 1.5 above a body of radius 10, and the
// cover is one, so that most of the sky is patch.
func TestCloudsAreWholeAtTheBottomAndThinAtTheTop(t *testing.T) {
	seen := nought
	low := picked(t, "mutants-clouds-low")
	high := picked(t, "mutants-clouds-high")
	cover := picked(t, "mutants-clouds-full-cover")
	wholeHeight := picked(t, "mutants-clouds-whole-height")
	bottomHeight := picked(t, "mutants-clouds-bottom-height")
	topHeight := picked(t, "mutants-clouds-top-height")
	thinShare := picked(t, "mutants-clouds-thin-share")
	tolerance := picked(t, "mutants-clouds-tolerance")
	seeds := uint64(pickedCount(t, "mutants-clouds-seeds"))
	for seed := range seeds {
		cloudy := mutantsCloudsLayer(t, low, high, cover, seed)
		for _, axis := range axes {
			whole := cloudy.Density(axis.Scale(wholeHeight))
			if whole == nought {
				continue
			}
			seen++
			bottom := cloudy.Density(axis.Scale(bottomHeight))
			if !closeTo(bottom, whole, tolerance) {
				t.Errorf("seed %d, %v: the bottom is %v, a fifth up %v", seed,
					axis, bottom, whole)
			}
			top := cloudy.Density(axis.Scale(topHeight))
			if !closeTo(top, thinShare*whole, tolerance) {
				t.Errorf("seed %d, %v: the top is %v, want %v", seed, axis, top,
					thinShare*whole)
			}
		}
	}
	if seen < pickedCount(t, "mutants-clouds-patches-least") {
		t.Fatalf("only %d patches in the sky", seen)
	}
}

// a cover of zero clouds none of the sky, and a layer of no thickness holds
// none, at the very height where its ratio would be zero over zero.
func TestCloudsOfNoCoverOrNoLayerAreNone(t *testing.T) {
	low := picked(t, "mutants-clouds-low")
	high := picked(t, "mutants-clouds-high")
	bareSeed := uint64(pickedCount(t, "mutants-clouds-bare-seed"))
	bare := mutantsCloudsLayer(t, low, high, nought, bareSeed)
	samples := pickedCount(t, "mutants-clouds-samples")
	bareHeight := picked(t, "mutants-clouds-bare-height")
	for place := range samples {
		at := scatter(t, place).Scale(bareHeight)
		if density := bare.Density(at); density != nought {
			t.Fatalf("a cover of nought has %v at %v", density, at)
		}
	}
	thinEdge := picked(t, "mutants-clouds-thin-edge")
	cover := picked(t, "mutants-clouds-full-cover")
	thinHeight := picked(t, "mutants-clouds-thin-height")
	seeds := uint64(pickedCount(t, "mutants-clouds-seeds"))
	for seed := range seeds {
		thin := mutantsCloudsLayer(t, thinEdge, thinEdge, cover, seed)
		for _, axis := range axes {
			density := thin.Density(axis.Scale(thinHeight))
			if density != nought {
				t.Fatalf("seed %d: a layer of no thickness has %v", seed,
					density)
			}
		}
	}
}
