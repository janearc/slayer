package hexes

import (
	"math"
	"sort"

	"github.com/janearc/slayer/maths"
)

// Sphere is the unit sphere with each icosahedron face split Splits times:
// 10*Splits^2+2 tiles, about Spacing apart. The zero Sphere is split once, the
// bare icosahedron's twelve pentagons. It holds nothing but Splits, so a
// tiling of any resolution costs nothing to make.
//
// A tile is named by the lattice corner at its middle: on face Face, Column
// steps toward the face's second corner and Row toward its third, with
// Column+Row at most Splits. A corner shared by several faces is named on the lowest
// numbered of them, so every tile has one name.
//
// Points are directions: Nearest takes any vector and reads its direction,
// Centre is a unit vector, and Spacing and Edge are chords of the unit
// sphere.
type Sphere struct {
	//ninja:vow gate a value, whose fields are what it is
	Splits int
}

// split is Splits, and one when Splits is below one.
func (sphere Sphere) split() int {
	if sphere.Splits < 1 {
		return 1
	}
	return sphere.Splits
}

// edge is the angle an icosahedron's edge subtends at its middle.
var edge = math.Atan(2)

// Spacing is about how far apart neighbouring middles are, in radians:
// the icosahedron's edge over Splits. Measured, neighbours are between 0.8 and
// 1.2 of it, closest near the pentagons and furthest mid-face.
func (sphere Sphere) Spacing() float64 {
	return edge / float64(sphere.split())
}

// finest is the most a sphere is split: its count of tiles, ten Splits squared
// and two, is the most an int64 holds with room to spare.
const finest = 500_000_000

// SphereFor is the sphere tiling whose tiles are about radians apart. a spacing
// that is no number is the coarsest sphere, and one of zero or less, or finer
// than finest splits, the finest.
func SphereFor(radians float64) Sphere {
	if math.IsNaN(radians) {
		return Sphere{Splits: 1}
	}
	if radians <= edge/finest {
		return Sphere{Splits: finest}
	}
	return Sphere{Splits: max(1, int(math.Round(edge/radians)))}
}

// Finer is the next rung down: split twice as often. Every middle of this
// tiling is a middle of the finer one, so a thing placed on a tile stays
// put when the rung changes.
//
// past half of finest it stops at finest, and a middle is no longer kept.
// a Sphere made by hand past finest does not count its tiles, which
// overflow.
func (sphere Sphere) Finer() Tiling {
	return Sphere{Splits: min(finest, 2*sphere.split())}
}

// Count is how many tiles there are.
func (sphere Sphere) Count() int {
	split := sphere.split()
	return 10*split*split + 2
}

// ID is a number for the tile, different for every tile of the tiling and the
// same forever, for hashing what is on it.
func (sphere Sphere) ID(tile Tile) int64 {
	split := int64(sphere.split())
	stepI, stepJ := int64(tile.Column), int64(tile.Row)
	perFace := (split + 1) * (split + 2) / 2
	return int64(tile.Face)*perFace +
		stepI*(split+1) - stepI*(stepI-1)/2 + stepJ
}

// Centre is the middle of a tile: a unit direction.
func (sphere Sphere) Centre(tile Tile) maths.Vec {
	return sphere.corner(tile.Face, tile.Column, tile.Row)
}

// Nearest is the tile a direction is in.
func (sphere Sphere) Nearest(point maths.Vec) Tile {
	point = point.Unit()
	split := sphere.split()
	side := face(point)
	_, secondWeight, thirdWeight := bary(side, point)
	looseI := secondWeight * float64(split)
	looseJ := thirdWeight * float64(split)
	floorI, floorJ := int(math.Floor(looseI)), int(math.Floor(looseJ))
	// the small triangle the direction crosses points up or down; its
	// three corners are the candidates
	candidates := [3][2]int{{floorI, floorJ}, {floorI + 1, floorJ},
		{floorI, floorJ + 1}}
	if looseI-float64(floorI)+looseJ-float64(floorJ) > 1 {
		candidates = [3][2]int{{floorI + 1, floorJ},
			{floorI, floorJ + 1}, {floorI + 1, floorJ + 1}}
	}
	best, bestI, bestJ := math.Inf(1), 0, 0
	for _, candidate := range candidates {
		stepI, stepJ := clamp(candidate[0], candidate[1], split)
		gap := apart(point, sphere.corner(side, stepI, stepJ))
		if gap < best {
			best, bestI, bestJ = gap, stepI, stepJ
		}
	}
	tile := sphere.walk(point, side, bestI, bestJ, best)
	return sphere.settle(point, tile)
}

