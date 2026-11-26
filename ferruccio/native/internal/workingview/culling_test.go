package workingview

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
)

func TestFrustumKeepsIntersectingSpheresAndRejectsClearMisses(t *testing.T) {
	view := viewFrustum(&placement.DisplayScene{Width: 100, Height: 100, Lens: film.Pinhole{Wide: math.Pi / 2}}, 100, 100)
	bounds := sphere{radius: 1}
	for _, test := range []struct {
		name string
		at   maths.Vec
		want bool
	}{
		{"centre", maths.V(0, 0, -5), true},
		{"crosses side", maths.V(6, 0, -5), true},
		{"outside side", maths.V(8, 0, -5), false},
		{"behind", maths.V(0, 0, 2), false},
		{"crosses near", maths.V(0, 0, -.01), true},
		{"outside far", maths.V(0, 0, -100002), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := view.visible(bounds, maths.Subspace{Origin: test.at}); got != test.want {
				t.Fatalf("visible = %v, want %v", got, test.want)
			}
		})
	}
}
