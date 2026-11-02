// Package forest builds ground meshes and seeded plant placements. Ground uses
// Slayer’s mesh representation and can follow the ordinary GPU geometry path.
// The same seed reproduces the same placements.
package forest

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/mesh"
)

// mostSide is the most cells along a ground's edge, or places along a
// forest's: what is made is side squared, so an unbounded side could ask
// for more memory than there is.
const mostSide = 256

// what the forest refuses
var (
	errGround = errors.New("forest: a ground needs from one to " +
		fmt.Sprint(mostSide) + " cells a side and a finite size " +
		"above nought")
	errPlaces = errors.New("forest: places need from one to " +
		fmt.Sprint(mostSide) + " to a side, a finite spacing above " +
		"nought and a jitter from nought to one")
)

// finite is whether a number is neither infinite nor nan.
func finite(number float64) bool {
	return !math.IsInf(number, 0) && !math.IsNaN(number)
}

// Ground is flat ground, side cells along each edge and size across, with
// its corner at the origin and two triangles to a cell, each facing up.
func Ground(side int, size float64) (*mesh.Mesh, error) {
	if side < 1 || side > mostSide || !(size > 0) || !finite(size) {
		return nil, errGround
	}
	across := side + 1
	points := make([]maths.Vec, 0, across*across)
	for row := range across {
		for column := range across {
			points = append(points, maths.V(
				size*float64(column)/float64(side), 0,
				size*float64(row)/float64(side)))
		}
	}
	faces := make([][3]int, 0, side*side*2)
	for row := range side {
		for column := range side {
			near := row*across + column
			far := near + across
			faces = append(faces, [3]int{near, far + 1, near + 1},
				[3]int{near, far, far + 1})
		}
	}
	return mesh.New(points, faces, nil, false)
}

// placement is where one piece of foliage stands: its place on the
// ground, how far it is turned about the up axis, and how large it is
// as a share of the usual size. it is read only through its methods.
type placement struct {
	at   maths.Vec
	turn float64
	size float64
}

// At is where it stands.
func (placed placement) At() maths.Vec { return placed.at }

// Turn is how far it is turned about the up axis, in radians.
func (placed placement) Turn() float64 { return placed.turn }

// Size is how large it is, as a share of the usual size.
func (placed placement) Size() float64 { return placed.size }

// Placements is a grid of side by side places, spacing apart, each
// moved off its grid point by up to half of jitter times the spacing,
// turned any way round, and sized between four fifths and six fifths.
// a seed and a grid give the same places every time.
func Placements(seed uint64, side int, spacing,
	jitter float64) ([]placement, error) {
	if side < 1 || side > mostSide || !(spacing > 0) || !finite(spacing) ||
		!(jitter >= 0 && jitter <= 1) {
		return nil, errPlaces
	}
	placed := make([]placement, 0, side*side)
	for index := range side * side {
		draw := func(which int64) float64 {
			return noise.Hash(seed, int64(index)*4+which)
		}
		column, row := float64(index%side), float64(index/side)
		placed = append(placed, placement{
			at: maths.V((column+(draw(0)-0.5)*jitter)*spacing, 0,
				(row+(draw(1)-0.5)*jitter)*spacing),
			turn: draw(2) * 2 * math.Pi,
			size: 0.8 + 0.4*draw(3),
		})
	}
	return placed, nil
}
