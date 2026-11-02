package sky

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// A sky of stars is dark almost everywhere, has stars in about the share
// of places it was asked for, is the same twice, and differs by seed.
func TestStars(t *testing.T) {
	var _ Sky = Stars{}
	a := Stars{Seed: 1, Share: 0.5, Size: 0.3}
	b := Stars{Seed: 2, Share: 0.5, Size: 0.3}
	lit, differ := 0, 0
	const n = 20000
	for k := range int64(n) {
		d := maths.V(noise.Hash(9, k)*2-1, noise.Hash(8, k)*2-1,
			noise.Hash(7, k)*2-1).Unit()
		x := a.Light(d)
		if x != a.Light(d) {
			t.Fatal("a star moved between two asks")
		}
		if x != (maths.Light{}) {
			lit++
			if x.R < 0 || x.G < 0 || x.B < 0 {
				t.Fatalf("a star is negative: %v", x)
			}
		}
		if x != b.Light(d) {
			differ++
		}
	}
	// half the tiles hold a star of radius 0.3 spacings: a few percent of
	// directions land on one
	if lit < n/100 || lit > n/5 {
		t.Errorf("%d of %d directions are lit", lit, n)
	}
	if differ < lit/2 {
		t.Errorf("two seeds agree on %d of %d", n-differ, n)
	}
	dark := Stars{Dark: maths.Grey(0.01), Share: 0.001}
	if got := dark.Light(maths.V(0.3, 0.2, 0.9)); got.R < 0.01 {
		t.Errorf("the dark between stars is %v", got)
	}
}
