// Package titan is the ground of a world: its heights, from a map as far
// as the map goes and made up below that.
//
// A Provider is the map. It is lidar for the land and soundings for the
// sea floor, or whatever else has been measured. It says how high the
// ground is at a place, and how far apart its own samples are there.
// Titan asks it for nothing finer than that.
//
// The world is laid out on grissom's mesh: an icosahedron's faces, each
// split in four as deep as asked. Down to the depth where a triangle's
// edge is as long as the map's samples are apart, every corner's height
// is the map's. That is measured.
//
// Below that depth the map has nothing to say, and titan makes the ground
// up. Each new corner is the middle of an edge. Its height is the mean of
// the edge's two ends, moved up or down by an amount that is fixed by the
// seed and the corner's place and nothing else. That is generated. It is
// arithmetic, with no model in it, and it is not the world: no survey ever
// saw it.
//
// How far a made corner moves comes from what the map showed nearby. Near
// each measured corner titan reads how much the map's heights differ over
// one sample's distance and over two. The first says how steep the ground
// is there. The two together say how jagged: whether a slope carries on
// or turns at every step.
//
// The made ground carries both on downward by one rule, which is titan's
// own: made ground, all its depths together, is about as steep as the map
// showed and no steeper. A corner made on an edge a hundredth as long
// moves a hundredth as far. So a made cliff is rough like the cliff the
// map saw, a made mudflat stays flat, and nothing made at a centimetre
// stands a metre tall.
//
// The made ground shows the mesh it was made on. Each corner is moved
// once and the ground is flat between corners, so the edges of the
// larger triangles stay as faint creases however fine the ground gets.
//
// Every Sample says which it is, and how many of its metres were made.
// Nothing made is ever passed off as measured.
package titan

import (
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/janearc/celestial/grissom/mesh"
)

// The lengths and counts titan is built with.
const (
	// Inchworm is the shortest edge titan splits to, in metres: a
	// centimetre. It's picked: the ground is asked for no finer.
	Inchworm = 0.01
	// around is how many places titan reads round a corner to learn how
	// rough the map is there. It's picked: one each sixth of a turn.
	around = 6
	// twice is the second distance the map is read at, in sample
	// spacings. It's picked: one doubling is what a halving undoes.
	twice = 2
	// most is how many measured corners are kept at once. It's picked.
	most = 1 << 16
	// none is no length, no depth, nothing. It's defined.
	none = 0
	// half is the middle of an edge. It's defined.
	half = 0.5
	// corners is how many corners a triangle has. It's defined.
	corners = 3
	// split is how many edges of a child make one of its parent. It's
	// defined by the mesh.
	split = 2
	// rough is the least a Hurst exponent may be. It's defined: ground
	// whose heights a step apart have nothing to do with each other.
	rough = 0
	// smooth is the most a Hurst exponent may be. It's defined: ground
	// whose slopes carry on unchanged.
	smooth = 1
	// uniform is the variance of a number drawn evenly from nought to
	// one, upside down. It's defined: one over it is twelve.
	uniform = 12
	// hint is how much smooth noise is laid over the made ground, as a
	// share of what a made corner moves by at each depth. It's picked: a
	// quarter breaks up the mesh's creases without becoming the ground.
	hint = 0.25
	// grain is how many of a depth's edges one wave of its noise spans.
	// It's picked: two, so the noise is never finer than the mesh.
	grain = 2
	// eighth and threeHalves and quarter are the cubic's own numbers:
	// the curve through two ends that leaves each at its slope. They're
	// defined by it.
	eighth      = 0.125
	threeHalves = 1.5
	quarter     = 0.25
	// turn is a whole turn in radians.
	turn = 2 * math.Pi
	// halfTurn is half a turn in degrees. It's defined.
	halfTurn = 180
)

// errRefused is what Refused gives.
var errRefused = errors.New("refused")

// Refused is wrapped by each refusal here. A ground with no map or no
// size is refused. So is a place the map has no height for.
func Refused() error { return errRefused }

