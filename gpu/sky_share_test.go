//go:build draw && darwin

package gpu

import (
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

// the sky fixture's picture and its tolerance.
const (
	// skyWidth and skyHeight are the fixture's picture. Picked
	// 2026-11-12: small, and enough pixels across the ball's shade.
	skyWidth, skyHeight = 96, 64
	// skyBudget is the most the shared picture's light may differ from
	// the exact one's at four samples a pixel, summed over the picture,
	// as a share of the exact picture's light. Picked 2026-11-12 as the
	// lamp store's own two percent.
	skyBudget = reachBudget
)

// underASky is the smoke under its lamps with a ball over the floor and
// a painted sky over all, so the floor's light is the sky's and the
// ball and the floor's own edge take some of it away.
func underASky(t *testing.T) (*thing.World, maths.Subspace) {
	t.Helper()
	world, eye := smokeUnderLamps(t)
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "ball", Colour: maths.Grey(0.5)},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(3, 1.2, 2),
			Scale: 1, Orientation: maths.Identity})})
	world.Sky = sky.Gradient{Horizon: maths.RGB(0.5, 0.5, 0.6),
		Zenith: maths.RGB(0.3, 0.4, 0.8)}
	return world, eye
}

// drawnAt is the fixture drawn by a per-call function at so many
// samples along a pixel's side.
func drawnAt(t *testing.T, world *thing.World, eye maths.Subspace, side int,
	by func(*thing.World, maths.Subspace, film.Sensor, time.Duration, int,
		int, int) (*frame.Frame, Profile, error)) (*frame.Frame, Profile) {
	t.Helper()
	picture, spent, err := by(world, eye,
		film.Pinhole{Wide: 50 * maths.Degree}, 0, skyWidth, skyHeight, side)
	mustDevice(t, err)
	return picture, spent
}

// TestSharedSkyLooksStayNearTheExactPicture draws a ball's shade under
// a sky with the looks shared, at two and four samples along a pixel's
// side, the counts that ship. Each differs from the store's own picture,
// so the looks were shared, and is within the budget of it; and the
// profile says which picture shared them.
func TestSharedSkyLooksStayNearTheExactPicture(t *testing.T) {
	world, eye := underASky(t)
	for _, side := range []int{2, 4} {
		whole, wholeSpent := drawnAt(t, world, eye, side, SampledFast)
		shared, sharedSpent := drawnAt(t, world, eye, side, SampledFaster)
		summed, worst := apart(t, whole, shared)
		t.Logf("%d samples a side: summed-relative %.9g, worst/peak %.9g",
			side, summed, worst)
		if summed == 0 {
			t.Fatal("shared looks drew the whole sky's picture bit for bit")
		}
		if summed > skyBudget {
			t.Fatalf("%d samples a side: shared looks are %.6f of the "+
				"picture's light from all of them, budget %v", side,
				summed, skyBudget)
		}
		if wholeSpent.SkyShared || !sharedSpent.SkyShared {
			t.Fatalf("the profiles say the sky was shared: %v and %v",
				wholeSpent.SkyShared, sharedSpent.SkyShared)
		}
	}
}

// TestSharedSkyLooksChangeNothingWhereThereIsNothingToShare is the near
// misses: at one sample a pixel a sample takes all sixteen looks, and
// with no sky there is no look to take; both are the store's picture
// bit for bit.
func TestSharedSkyLooksChangeNothingWhereThereIsNothingToShare(t *testing.T) {
	world, eye := underASky(t)
	whole, _ := drawnAt(t, world, eye, 1, SampledFast)
	shared, spent := drawnAt(t, world, eye, 1, SampledFaster)
	sameBits(t, "one sample a pixel", whole, shared)
	if spent.SkyShared {
		t.Error("one sample a pixel says it shared the sky")
	}
	world, eye = smokeUnderLamps(t)
	whole, _ = drawnAt(t, world, eye, 2, SampledFast)
	shared, _ = drawnAt(t, world, eye, 2, SampledFaster)
	sameBits(t, "no sky", whole, shared)
}
