package manufacture

import (
	"errors"
	"github.com/janearc/slayer/manufacture/geometry"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// lump is a stone: a ball cut into triangles, pushed in by noise, squared off,
// cleaved by planes as broken stone is, flattened underneath, stretched to its
// three axes and set on the ground. how much of each is done follows from one
// number, how worn it is:
//
// from zero, freshly broken, to one, worn smooth, which is exactly the
// ellipsoid of its axes. a pebble, a cobble and a boulder are all this. the
// making is ninja-gaiden's, from lesson/debris.go.txt, with maths/noise in
// place of its own.
//
// its longest axis lies along x, its shortest stands up, and its seed is which
// stone it is.
//
// made only by newLump, and made a plate only by asPlate.
type lump struct {
	long, middle, short measure.Length
	worn                float64
	from                seed
	broken              breaking
}

// breaking is how a stone not worn at all is roughened and broken; a worn
// one gets less of each, down to none.
//
// how much of its radius noise may take, the power its coordinates are
// raised to across and up, which swell it toward a box, how many planes
// cleave it, how far from its middle the nearest lies and how far further
// they spread, as shares of its radius, and whether the planes stand upright.
//
// what is left of one when each power is taken from it, squareless and
// flatless, is kept as made, not worked out where a stone is shaped: a
// stone's are worked out as constants, exactly, and a plate's from numbers,
// in float64, and the two differ in the last place.
type breaking struct {
	roughest, squarest, flattest float64
	squareless, flatless         float64
	cleaves                      int
	nearest, spread              float64
	upright                      bool
}

// stoneBreaking is how a stone of any shape is broken: the numbers below,
// with what is left of each power worked out as a constant.
func stoneBreaking() breaking {
	return breaking{roughest: roughest, squarest: squarest,
		flattest: squarest, squareless: 1 - squarest, flatless: 1 - squarest,
		cleaves: mostCleaves, nearest: nearest, spread: cleaveRange}
}

// complement is what is left of a share when it is taken from one, worked
// out in float64 as a plate's always has been.
func complement(share float64) float64 { return 1 - share }

// how a plate of a layered rock is broken, where it differs from a stone:
// picked by looking at the photographs a venus lander took of its ground,
// on 2026-11-15. its top and bottom are the rock's layers, nearly flat,
// and it breaks along joints that stand upright, into polygons.
const (
	plateRough   = 0.08 // picked: a layer's face is nearly smooth
	plateFlat    = 0.15 // picked: its top swollen nearly flat
	plateCleaves = 7    // picked: five to seven broken sides
	plateNearest = 0.55 // picked: each side cut a little in
)

// asPlate is the stone broken as a plate is.
func (made lump) asPlate() lump {
	made.broken = breaking{roughest: plateRough, squarest: squarest,
		flattest: plateFlat, squareless: complement(squarest),
		flatless: complement(plateFlat), cleaves: plateCleaves,
		nearest: plateNearest, spread: cleaveRange, upright: true}
	return made
}

var errLump = errors.New("manufacture: a lump needs three axes above " +
	"nought, each with an end, the long no shorter than the middle " +
	"and the middle no shorter than the short, a wear from nought " +
	"to one, and a seed that was sown")

// newLump is a stone of three axes, so worn, from a seed, or why there
// can be none.
func newLump(long, middle, short measure.Length, worn float64,
	from seed) (lump, error) {
	across, deep, tall := inScene(long), inScene(middle), inScene(short)
	sound := positive(across) && positive(deep) && positive(tall) &&
		across >= deep && deep >= tall && share(worn) && from.made()
	if !sound {
		return lump{}, errLump
	}
	return lump{long: long, middle: middle, short: short, worn: worn,
		from: from, broken: stoneBreaking()}, nil
}

// how a stone is roughened and broken. these are tuning, not
// measurement: ninja-gaiden's, set by eye against a picture of a
// mountainside on 2026-10-06. each is what a stone not worn at all
// gets, and a worn one gets less, down to none.
const (
	// roughest is how much of its radius noise may take from a stone.
	roughest = 0.5
	// grain is how fast the noise changes over the ball, and octaves
	// how many scales of it there are: lumps, bumps and grit.
	grain   = 1.1
	octaves = 3
	// squarest is the power a ball's coordinates are raised to, which
	// swells it toward a box; one leaves it round.
	squarest = 0.7
	// mostCleaves is how many planes cleave a stone, nearest and
	// spread how far from its middle each lies, as shares of its
	// radius.
	mostCleaves = 6
	nearest     = 0.45
	cleaveRange = 0.35
	// sits is how far below its middle a stone is cut flat to sit on,
	// as a share of its radius.
	sits = 0.3
)

// the parts of a seed a lump draws from: one for each cleaving plane,
// counted from here, so they are no arrangement's.
const cleaving = 100

// offLattice moves the ball off the noise's own lattice, where gradient noise
// is zero and every stone would have the same six dimples.
var offLattice = maths.V(7.3, 3.1, 5.7)

func ball(fine cut) surface { return surface(geometry.Ball[face](fine.around)) }

// plane is one cleaving of a stone: which way it faces and how far
// from the middle it lies, on the unit ball.
type plane struct {
	facing maths.Vec
	out    float64
}

// cleaves is the planes that cleave this stone: fewer the more worn it
// is, each facing a way and lying a distance drawn from the seed.
func (made lump) cleaves() []plane {
	broken := made.broken
	count := int(math.Round(float64(broken.cleaves) * (1 - made.worn)))
	planes := make([]plane, count)
	for index := range planes {
		part, from := cleaving+index, made.from
		planes[index] = plane{facing: made.cleaving(part),
			out: broken.nearest + broken.spread*from.draw(part, 1, 0)}
	}
	return planes
}

// cleaving is the way a cleaving plane of a part of the seed faces: any way,
// or for planes that stand upright, a way round the level drawn as a share
// of a turn.
func (made lump) cleaving(part int) maths.Vec {
	from := made.from
	if made.broken.upright {
		round := fullTurn * from.draw(part, 0, 0)
		return maths.V(math.Cos(round), 0, math.Sin(round))
	}
	return maths.V(from.swing(part, 0, 0), from.swing(part, 0, 1),
		from.swing(part, 0, 2)).Unit()
}

// squared is a coordinate of the ball swollen toward a box: its size raised to
// a power under one, which leaves zero and one alone.
func squared(coordinate, power float64) float64 {
	return math.Copysign(math.Pow(math.Abs(coordinate), power),
		coordinate)
}

// shaped is where a point of the unit ball goes on this stone, before
// it is stretched to its axes: in by the noise, squared, back behind
// every cleaving plane it is past, and no lower than the flat.
func (made lump) shaped(on maths.Vec, planes []plane) maths.Vec {
	fresh, broken := 1-made.worn, made.broken
	at := on.Scale(grain).Add(offLattice)
	bump := noise.Fractal(made.from.number, at.X, at.Y, at.Z, octaves)
	reach := 1 - broken.roughest*fresh*asShare(bump)
	power := broken.squarest + broken.squareless*made.worn
	lift := broken.flattest + broken.flatless*made.worn
	point := maths.V(squared(on.X, power), squared(on.Y, lift),
		squared(on.Z, power)).Scale(reach)
	for _, cleft := range planes {
		if past := point.Dot(cleft.facing) - cleft.out; past > 0 {
			point = point.Sub(cleft.facing.Scale(past))
		}
	}
	point.Y = math.Max(point.Y, made.flat())
	return point
}

// flat is how low the unit ball of this stone may reach before it is
// cut flat to sit on: sits below its middle for a stone not worn at
// all, and the whole way down, which is no flat, for one worn smooth.
func (made lump) flat() float64 {
	return -1 + (1-sits)*(1-made.worn)
}

// each is two points taken axis by axis: the lesser of each pair, the
// greater, or one times the other.
func each(one, other maths.Vec,
	of func(one, other float64) float64) maths.Vec {
	return maths.V(of(one.X, other.X), of(one.Y, other.Y),
		of(one.Z, other.Z))
}

// times is one number times another.
func times(one, other float64) float64 { return one * other }

// over is one number over another.
func over(one, other float64) float64 { return one / other }

// asShare is noise, which runs either side of zero, as a share: halved and
// lifted, and held to a share where it strays past one.
func asShare(wave float64) float64 {
	return math.Min(1, math.Max(0, (1+wave)/2))
}

// bounds is how far a surface reaches each way: its least and its
// most along each axis.
func bounds(skin surface) (maths.Vec, maths.Vec) {
	inf := math.Inf(1)
	low, high := maths.V(inf, inf, inf), maths.V(-inf, -inf, -inf)
	for _, one := range skin {
		for _, at := range one {
			low = each(low, at, math.Min)
			high = each(high, at, math.Max)
		}
	}
	return low, high
}

// skin is the stone's surface in its own space, cut so finely:
// standing on the ground, its middle over the space's start.
//
// breaking a stone takes some of each axis away, so once shaped it is
// stretched back until it reaches as far each way as the ball it was
// cut from: a stone asked for at three axes has them, however broken,
// and a stone worn smooth is not moved at all.
//
// a plate's upright joints and its flat meet in lines, and the faces
// pushed onto a line, three corners to it, have no area: they are left out.
func (made lump) skin(fine cut) surface {
	planes := made.cleaves()
	round := ball(fine)
	stone := make(surface, len(round))
	for index, one := range round {
		for corner, on := range one {
			stone[index][corner] = made.shaped(on, planes)
		}
	}
	roundLow, roundHigh := bounds(round)
	low, high := bounds(stone)
	back := each(roundHigh.Sub(roundLow), high.Sub(low), over)
	// the unit ball is two across, so half of each axis asked for is
	// one of the ball's
	half := maths.V(inScene(made.long), inScene(made.short),
		inScene(made.middle)).Scale(0.5)
	fit := each(back, half, times)
	middle := low.Add(high).Scale(0.5)
	foot := maths.V(middle.X, low.Y, middle.Z)
	for index, one := range stone {
		for corner, at := range one {
			stone[index][corner] = each(at.Sub(foot), fit, times)
		}
	}
	if made.broken.upright {
		return stone.withArea()
	}
	return stone
}
