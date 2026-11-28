package world

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/maths/strategies/scrooge"
)

// The tree tests' numbers: the levels they walk, how many parents of each
// they ask, how near float32 sums must come to nought, and how many
// children a tile has at least and at most.
const (
	levelsAsked   = 5
	parentsAsked  = 60
	sumAllowed    = 1e-6
	fewestKids    = 1
	mostKids      = 7
	frugalAllowed = 1e-9
)

// parentsAt are tiles of a level spread over the sphere, as many as a
// test asks of it.
func parentsAt(tree *tree, level int) []hexes.Tile {
	tiles := make([]hexes.Tile, 0, parentsAsked)
	for place := range parentsAsked {
		tiles = append(tiles, tree.levels[level].Nearest(
			scatterOn(place, parentsAsked)))
	}
	return tiles
}

// A tile's children are its own: each is a tile of the next level whose
// parent is the tile, and every such tile near it is among them, so the
// tree is a tree and its levels are whole.
func TestATilesChildrenAreAllItsOwn(t *testing.T) {
	tree := New(terran).tree()
	for level := range levelsAsked {
		fine := tree.levels[level+1]
		for _, parent := range parentsAt(tree, level) {
			kids := tree.children(level, parent)
			owned := map[int64]bool{}
			for _, kid := range kids {
				owned[fine.ID(kid)] = true
				if !sameTile(tree, level, tree.parentOf(level+1, kid),
					parent) {
					t.Fatalf("level %d: a child of %v is not its own", level,
						parent)
				}
			}
			if len(kids) < fewestKids || len(kids) > mostKids {
				t.Fatalf("level %d: %v has %d children", level, parent,
					len(kids))
			}
			assertNoneMissed(t, tree, level, parent, owned)
		}
	}
}

// sameTile is whether two tiles of a level are one.
func sameTile(tree *tree, level int, one, other hexes.Tile) bool {
	tiling := tree.levels[level]
	return tiling.ID(one) == tiling.ID(other)
}

// assertNoneMissed fails a test where a tile of the next level within two
// rings of a parent's middle child is the parent's and not among its
// children.
func assertNoneMissed(t *testing.T, tree *tree, level int,
	parent hexes.Tile, owned map[int64]bool) {
	t.Helper()
	fine := tree.levels[level+1]
	middle := fine.Nearest(tree.levels[level].Centre(parent))
	for _, near := range fine.Neighbours(middle) {
		for _, further := range append(fine.Neighbours(near), near) {
			mine := sameTile(tree, level,
				tree.parentOf(level+1, further), parent)
			if mine && !owned[fine.ID(further)] {
				t.Fatalf("level %d: %v's child %v is missed", level, parent,
					further)
			}
		}
	}
}

// A parent is the mean of its children: their details, attribute by
// attribute, sum to nought, so each child's attributes are its parent's
// plus a detail whose mean over them is nought.
func TestAParentIsTheMeanOfItsChildren(t *testing.T) {
	ground := grounded(New(terran))
	tree := ground.tree
	for level := range levelsAsked {
		for _, parent := range parentsAt(tree, level) {
			made := node{ground: ground, level: level, tile: parent}
			own := made.values()
			var mean [attributes]float64
			kids := tree.children(level, parent)
			for _, kid := range kids {
				got := node{ground: ground, level: level + 1,
					tile: kid}.values()
				for which := range attributes {
					mean[which] += got[which] / float64(len(kids))
				}
			}
			for which := range attributes {
				if math.Abs(mean[which]-own[which]) > sumAllowed {
					t.Fatalf("level %d: attribute %d's children average "+
						"%v, the parent is %v", level, which, mean[which],
						own[which])
				}
			}
		}
	}
}

// The tree keeps scrooge's contract at every level, as scrooge.Frugal
// checks it: a node opened by a footprint gives children within its box,
// none finer than the footprint that gives a layer under it, and the
// same light.
func TestTheTreeIsFrugalAtEveryLevel(t *testing.T) {
	ground := grounded(New(terran))
	for level := range levelsAsked {
		for _, tile := range parentsAt(ground.tree, level) {
			made := node{ground: ground, level: level, tile: tile}
			grain := made.Aggregate().Grain
			for _, width := range []float64{grain / 2, grain, grain * 3} {
				at := scrooge.Footprint{From: made.middle(), Width: width}
				if err := scrooge.Frugal(made, at, frugalAllowed); err != nil {
					t.Fatalf("level %d at %v: %v", level, width, err)
				}
			}
		}
	}
}

