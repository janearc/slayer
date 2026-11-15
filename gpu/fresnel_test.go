//go:build draw

package gpu

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// lapped is two balls of water lapped over one another before a glowing
// backdrop. A ray that bends into the first meets the second's surface
// from within, at angles steep enough to be past the critical angle
// for a good share of the picture, so the chain turns rays back whole
// and Fresnel's share is read from both sides of a surface.
func lapped() (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	water := &thing.Material{Name: "water", Gloss: 1, Index: 1.33,
		Through: 1}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: water})
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.7},
		Material: water, Motion: at(0.6, 0.1, -0.2)})
	wall := &thing.Material{Name: "wall", Glow: maths.RGB(0.5, 0.7, 0.9)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 8},
		Material: wall, Motion: at(0, 0, -14)})
	from := maths.V(0.3, 0.2, 4)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
}

// the gpu turns rays back whole past the critical angle, and reads
// Fresnel's share by the side a ray arrives from, as slayer's tracer does
func TestTheGPUReflectsWithinWaterAsSlayerDoes(t *testing.T) {
	world, eye := lapped()
	against(t, world, eye)
}

// bubbled is a ball of index 0.75, the inside of a bubble in water,
// between two glowing walls, one behind it and one behind the eye. From
// the air its rim is past the critical angle: the sine of the
// incidence is more than 0.75, so the rim reflects whole and is a
// mirror of the wall behind the eye.
func bubbled() (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	bubble := &thing.Material{Name: "bubble", Gloss: 1, Index: 0.75,
		Through: 1}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: bubble})
	far := &thing.Material{Name: "far", Glow: maths.RGB(0.5, 0.7, 0.9)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 8},
		Material: far, Motion: at(0, 0, -14)})
	near := &thing.Material{Name: "near", Glow: maths.RGB(0.9, 0.6, 0.2)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 6},
		Material: near, Motion: at(0, 0, 14)})
	from := maths.V(0.3, 0.2, 4)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
}

// the gpu reflects whole from the air side of an index below one, as
// slayer's tracer does
func TestTheGPUReflectsWholeFromTheAirSideOfABubbleAsSlayerDoes(t *testing.T) {
	world, eye := bubbled()
	against(t, world, eye)
}

// the gpu turns rays back whole past the critical angle at a gloss
// below one, as slayer's tracer does: total reflection does not wait
// on gloss. The water lets a part of the light through and glows, so
// that what a gloss would leave of the surface shows.
func TestTheGPUReflectsWholeAtAHalfGlossAsSlayerDoes(t *testing.T) {
	world, eye := lapped()
	for _, placed := range world.Things {
		if placed.Material.Through > 0 {
			placed.Material.Gloss = 0.5
			placed.Material.Through = 0.6
			placed.Material.Glow = maths.Grey(0.2)
		}
	}
	against(t, world, eye)
}

// soup is a heap of lapped balls of water before a glowing backdrop,
// so many that rays turned back whole inside it run out of the bends a
// look may take, and meet a surface that reflects all of what arrives
// with none left to follow it. The water glows a little, so that a
// surface lit as itself shows against one that is not.
func soup() (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	water := &thing.Material{Name: "water", Gloss: 1, Index: 1.33,
		Through: 1, Glow: maths.Grey(0.3)}
	for _, place := range []maths.Vec{
		maths.V(0, 0, 0), maths.V(0.7, 0.2, -0.3), maths.V(-0.6, 0.3, -0.2),
		maths.V(0.1, -0.7, -0.4), maths.V(0.2, 0.7, -0.5),
		maths.V(-0.3, -0.2, -0.9), maths.V(0.5, -0.4, 0.3),
	} {
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.8},
			Material: water, Motion: at(place.X, place.Y, place.Z)})
	}
	wall := &thing.Material{Name: "wall", Glow: maths.RGB(0.5, 0.7, 0.9)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 8},
		Material: wall, Motion: at(0, 0, -14)})
	from := maths.V(0.3, 0.2, 5)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
}

// the gpu stops a look at the end of its bends as slayer's tracer
// does, a surface there that reflects all of what arrives lit as
// itself
func TestTheGPUEndsTheBendsOfASoupAsSlayerDoes(t *testing.T) {
	world, eye := soup()
	against(t, world, eye)
}
