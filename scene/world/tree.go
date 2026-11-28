package world

import (
	"iter"
	"math"
	"slices"
	"sync"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/maths/strategies/scrooge"
)

// The ground's tree: scrooge's frugal level-of-detail tree over the
// world's own tiling. A node is a tile at a level; level nought is the
// world's tiling and each level under it is the one before made Finer.
// A tile's children are the finer tiles whose middles lie in it.
//
// Every node carries a few numbers, its attributes: how high it is, the
// cold and the rain, and the noise its biome and its light are mixed by.
// A child's are its parent's plus detail whose mean over the parent's
// children is nought, so a parent is the average of its children and a
// ground seen coarse is where the fine ground averages to.
//
// Nothing is made until a pixel asks for it, and nothing finer than the
// pixel: scrooge.Resolve opens a node only while its children are as
// wide as the footprint. What is made is kept, and forgotten when there
// is too much of it; a forgotten node is made again from its seed, the
// same as before.

// attribute is one of the numbers a node carries.
type attribute int

// The attributes: the height land and sea are decided by; the smoother
// relief rock and snow are decided by; the ridges high ground has; the
// cold and the rain; the kin noise a biome takes its neighbour by, and
// its small patches; the noise a shore's strip comes and goes by; and
// the light's grain.
const (
	heightAttribute attribute = iota
	reliefAttribute
	ridgeAttribute
	coldAttribute
	rainAttribute
	kinAttribute
	patchAttribute
	shoreAttribute
	grainAttribute
	attributes // how many there are
)

// shape is how an attribute's detail goes down the levels: the first and
// last levels it has detail at, how tall the first level's is, as a share
// of the level before each further one is, and how many pixels wide its
// finest level is at least.
type shape struct {
	first, last int
	gain, fall  float64
	span        float64
}

// The levels: a texture asked for no footprint is worked to restLevels,
// and none is worked past mostLevels, about a centimetre on a world of
// Earth's size.
const (
	restLevels = 10
	mostLevels = 26
)

// How far a level's detail spreads: a node's draw is from minus one to
// one less its siblings' mean, and blended over its rings it spreads by
// blendSpread of its amplitude, where gradient noise's octave of the same
// gain spreads by gradientSpread (both measured over 3000 places at
// levels 2 to 12, 2026-11-26). valueScale makes a level as tall as the
// octave of the noise ground before the tree, whose look the tree keeps;
// at a half, by eye, every attribute stood at under half its height.
const (
	blendSpread    = 0.23
	gradientSpread = 0.27
	valueScale     = gradientSpread / blendSpread
)

// The shapes, as the octaves of the noise ground before the tree had
// them (rugged.go, retired 2026-11-26; see the constants of each): a
// level's tiles are about half a wave apart, so level one is its first
// octave, a wave of one coarse spacing. Set by eye (2026-11-26) from
// that ground's, which were set by eye (2026-11-27).
var shapes = [attributes]shape{
	heightAttribute: {first: 1, last: mostLevels, gain: 1,
		fall: heightFall, span: heightSpan},
	reliefAttribute: {first: 1, last: reliefOctaves, gain: reliefGain,
		fall: reliefFall, span: 1},
	ridgeAttribute: {first: ridgeLevel, last: ridgeLast, gain: 1,
		fall: peakFall, span: peakSpan},
	coldAttribute: {first: 1, last: chillLevels, gain: coldFineGain,
		fall: heightFall, span: 1},
	rainAttribute: {first: 1, last: biomeLevels, gain: rainFineGain,
		fall: rainFall, span: wetSpan},
	kinAttribute: {first: 1, last: biomeLevels, gain: 1, fall: kinFall,
		span: wetSpan},
	patchAttribute: {first: patchLevel, last: biomeLevels,
		gain: textureGain, fall: kinFall, span: textureSpan},
	shoreAttribute: {first: shoreLevel, last: biomeLevels, gain: 1,
		fall: shoreFall, span: shoreSpan},
	grainAttribute: {first: 1, last: grainMost, gain: 1, fall: grainFall,
		span: grainSpan},
}