// apart is how far two unit directions are, as the square of the chord. A dot
// product is one less a hair for every neighbour at a fine rung and cannot tell
// them apart; the chord keeps its digits.
func apart(one, other maths.Vec) float64 {
	gap := one.Sub(other)
	return gap.Dot(gap)
}

// walk steps to a neighbouring corner of the same face while one is nearer,
// and stops at the nearest. In practice it takes no step at all: none in
// 320,000 points tried; the bound is for safety.
func (sphere Sphere) walk(point maths.Vec, side, stepI, stepJ int,
	best float64) Tile {
	split := sphere.split()
	for range 8 {
		nextI, nextJ, moved := stepI, stepJ, false
		for _, shift := range steps {
			tryI, tryJ := stepI+shift[0], stepJ+shift[1]
			if tryI < 0 || tryJ < 0 || tryI+tryJ > split {
				continue
			}
			gap := apart(point, sphere.corner(side, tryI, tryJ))
			if gap < best {
				best, nextI, nextJ, moved = gap, tryI, tryJ, true
			}
		}
		if !moved {
			break
		}
		stepI, stepJ = nextI, nextJ
	}
	return sphere.canonical(side, stepI, stepJ)
}

// settle is one more look, across faces: a corner on the edge of a face
// has neighbours on the next face, and one of them may be nearer still.
func (sphere Sphere) settle(point maths.Vec, tile Tile) Tile {
	if sphere.inside(tile) {
		return tile
	}
	best := apart(point, sphere.Centre(tile))
	var room [6]Tile
	for _, neighbour := range room[:sphere.near(tile, &room)] {
		if gap := apart(point, sphere.Centre(neighbour)); gap < best {
			best, tile = gap, neighbour
		}
	}
	return tile
}

// steps are the six lattice directions from a corner.
var steps = [6][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, -1}, {-1, 1}}

// clamp keeps a corner on its face: the steps at least zero, their sum
// at most the split.
func clamp(stepI, stepJ, split int) (int, int) {
	stepI, stepJ = max(stepI, 0), max(stepJ, 0)
	if stepI+stepJ > split {
		stepJ = max(split-stepI, 0)
		stepI = split - stepJ
	}
	return stepI, stepJ
}

// inside is whether a tile's corner is strictly inside its face, where it
// belongs to that face alone.
func (sphere Sphere) inside(tile Tile) bool {
	return tile.Column > 0 && tile.Row > 0 && tile.Column+tile.Row < sphere.split()
}

// corner is a lattice corner of a face, pushed out to the sphere.
func (sphere Sphere) corner(side, stepI, stepJ int) maths.Vec {
	split := sphere.split()
	first := verts[faces[side][0]]
	second := verts[faces[side][1]]
	third := verts[faces[side][2]]
	return first.Scale(float64(split - stepI - stepJ)).
		Add(second.Scale(float64(stepI))).
		Add(third.Scale(float64(stepJ))).Unit()
}

// canonical is the one name of a corner: itself when it is inside its
// face, and otherwise the same corner on the lowest numbered face it is on.
func (sphere Sphere) canonical(side, stepI, stepJ int) Tile {
	tile := Tile{side, stepI, stepJ}
	if sphere.inside(tile) {
		return tile
	}
	direction := sphere.corner(side, stepI, stepJ)
	for other := 0; other < side; other++ {
		if otherI, otherJ, ok := sphere.on(other, direction); ok {
			return Tile{other, otherI, otherJ}
		}
	}
	return tile
}

