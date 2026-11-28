package world

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// How far apart, in coarse spacings, two points either side of a tile's
// edge are, and the most the smoothed height may differ across them.
const (
	acrossStep    = 0.001
	acrossAllowed = 0.01
)

// Every tile's reach is itself and its rings, once each, and begins with
// itself.
func TestEveryTileReachesItsRingsOnce(t *testing.T) {
	world := New(terran)
	for index := range world.tiles {
		list := world.reach[index]
		if list[0] != index {
			t.Fatalf("tile %d's reach begins with %d", index, list[0])
		}
		seen := map[int]bool{}
		for _, other := range list {
			if seen[other] {
				t.Fatalf("tile %d reaches %d twice", index, other)
			}
			seen[other] = true
		}
		if len(list) < 1+len(world.near[index]) {
			t.Fatalf("tile %d reaches %d tiles, fewer than its "+
				"neighbours and itself", index, len(list))
		}
	}
}

// The bump is one at the middle, nought at and past the reach, and never
// rises with the distance.
func TestTheBumpFallsFromOneToNought(t *testing.T) {
	if bump(0) != 1 || bump(1) != 0 || bump(3) != 0 {
		t.Fatalf("bump is %v, %v, %v, not 1, 0, 0", bump(0), bump(1), bump(3))
	}
	last := 1.0
	for step := 1; step <= 100; step++ {
		now := bump(float64(step) / 100)
		if now > last {
			t.Fatalf("the bump rises to %v from %v at %d", now, last, step)
		}
		last = now
	}
}

// At a tile's middle the tile is the nearest, and the smoothed height is
// a mean of the tiles round it: no higher than the highest and no lower
// than the lowest, and nearer the tile's own than the far side of the
// range.
func TestSmoothedAtAMiddleIsAMeanOfItsReach(t *testing.T) {
	world := New(terran)
	for index := 0; index < len(world.tiles); index += 7 {
		ground := world.Smoothed(world.middle[index])
		if ground.Nearest != index {
			t.Fatalf("at tile %d's middle the nearest is %d", index,
				ground.Nearest)
		}
		low, high := math.Inf(1), math.Inf(-1)
		for _, other := range world.reach[index] {
			low = math.Min(low, world.height[other])
			high = math.Max(high, world.height[other])
		}
		if ground.Height < low-1e-9 || ground.Height > high+1e-9 {
			t.Fatalf("tile %d: smoothed %v is outside its reach's %v to %v",
				index, ground.Height, low, high)
		}
	}
}

// Anti-test: no tile's edge shows in the smoothed height. Two points a
// hair either side of the edge between two tiles have heights a hair
// apart, as the height does within a tile.
func TestSmoothedHasNoStepAtATilesEdge(t *testing.T) {
	world := New(terran)
	hair := acrossStep * world.Tiling.Spacing()
	for index := 0; index < len(world.tiles); index += 3 {
		for _, other := range world.near[index] {
			a, b := world.middle[index], world.middle[other]
			edge := a.Add(b).Unit()
			along := b.Sub(a).Unit().Scale(hair)
			before := world.Smoothed(edge.Sub(along)).Height
			after := world.Smoothed(edge.Add(along)).Height
			if math.Abs(before-after) > acrossAllowed {
				t.Fatalf("height steps from %v to %v across the edge "+
					"of tiles %d and %d", before, after, index, other)
			}
		}
	}
}

// Beyond the world's edge there is nothing to smooth: the deep.
func TestSmoothedPastTheEdgeIsTheDeep(t *testing.T) {
	world := New(Terrain{Tiling: hexes.Flat{Step: 1, Across: 1, Down: 1},
		Seed: 3, Ocean: 0.5})
	ground := world.Smoothed(maths.V(0, 0, 1))
	if ground.Nearest != -1 || ground.Height != -1 {
		t.Errorf("past the edge: %+v", ground)
	}
}
