package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/texture"
)

// the numbers the plate tests hold the rows to
const (
	// seedsTried is how many seeds a test makes each kind from. picked:
	// enough for a lean to be drawn from across its span.
	seedsTried = 40 // picked: see above
	// thinnest is the most a plate's thickness may be of its length.
	// defined: the rows' widest middle share times their thickest short
	// one, the chip's.
	thinnest = 0.9 * 0.5 // defined: the chip row's middle and short
	// someLean is a lean, in degrees, that a few of forty slabs reach.
	// picked: well clear of lying flat.
	someLean = 5 // picked: see above
	// greyest is the most a plate's red may be of its blue and still be
	// grey brown and not orange. picked: half as much again.
	greyest = 1.5 // picked: see above
	// darkest is the most of any light a plate gives back. picked: the
	// ground the landers saw gave back under a tenth.
	darkest = 0.1 // picked: see above
)

// leanOf is how far a made thing's first part leans from standing up, in
// degrees.
func leanOf(made good) float64 {
	up := made.parts[0].space.Orientation.Out(maths.YAxis)
	return math.Acos(math.Min(1, up.Y)) / maths.Degree
}

// a slab and a chip are plates: far thinner than they are long.
func TestAPlateIsThinnerThanItIsLong(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"slab", "chip"} {
		for number := range uint64(seedsTried) {
			stone := mustMake(t, name, number).parts[0].shape.(lump)
			if inScene(stone.short) > thinnest*inScene(stone.long) {
				t.Errorf("%s from seed %d is %v thick and %v long",
					name, number, stone.short, stone.long)
			}
		}
	}
}

// a slab leans no further than its row allows, and some slabs lean.
func TestASlabLeansWithinItsRow(t *testing.T) {
	t.Parallel()
	most, row := 0.0, plateRows()["slab"]
	for number := range uint64(seedsTried) {
		lean := leanOf(mustMake(t, "slab", number))
		if lean > row.lean.most+1e-9 { // picked: a rounding's slack
			t.Errorf("slab from seed %d leans %v degrees, past %v",
				number, lean, row.lean.most)
		}
		most = math.Max(most, lean)
	}
	if most < someLean {
		t.Errorf("forty slabs lean at most %v degrees", most)
	}
}

// a plate breaks along upright joints: every cleaving plane of a slab
// faces level.
func TestAPlateBreaksAlongUprightJoints(t *testing.T) {
	t.Parallel()
	for number := range uint64(seedsTried) {
		stone := mustMake(t, "slab", number).parts[0].shape.(lump)
		planes := stone.cleaves()
		if len(planes) == 0 {
			t.Fatalf("slab from seed %d is not broken", number)
		}
		for _, cleft := range planes {
			if math.Abs(cleft.facing.Y) > 1e-12 { // picked: a rounding
				t.Errorf("slab from seed %d breaks along %v", number,
					cleft.facing)
			}
		}
	}
}

// anti: a cobble is broken as every stone was before plates: its planes
// face any way, and its breaking is the stone's own numbers.
func TestACobbleBreaksAsAStone(t *testing.T) {
	t.Parallel()
	stone := mustMake(t, "cobble", 3).parts[0].shape.(lump) // picked: any
	if stone.broken != stoneBreaking() || stone.broken.upright {
		t.Errorf("a cobble is broken as %+v", stone.broken)
	}
}

// TestASlabsDustFollowsItsRow holds a slab's dust to its row. A row with
// no dust gives a slab of plain rock. A row with dust gives a share of it
// no more than the row allows, darker than the rock. The rows have no
// dust since 2026-11-15, when the patches drew as even spots.
func TestASlabsDustFollowsItsRow(t *testing.T) {
	t.Parallel()
	row := plateRows()["slab"]
	for number := range uint64(seedsTried) {
		look := mustMake(t, "slab", number).dressed[body]
		dusted, ok := look.Texture.(texture.Dusted)
		if row.dust.most == 0 {
			if ok {
				t.Fatalf("slab from seed %d is dusted %+v and its row gives none",
					number, dusted)
			}
			continue
		}
		if !ok {
			t.Fatalf("slab from seed %d has no dust", number)
		}
		if dusted.Share < row.dust.least || dusted.Share > row.dust.most ||
			dusted.Dust.G >= dusted.Rock.G {
			t.Errorf("slab from seed %d is dusted %+v", number, dusted)
		}
	}
}

// anti: a cobble is not dusted: its look is its row's granite, plain.
func TestACobbleIsNotDusted(t *testing.T) {
	t.Parallel()
	look := mustMake(t, "cobble", 3).dressed[body] // picked: any seed
	if look.Texture != nil || look.Name != "granite" {
		t.Errorf("a cobble is dressed %+v", look)
	}
}

// a slab is sunk into the ground by no more of its thickness than its row
// allows, and never lifted off it.
func TestASlabIsSunkWithinItsRow(t *testing.T) {
	t.Parallel()
	row := plateRows()["slab"]
	for number := range uint64(seedsTried) {
		made := mustMake(t, "slab", number)
		thick := inScene(made.parts[0].shape.(lump).short)
		depth := -made.parts[0].space.Origin.Y
		if depth < 0 || depth > row.sunk.most*thick+1e-9 { // picked: slack
			t.Errorf("slab from seed %d is sunk %v of %v", number, depth,
				thick)
		}
	}
}

// anti: a pebble, a cobble and a boulder are neither leaned nor sunk; the
// plates' lean leaves the rounder stones as they were.
func TestARoundStoneDoesNotLean(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"pebble", "cobble", "boulder"} {
		made := mustMake(t, name, 7) // picked: any seed
		if lean := leanOf(made); lean != 0 {
			t.Errorf("%s leans %v degrees", name, lean)
		}
		if at := made.parts[0].space.Origin; at != (maths.Vec{}) {
			t.Errorf("%s is moved to %v", name, at)
		}
	}
}

// anti: a plate is dark and grey brown, never the orange the sky's light
// makes of it.
func TestAPlateIsDarkAndNotOrange(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"slab", "chip"} {
		for _, look := range mustMake(t, name, 1).dressed {
			colour := look.Colour
			if math.Max(colour.R, math.Max(colour.G, colour.B)) > darkest {
				t.Errorf("%s's %s gives back %v", name, look.Name, colour)
			}
			if colour.R > greyest*colour.B {
				t.Errorf("%s's %s is orange: %v", name, look.Name, colour)
			}
		}
	}
}