// on is the lattice corner of a face that a direction is, if it is one.
func (sphere Sphere) on(side int, direction maths.Vec) (stepI, stepJ int,
	ok bool) {
	if direction.Dot(normals[side]) < 0.7 {
		return 0, 0, false // not near that face at all
	}
	split := float64(sphere.split())
	firstWeight, secondWeight, thirdWeight := bary(side, direction)
	const slack = 1e-9
	if firstWeight < -slack || secondWeight < -slack ||
		thirdWeight < -slack {
		return 0, 0, false
	}
	looseI, looseJ := secondWeight*split, thirdWeight*split
	stepI, stepJ = int(math.Round(looseI)), int(math.Round(looseJ))
	if math.Abs(looseI-float64(stepI)) > 1e-6 ||
		math.Abs(looseJ-float64(stepJ)) > 1e-6 {
		return 0, 0, false
	}
	return stepI, stepJ, true
}

// near fills the room with the tile's neighbours, five or six, and says how
// many: the lattice neighbours on its own face, and on every other face
// the corner is on.
func (sphere Sphere) near(tile Tile, room *[6]Tile) int {
	count := 0
	add := func(side, stepI, stepJ int) {
		if stepI < 0 || stepJ < 0 || stepI+stepJ > sphere.split() {
			return
		}
		named := sphere.canonical(side, stepI, stepJ)
		for _, have := range room[:count] {
			if have == named {
				return
			}
		}
		if count < len(room) {
			room[count] = named
			count++
		}
	}
	for _, shift := range steps {
		add(tile.Face, tile.Column+shift[0], tile.Row+shift[1])
	}
	if sphere.inside(tile) {
		return count
	}
	direction := sphere.Centre(tile)
	for other := range faces {
		if other == tile.Face {
			continue
		}
		if otherI, otherJ, ok := sphere.on(other, direction); ok {
			for _, shift := range steps {
				add(other, otherI+shift[0], otherJ+shift[1])
			}
		}
	}
	return count
}

// Neighbours is the tiles sharing an edge with this one: five for a
// pentagon, six for a hexagon, in order of ID.
func (sphere Sphere) Neighbours(tile Tile) []Tile {
	var room [6]Tile
	kept := append([]Tile(nil), room[:sphere.near(tile, &room)]...)
	sort.Slice(kept, func(left, right int) bool {
		return sphere.ID(kept[left]) < sphere.ID(kept[right])
	})
	return kept
}

// Edge is how far a direction is from the edge of a tile, as a chord of the
// unit sphere: zero on the edge, growing toward the middle, and less than zero
// outside the tile. The edge with a neighbour is the plane square to the chord
// between the two middles, halfway along it.
func (sphere Sphere) Edge(point maths.Vec, tile Tile) float64 {
	point = point.Unit()
	middle := sphere.Centre(tile)
	var room [6]Tile
	best := math.Inf(1)
	for _, neighbour := range room[:sphere.near(tile, &room)] {
		nearMiddle := sphere.Centre(neighbour)
		chord := middle.Sub(nearMiddle)
		between := middle.Add(nearMiddle).Scale(0.5)
		best = math.Min(best,
			point.Sub(between).Dot(chord)/chord.Len())
	}
	return best
}

// Tiles is every tile, in order of ID. It is 10N^2+2 long, so it is for
// coarse tilings: a world's plates are decided on one, a sky's sample is
// not.
func (sphere Sphere) Tiles() []Tile {
	split := sphere.split()
	kept := make([]Tile, 0, sphere.Count())
	for side := range faces {
		for stepI := 0; stepI <= split; stepI++ {
			for stepJ := 0; stepI+stepJ <= split; stepJ++ {
				named := sphere.canonical(side, stepI, stepJ)
				if named == (Tile{side, stepI, stepJ}) {
					kept = append(kept, named)
				}
			}
		}
	}
	return kept
}

// face is which face a direction looks through: the one whose middle is
// nearest, which for an icosahedron is the one the direction crosses.
func face(direction maths.Vec) int {
	best, at := -2.0, 0
	for side, normal := range normals {
		if dot := direction.Dot(normal); dot > best {
			best, at = dot, side
		}
	}
	return at
}

