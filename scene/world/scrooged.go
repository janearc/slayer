package world

import (
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/texture"
)

// Scrooged is the world's surface read from its tree (tree.go), as finely
// as the pixel asking can see and no finer. The coarse tiles' heights,
// smoothed between them and bent by a coarse warp, are where the ground
// starts; the tree's levels add their detail to it, each level's blended
// over its tiles so no tile's shape shows, down to the level whose tiles
// are a pixel apart, which fades in as the pixel narrows.
//
// Land, sea and biome are decided at the point from the height, the cold
// and the rain so made. A level's detail has a mean of nought over each
// parent's children, so the ground seen coarse is where the fine ground
// averages to, and a coast breaks into finer coast as the eye comes
// down without moving. Where a pixel straddles a shore its paint is a
// blend of sea and land by how much of the height is still left out.
//
// Vary is how much the light may differ from its class's paint, from
// zero to one; Mix how readily a neighbouring class is taken, forest in
// grass, rock in desert. With no mix the ground is the coarse tiles' own.
func (w *World) Scrooged(look Look, vary, mix float64) texture.Texture {
	return w.ScroogedRidges(look, vary, mix, 1)
}

// ScroogedRidges varies the painted ridge contribution. One keeps the
// original appearance; zero suppresses ridges. This is not terrain geometry.
func (w *World) ScroogedRidges(look Look, vary, mix, ridges float64) texture.Texture {
	return &scrooged{w: w, tree: w.tree(), look: look, vary: vary, mix: mix, ridges: ridges}
}

// scrooged is a world's surface read from its tree.
type scrooged struct {
	w         *World
	tree      *tree
	look      Look
	vary, mix float64
	ridges    float64
}

// How the height is worked: its first wave in coarse spacings, the gain
// of its first level away from a coast, which cannot show, how tall each
// level is as a share of the one before, 2 to the power of minus a
// roughness of about 0.5, and how many pixels apart its finest level
// read is at least: finer levels are left to the land's share, so a
// coast is not decided pixel by pixel. Set by eye (2026-11-26).
const (
	heightWave = 1
	heightGain = 0.06
	heightFall = 0.72
	heightSpan = 2
)

// Near a coast the height's gain is the coarse ground's slope, found
// over slopeStep coarse spacings, times the first wave, times
// slopeRoughness, between slopeFloor and slopeCeil. coastReach is how
// far from the sea, in height, that is used. axisLimit is how near the
// x axis a point is before the tangents are made against y. Set by eye
// (2026-11-27): a coast moves about a quarter of a wave at any wave.
const (
	coastReach     = 0.5
	slopeStep      = 0.1
	slopeRoughness = 0.5
	slopeFloor     = 0.02
	slopeCeil      = 0.6
	axisLimit      = 0.9
)

// The tiles' height is bent by warpOctaves octaves of noise of warpGain,
// from a first wave of warpWave coarse spacings, and moved by moveGain
// spacings of noise of moveWave: the big bends of a coast, so a
// continent's sides are not straight. The relief the class is decided on
// has its own gain and fall. Set by eye (2026-11-27).
const (
	warpWave      = 1
	warpGain      = 0.1
	warpOctaves   = 4
	moveGain      = 0.9
	moveWave      = 2.5
	moveSeed      = 110
	warpSeedShift = 99
	reliefGain    = 0.3
	reliefFall    = 0.61
)

// A range's ridges: how tall each level is as a share of the last, and
// how many pixels wide the finest is at least (by eye, 2026-11-26: a
// level is blended over its tiles and does not speckle at four pixels).
// Their noise over its spread is a place's crest, how far it stands up
// from the ground round it. Rock is where the crest and the plain relief
// together pass crestRock, snow where they pass crestSnow, each soft by
// crestSoft either side; a plain crestPlain higher needs a crest one
// spread lower. So rock follows a range's crests, snow its highest, and
// the low ground between is clear. Set by eye (2026-11-26): rock decided
// on the height alone was a speckle over all of seed 3's high ground.
const (
	peakFall   = 0.6
	peakSpan   = 4
	crestRock  = 1
	crestSnow  = 2
	crestSoft  = 0.3
	crestPlain = 0.12
)

