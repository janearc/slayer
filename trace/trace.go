// Package trace renders a world from a camera at a specified time. Render
// traces one ray per pixel; Sampled averages a regular grid of rays; RenderIn
// records a spectral band.
//
// Rays travel from the camera to the nearest surface, accumulating emission,
// direct lighting, sky illumination, and one mirror bounce.
//
// A surface with Material.Through above zero bends the ray by Snell's law and
// reflects by the unpolarised Fresnel share, all of the light past the
// critical angle. A partial reflection on a ray that has already bent is paid
// and not followed, which dims a water ball seen against a bright surround by
// about two in a hundred head on and a quarter at the rim.
//
// Render and Sampled start rays in air. Their additive WithInitialMedia forms
// accept an explicit ordered index/rank stack for a camera already inside clear
// media; they do not infer media from geometry.
//
// The CPU-only named ClearMedium option applies Beer-Lambert extinction to each
// camera/refracted segment. A medium that scatters also adds single
// in-scattering along the segment. The tagged ocean's mean-plane source
// refracts its one distant lamp through the sea's mean plane to points under
// it; no other lamp's direct path is refracted.
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
// Rendering entry points return an error beside the picture.
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
	// leastFacing is the cosine between the way the eye looks from and
	// the normal a smooth mesh is shaded by, where the smooth normal is
	// opposed to its face and, turned over, still leans away from the
	// eye: the normal is leaned back until its cosine is this before it
	// is made a unit length, and under this after, by a face seen at a
	// grazing angle most.
	//
	// It was picked: a hundredth, about half a degree off the view's own
	// plane, over a rounding of a normal's length, so the side is sure.
	leastFacing = 1e-2
	// leastOpposed is how far the cosine between a smooth normal and the
	// face the eye sees must be under nought for the smooth normal to be
	// called opposed to it: a half, so a hundred and twenty degrees apart.
	//
	// It was picked: a smooth normal that far from its face is the normal of
	// another surface. A survey with four metres between samples has faces
	// on a ridge whose corners lean about a hundred and forty degrees from
	// the face (measured at one cliff); no other mesh was measured.
	leastOpposed = 0.5
	// marchSteps is the most steps a march toward a surface takes
	// before it gives up as a miss.
	marchSteps = 256
	// bounces is how many times a ray may mirror off a glossy surface:
	// once, so a hall of mirrors ends.
	bounces = 1
	// refractions is how many clear surfaces a look may bend through:
	// enough for a bubble in a drink in a glass, which is six, air to
	// glass to drink to bubble and back out, with two to spare; and an
	// end, so two facing panes do not trap a ray. It was four until
	// 2026-11-12, enough for a glass seen through both walls and no more.
	refractions = 8
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
