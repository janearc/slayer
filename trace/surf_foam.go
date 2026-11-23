package trace

// A sea with surf says how much of its surface is foam at a point and
// what the foam reflects; this file lays that foam over the water's own
// light, lit by the scene's lamps and sky as any matte surface is.

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// foamer is a shape that says how much of its surface at a point, in its
// own frame, is foam, from none to all, and what the foam reflects:
// arid.Sea with surf is one.
type foamer interface {
	Foam(point maths.Vec) (float64, maths.Light)
}

// surfFoam is the light of a clear surface, sum, with the foam its shape
// says lies at local laid over it: the foam's share of the light is the
// lamps' and the sky's on a matte surface of the foam's reflectance at
// point, facing normal, and the rest is the water's. A shape that says no
// foam leaves the light as it is.
func (scene *staged) surfFoam(sum maths.Light, local, point,
	normal maths.Vec, seen look, target *thing.Thing) maths.Light {
	surfed, has := target.Shape.(foamer)
	if !has {
		return sum
	}
	coverage, reflectance := surfed.Foam(local)
	if !(coverage > 0) {
		return sum
	}
	light := scene.oceanLamplight(point, normal, reflectance, seen,
		target).Add(scene.skylight(point, normal, reflectance))
	return sum.Scale(1 - coverage).Add(light.Scale(coverage))
}