// What the cold and the rain are before the tree's detail: the world's
// noise at its own scale and octaves, the latitude's cold and the
// height's, and the rain falling off with the distance from the sea; the
// detail's first gains, and how tall each level of the rain's is as a
// share of the last. Set by eye (2026-11-26): the noise ground's gain of
// 0.1, now the tree's levels stand as tall as its octaves; at 0.15 and
// 0.65 on half-height levels grass and forest met on a smooth curve.
const (
	coldFineGain  = 0.08
	rainFineGain  = 0.1
	rainFall      = 0.6
	coldNoiseSeed = 60
	rainNoiseSeed = 61
	coldPerHeight = 0.45
	latitudeCold  = 1.05
	coldRough     = 0.12
	rainRough     = 0.3
	coldRoughWave = 4
	rainRoughWave = 3
	roughOctaves  = 3
)

// The shore is a strip of ground within a height of the sea, in a tile
// within beachWet tiles of it. Its width is shoreHeight times shoreLeast
// to shoreLeast plus shoreMore by its noise, seen to shoreSpan pixels,
// soft by shoreSoft; twice as wide on flat ground as on steep, and rock
// where the gain is over steepHalf of the way from steepFrom to steepTo.
// Set by eye (2026-11-27): a few pixels at the most, and gone in places;
// the height by eye (2026-11-26), as a pixel's share of it shows.
const (
	shoreHeight  = 0.0006
	shoreLeast   = 0
	shoreMore    = 2
	shoreMost    = shoreHeight * (shoreLeast + shoreMore) * 2
	shoreSpan    = 4
	shoreSoft    = 0.25
	shoreFall    = 0.6
	steepFrom    = 0.15
	steepTo      = 0.45
	steepHalf    = 0.5
	beachWet     = 1.5
	farFromShore = 2
)

// reliefOctaves is how many levels of detail the relief has: a mountain
// is a connected range and a plain a plain. Set by eye (2026-11-27):
// about three hundred km at the finest.
const reliefOctaves = 3

// The kin mix: the level its noise is over for the kin class to be
// taken, how tall each level is as a share of the last, and the most of
// a landscape that is its kin's. Set by eye (2026-11-27).
const (
	kinThreshold = 0.3
	kinFall      = 0.7
	kinMost      = 0.6
)

// The kin mix's small patches: how far they move the kin noise, how many
// pixels wide they are at least, and the regions they come in, a noise
// of textureRegion coarse spacings and maskOctaves octaves over which
// they fade in between minus and plus textureMask. Set by eye
// (2026-11-27); the fade from 0.1 to 0.4 by eye (2026-11-26), as a
// region with patches met one without on a crisp curve from 320 km up.
const (
	textureGain      = 1.2
	textureSpan      = 12
	textureRegion    = 0.15
	textureMask      = 0.4
	maskOctaves      = 2
	textureSeedShift = 97
)

// How far a class's edge is soft: the height, cold, rain and kin noise
// are each moved by up to this much, up and down, in softMixes mixes,
// and the paints of the classes found are averaged, so one biome goes
// into the next by a band and not a line. Set by eye (2026-11-27). Four
// mixes of signs made the band four steps, each a crisp line where the
// cold and rain are smooth; sixteen spread evenly make each step a
// sixteenth (2026-11-26).
const (
	softHeight = 0.06
	softCold   = 0.04
	softRain   = 0.07
	softKin    = 0.3
	softMixes  = 16
)

// softSteps are irrational steps, one for each of the height, cold, rain
// and kin, by which the mixes' moves are spread: the k-th mix moves each
// by the fraction of k and a half times its step, so each runs evenly
// from down to up and no two run together.
var softSteps = [4]float64{0.6180339887, 0.4142135624, 0.7320508076,
	0.2360679775}

// mountainSoft is how far under pick's rock line, in relief, the lowland
// class is picked, so pick leaves rock and snow to the ranges. Set by eye
// (2026-11-27).
const mountainSoft = 0.12

// softSigns are the mixes' moves, from minus one to one, for the height,
// cold, rain and kin, that the soft edge is averaged over.
var softSigns = spreadMixes()

// spreadMixes is softMixes moves of each of four numbers, each from
// minus one to one evenly, by softSteps.
func spreadMixes() [softMixes][4]float64 {
	var mixes [softMixes][4]float64
	for mix := range softMixes {
		for which, step := range softSteps {
			share := math.Mod((float64(mix)+0.5)*step, 1)
			mixes[mix][which] = 2*share - 1
		}
	}
	return mixes
}