// The levels the finer attributes start at, the first whose tiles are
// about half the wave the noise ground gave them apart: ridges of about
// 700 km, shores of about 30 km and patches of about 8 km on a world of
// Earth's size. biomeLevels is the deepest the biome's own noise goes,
// about twenty metres there; chillLevels the cold's, about twenty km.
// ridgeLast is the deepest a range's ridges go, about eighty km. By eye
// (2026-11-26): rock follows a range's spine; ridges a level finer drew
// the spines as a close net, and ridges to biomeLevels speckled rock
// over the whole of seed 3's high ground.
const (
	ridgeLevel  = 2
	ridgeLast   = 5
	shoreLevel  = 6
	patchLevel  = 8
	biomeLevels = 16
	chillLevels = 6
)

// The tree's blend and boxes: a point takes the bump-weighted mean of a
// level's detail over the tiles within treeReach of that level's spacing,
// as Smoothed does the world's own, the nearest and two rings round it,
// so a tile's shape never shows; and a node's box reaches capReach
// spacings from its middle each way, which holds every tile under it
// however the levels' tiles straddle their parents. A reach of one, one
// ring, left the edges between tiles steeper than along them
// (2026-11-26).
const (
	treeReach = blendReach
	capReach  = 1.4
)

// treeSeed keeps the tree's draws from the world's other noise.
const treeSeed = 0x7ee5

// tree is a world's ground as scrooge's tree: the tilings of its levels
// and what has been made of them.
type tree struct {
	w       *World
	levels  []hexes.Tiling
	records keep[record]
	parents keep[parentStats]
	// views are what each goroutine last read of each level
	views sync.Pool
}

// record is what a made node keeps: the detail it adds to its parent's
// attributes, and the height gain its parent gave its children.
type record struct {
	detail [attributes]float32
	gain   float32
}

// parentStats is what a parent knows of its children: the mean of their
// draws, which their detail is taken from, and the height's gain there.
type parentStats struct {
	mean [attributes]float32
	gain float32
}

// tree is the world's tree, made once, when it is first asked for.
func (w *World) tree() *tree {
	w.treeOnce.Do(func() { w.grown = newTree(w) })
	return w.grown
}

// newTree is the tree of a world: its levels, down to where the tiling
// stops growing finer or to mostLevels.
func newTree(w *World) *tree {
	levels := []hexes.Tiling{w.Tiling}
	for len(levels) <= mostLevels {
		last := levels[len(levels)-1]
		finer := last.Finer()
		if !(finer.Spacing() < last.Spacing()) {
			break
		}
		levels = append(levels, finer)
	}
	made := &tree{w: w, levels: levels, records: newKeep[record](),
		parents: newKeep[parentStats]()}
	made.views.New = func() any { return new(view) }
	return made
}

// view is what one goroutine last read of each level: the nearest tile
// to the point it asked, and the middles and records of that tile's two
// rings. The next point a picture asks is nearly always under the same
// tiles at all but the finest levels, and is read without asking the
// tiling or the keep again.
type view struct {
	levels [mostLevels + 1]levelView
}

// levelView is a view of one level.
type levelView struct {
	held    bool
	nearest int64
	middle  maths.Vec
	count   int
	tiles   [ringTiles]hexes.Tile
	middles [ringTiles]maths.Vec
	records [ringTiles]record
}

// stillNear is how far, in a level's spacings, a point may be from the
// middle of the tile it was last nearest to and be sure that tile is
// still the nearest: under half the closest two middles come, which on
// a sphere's tiling is 0.8 of its spacing.
const stillNear = 0.38

// read is a level's view for a point: the one held where the point is
// surely nearest to the same tile, and otherwise the tile nearest it and
// its rings, read again only where the nearest tile is another.
func (t *tree) read(level int, seen *levelView, p maths.Vec) {
	tiling := t.levels[level]
	if seen.held && p.Sub(seen.middle).Len() < stillNear*tiling.Spacing() {
		return
	}
	nearest := tiling.Nearest(p)
	id := tiling.ID(nearest)
	if seen.held && id == seen.nearest {
		return
	}
	t.refill(level, seen, nearest, id)
}