// Anti-test: nothing is made finer than asked. A node whose children
// would be closer together than the footprint is wide gives none, and the
// deepest level a pixel's paint reads is one scrooge opens a node for,
// and the level under it is one scrooge does not.
func TestNothingIsMadeFinerThanAsked(t *testing.T) {
	ground := grounded(New(terran))
	tree := ground.tree
	for level := range levelsAsked {
		for _, tile := range parentsAt(tree, level) {
			made := node{ground: ground, level: level, tile: tile}
			wide := scrooge.Footprint{From: made.middle(),
				Width: made.Aggregate().Grain * 1.01}
			for range made.Detail(wide) {
				t.Fatalf("level %d gives children finer than asked", level)
			}
		}
	}
	for place := range pointsAsked {
		point := scatterOnSphere(place)
		angle := tree.spacing(0) * math.Pow(0.37, float64(place%12))
		assertReadAsOpened(t, ground, point, angle)
	}
}

// assertReadAsOpened fails a test where the deepest level a pixel's paint
// reads at a point is not one the node over it, a level up, is opened
// for, or the level under it is.
func assertReadAsOpened(t *testing.T, ground *scrooged, point maths.Vec,
	angle float64) {
	t.Helper()
	tree := ground.tree
	newest, _ := tree.newest(angle)
	at := scrooge.Footprint{From: point, Width: angle}
	if newest > 0 {
		over := node{ground: ground, level: newest - 1,
			tile: tree.levels[newest-1].Nearest(point)}
		if scrooge.Resolve(over, at) != scrooge.Open {
			t.Fatalf("level %d is read at %v but its parent is not opened",
				newest, angle)
		}
	}
	if newest < tree.deepest() {
		under := node{ground: ground, level: newest,
			tile: tree.levels[newest].Nearest(point)}
		if scrooge.Resolve(under, at) == scrooge.Open &&
			weight(tree.spacing(newest+1), angle) > 0 {
			t.Fatalf("level %d is opened at %v but not read", newest+1,
				angle)
		}
	}
}

// The coarse-and-fine test's numbers: the coarse pixel as a share of the
// world's spacing, how many times finer the fine pixel is, how many fine
// pixels a side of a coarse one is asked at, how many places are tried
// and kept, how many residues from the sea a kept place's height is at
// most, and the share of them whose classes must agree.
const (
	coarseShare  = 1.0 / 8
	fineTimes    = 64
	underSide    = 8
	placesTried  = 20000
	placesKept   = 200
	nearResidues = 4
	agreeAtLeast = 0.95
)

// The coast holds still: asked at a coarse pixel, a place near a coast
// whose class is clear is the class most of the fine pixels under it
// are. A coast that moved with the pixel would put the coarse class
// where the fine ones are mostly the other.
func TestTheCoarseClassIsWhatMostFineOnesAre(t *testing.T) {
	world := New(terran)
	ground := grounded(world)
	coarse := coarseShare * world.Tiling.Spacing()
	kept, agreed := 0, 0
	for place := 0; place < placesTried && kept < placesKept; place++ {
		point := scatterOn(place, placesTried)
		at, _ := ground.here(point, coarse)
		clear := math.Abs(at.height) > at.residue
		if !clear || math.Abs(at.height) > nearResidues*at.residue ||
			at.held {
			continue
		}
		kept++
		land := at.height > 0
		if (fineLandShare(ground, point, coarse) > 0.5) == land {
			agreed++
		}
	}
	if kept < placesKept/2 {
		t.Fatalf("only %d places near a coast found", kept)
	}
	if float64(agreed) < agreeAtLeast*float64(kept) {
		t.Errorf("%d of %d places near a coast have the class most of "+
			"their fine pixels have", agreed, kept)
	}
}