// The depth of water: how far under the sea's level the paint starts to
// go from the sea's to the deep's, and how far it has gone all the way.
// Set by eye (2026-11-26).
const (
	shallowest = 0.08
	deepest    = 0.5
)

// fixedReach is how many coarse spacings from a fixed tile's middle its
// class is held, so a feature's patch is round and not a hexagon.
const fixedReach = 0.45

// The wet that decides the rain is a count of tiles from the sea, whose
// contours are hexagonal: it is moved by noise of wetGain tiles over
// wetOctaves octaves from a first wave of wetWave coarse spacings. The
// rain's detail is seen to wetSpan pixels. Set by eye (2026-11-26).
const (
	wetGain      = 1.6
	wetSpan      = 12
	wetFall      = 0.5
	wetWave      = 1.2
	wetOctaves   = 4
	wetSeedShift = 96
)

// The light's grain: how far each level's detail lightens or darkens the
// paint for a vary of one, the share of it the sea has (by eye,
// 2026-11-26: at 0.3 the open sea was blotched in a paint map), the deepest
// level it goes to, how tall each level is as a share of the last, and
// how many pixels apart its finest level's tiles are at least. Set by eye
// (2026-11-27) for the noise ground, and kept: a few per cent a level,
// every level as tall, so the ground has grain at whatever scale the eye
// is. At a span of one, the finest level's tiles, two pixels apart,
// showed as a lattice of dots; at two, a dither pixel by pixel; at four
// the finest grain is a few pixels across and what is finer is averaged
// away, as the pixel sees it (2026-11-26).
const (
	grainLevel = 1.5
	seaGrain   = 0.15
	grainMost  = 20
	grainFall  = 1
	grainSpan  = 4
)

// cdfReach is how many spreads of the height left out below a pixel a
// shore blends over either side of the sea's level: a smooth step that
// wide is as steep at its middle as the normal's share is, so a pixel
// straddling a shore is the share of land and sea under it.
const cdfReach = 1.88

// The octaves of the coarse noise: each is riseOctave times the
// frequency and fallOctave the height of the one before, and takes its
// own seed by octaveStep. 2.03, not 2, keeps the lattices apart.
const (
	riseOctave = 2.03
	fallOctave = 0.5
	octaveStep = 0x2545f4914f6cdd1d
)

// angular is a footprint, in the thing's own length, as the angle it
// subtends at the middle of a body whose surface is where p is: the
// tiling's measure. A point at the middle, or a footprint of none, has
// no angle.
func angular(p maths.Vec, footprint float64) float64 {
	if p.Len() <= 0 || footprint <= 0 {
		return 0
	}
	return footprint / p.Len()
}

// At is the paint at p, worked as finely as a point is: with no pixel.
func (s *scrooged) At(p maths.Vec) maths.Light { return s.Within(p, 0) }

// Within is the paint at p for a pixel footprint wide, in the length p
// is in: the tree read at p down to the footprint, and no finer.
func (s *scrooged) Within(p maths.Vec, footprint float64) maths.Light {
	point := p.Unit()
	paint, land, grain := s.ground(point, angular(p, footprint))
	if s.vary <= 0 {
		return paint
	}
	calm := seaGrain + (1-seaGrain)*land
	return paint.Scale(math.Max(0, 1+calm*s.vary*grainLevel*grain))
}

// landShare is how much of what is under a pixel of an angle is land.
func (s *scrooged) landShare(p maths.Vec, angle float64) float64 {
	_, land, _ := s.ground(p, angle)
	return land
}

// ground is the paint of the class at p, before the light's grain, how
// much of what is under the pixel is land, from nought to one, and the
// grain's detail there.
func (s *scrooged) ground(p maths.Vec, angle float64) (maths.Light,
	float64, float64) {
	if s.mix <= 0 {
		return s.look.Paints[s.w.Biome(s.w.Tiling.Nearest(p))], 1, 0
	}
	at, grain := s.here(p, angle)
	if at.held {
		return s.look.Paints[s.w.fixed[at.smooth.Nearest]], 1, grain
	}
	paint, land := s.shared(at)
	return paint, land, grain
}