// refill is a level's view read again round a new nearest tile: the
// tiles of its rings that the old view held are taken from it, and only
// the others from the tiling and the keep.
func (t *tree) refill(level int, seen *levelView, nearest hexes.Tile,
	id int64) {
	tiling := t.levels[level]
	was := *seen
	if !was.held {
		was.count = 0
	}
	seen.held, seen.nearest = true, id
	seen.middle = tiling.Centre(nearest)
	seen.count = ringed(tiling, nearest, &seen.tiles)
	for index, tile := range seen.tiles[:seen.count] {
		if kept := slices.Index(was.tiles[:was.count], tile); kept >= 0 {
			seen.middles[index] = was.middles[kept]
			seen.records[index] = was.records[kept]
			continue
		}
		seen.middles[index] = tiling.Centre(tile)
		seen.records[index] = t.record(level, tile)
	}
}

// deepest is the last level of the tree.
func (t *tree) deepest() int { return len(t.levels) - 1 }

// spacing is a level's spacing, as an angle.
func (t *tree) spacing(level int) float64 {
	return t.levels[level].Spacing()
}

// children are a tile's children: the tiles of the next level whose
// middles are nearer the tile's middle than any other tile's. The finer
// tiling has every middle of this one, and the middles between, so they
// are the tile's own middle and those of the ring round it that it wins.
func (t *tree) children(level int, tile hexes.Tile) []hexes.Tile {
	coarse, fine := t.levels[level], t.levels[level+1]
	middle := fine.Nearest(coarse.Centre(tile))
	own := coarse.ID(tile)
	kids := []hexes.Tile{middle}
	for _, near := range fine.Neighbours(middle) {
		if coarse.ID(coarse.Nearest(fine.Centre(near))) == own {
			kids = append(kids, near)
		}
	}
	return kids
}

// parentOf is the tile a level's tile is a child of, a level up.
func (t *tree) parentOf(level int, tile hexes.Tile) hexes.Tile {
	return t.levels[level-1].Nearest(t.levels[level].Centre(tile))
}

// draws are a tile's draws, one for each attribute, from minus one to
// one: its seed, from which its detail is made.
func (t *tree) draws(level int, tile hexes.Tile) [attributes]float64 {
	id := t.levels[level].ID(tile)
	var drawn [attributes]float64
	for which := range attributes {
		drawn[which] = 2*noise.Hash2(t.w.Seed+treeSeed+uint64(which),
			int64(level), id) - 1
	}
	return drawn
}

// amplitude is how tall an attribute's detail is at a level, for a
// height gain: nought outside the levels it has detail at.
func amplitude(which attribute, level int, gain float64) float64 {
	form := shapes[which]
	if level < form.first || level > form.last {
		return 0
	}
	first := form.gain
	if which == heightAttribute {
		first = gain
	}
	return valueScale * first * math.Pow(form.fall, float64(level-form.first))
}

// stats are what a parent knows of its children, made once and kept.
func (t *tree) stats(level int, parent hexes.Tile) parentStats {
	key := tileKey{level, t.levels[level].ID(parent)}
	if kept, found := t.parents.get(key); found {
		return kept
	}
	var made parentStats
	kids := t.children(level, parent)
	for _, kid := range kids {
		drawn := t.draws(level+1, kid)
		for which := range attributes {
			made.mean[which] += float32(drawn[which] / float64(len(kids)))
		}
	}
	made.gain = float32(t.w.coastGain(t.levels[level].Centre(parent)))
	t.parents.put(key, made)
	return made
}

// record is a node's record, made once and kept: its draws less the mean
// of its parent's children's, times the attribute's amplitude there.
func (t *tree) record(level int, tile hexes.Tile) record {
	key := tileKey{level, t.levels[level].ID(tile)}
	if kept, found := t.records.get(key); found {
		return kept
	}
	parent := t.stats(level-1, t.parentOf(level, tile))
	drawn := t.draws(level, tile)
	made := record{gain: parent.gain}
	for which := range attributes {
		size := amplitude(which, level, float64(parent.gain))
		made.detail[which] = float32(size *
			(drawn[which] - float64(parent.mean[which])))
	}
	t.records.put(key, made)
	return made
}

// node is one tile of a level of the ground's tree, as scrooge asks it.
type node struct {
	ground *scrooged
	level  int
	tile   hexes.Tile
}

// Top is the ground's tree as scrooge starts it: a node for every tile
// of the world's tiling.
func (s *scrooged) Top() []scrooge.Node {
	tiles := s.w.Tiling.Tiles()
	top := make([]scrooge.Node, 0, len(tiles))
	for _, tile := range tiles {
		top = append(top, node{ground: s, level: 0, tile: tile})
	}
	return top
}