// Provider is a map of a world's heights. Titan reads the world from it
// and never asks it for more than it has.
type Provider interface {
	// Height is how high the ground is at a place, in metres above the
	// map's own zero, and whether the map has a height there. The sea
	// floor is below nought.
	Height(at mesh.Point) (metres float64, known bool, err error)
	// Finest is how far apart the map's own samples are near a place,
	// in metres. Closer than this the map has nothing to say.
	Finest(at mesh.Point) float64
	// Source is what the map is and where it came from, for whoever
	// reads a Sample and asks.
	Source() string
}

// Kind is where a height came from.
type Kind int

// The kinds.
const (
	// Measured heights are the map's: read at its own samples, and
	// joined flat across the triangles between them.
	Measured Kind = iota
	// Generated heights hold ground the map never saw. Titan made it,
	// by rule, below the map's finest.
	Generated
)

// String is the kind in a word.
func (kind Kind) String() string {
	if kind == Measured {
		return "measured"
	}
	return "generated"
}

// Sample is the ground at one place, and how much of it is real.
type Sample struct {
	height, made, edge float64
	depth              int
	kind               Kind
	source             string
}

// Height is how high the ground is, in metres above the map's zero.
func (sample Sample) Height() float64 { return sample.height }

// Made is how many of those metres titan made up, up or down. A measured
// sample has none.
func (sample Sample) Made() float64 { return sample.made }

// Kind is whether the height is the map's or holds made ground.
func (sample Sample) Kind() Kind { return sample.kind }

// Depth is how many times the mesh was split to reach the triangle the
// place is in.
func (sample Sample) Depth() int { return sample.depth }

// Edge is about how long that triangle's edges are, in metres.
func (sample Sample) Edge() float64 { return sample.edge }

// Source is what the map under the sample says it is.
func (sample Sample) Source() string { return sample.source }

// Ground is a world's surface: a map, the size of the world it maps, and
// a seed for what the map leaves out.
type Ground struct {
	from   Provider
	radius float64
	seed   uint64
	face   float64 // an icosahedron's edge on this world, in metres

	guard sync.Mutex
	kept  map[place]read
}

// New is the ground of a world of a radius in metres, read from a map.
// The seed picks the made ground; the same seed makes the same ground for
// ever. It refuses a ground with no map, or with no size.
func New(from Provider, radius float64, seed uint64) (*Ground, error) {
	if from == nil || !(radius > none) || math.IsInf(radius, none) {
		return nil, fmt.Errorf("%w: a ground has a map, and a radius "+
			"above nought in metres", errRefused)
	}
	first, second, _ := mesh.Faces()[none].Corners()
	arc := math.Acos(vectorOf(first).dot(vectorOf(second)))
	return &Ground{from: from, radius: radius, seed: seed,
		face: arc * radius, kept: map[place]read{}}, nil
}

// Edge is about how long a triangle's edge is at a depth, in metres.
func (ground *Ground) Edge(depth int) float64 {
	return ground.face / math.Exp2(float64(depth))
}

// Depth is the first depth whose triangles have edges no longer than a
// length in metres, and never one finer than Inchworm.
func (ground *Ground) Depth(metres float64) int {
	metres = math.Max(metres, Inchworm)
	depth := int(math.Ceil(math.Log2(ground.face / metres)))
	return max(depth, none)
}

// Kept is how many of the map's corners the ground is holding.
func (ground *Ground) Kept() int {
	ground.guard.Lock()
	defer ground.guard.Unlock()
	return len(ground.kept)
}

// At is the ground at a place, in as much detail as asked: detail is the
// length in metres of the smallest thing the asker wants to see, a
// pixel's footprint say. Detail the map has is measured. Finer than that
// is generated, down to Inchworm and no further.
//
// It refuses a place the map has no height for.
func (ground *Ground) At(at *mesh.Point, detail float64) (*Sample, error) {
	if at == nil {
		return nil, fmt.Errorf("%w: the ground at no place", errRefused)
	}
	deep := ground.Depth(detail)
	toward := vectorOf(*at)
	cell, err := mesh.Locate(at, none)
	if err != nil {
		return nil, err
	}
	first, second, third := cell.Corners()
	var shape [corners]corner
	for i, each := range []mesh.Point{first, second, third} {
		shape[i], err = ground.read(vectorOf(each))
		if err != nil {
			return nil, err
		}
	}
	for depth := range deep {
		shape, err = ground.step(shape, toward, depth+1)
		if err != nil {
			return nil, err
		}
	}
	return ground.sample(shape, toward, deep), nil
}

