package session

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/terrain"
)

// shoreSea is the open sea the shore tests lay over a ground: a wind sea
// at time nought, seed 7.
func shoreSea(t *testing.T) *arid.Sea {
	t.Helper()
	sea, err := arid.NewSea(oceantex.Sea{Wind: 7, Fetch: 500000,
		Toward: 1.5708}, 0, 7)
	if err != nil {
		t.Fatal(err)
	}
	return sea
}

// shoreWindow is a window of across by down samples ten metres apart, each
// height from its column and row.
func shoreWindow(t *testing.T, across, down int,
	height func(column, row int) float64) terrain.Window {
	t.Helper()
	const apart = 10.0
	window, err := terrain.BuildWindow(0, 0, across, down, apart,
		func(column, row int) (float64, error) {
			return height(column, row), nil
		})
	if err != nil {
		t.Fatal(err)
	}
	return window
}

// basinShore is a 7 by 7 ground: 5 m deep sea at the edge, a ring of land
// 2 m up inside it, and a hollow 1 m below the sea's level inside that.
func basinShore(column, row int) float64 {
	switch max(max(column-3, 3-column), max(row-3, 3-row)) {
	case 3:
		return -5
	case 2:
		return 2
	}
	return -1
}

// groundedSet is a set with terrain, read from the guide's own file: only
// the fact that it has ground is used.
func groundedSet(t *testing.T) set.Set {
	t.Helper()
	raw, err := os.ReadFile("../doc/observatory-ecosystem.yaml")
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read("shore.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	return read
}

// The ground is put in the sea's frame as the stage says: the ground's first
// sample at the world's origin, its heights less the scene's lowest, and the
// sea's middle at the ocean's Mean. A ray straight down over the hollow
// meets no water, one over the sea at the edge does, and the open sea,
// which stands over both, is the check that the ray is over water at all.
func TestShoredPutsTheGroundInTheSeasFrame(t *testing.T) {
	window := shoreWindow(t, 7, 7, basinShore)
	open := shoreSea(t)
	mean := maths.V(35, 5, 35)
	making := testStaging()
	making.ground, making.lowest = window, window.Lowest()
	making.world.OceanSurface = &thing.OceanSurface{
		Boundary: &thing.Thing{Shape: open}, Mean: mean}
	if err := making.shored(groundedSet(t)); err != nil {
		t.Fatal(err)
	}
	shored, ok := making.world.OceanSurface.Boundary.Shape.(*arid.Sea)
	if !ok || shored == open {
		t.Fatal("the ocean's sea was not put over the ground")
	}
	down := func(east, south float64) maths.Ray {
		return maths.Ray{At: maths.V(east-mean.X, 10, south-mean.Z),
			Dir: maths.V(0, -1, 0)}
	}
	if _, hit := open.Hit(down(30, 30), 0, 100); !hit {
		t.Fatal("the open sea does not stand over the hollow")
	}
	if along, hit := shored.Hit(down(30, 30), 0, 100); hit {
		t.Fatalf("water met %g m down over the hollow at 30, 30", along)
	}
	if _, hit := shored.Hit(down(0, 0), 0, 100); !hit {
		t.Fatal("no water over the sea at the ground's first sample")
	}
}

// diagonalShore is a 5 by 4 ground 1 m up but for two pairs of samples 1 m
// down: one pair along the diagonal the ground's triangles draw, from 0, 0
// to 1, 1, and one along the other, from 4, 0 to 3, 1.
func diagonalShore(column, row int) float64 {
	switch [2]int{column, row} {
	case [2]int{0, 0}, [2]int{1, 1}, [2]int{4, 0}, [2]int{3, 1}:
		return -1
	}
	return 1
}

// The coast joins the samples the ground's mesh joins. The mesh the window
// draws has the edge from 0, 0 to 1, 1 and not the one from 4, 0 to 3, 1,
// and the sea over that window has one hollow cut off from it, the sample
// the undrawn diagonal alone would join.
func TestCoastFollowsTheDiagonalTheGroundIsDrawnOn(t *testing.T) {
	const across, down = 5, 4
	window := shoreWindow(t, across, down, diagonalShore)
	drawn, err := window.GroundOn(0, 0, window.Lowest(), nil)
	if err != nil {
		t.Fatal(err)
	}
	joined := func(first, second int) bool {
		for _, face := range drawn.Faces() {
			if slices.Contains(face[:], first) &&
				slices.Contains(face[:], second) {
				return true
			}
		}
		return false
	}
	if !joined(0, 1*across+1) {
		t.Error("the mesh does not join 0, 0 to 1, 1")
	}
	if joined(4, 1*across+3) {
		t.Error("the mesh joins 4, 0 to 3, 1")
	}
	_, counted, err := shoreSea(t).Ashore(window, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if counted.Hollow != 1 || counted.Land != across*down-4 {
		t.Errorf("the coast counts %d hollows and %d land, want 1 and %d",
			counted.Hollow, counted.Land, across*down-4)
	}
}

func TestTallySaysTheSeaOverGround(t *testing.T) {
	made := tally{sea: "sea over ground: of 4 heights"}
	if said := made.String(); !strings.HasSuffix(said,
		". sea over ground: of 4 heights") {
		t.Fatalf("the tally %q does not end with the sea", said)
	}
}

func TestTallySaysNoSeaWhereThereIsNone(t *testing.T) {
	if said := (tally{}).String(); strings.Contains(said, "sea over ground") {
		t.Fatalf("the tally %q speaks of a sea", said)
	}
}
