package medium

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// axes are the six directions along the axes: a point on one of them has
// a length that is exactly its coordinate, and a direction that is exactly
// the axis, so a patch of cloud is the same at every height along it.
var axes = []maths.Vec{maths.V(1, 0, 0), maths.V(-1, 0, 0), maths.V(0, 1, 0),
	maths.V(0, -1, 0), maths.V(0, 0, 1), maths.V(0, 0, -1)}

// a layer of cloud is whole to a third of the way up, from its very bottom
// at Low, and thins to 0.15 of that at its very top, High, where it stops.
//
// along an axis the patch is one patch at every height, so for a patch
// that is there at 0.2 up, the bottom is the same and the top is 0.15 of
// it. the layer is from 0.5 to 1.5 above a body of radius 10, and the
// cover is one, so that most of the sky is patch.
func TestCloudsAreWholeAtTheBottomAndThinAtTheTop(t *testing.T) {
	seen := 0
	for seed := range uint64(60) {
		c := Clouds{Radius: 10, Low: 0.5, High: 1.5, Cover: 1, Thick: 1, Seed: seed}
		for _, axis := range axes {
			whole := c.Density(axis.Scale(10.7))
			if whole == 0 {
				continue
			}
			seen++
			if low := c.Density(axis.Scale(10.5)); !closeTo(low, whole, 1e-12) {
				t.Errorf("seed %d, %v: the bottom is %v, a fifth up %v", seed,
					axis, low, whole)
			}
			if top := c.Density(axis.Scale(11.5)); !closeTo(top, 0.15*whole, 1e-12) {
				t.Errorf("seed %d, %v: the top is %v, want %v", seed, axis, top,
					0.15*whole)
			}
		}
	}
	if seen < 50 {
		t.Fatalf("only %d patches in the sky", seen)
	}
}

// a cover of zero clouds none of the sky, and a layer of no thickness holds
// none, at the very height where its ratio would be zero over zero.
func TestCloudsOfNoCoverOrNoLayerAreNone(t *testing.T) {
	bare := Clouds{Radius: 10, Low: 0.5, High: 1.5, Cover: 0, Thick: 1, Seed: 10}
	for i := range 400 {
		at := scatter(i).Scale(10.8)
		if d := bare.Density(at); d != 0 {
			t.Fatalf("a cover of nought has %v at %v", d, at)
		}
	}
	for seed := range uint64(60) {
		thin := Clouds{Radius: 10, Low: 1, High: 1, Cover: 1, Thick: 1, Seed: seed}
		for _, axis := range axes {
			if d := thin.Density(axis.Scale(11)); d != 0 {
				t.Fatalf("seed %d: a layer of no thickness has %v", seed, d)
			}
		}
	}
}