// fineLandShare is the share of fine pixels under a coarse pixel at a
// point that are land: a grid of them across the coarse pixel, each
// asked at a pixel fineTimes narrower.
func fineLandShare(ground *scrooged, point maths.Vec, coarse float64) float64 {
	first, second := tangents(point)
	land := 0
	for across := range underSide {
		for down := range underSide {
			u := (float64(across)+0.5)/underSide - 0.5
			v := (float64(down)+0.5)/underSide - 0.5
			under := point.Add(first.Scale(u * coarse)).Add(
				second.Scale(v * coarse)).Unit()
			if ground.landShare(under, coarse/fineTimes) >= 0.5 {
				land++
			}
		}
	}
	return float64(land) / (underSide * underSide)
}

// The children-and-parent light test's numbers: the levels asked, and
// how far the mean of children's light may be from their parents', on
// average over many parents, in the ramp look's greys.
const (
	lightLevelFrom = 1
	lightLevelTo   = 4
	lightAllowed   = 0.02
)

// Anti-test: a child is not brighter than its parent on average, nor
// darker. Over many parents at each level, the mean of their children's
// paint, each seen whole, is the parents' own, seen whole, within a
// small share.
func TestAChildIsNotBrighterThanItsParentOnAverage(t *testing.T) {
	ground := grounded(New(terran))
	for level := lightLevelFrom; level <= lightLevelTo; level++ {
		drift := 0.0
		parents := parentsAt(ground.tree, level)
		for _, tile := range parents {
			made := node{ground: ground, level: level, tile: tile}
			own := made.Aggregate().Albedo.R
			kids, sum := 0, 0.0
			for kid := range made.Detail(scrooge.Footprint{
				From: made.middle(), Width: made.Aggregate().Grain}) {
				sum += kid.Aggregate().Albedo.R
				kids++
			}
			drift += sum/float64(kids) - own
		}
		if mean := drift / float64(len(parents)); math.Abs(mean) >
			lightAllowed {
			t.Errorf("level %d: children are %v brighter than their "+
				"parents on average", level, mean)
		}
	}
}

// A node's box holds its tile, its coverage is a share, and the deepest
// level gives no grain.
func TestANodesAggregateIsItsTiles(t *testing.T) {
	ground := grounded(New(terran))
	tree := ground.tree
	for _, level := range []int{0, 3, tree.deepest()} {
		tile := tree.levels[level].Nearest(maths.V(0.3, 0.5, 0.8))
		made := node{ground: ground, level: level, tile: tile}
		box := made.Aggregate().Box
		middle := made.middle()
		if middle.X < box.Min.X || middle.X > box.Max.X ||
			middle.Z < box.Min.Z || middle.Z > box.Max.Z {
			t.Errorf("level %d: the box %v does not hold %v", level, box,
				middle)
		}
		if got := made.Coverage(tree.spacing(level) * 100); got <= 0 ||
			got >= 1 {
			t.Errorf("level %d: a wide cell is %v covered", level, got)
		}
		if made.Coverage(0) != 1 {
			t.Errorf("level %d: no cell is not wholly covered", level)
		}
	}
	deepest := node{ground: ground, level: tree.deepest(),
		tile: tree.levels[tree.deepest()].Nearest(maths.V(1, 0, 0))}
	if grain := deepest.Aggregate().Grain; grain != 0 {
		t.Errorf("the deepest level has a grain of %v", grain)
	}
	if len(ground.Top()) != len(New(terran).tiles) {
		t.Error("the top is not every tile of the world")
	}
}

// Anti-test: what the tree forgets it makes again the same. A keep that
// is full forgets, and a record made after is the record made before.
func TestAForgottenNodeIsMadeTheSame(t *testing.T) {
	tree := New(terran).tree()
	tile := tree.levels[3].Nearest(maths.V(0.3, 0.5, 0.8))
	before := tree.record(3, tile)
	fresh := newKeep[record]()
	tree.records = fresh
	if after := tree.record(3, tile); after != before {
		t.Errorf("a forgotten record is %v, before %v", after, before)
	}
	held := fresh.of(tileKey{1, 1})
	added := 0
	for id := int64(0); added <= shardHold; id++ {
		if key := (tileKey{1, id}); fresh.of(key) == held {
			fresh.put(key, record{})
			added++
		}
	}
	if len(held.held) >= shardHold {
		t.Errorf("a shard given %d holds %d, and forgot nothing", added,
			len(held.held))
	}
}