// step is the child of a triangle that a direction falls in: one split
// down, at a depth. Its new corners are the middles of the triangle's
// edges, each measured if the map reaches that deep there and made if not.
func (ground *Ground) step(shape [corners]corner, toward vector,
	depth int) (into [corners]corner, err error) {
	var middles [corners]corner
	for i := range shape {
		from, to := shape[i], shape[(i+1)%corners]
		middles[i], err = ground.middle(from, to, depth)
		if err != nil {
			return into, err
		}
	}
	ab, bc, ca := middles[0], middles[1], middles[2]
	// one child at each corner, in the corners' order, then the middle:
	// the order grissom's mesh splits in
	children := [][corners]corner{{shape[0], ab, ca}, {ab, shape[1], bc},
		{ca, bc, shape[2]}, {ab, bc, ca}}
	best, margin := none, math.Inf(-1)
	for i, child := range children {
		if inside := toward.margin(child); inside > margin {
			best, margin = i, inside
		}
	}
	return children[best], nil
}

// middle is the corner at the middle of an edge, at a depth.
//
// Where the map's samples are no further apart than this depth's edges,
// the corner is read from the map. Where they are, the corner is made:
// the mean of the edge's ends, moved.
//
// The move's spread is D (L / spacing) sqrt(1 - 2^(2H-2)) / 2^H. D is how
// much the map's heights differ a spacing apart, L is the edge's length
// and H is the Hurst exponent the map showed. The last two factors are
// the midpoint displacement of a self-affine surface: how far the middle
// of an edge strays from the mean of its ends. It is none where H is
// one, since there the slope carries straight on.
//
// The (L / spacing) is titan's rule, and it is picked. A self-affine
// surface would have (L / spacing)^H, which keeps the map's jaggedness
// exactly and so, carried down four or five powers of ten, makes ground
// a hundred times steeper at a centimetre than the map saw at a hundred
// metres. Ground is not like that. Scaling by L gives every depth the
// same steepness.
//
// There are as many depths as halvings between the map's spacing and an
// inchworm, each adding its own, so the spread is divided by the root of
// that count: all the made depths together are about as steep as the map
// showed, and a test holds them to it.
//
// Before it is moved, the middle is not the plain mean of the ends. It
// lies on the cubic curve along the edge that leaves each end at that
// end's slope, and takes the curve's slope there. A measured corner's
// slope is the map's. So between the map's samples the ground bends from
// one to the next and does not lie in flat plates, and a made corner is
// a rise with a top and not a point with creases.
//
// the midpoint displacement is the standard one for fractional Brownian
// surfaces (Fournier, Fussell and Carpenter 1982; Peitgen and Saupe
// 1988). it is written here from memory and was not read from either.
//
// The move depends on the corner's place and the seed alone, so two
// triangles that share an edge make the same corner and the ground has no
// cracks.
func (ground *Ground) middle(from, to corner, depth int) (corner, error) {
	at := from.at.plus(to.at).unit()
	if ground.Edge(depth) >= ground.from.Finest(at.point()) {
		return ground.read(at)
	}
	// the curve along the edge that leaves each end at its own slope:
	// its middle is the mean of the ends, raised by an eighth of the
	// edge times how much the slopes close on each other
	chord := to.at.minus(from.at)
	span := math.Sqrt(chord.dot(chord))
	made := corner{at: at, kind: Generated,
		height:  half * (from.height + to.height),
		made:    half * (from.made + to.made),
		differs: half * (from.differs + to.differs),
		hurst:   half * (from.hurst + to.hurst),
		spacing: half * (from.spacing + to.spacing),
		slope:   from.slope.plus(to.slope).scaled(half)}
	if span > none {
		along := chord.scaled(1 / span)
		metres := span * ground.radius
		leaving, arriving := from.slope.dot(along), to.slope.dot(along)
		made.height += eighth * metres * (leaving - arriving)
		// and the curve's own slope at its middle, along the edge
		middle := threeHalves*(to.height-from.height)/metres -
			quarter*(leaving+arriving)
		made.slope = made.slope.plus(
			along.scaled(middle - half*(leaving+arriving)))
	}
	if made.spacing > none {
		move := ground.spread(made, depth) * ground.drawn(at)
		made.height += move
		made.made += move
	}
	return made, nil
}

