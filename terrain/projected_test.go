package terrain

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/footprint"
)

// The synthetic source spans 64 by 64 units facing the camera. Its supplied
// representation error is at most one eighth of the patch width. This is an
// authored bound, independent of DEM sampling or production overview data.
func projectedGeometry(p Patch) (PatchGeometry, error) {
	return PatchGeometry{Bounds: maths.Box{
		Min: maths.V(float64(p.MinColumn)-32, float64(p.MinRow)-32, 0),
		Max: maths.V(float64(p.MaxColumn)-32, float64(p.MaxRow)-32, 0)},
		Error: measure.Metres(float64(p.MaxColumn-p.MinColumn) / 8)}, nil
}

func projectedSelection(t *testing.T, depth, field float64,
	patches int,
) Selection {
	t.Helper()
	view, err := footprint.New(maths.Subspace{Origin: maths.V(0, 0, depth)},
		measure.Radians(field*math.Pi/180), 320, 180, 1, measure.Metres(0.1))
	if err != nil {
		t.Fatal(err)
	}
	got, err := SelectProjectedPatches(Patch{MaxColumn: 64, MaxRow: 64},
		SelectionBudget{Patches: patches, Evaluations: 1024,
			Depth: 6, Target: 4}, view, projectedGeometry)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestProjectedTerrainChangesDetailWithoutLosingCoverage(t *testing.T) {
	near := projectedSelection(t, 100, 60, 256)
	far := projectedSelection(t, 1000, 60, 256)
	narrow := projectedSelection(t, 1000, 10, 256)
	if len(far.Patches) != 1 || len(near.Patches) <= 1 ||
		len(narrow.Patches) <= len(far.Patches) {
		t.Fatalf("detail counts near=%d far=%d narrow=%d",
			len(near.Patches), len(far.Patches), len(narrow.Patches))
	}
	for _, selected := range []Selection{near, far, narrow} {
		area := 0
		for _, p := range selected.Patches {
			area += (p.MaxColumn - p.MinColumn) * (p.MaxRow - p.MinRow)
		}
		if area != 4096 || selected.Limited {
			t.Fatalf("source coverage area=%d limited=%v", area, selected.Limited)
		}
	}
}

func TestProjectedTerrainKeepsCoarseParentWhenBudgetIsFull(t *testing.T) {
	got := projectedSelection(t, 100, 60, 1)
	if !got.Limited || len(got.Patches) != 1 ||
		got.Patches[0] != (Patch{MaxColumn: 64, MaxRow: 64}) {
		t.Fatalf("coarse source coverage lost: %+v", got)
	}
}

func TestProjectedTerrainPropagatesBoundsFailure(t *testing.T) {
	want := errors.New("source bounds unavailable")
	_, err := SelectProjectedPatches(Patch{MaxColumn: 64, MaxRow: 64},
		SelectionBudget{Patches: 1, Evaluations: 1, Depth: 1, Target: 1},
		footprint.Perspective{}, func(Patch) (PatchGeometry, error) {
			return PatchGeometry{}, want
		})
	if !errors.Is(err, want) {
		t.Fatalf("bounds failure: %v", err)
	}
}

func TestProjectedTerrainReportsNearPlaneUncertainty(t *testing.T) {
	view, err := footprint.New(maths.Subspace{}, measure.Radians(math.Pi/3),
		320, 180, 1, measure.Metres(0.1))
	if err != nil {
		t.Fatal(err)
	}
	got, err := SelectProjectedPatches(Patch{MaxColumn: 64, MaxRow: 64},
		SelectionBudget{Patches: 1, Evaluations: 1, Depth: 1,
			Target: math.MaxFloat64}, view, projectedGeometry)
	if err != nil || !got.Limited || len(got.Patches) != 1 {
		t.Fatalf("near-plane uncertainty: %+v, %v", got, err)
	}
}