// shared is a pixel's paint at a place and how much of it is land: the
// sea, the shore's strip and the land past it, each by the share of the
// ground under the pixel that it is. That ground's height spreads by the
// residue either side of the height read, so the land is the share of it
// over the sea's level, and the land past the strip the share over the
// strip's top. A pixel straddling a shore is so a blend, never a pick.
func (s *scrooged) shared(at here) (maths.Light, float64) {
	land := maths.Smooth(-at.residue, at.residue, at.height)
	inland := maths.Smooth(-at.residue, at.residue,
		at.height-s.stripWidth(at))
	var paint maths.Light
	if land < 1 {
		paint = s.water(at, math.Min(0, at.height)).Scale(1 - land)
	}
	if inland > 0 {
		paint = paint.Add(s.landAt(at, false).Scale(inland))
	}
	if land > inland {
		paint = paint.Add(s.landAt(at, true).Scale(land - inland))
	}
	return paint, land
}

// here is what is worked out at a point: the coarse world there, and the
// height, the plain relief and the ranges' crests, the cold, the rain,
// the kin noise and the shore's mix, each its coarse field and the
// tree's detail. The rain and the kin
// are left out deep under the sea, where nothing shows them.
type here struct {
	smooth   Ground
	height   float64
	residue  float64
	chill    float64
	rain     float64
	kinNoise float64
	plain    float64
	crest    float64
	steep    float64
	shoreMix float64
	held     bool
}

// here is the point's coarse world and its tree's detail, and the
// grain's detail, for a pixel of an angle.
func (s *scrooged) here(p maths.Vec, angle float64) (here, float64) {
	smooth := s.w.Smoothed(p)
	got := s.tree.gather(p, angle)
	base := s.w.lifted(p)
	at := here{smooth: smooth, held: s.holds(p, smooth),
		height: base + got.detail[heightAttribute], residue: got.residue,
		steep: maths.Smooth(steepFrom, steepTo, got.gain),
		chill: s.w.chill(p) + got.detail[coldAttribute]}
	if at.height < -at.residue-deepest {
		return at, got.detail[grainAttribute]
	}
	at.rain = s.w.rainAt(p, smooth) + got.detail[rainAttribute]
	at.kinNoise = got.detail[kinAttribute] +
		s.w.patchMask(p)*got.detail[patchAttribute]
	at.shoreMix = maths.Smooth(-shoreSoft, shoreSoft,
		got.detail[shoreAttribute])
	at.plain = base + got.detail[reliefAttribute]
	at.crest = crestOf(got.detail[ridgeAttribute],
		s.tree.ridgeSpread(angle)) * s.ridges
	return at, got.detail[grainAttribute]
}

// The fold of a normal number, its size: its mean and its spread, for a
// number of a spread of one.
const (
	foldMean   = 0.7978845608
	foldSpread = 0.6028103203
)

// crestOf is a place's crest from its ridges' noise and the spread of
// the noise read: highest along the lines where the noise is nought,
// which run unbroken as a range's spines, and falling either side, in
// spreads of the fold, so it has a mean of nought and a spread of one,
// the same share of the ranges at every distance. With no ridges read
// there is no crest.
func crestOf(noise, spread float64) float64 {
	if spread <= 0 {
		return 0
	}
	return (foldMean - math.Abs(noise)/spread) / foldSpread
}

// ridgeSpread is how far the ridges a pixel of an angle reads spread:
// blendSpread of each of their levels' amplitude, as much of it as the
// pixel sees, the levels adding as squares do.
func (t *tree) ridgeSpread(angle float64) float64 {
	angle = t.asked(angle)
	form := shapes[ridgeAttribute]
	square := 0.0
	for level := form.first; level <= min(form.last, t.deepest()); level++ {
		seen := weight(t.spacing(level), angle*form.span) *
			amplitude(ridgeAttribute, level, 1)
		square += seen * seen
	}
	return blendSpread * math.Sqrt(square)
}

// asked is the angle a pixel is read at: its own, or for no pixel, the
// rest levels' spacing.
func (t *tree) asked(angle float64) float64 {
	if !(angle > 0) {
		return t.spacing(0) / math.Exp2(restLevels)
	}
	return angle
}

