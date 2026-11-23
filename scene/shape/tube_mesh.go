package shape

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// TubeChordShare is how far a flat side of a tube's mesh may lie inside
// the true circle, as a share of the circle's radius. A side of n to the
// circle sags by r*(1-cos(pi/n)) at its middle, and the shading hides
// the sag everywhere but at the outline. There the sag is a step of one
// pixel when the radius spans 1024 pixels, which is a tube that fills a
// frame two thousand pixels wide.
//
// Nothing drawn at a size slayer makes
// shows a step, so one share serves every tube. Its value is read from
// picks.yaml.
func TubeChordShare() float64 { return pick("tube-chord-share") }

// tubeLeastSides is the fewest sides a ring can have and still be a
// ring: the fewest corners of a closed flat figure.
const tubeLeastSides = 3 // defined: a triangle

// tubeMostSides is the most sides a ring may have: a finer share would only
// grow the mesh without bound. It is chosen, in picks.yaml.
var tubeMostSides = int(pick("tube-most-sides"))

// The shapes a ring and its faces are made of.
const (
	turn           = 2 * math.Pi // defined: a full circle in radians
	cornersOfAFace = 3           // defined: a triangle has three corners
	secondCorner   = 1           // defined: where a face names its second corner
	thirdCorner    = 2           // defined: where a face names its third corner
	facingOut      = unity       // defined: a normal part along the radius, out
	facingIn       = -unity      // defined: a normal part along the radius, in
	facingUp       = unity       // defined: a normal part along the axis, up
	facingDown     = -unity      // defined: a normal part along the axis, down
	facingNowhere  = nought      // defined: a normal with no part that way
)

// triangleCorners is one face: three indices into the mesh's corners.
type triangleCorners = [cornersOfAFace]int

// sidesNeeded is how many sides a ring needs for its sides to sag no more
// than share of its radius, whole, or not a number or infinite when share
// is not between nought and one or is so small the circle has no flat
// side fine enough. The sag is a share of the radius, so the count does
// not depend on the radius.
func sidesNeeded(share float64) float64 {
	return math.Ceil(math.Pi / math.Acos(unity-share))
}

// tubeSides is how many sides a ring needs for its sides to sag no more
// than share of its radius. It is for a share that checkChordShare has
// passed.
func tubeSides(share float64) int {
	needed := int(sidesNeeded(share))
	return max(needed, tubeLeastSides)
}

// checkChordShare refuses a share that is not between nought and one, not
// a number, or so small a ring would need more than tubeMostSides.
func checkChordShare(share float64) error {
	if !(share > nought && share < unity) {
		return errors.New("a tube's chord share is between 0 and 1")
	}
	if !(sidesNeeded(share) <= float64(tubeMostSides)) {
		return fmt.Errorf("a tube's chord share of %g needs more than %d "+
			"sides a ring", share, tubeMostSides)
	}
	return nil
}

// tubeBuilder is a tube's mesh as it grows: rings of corners with their
// normals, and the triangles that join them.
type tubeBuilder struct {
	sides    int
	vertices []maths.Vec
	normals  []maths.Vec
	faces    []triangleCorners
}

// ring adds one ring of corners at a radius and a height and gives the
// index of the first. Each corner faces radial parts along the radius
// and axial parts along the axis, which are one or zero between them.
func (grown *tubeBuilder) ring(radius, height, radial, axial float64) int {
	first := len(grown.vertices)
	for column := range grown.sides {
		angle := turn * float64(column) / float64(grown.sides)
		across, along := math.Cos(angle), math.Sin(angle)
		grown.vertices = append(grown.vertices,
			maths.V(radius*across, height, radius*along))
		grown.normals = append(grown.normals,
			maths.V(radial*across, axial, radial*along))
	}
	return first
}

// heart adds the inner edge of a cap: a ring where the tube has a hole,
// and the one corner on the axis where it is solid.
func (grown *tubeBuilder) heart(inner, height, axial float64) int {
	if inner > nought {
		return grown.ring(inner, height, facingNowhere, axial)
	}
	first := len(grown.vertices)
	grown.vertices = append(grown.vertices, maths.V(nought, height, nought))
	grown.normals = append(grown.normals, maths.V(nought, axial, nought))
	return first
}

// wall joins two rings, one over the other, into sides. Outward wound
// they face away from the axis; otherwise they face toward it.
func (grown *tubeBuilder) wall(low, high int, outward bool) {
	for column := range grown.sides {
		next := (column + unity) % grown.sides
		foot, head := low+column, high+column
		headNext, footNext := high+next, low+next
		if outward {
			grown.faces = append(grown.faces, triangleCorners{foot, head, headNext},
				triangleCorners{foot, headNext, footNext})
		} else {
			grown.faces = append(grown.faces, triangleCorners{foot, headNext, head},
				triangleCorners{foot, footNext, headNext})
		}
	}
}

// lid closes an end with a flat annulus from an outer ring to an inner
// one, or to a single corner on the axis when there is no hole. Wound
// to face up when up is set and down otherwise.
func (grown *tubeBuilder) lid(outer, inner int, hollow, up bool) {
	for column := range grown.sides {
		next := (column + unity) % grown.sides
		edge, edgeNext := outer+column, outer+next
		var made []triangleCorners
		if hollow {
			made = []triangleCorners{{edge, inner + next, edgeNext},
				{edge, inner + column, inner + next}}
		} else {
			made = []triangleCorners{{inner, edgeNext, edge}}
		}
		for _, face := range made {
			if !up {
				face[secondCorner], face[thirdCorner] = face[thirdCorner],
					face[secondCorner]
			}
			grown.faces = append(grown.faces, face)
		}
	}
}

// Mesh is the tube as a closed, smooth-shaded triangle mesh in the
// tube's own space: the outer wall, the inner wall if the tube has a
// hole, and a flat ring closing each end, as the tube's own surface is
// closed. The sides of every ring are enough that none lies inside the
// true circle by more than share of the circle's radius; TubeChordShare
// is the share slayer draws with.
//
// A share that is not a number, is not between 0 and 1, or would need more
// than tubeMostSides sides to a ring is refused, as is a tube with no wall
// or no length.
//
// The wall and the ends do not share corners, so each corner has its own
// normal and the edge between them stays sharp. A tube whose wall is
// wider than its radius has no hole and its ends are discs.
func (tube Tube) Mesh(share float64) (*Mesh, error) {
	if !tube.meshable() {
		return nil, fmt.Errorf("a tube needs a wall and a length: %+v", tube)
	}
	if err := checkChordShare(share); err != nil {
		return nil, err
	}
	outer, inner := tube.Radius+tube.Thick*half, tube.Radius-tube.Thick*half
	low, high, hollow := -tube.Height*half, tube.Height*half, inner > nought
	grown := tubeBuilder{sides: tubeSides(share)}
	grown.wall(grown.ring(outer, low, facingOut, facingNowhere),
		grown.ring(outer, high, facingOut, facingNowhere), true)
	if hollow {
		grown.wall(grown.ring(inner, low, facingIn, facingNowhere),
			grown.ring(inner, high, facingIn, facingNowhere), false)
	}
	top := grown.ring(outer, high, facingNowhere, facingUp)
	grown.lid(top, grown.heart(inner, high, facingUp), hollow, true)
	bottom := grown.ring(outer, low, facingNowhere, facingDown)
	grown.lid(bottom, grown.heart(inner, low, facingDown), hollow, false)
	return NewMesh(grown.vertices, grown.faces, grown.normals, true)
}
