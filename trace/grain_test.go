package trace

import (
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// grainWide is a ray's angular width the grain tests are given: a
// thousandth of a radian, picked.
const grainWide = 1e-3

// grainView is a view of an empty world 8 by 6 pixels at rays a side.
func grainView(t *testing.T, rays int) view {
	t.Helper()
	seen, err := newView(&thing.World{}, maths.Subspace{Scale: 1},
		film.Pinhole{}, 0, 8, 6, rays)
	if err != nil {
		t.Fatal(err)
	}
	return seen
}

// a fine view makes every ray a pixel and keeps the rays a pixel's side
// had, so a wrinkle fades by the finished pixel: four rays a side, four
// rays wide.
func TestAFineViewFadesByTheFinishedPixel(t *testing.T) {
	fine := grainView(t, 4).fine()
	if fine.rays != 1 || fine.width != 32 || fine.side != 4 {
		t.Fatalf("a fine view of 8 pixels at four rays is %d wide at %d "+
			"rays, side %d; want 32, 1 and 4", fine.width, fine.rays,
			fine.side)
	}
	if got, want := fine.grain(grainWide), 4*grainWide; got != want {
		t.Errorf("the grain at four rays a side is %v; want %v", got, want)
	}
}

// at one ray a pixel the grain is the ray's own width, as it always was.
func TestOneRayAPixelFadesByItsOwnWidth(t *testing.T) {
	if got := grainView(t, 1).fine().grain(grainWide); got != grainWide {
		t.Errorf("the grain at one ray a side is %v; want %v", got,
			grainWide)
	}
}