// sums are what the tree's levels add at a point: each attribute's
// detail over the levels the footprint opens, the height's gain there,
// and the residue, how far either side of the sea's level the height
// left out below the pixel blends land into sea.
type sums struct {
	detail        [attributes]float64
	gain, residue float64
}

// gather is the tree's detail at p for a pixel of an angle: every level
// whose tiles are wider apart than the pixel, each attribute's from its
// first level to its last, the newest fading in from nought where its
// tiles are as far apart as its span of pixels is wide.
func (t *tree) gather(p maths.Vec, angle float64) sums {
	angle = t.asked(angle)
	var got sums
	newest, _ := t.newest(angle)
	seen := t.views.Get().(*view)
	defer t.views.Put(seen)
	for level := 1; level <= max(newest, 1); level++ {
		t.read(level, &seen.levels[level], p)
		blended, gain := t.blend(level, &seen.levels[level], p)
		if level == 1 {
			got.gain = gain
		}
		for which := range attributes {
			width := angle * shapes[which].span
			got.detail[which] += weight(t.spacing(level), width) *
				blended[which]
		}
	}
	read, fade := t.newest(angle * shapes[heightAttribute].span)
	got.residue = leftOut(max(read, 1), fade, got.gain)
	return got
}

// leftOut is how far either side of the sea's level a pixel's height
// blends land into sea, for the height's gain: cdfReach spreads of the
// height left out below it, what the newest level read does not show of
// itself and every level under it, each a share heightFall of the last,
// their spreads adding as squares do.
func leftOut(newest int, fade, gain float64) float64 {
	unseen := (1 - fade) * amplitude(heightAttribute, newest, gain)
	under := amplitude(heightAttribute, newest+1, gain)
	square := unseen*unseen + under*under/(1-heightFall*heightFall)
	return cdfReach * blendSpread * math.Sqrt(square)
}

// newest is the deepest level a pixel of an angle sees, the last whose
// tiles are further apart than it is wide, and how much of it the pixel
// sees; level nought, the world's own, where it sees none of the tree.
func (t *tree) newest(angle float64) (int, float64) {
	level, fade := 0, 0.0
	for next := 1; next <= t.deepest(); next++ {
		share := weight(t.spacing(next), angle)
		if share <= 0 {
			break
		}
		level, fade = next, share
	}
	return level, fade
}

// weight is how much of a level's detail a pixel of a width sees: none
// where the level's tiles are no further apart than it is wide, all of
// it where they are twice that, and smoothly between.
func weight(spacing, width float64) float64 {
	return math.Max(0, math.Min(1, spacing/width-1))
}

// blend is a level's detail at p, from its view there: the bump-weighted
// mean of the nearest tile's and its two rings', and of the height's gain
// they were given.
func (t *tree) blend(level int, seen *levelView, p maths.Vec) (
	[attributes]float64, float64) {
	reach := treeReach * t.spacing(level)
	var sum [attributes]float64
	var total, gain float64
	for index := range seen.count {
		part := bump(p.Sub(seen.middles[index]).Len() / reach)
		if part <= 0 {
			continue
		}
		made := &seen.records[index]
		total += part
		gain += part * float64(made.gain)
		for which := range attributes {
			sum[which] += part * float64(made.detail[which])
		}
	}
	for which := range attributes {
		sum[which] /= total
	}
	return sum, gain / total
}

// ringTiles is how many tiles a tile and its blendRings rings hold at
// most: one, six and twelve.
const ringTiles = 19

// ringed fills tiles with a tile and the two rings of tiles round it,
// each once, and says how many, making no list on the way.
func ringed(tiling hexes.Tiling, middle hexes.Tile,
	tiles *[ringTiles]hexes.Tile) int {
	var ring, further [6]hexes.Tile
	inner := neighboursOf(tiling, middle, &ring)
	count := copy(tiles[:], ring[:inner])
	tiles[count] = middle
	count++
	for _, near := range ring[:inner] {
		for _, tile := range further[:neighboursOf(tiling, near, &further)] {
			if !slices.Contains(tiles[:count], tile) && count < ringTiles {
				tiles[count] = tile
				count++
			}
		}
	}
	return count
}

// neighbourer is a tiling that fills a room with a tile's neighbours, as
// a sphere's does, without making a list.
type neighbourer interface {
	NeighboursInto(tile hexes.Tile, room *[6]hexes.Tile) int
}

