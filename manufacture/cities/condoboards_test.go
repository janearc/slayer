package cities

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestATelescopeStandsInsideItsRoofAndOnlyWhereItsDomeFits(t *testing.T) {
	mounted := 0
	for seed := range Seed(seeds) {
		drawing := homeDrawer(t, seed)
		radius := drawing.fixed("telescope-diameter") *
			drawing.fixed("telescope-scale") / half
		for bay := range 4 {
			for _, depth := range []float64{3, 8, 11} {
				area := roofArea{x0: 0.2, z0: 0.2, x1: 28, z1: depth, y: 14}
				mounts, taken := drawing.telescope(place{Bay: bay}, area)
				for _, mount := range mounts {
					mounted++
					at := mount.Place.Origin
					if mount.Kind != telescopeKind ||
						at.X-radius < area.x0-tolerance ||
						at.X+radius > area.x1+tolerance ||
						at.Z-radius < area.z0-tolerance ||
						at.Z+radius > area.z1+tolerance ||
						at.Y != area.y || !(taken >= 2*radius) {
						t.Errorf("seed %d bay %d: a telescope at %v, taking "+
							"%v of a roof %v deep", seed, bay, at, taken, depth)
					}
				}
				if len(mounts) == 0 && taken != 0 {
					t.Errorf("seed %d: room taken for no telescope", seed)
				}
			}
		}
	}
	if mounted == 0 {
		t.Error("no telescope on any roof")
	}
}

func TestADesignCarriesItsTelescopesAndTheyMoveWithIt(t *testing.T) {
	district := Seed(drawerOf(t, condominiumCulture, 1).district)
	found := false
	for index := range Seed(seeds) {
		design, err := CondominiumAt(index*district, nil, demoLevel)
		if err != nil {
			t.Fatal(err)
		}
		for _, mount := range design.Mounts {
			t.Logf("seed %d keeps a telescope at %v", index*district,
				mount.Place.Origin)
		}
		if len(design.Mounts) == 0 || found {
			continue
		}
		found = true
		by := maths.V(100, 0, -50)
		moved := design.at(standingAt(by))
		if len(moved.Mounts) != len(design.Mounts) ||
			moved.Mounts[0].Place.Origin != design.Mounts[0].Place.Origin.Add(
				by) {
			t.Errorf("seed %d: the telescopes did not move with the design",
				index*district)
		}
	}
	if !found {
		t.Error("no condominium keeps a telescope")
	}
}

func TestACondominiumBelowTheDemoLevelKeepsNoTelescope(t *testing.T) {
	for index := range Seed(homeSeeds) {
		design, err := Condominium(index)
		if err != nil {
			t.Fatal(err)
		}
		if len(design.Mounts) != 0 {
			t.Errorf("seed %d: %d telescopes at the quick level", index,
				len(design.Mounts))
		}
	}
}

func TestARoofBoardStandsInsideItsRoofOnItsPosts(t *testing.T) {
	area := roofArea{x0: 0.2, z0: 0.2, x1: 28, z1: 11, y: 14}
	boards := 0
	for seed := range Seed(seeds) {
		drawing := homeDrawer(t, seed)
		for bay := range 4 {
			for _, part := range drawing.roofBoard(place{Bay: bay}, area) {
				low, high := reachOf(t, part)
				boards++
				if !insideArea(low, high, area) {
					t.Errorf("seed %d: a roof board at %v to %v is off the "+
						"roof", seed, low, high)
				}
			}
		}
	}
	if boards == 0 {
		t.Error("no roof board in any seed")
	}
}

func TestAWallBoardCoversOnlyTheBlankEndOfTheLowTier(t *testing.T) {
	shape := massShape{width: 28, depth: 11.5}
	boards := 0
	for seed := range Seed(seeds) {
		drawing := homeDrawer(t, seed)
		rise := drawing.fixed("wall-board-rise")
		for bay := range 4 {
			for _, left := range []bool{true, false} {
				for _, part := range drawing.wallBoard(place{Bay: bay}, shape,
					12, left) {
					low, high := reachOf(t, part)
					boards++
					outside := high.X <= tolerance || low.X >= shape.width-
						tolerance
					if !outside || low.Z < -tolerance ||
						high.Z > shape.depth+tolerance || low.Y < rise-
						tolerance || high.Y > 12+tolerance {
						t.Errorf("seed %d: a wall board at %v to %v is not "+
							"on the end wall", seed, low, high)
					}
				}
			}
		}
	}
	if boards == 0 {
		t.Error("no wall board in any seed")
	}
	if got := homeDrawer(t, 1).wallBoard(place{}, shape, 5, true); len(got) != 0 {
		t.Errorf("a board on a wall too low for it: %d parts", len(got))
	}
}
