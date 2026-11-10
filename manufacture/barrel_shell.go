package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// barrelShell closes an elliptical half-barrel with open air below its roof.
// Its inner and outer skins, end arches and feet form one watertight part.
type barrelShell struct{ radius, height, wall, depth float64 }

// edge shares exact foot coordinates between the skins and the end closures.
func (shell barrelShell) edge(index, sides int, inner bool,
	z float64,
) maths.Vec {
	radius, height := shell.radius, shell.height
	if inner {
		radius -= shell.wall
		height -= shell.wall
	}
	if index == spaceZero {
		return maths.V(radius, spaceZero, z)
	}
	if index == sides {
		return maths.V(-radius, spaceZero, z)
	}
	angle := math.Pi * float64(index) / float64(sides)
	sine, cosine := math.Sincos(angle)
	return maths.V(radius*cosine, height*sine, z)
}

// skin joins all four surfaces at each arch interval.
func (shell barrelShell) skin(fine cut) surface {
	sides := fine.around
	var skin surface
	for index := range sides {
		next := index + spaceOne
		outerNear := shell.edge(index, sides, false, spaceZero)
		outerNext := shell.edge(next, sides, false, spaceZero)
		outerFar := shell.edge(index, sides, false, shell.depth)
		outerAhead := shell.edge(next, sides, false, shell.depth)
		innerNear := shell.edge(index, sides, true, spaceZero)
		innerNext := shell.edge(next, sides, true, spaceZero)
		innerFar := shell.edge(index, sides, true, shell.depth)
		innerAhead := shell.edge(next, sides, true, shell.depth)
		skin = quad(skin, outerNear, outerNext, outerAhead, outerFar)
		skin = quad(skin, innerNear, innerFar, innerAhead, innerNext)
		skin = quad(skin, outerNear, innerNear, innerNext, outerNext)
		skin = quad(skin, outerFar, outerAhead, innerAhead, innerFar)
	}
	return shell.feet(skin, sides)
}

// feet close the shell's thickness at both spring points of the arch.
func (shell barrelShell) feet(skin surface, sides int) surface {
	rightNear := shell.edge(spaceZero, sides, false, spaceZero)
	rightFar := shell.edge(spaceZero, sides, false, shell.depth)
	rightInside := shell.edge(spaceZero, sides, true, spaceZero)
	rightInsideFar := shell.edge(spaceZero, sides, true, shell.depth)
	skin = quad(skin, rightNear, rightFar, rightInsideFar, rightInside)
	leftNear := shell.edge(sides, sides, false, spaceZero)
	leftFar := shell.edge(sides, sides, false, shell.depth)
	leftInside := shell.edge(sides, sides, true, spaceZero)
	leftInsideFar := shell.edge(sides, sides, true, shell.depth)
	return quad(skin, leftNear, leftInside, leftInsideFar, leftFar)
}