// middle is the node's middle, a unit direction.
func (n node) middle() maths.Vec {
	return n.ground.tree.levels[n.level].Centre(n.tile)
}

// Aggregate is the node's box, the paint it comes to seen whole, and how
// far apart its children are. The ground gives no light of its own.
func (n node) Aggregate() scrooge.Aggregate {
	spacing := n.ground.tree.spacing(n.level)
	middle := n.middle()
	half := maths.V(capReach*spacing, capReach*spacing, capReach*spacing)
	made := scrooge.Aggregate{Box: maths.Box{Min: middle.Sub(half),
		Max: middle.Add(half)}, Albedo: n.ground.Within(middle, spacing)}
	if n.level < n.ground.tree.deepest() {
		made.Grain = n.ground.tree.spacing(n.level + 1)
	}
	return made
}

// hexArea is a hexagon's area for a spacing of one between middles.
var hexArea = math.Sqrt(3) / 2

// Coverage is how much of a cell the node's tile covers: all of a cell
// no wider than it, and its own area's share of a wider one.
func (n node) Coverage(cell float64) float64 {
	spacing := n.ground.tree.spacing(n.level)
	if cell <= 0 {
		return 1
	}
	return math.Min(1, hexArea*spacing*spacing/(cell*cell))
}

// Detail is the node's children, each made when taken, where scrooge
// opens the node at the footprint; none where it does not.
func (n node) Detail(at scrooge.Footprint) iter.Seq[scrooge.Node] {
	return func(yield func(scrooge.Node) bool) {
		if scrooge.Resolve(n, at) != scrooge.Open {
			return
		}
		for _, kid := range n.ground.tree.children(n.level, n.tile) {
			if !yield(node{ground: n.ground, level: n.level + 1,
				tile: kid}) {
				return
			}
		}
	}
}

// values are the node's attributes as the tree has them: its parent's
// and its own detail, from nought at level nought, where the world's own
// fields are, which the tree adds to.
func (n node) values() [attributes]float64 {
	var sum [attributes]float64
	if n.level == 0 {
		return sum
	}
	tree := n.ground.tree
	parent := node{ground: n.ground, level: n.level - 1,
		tile: tree.parentOf(n.level, n.tile)}
	sum = parent.values()
	made := tree.record(n.level, n.tile)
	for which := range attributes {
		sum[which] += float64(made.detail[which])
	}
	return sum
}

// tileKey names a tile at a level of the tree.
type tileKey struct {
	level int
	id    int64
}

// How much the tree keeps: its kept things are split into keepShards
// shards, each of which forgets all it holds when it holds shardHold, so
// a draw of any size holds at most keepShards times shardHold of each.
const (
	keepShards = 256
	shardHold  = 8192
)

// keep is what the tree has made, by tile, safe to share between the
// goroutines a draw paints with.
type keep[Kept any] struct {
	shards []shard[Kept]
}

// shard is one part of a keep.
type shard[Kept any] struct {
	lock sync.RWMutex
	held map[tileKey]Kept
}

// newKeep is an empty keep.
func newKeep[Kept any]() keep[Kept] {
	made := keep[Kept]{shards: make([]shard[Kept], keepShards)}
	for index := range made.shards {
		made.shards[index].held = map[tileKey]Kept{}
	}
	return made
}

// of is the shard a key is kept in.
func (k keep[Kept]) of(key tileKey) *shard[Kept] {
	mixed := noise.Mix(uint64(key.id)*keepShards + uint64(key.level))
	return &k.shards[mixed%keepShards]
}

// get is what is kept for a key, and whether there is anything.
func (k keep[Kept]) get(key tileKey) (Kept, bool) {
	held := k.of(key)
	held.lock.RLock()
	defer held.lock.RUnlock()
	kept, found := held.held[key]
	return kept, found
}

// put keeps a thing for a key, forgetting the shard's whole store first
// when it is full: what is forgotten is made again, the same, if asked.
func (k keep[Kept]) put(key tileKey, kept Kept) {
	held := k.of(key)
	held.lock.Lock()
	defer held.lock.Unlock()
	if len(held.held) >= shardHold {
		held.held = make(map[tileKey]Kept, shardHold)
	}
	held.held[key] = kept
}