// spread is how far, on average, a corner made at a depth is moved: the
// rule middle's comment sets out, from what the map showed near it.
func (ground *Ground) spread(near corner, depth int) float64 {
	if !(near.spacing > none) {
		return none
	}
	long := ground.Edge(depth - 1) // the edge whose middle is made
	over := near.differs * long / near.spacing
	// every depth from the map's to an inchworm's adds its own slope,
	// so each is given its share of the map's
	depths := math.Max(math.Log2(near.spacing/Inchworm), 1)
	// a Hurst exponent carried between corners can land a hair past
	// one, where there is nothing to take a root of
	strays := math.Max(1-math.Exp2(split*near.hurst-split), none)
	return over * math.Sqrt(strays) /
		(math.Exp2(near.hurst) * math.Sqrt(depths))
}

// drawn is a number for a place, with a mean of nought and a spread of
// one, the same for a seed and a place for ever.
func (ground *Ground) drawn(at vector) float64 {
	even := unit(at.hashed(ground.seed))
	return math.Sqrt(uniform) * (even - half)
}

// sample is the ground at a direction inside a triangle: its corners'
// heights, weighed by how near each the direction is. The triangle is
// flat between its corners.
func (ground *Ground) sample(shape [corners]corner, toward vector,
	depth int) *Sample {
	var weights [corners]float64
	var whole float64
	for i := range shape {
		// a corner's weight is the area of the triangle the direction
		// makes with the other two. the two are measured from the
		// direction first: three directions a centimetre apart on a
		// world are alike to nine digits, and their plain triple
		// product has none left.
		next := shape[(i+1)%corners].at.minus(toward)
		last := shape[(i+2)%corners].at.minus(toward)
		weights[i] = math.Max(toward.dot(next.cross(last)), none)
		whole += weights[i]
	}
	found := &Sample{depth: depth, edge: ground.Edge(depth),
		source: ground.from.Source()}
	var here corner
	for i, each := range shape {
		share := 1.0 / corners
		if whole > none {
			share = weights[i] / whole
		}
		found.height += share * each.height
		found.made += share * each.made
		here.differs += share * each.differs
		here.hurst += share * each.hurst
		here.spacing += share * each.spacing
		if each.kind == Generated {
			found.kind = Generated
		}
	}
	if found.kind == Generated {
		over := ground.noise(here, toward, depth)
		found.height += over
		found.made += over
	}
	return found
}

// noise is a hint of smooth noise over the made ground at a direction:
// one layer for each made depth down to the one asked for, each a share
// of what a corner made at that depth moves by, in waves a couple of that
// depth's edges long.
//
// It is there for the look of the thing. A corner is moved once and the
// ground between corners is flat, so the mesh shows through as creases;
// noise that knows nothing of the mesh breaks them up. It is made ground
// like the rest: counted in Made, laid only below the map's finest, and
// fixed by the seed and the place.
//
// Each layer belongs to a depth and not to the asking, so looking closer
// adds finer layers and leaves the coarser ones as they were.
func (ground *Ground) noise(near corner, toward vector, deep int) float64 {
	if !(near.spacing > none) {
		return none
	}
	var over float64
	for depth := 1; depth <= deep; depth++ {
		if ground.Edge(depth) >= near.spacing {
			continue
		}
		waves := ground.radius / (grain * ground.Edge(depth))
		over += hint * ground.spread(near, depth) *
			smoothed(ground.seed+uint64(depth), toward.scaled(waves))
	}
	return over
}