// neighboursOf fills room with a tile's neighbours and says how many: by
// the tiling's own room where it has one, and its list where it has not.
func neighboursOf(tiling hexes.Tiling, tile hexes.Tile,
	room *[6]hexes.Tile) int {
	if filler, fills := tiling.(neighbourer); fills {
		return filler.NeighboursInto(tile, room)
	}
	return copy(room[:], tiling.Neighbours(tile))
}

// spectrum is noise at p of wave coarse spacings, summed over octaves,
// each a share fall as tall as the one before.
func (w *World) spectrum(seed uint64, p maths.Vec, wave float64,
	octaves int, fall float64) float64 {
	k := 1 / (wave * w.Tiling.Spacing())
	x, y, z := p.X*k, p.Y*k, p.Z*k
	sum, amp := 0.0, 1.0
	for octave := range octaves {
		sum += amp * noise.Gradient(seed+uint64(octave)*octaveStep, x, y, z)
		amp *= fall
		x, y, z = x*riseOctave, y*riseOctave, z*riseOctave
	}
	return sum
}

// lifted is the coarse ground's height at p, its tiles smoothed and bent
// by the warp: what the tree's detail is added to.
func (w *World) lifted(p maths.Vec) float64 {
	spacing := w.Tiling.Spacing()
	var move [3]float64
	for axis := range move {
		move[axis] = moveGain * spacing * w.spectrum(
			w.Seed+moveSeed+uint64(axis), p, moveWave, warpOctaves,
			fallOctave)
	}
	moved := maths.V(p.X+move[0], p.Y+move[1], p.Z+move[2]).Unit()
	return w.Smoothed(moved).Height + warpGain*w.spectrum(
		w.Seed+warpSeedShift, p, warpWave, warpOctaves, fallOctave)
}

// coastGain is how tall the height's first level of detail stands at p.
// Near a coast it is a share of how far the ground climbs over that
// level's wave, the coarse ground's slope times the wave, so a coast
// moves by about the same share of any wave whether the land beside the
// sea is steep or flat; away from the coast it is heightGain.
func (w *World) coastGain(p maths.Vec) float64 {
	base := w.lifted(p)
	if math.Abs(base) > coastReach {
		return heightGain
	}
	step := slopeStep * w.Tiling.Spacing()
	first, second := tangents(p)
	along := w.lifted(p.Add(first.Scale(step)).Unit()) - base
	across := w.lifted(p.Add(second.Scale(step)).Unit()) - base
	slope := math.Hypot(along, across) / step
	return math.Min(slopeCeil, math.Max(slopeFloor,
		slopeRoughness*slope*heightWave*w.Tiling.Spacing()))
}

// tangents are two unit vectors at right angles to each other and to a
// direction on the sphere.
func tangents(p maths.Vec) (first, second maths.Vec) {
	away := maths.V(1, 0, 0)
	if math.Abs(p.X) > axisLimit {
		away = maths.V(0, 1, 0)
	}
	first = p.Cross(away).Unit()
	return first, p.Cross(first)
}

// chill is the cold at p before the tree's detail and the height's: the
// latitude's, the world's noise, and the world's cold.
func (w *World) chill(p maths.Vec) float64 {
	rough := noise.Fractal(w.Seed+coldNoiseSeed, p.X*coldRoughWave,
		p.Y*coldRoughWave, p.Z*coldRoughWave, roughOctaves)
	return w.Tiling.Latitude(p, w.north())*latitudeCold + coldRough*rough +
		w.Cold
}

// rainAt is the rain at p before the tree's detail: falling off with the
// distance from the sea, its contours bent, and wandering by the world's
// noise.
func (w *World) rainAt(p maths.Vec, smooth Ground) float64 {
	rough := noise.Fractal(w.Seed+rainNoiseSeed, p.X*rainRoughWave,
		p.Y*rainRoughWave, p.Z*rainRoughWave, roughOctaves)
	bend := w.spectrum(w.Seed+wetSeedShift, p, wetWave, wetOctaves, wetFall)
	wet := math.Max(0, smooth.Wet+wetGain*bend)
	return math.Exp(-wet/w.rainBy) + rainRough*rough
}

