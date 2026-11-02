// Package trace renders a world from a camera at a specified time. Render
// traces one ray per pixel; Sampled averages a regular grid of rays; RenderIn
// records a spectral band.
//
// Rays travel from the camera to the nearest surface, accumulating emission,
// direct lighting, sky illumination, and one mirror bounce.
//
// Gravitational fields bend visible-light rays along geodesics. Volume
// integration adds emission and scattering while attenuating the light behind
// each sample. Gas media use their own scattering coefficients and currently do
// not emit.
//
// Surfaces and volumes can shadow one another. Indirect illumination from
// volumes onto other objects is omitted, as is attenuation of lamp light by the
// global haze model.
//
// This package was derived from the tracer at c5c2169. It includes field
// traversal and volume-shadow fixes absent from the audited render/trace copy.
// Callers migrating from that package must handle rendering errors.
package trace

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// the tracer's numbers
const (
	// touch is how close a march has to come to a surface to count as
	// meeting it, as a share of the shape's reach. A share and not a
	// length, so a scene a thousandth or a thousand times as big draws
	// the same.
	//
	// A millionth keeps a shape a hundred metres across at the tenth of
	// a millimetre it was.
	touch = 1e-6
	// lift is how far a shadow ray starts off the surface it leaves, so
	// it does not meet the point it left, as a share of the scene's
	// reach or the point's distance from the origin, whichever is
	// more. A share for touch's reason, and the same one.
	lift = 1e-6
	// marchSteps is the most steps a march toward a surface takes
	// before it gives up as a miss.
	marchSteps = 256
	// bounces is how many times a ray may mirror off a glossy surface:
	// once, so a hall of mirrors ends.
	bounces = 1
	// nowhere is the distance of a ray that meets nothing.
	nowhere = math.MaxFloat64
	// strides is how many steps a ray takes across a volume it crosses,
	// and shades how many it takes toward a lamp from each of those
	strides = 48
	shades  = 10
	// faint is how little of the light behind has to be left for a march
	// to stop early. It is a share and not an amount: what is dropped
	// is under a thousandth of whatever lies behind, which shows if
	// what lies behind is a thousand times white.
	faint = 1e-3
	// evenly is how much of a lamp an isotropic scatterer sends toward the
	// eye, against a surface's cosine: a lit cloud beside a lit wall
	evenly = 0.5
)

// grey is what a thing with no material is made of.
var grey = &thing.Material{Name: "grey", Colour: maths.Grey(0.8)}

// pixelOf is a pixel of a light, covered and as deep as given.
func pixelOf(light maths.Light, cover, depth float32) frame.Pixel {
	red, green, blue := float32(light.R), float32(light.G),
		float32(light.B)
	return frame.Pixel{R: red, G: green, B: blue, Cover: cover,
		Depth: depth}
}
