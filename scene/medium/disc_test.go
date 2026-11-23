package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// discDensityFile is the table of points the disc's density is read at, each
// with the density wanted there.
const discDensityFile = "testdata/disc-density.csv"

// discPlain is the disc of these tests with no profile, no Dense and no
// grain, read from the chosen numbers.
func discPlain(t *testing.T) Disc {
	t.Helper()
	return Disc{Inner: picked(t, "disc-inner"),
		Outer: picked(t, "disc-outer"),
		Thick: picked(t, "disc-thick")}
}

// discRings is the plain disc with a profile that has a gap in the middle.
func discRings(t *testing.T) Disc {
	t.Helper()
	rings := discPlain(t)
	rings.Profile = []float64{unity, unity, nought, unity, unity}
	return rings
}

// discDownward is a ray straight down at a distance out along x, from a
// height.
func discDownward(out, height float64) maths.Ray {
	return maths.Ray{At: maths.V(out, height, nought),
		Dir: maths.V(nought, -unity, nought)}
}

// discDensityRows holds the disc's density to the table of points: zero
// inside its hole, above its faces and beyond its edge, its profile's density
// in the ring and zero in the gap.
func discDensityRows(t *testing.T, rings Disc) {
	t.Helper()
	tolerance := picked(t, "disc-tolerance")
	for _, row := range readSheet(t, discDensityFile) {
		at := maths.V(cell(t, row, "x"), cell(t, row, "y"), cell(t, row, "z"))
		want := cell(t, row, "want")
		if got := rings.Density(at); !(math.Abs(got-want) <= tolerance) {
			t.Errorf("at %v: %v, want %v", at, got, want)
		}
	}
}

// A disc is a medium with a span: zero inside its hole, above its faces and
// beyond its edge; its profile's density in the ring; zero in the gap; and a
// ray straight down through the ring crosses its slab.
func TestDisc(t *testing.T) {
	var _ Spanner = Disc{}
	rings := discRings(t)
	tolerance := picked(t, "disc-tolerance")
	discDensityRows(t, rings)
	downOut := picked(t, "disc-down-out")
	downHeight := picked(t, "disc-down-height")
	down := discDownward(downOut, downHeight)
	enter, leave, ok := rings.Span(down, nought, math.Inf(unity))
	if !ok || !(math.Abs(enter-picked(t, "disc-down-enter")) <= tolerance) ||
		!(math.Abs(leave-picked(t, "disc-down-leave")) <= tolerance) {
		t.Errorf("down through the ring spans %v to %v, %v", enter, leave, ok)
	}
	miss := discDownward(picked(t, "disc-miss-out"), downHeight)
	if _, _, ok := rings.Span(miss, nought, math.Inf(unity)); ok {
		t.Error("a ray beyond the edge spans the disc")
	}
	alongStart := maths.V(picked(t, "disc-along-start"), nought, nought)
	along := maths.Ray{At: alongStart, Dir: maths.V(unity, nought, nought)}
	enter, leave, ok = rings.Span(along, nought, math.Inf(unity))
	if !ok || !(math.Abs(enter-picked(t, "disc-along-enter")) <= tolerance) ||
		!(math.Abs(leave-picked(t, "disc-along-leave")) <= tolerance) {
		t.Errorf("along the disc spans %v to %v, %v", enter, leave, ok)
	}
}

// Ringlets thin the disc from place to place and never thicken it, and a
// disc with no profile is one everywhere.
func TestRinglets(t *testing.T) {
	plain := discPlain(t)
	dense := discPlain(t)
	dense.Dense = picked(t, "disc-dense")
	middle := maths.V(picked(t, "disc-dense-radius"), nought, nought)
	if got := dense.Density(middle); got != dense.Dense {
		t.Fatalf("dense is %v", got)
	}
	grainy := plain
	grainy.Grain = picked(t, "disc-grain")
	grainy.Seed = uint64(pickedCount(t, "disc-grain-seed"))
	seen := map[float64]bool{}
	scale := picked(t, "disc-ringlet-scale")
	end, step := picked(t, "disc-ringlet-end"), picked(t, "disc-ringlet-step")
	for radius := picked(t, "disc-ringlet-start"); radius < end; radius += step {
		point := maths.V(radius, nought, nought)
		if plain.Density(point) != unity {
			t.Fatalf("a plain disc is %v at %v", plain.Density(point), radius)
		}
		grain := grainy.Density(point)
		if grain > unity || grain < nought {
			t.Fatalf("ringlets gave %v at %v", grain, radius)
		}
		seen[math.Round(grain*scale)] = true
	}
	if len(seen) < pickedCount(t, "disc-ringlet-values-least") {
		t.Errorf("the ringlets hardly vary: %d values", len(seen))
	}
}