// bary is where a direction crosses the plane of a face, as weights on its
// three corners: all three from zero to one inside the face.
func bary(side int, direction maths.Vec) (firstWeight, secondWeight,
	thirdWeight float64) {
	held := planes[side]
	first := verts[faces[side][0]]
	landing := direction.Scale(
		normals[side].Dot(first) / normals[side].Dot(direction))
	fromFirst := landing.Sub(first)
	d20, d21 := fromFirst.Dot(held.v0), fromFirst.Dot(held.v1)
	secondWeight = (held.d11*d20 - held.d01*d21) / held.denom
	thirdWeight = (held.d00*d21 - held.d01*d20) / held.denom
	return 1 - secondWeight - thirdWeight, secondWeight, thirdWeight
}

// plane is what bary needs of a face, worked out once.
type plane struct {
	v0, v1               maths.Vec
	d00, d01, d11, denom float64
}

// The icosahedron: twelve corners on the unit sphere, twenty faces, each
// face's outward middle, and what bary needs of each.
var (
	verts   [12]maths.Vec
	faces   [20][3]int
	normals [20]maths.Vec
	planes  [20]plane
)

// icosahedron fills the tables.
func init() {
	golden := (1 + math.Sqrt(5)) / 2
	raw := [12]maths.Vec{
		{X: -1, Y: golden}, {X: 1, Y: golden},
		{X: -1, Y: -golden}, {X: 1, Y: -golden},
		{Y: -1, Z: golden}, {Y: 1, Z: golden},
		{Y: -1, Z: -golden}, {Y: 1, Z: -golden},
		{X: golden, Z: -1}, {X: golden, Z: 1},
		{X: -golden, Z: -1}, {X: -golden, Z: 1},
	}
	for place, vertex := range raw {
		verts[place] = vertex.Unit()
	}
	faces = [20][3]int{
		{0, 11, 5}, {0, 5, 1}, {0, 1, 7}, {0, 7, 10}, {0, 10, 11},
		{1, 5, 9}, {5, 11, 4}, {11, 10, 2}, {10, 7, 6}, {7, 1, 8},
		{3, 9, 4}, {3, 4, 2}, {3, 2, 6}, {3, 6, 8}, {3, 8, 9},
		{4, 9, 5}, {2, 4, 11}, {6, 2, 10}, {8, 6, 7}, {9, 8, 1},
	}
	for side, triangle := range faces {
		first := verts[triangle[0]]
		second := verts[triangle[1]]
		third := verts[triangle[2]]
		normals[side] = first.Add(second).Add(third).Unit()
		v0, v1 := second.Sub(first), third.Sub(first)
		d00, d01, d11 := v0.Dot(v0), v0.Dot(v1), v1.Dot(v1)
		planes[side] = plane{v0, v1, d00, d01, d11,
			d00*d11 - d01*d01}
	}
}

// Distance is the angle between two directions one unit long, in radians, by
// the arc cosine of their dot: below about 1.5e-8 it is zero, so at the finest
// rungs neighbours are no distance apart, and a direction that is not one unit
// long is not measured, its dot held to one.
func (sphere Sphere) Distance(one, other maths.Vec) float64 {
	return math.Acos(math.Max(-1, math.Min(1, one.Dot(other))))
}

// Latitude is how far a direction lies toward either pole along north: zero on
// the equator, one at a pole, for a direction and a north one unit long. others
// are not made so, and give more than one.
func (sphere Sphere) Latitude(direction, north maths.Vec) float64 {
	return math.Abs(direction.Dot(north))
}

// Somewhere is the direction (2a-1, 2b-1, 2c-1): the sphere's points
// from three even numbers, thicker toward the cube's corners, which is
// how a world has scattered its plates since its first picture. the
// cube's middle, which is no direction, is up.
func (sphere Sphere) Somewhere(shareX, shareY, shareZ float64) maths.Vec {
	at := maths.V(shareX*2-1, shareY*2-1, shareZ*2-1)
	if at == (maths.Vec{}) {
		return maths.Up
	}
	return at.Unit()
}
