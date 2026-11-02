package trace

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// skyLooks is how many ways round a point of a surface the sky's light
// is gathered from: picked for time. the gpu's kernel gathers from the
// same ways, so the two draw alike.
const skyLooks = 16

// goldenTurn is the golden angle, the turn between one look and the next:
// a full turn times the golden ratio less one, so no two looks line up.
var goldenTurn = math.Pi * (3 - math.Sqrt(5))

// skyWays is the looks, in a surface's own frame, z out along its
// normal: spread over the disc under the half of the sky the surface
// sees, a look to each equal share of the disc, then lifted onto the
// half sphere above it.
//
// lifted so, they fall as the cosine to the normal falls, which is how
// a matte surface weighs the light landing on it (malley's method).
var skyWays = func() [skyLooks]maths.Vec {
	var ways [skyLooks]maths.Vec
	for look := range skyLooks {
		share := (float64(look) + 0.5) / skyLooks
		out, turn := math.Sqrt(share), goldenTurn*float64(look)
		ways[look] = maths.V(out*math.Cos(turn), out*math.Sin(turn),
			math.Sqrt(1-share))
	}
	return ways
}()

// across is two ways at right angles to a normal and to each other, so a
// surface's own frame can be built round it without a division by zero
// anywhere: duff and others' basis (journal of computer graphics techniques,
// 2017).
func across(normal maths.Vec) (first, second maths.Vec) {
	sign := math.Copysign(1, normal.Z)
	a := -1 / (sign + normal.Z)
	b := normal.X * normal.Y * a
	first = maths.V(1+sign*normal.X*normal.X*a, sign*b, -sign*normal.X)
	second = maths.V(b, sign+normal.Y*normal.Y*a, -normal.Y)
	return first, second
}

// skylight is what the sky lights a matte point of a surface with: its
// colour times the mean of the sky's light along each look, dimmed by
// whatever stands in the way.
//
// the looks fall as the cosine to the normal, so their mean is the
// irradiance from the sky over pi, the radiance a matte surface of
// that colour sends back. a white surface under a sky of one radiance
// everywhere is that radiance.
func (scene staged) skylight(point, normal maths.Vec,
	colour maths.Light) maths.Light {
	if scene.lighting == nil || colour == (maths.Light{}) {
		return maths.Light{}
	}
	first, second := across(normal)
	from := scene.lifted(point, normal)
	var sum maths.Light
	for _, look := range skyWays {
		way := first.Scale(look.X).Add(second.Scale(look.Y)).
			Add(normal.Scale(look.Z))
		reached := scene.transmit(from, way, math.Inf(1))
		sum = sum.Add(scene.lighting.Light(way).Times(reached))
	}
	return colour.Times(sum.Scale(1.0 / skyLooks))
}