// patchMask is how much of the small patches a place has, from nought to
// one: regions of a few hundred km, some cleared in patches and some not.
func (w *World) patchMask(p maths.Vec) float64 {
	return maths.Smooth(-textureMask, textureMask, w.spectrum(
		w.Seed+textureSeedShift+1, p, textureRegion, maskOctaves,
		fallOctave))
}

// holds is whether p is within a fixed tile's round patch.
func (s *scrooged) holds(p maths.Vec, smooth Ground) bool {
	if smooth.Nearest < 0 || s.w.fixed[smooth.Nearest] == unfixed {
		return false
	}
	return p.Sub(s.w.middle[smooth.Nearest]).Len() <
		fixedReach*s.w.Tiling.Spacing()
}

// water is the paint of the water at a place a height under the sea's
// level: ice where it is cold enough, and otherwise graded from the
// sea's paint to the deep's by how far down it is.
func (s *scrooged) water(at here, height float64) maths.Light {
	if pick(height, at.chill, 0, farFromShore) == Ice {
		return s.look.Paints[Ice]
	}
	share := maths.Smooth(shallowest, deepest, -height)
	sea, abyss := s.look.Paints[Sea], s.look.Paints[Deep]
	return sea.Scale(1 - share).Add(abyss.Scale(share))
}

// stripWidth is how high over the sea's level the shore's strip reaches
// at a place: none away from the sea, and otherwise by the shore's own
// noise, from none to its widest, twice as wide where the coast is flat
// as where it is steep.
func (s *scrooged) stripWidth(at here) float64 {
	if at.smooth.Wet >= beachWet {
		return 0
	}
	return shoreHeight * (shoreLeast + shoreMore*at.shoreMix) *
		(1 + (1 - at.steep))
}

// shoreClass is what the strip at the water's edge is: a beach on a
// flat shore and bare rock on a steep one.
func (s *scrooged) shoreClass(at here) Biome {
	if at.steep > steepHalf {
		return Rock
	}
	return Beach
}

// landAt is the paint of land at a place: its class from the smoother
// relief, cold and rain, averaged over the soft mixes, the cold taken
// from the relief before its ridges, so a cold range's spines are not a
// lace of snow and only the highest summits have it; and on the shore's
// strip, where edge says it is, the strip's own class, but where the land
// is snow, rock or tundra, which runs to the water as it is.
func (s *scrooged) landAt(at here, edge bool) maths.Light {
	var sum maths.Light
	shore := s.shoreClass(at)
	level := kinThreshold - 0.5*s.mix
	share := kinMost * maths.Smooth(level-softKin, level+softKin, at.kinNoise)
	for _, sign := range softSigns {
		plain := math.Max(0, at.plain+sign[0]*softHeight)
		cold := at.chill + plain*coldPerHeight + sign[1]*softCold
		low := math.Min(plain, rockHeight-mountainSoft)
		lowCold := at.chill + low*coldPerHeight + sign[1]*softCold
		class := pick(low, lowCold, at.rain+sign[2]*softRain, farFromShore)
		paint := s.look.Paints[class]
		if edge && class != Snow && class != Rock && class != Tundra {
			paint = s.look.Paints[shore]
		} else {
			other := s.look.Paints[kin[class]]
			paint = paint.Scale(1 - share).Add(other.Scale(share))
		}
		sum = sum.Add(s.mountain(paint, plain, at.crest, cold))
	}
	return sum.Scale(1.0 / softMixes)
}

// mountain is a lowland paint taken up a range: to bare rock where its
// crest and plain relief pass crestRock, a plain at rockHeight needing a
// crest of that many spreads, and to snow where they pass crestSnow, a
// plain at snowHeight needing that many. Cold ground goes to snow by its
// cold as pick has it.
func (s *scrooged) mountain(lowland maths.Light,
	plain, crest, cold float64) maths.Light {
	rock := maths.Smooth(crestRock-crestSoft, crestRock+crestSoft,
		crest+(plain-rockHeight)/crestPlain)
	snow := maths.Smooth(crestSnow-crestSoft, crestSnow+crestSoft,
		crest+(plain-snowHeight)/crestPlain)
	if cold > snowCold {
		snow = 1
		rock = 1
	}
	flank := s.look.Paints[Rock].Scale(1 - snow).Add(
		s.look.Paints[Snow].Scale(snow))
	return lowland.Scale(1 - rock).Add(flank.Scale(rock))
}
