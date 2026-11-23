package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// ballUnderAPane is the light of a look straight down, 0.9 out from the
// middle of a black glossy ball, under a sky of one everywhere; with a
// pane of all but air's own index across the look three up, or without.
//
// The look meets the ball where its normal is 0.9, 0.436, 0, at a
// cosine of 0.436, and is mirrored down and outward, along 0.785,
// -0.62, 0: to the sky, and never back to the pane.
func ballUnderAPane(pane bool) float64 {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Gloss: 1}})
	if pane {
		planeAt := resting(maths.V(0, 3, 0))
		world.Add(thing.Thing{Shape: shape.Plane{},
			Material: &thing.Material{Through: 1, Index: 1.0001},
			Motion:   planeAt})
	}
	world.Sky = even(maths.Grey(1))
	look := maths.Ray{At: maths.V(0.9, 5, 0), Dir: maths.V(0, -1, 0)}
	return bright(lone(world, look, 0))
}

// A pane that bends nothing changes nothing behind it. A black glossy
// ball mirrors the sky by fresnel's share at the look's cosine, some
// nine in a hundred of a sky of one here, and seen through a pane whose
// index is air's to a ten-thousandth it mirrors the same, to a few
// parts in a hundred.
//
// A look that has bent through a clear surface still follows the mirror
// of a surface that is not clear, and is given the sky for it: the
// ball's mirrored share is kept. Dropped, the ball would read black.
func TestAMirrorBallSeenThroughAPaneStillMirrors(t *testing.T) {
	bare, through := ballUnderAPane(false), ballUnderAPane(true)
	if !(bare > 0.05) {
		t.Fatalf("the bare ball mirrors %v of the sky", bare)
	}
	if !within(through, bare, 0.05*bare) {
		t.Errorf("through a pane the ball reads %v, and bare %v", through,
			bare)
	}
}

// Anti: the bare ball mirrors by fresnel's share and no more. Its
// material's own share at the look's cosine, the square root of
// 1 - 0.81, of a sky of one is what the look reads.
func TestABareMirrorBallReadsFresnelsShareOfTheSky(t *testing.T) {
	gloss := &thing.Material{Gloss: 1}
	want := gloss.Mirrors(math.Sqrt(1 - 0.81))
	if got := ballUnderAPane(false); !within(got, want, 1e-6) {
		t.Errorf("the bare ball reads %v, want fresnel's %v", got, want)
	}
}
