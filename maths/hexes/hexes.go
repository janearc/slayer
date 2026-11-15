// Package hexes tiles supported surfaces with hexagons and maps positions to
// tiles. Refining a tiling preserves existing centres so attached details
// remain in place as resolution changes.
package hexes

import "github.com/janearc/slayer/maths"

// Tile is one tile of a tiling. On a Sphere it is a lattice corner of face
// Face; on a Band or a Flat, Face is zero and Column and Row are the column and
// the row.
type Tile struct {
	//ninja:vow gate a value, whose fields are what it is
	Face, Column, Row int
}

// Tiling is a surface in hexagons. Points are in the surface's own
// space: directions for a sphere, positions for a band or a plane.
type Tiling interface {
	// Nearest is the tile a point is in.
	Nearest(point maths.Vec) Tile
	// Centre is the middle of a tile.
	Centre(tile Tile) maths.Vec
	// Neighbours is the tiles sharing an edge with this one, in order of
	// ID: six, five at a sphere's pentagons, fewer at a plane's edge.
	Neighbours(tile Tile) []Tile
	// Edge is how far a point is from the edge of a tile: zero on the edge,
	// growing toward the middle, less than zero outside the tile.
	Edge(point maths.Vec, tile Tile) float64
	// ID is a number for the tile, different for every tile of the tiling
	// and the same forever, for hashing what is on it.
	ID(tile Tile) int64
	// Spacing is about how far apart neighbouring middles are, in the
	// units points are in.
	Spacing() float64
	// Tiles is every tile the tiling holds, in order of ID. A sphere's
	// are all of them; a band's and a plane's are the ones inside their
	// extent. It is for coarse tilings.
	Tiles() []Tile
	// Finer is the next rung down, with every middle of this one.
	Finer() Tiling
	// Distance is how far apart two points of the surface are, along it,
	// in the units points are in: radians on a sphere, lengths on a band
	// or a plane.
	Distance(one, other maths.Vec) float64
	// Latitude is how far a point is from the surface's equator toward its
	// poles, zero to one. A sphere's poles lie along north, which is the
	// world's; a band's are its rims; a plane's lie along north at the edge
	// of its extent.
	//
	// So a plane whose north points straight up has no latitude at all.
	Latitude(point, north maths.Vec) float64
	// Somewhere is a point of the surface from three numbers in [0, 1),
	// for scattering things over it, as a world scatters its plates.
	Somewhere(shareA, shareB, shareC float64) maths.Vec
}
