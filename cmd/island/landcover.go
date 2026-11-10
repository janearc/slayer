package main

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/frame"
)

// altitudePaint is an authored radial-height palette. It reads no source file
// during tracing, and keeps its world anchor as the render origin changes.
type altitudePaint struct {
	frame      frame.Sphere
	origin     maths.Vec
	bands      []heightBand
	base       maths.Light
	forest     *forestPaint
	transition float64
}

type forestPaint struct {
	areaDensity                                   float64
	areaSampled                                   bool
	space                                         maths.Subspace
	sphere                                        frame.Sphere
	edgeFade                                      float64
	heightFade                                    float64
	bounds                                        localBounds
	minimum, maximum, fadeFrom, fadeTo, clearance float64
	colour                                        maths.Light
	clearings                                     []clearing
}

// At samples the height palette at a point, before any distant-canopy blend.
func (paint altitudePaint) At(local maths.Vec) maths.Light { return paint.Within(local, 0) }

// Within blends an authored canopy appearance as individual roots become
// subpixel. This is a scenery approximation, not a leaf-area radiative model.
func (paint altitudePaint) Within(local maths.Vec, footprint float64) maths.Light {
	world := local.Add(paint.origin)
	position, err := paint.frame.Location(world)
	if err != nil {
		return maths.Grey(math.NaN())
	}
	colour := paint.heightColour(position.Height.InMetres())
	forest := paint.forest
	if forest == nil || footprint <= forest.fadeFrom {
		return colour
	}
	at := forest.chart(world)
	b := forest.bounds
	if at.X < b.MinEast || at.X >= b.MaxEast || at.Z < b.MinSouth || at.Z >= b.MaxSouth ||
		position.Height.InMetres() < forest.minimum || position.Height.InMetres() > forest.maximum {
		return colour
	}
	for _, clearing := range forest.clearings {
		if math.Hypot(at.X-clearing.east, at.Z-clearing.south) < clearing.radius+forest.clearance {
			return colour
		}
	}
	density := edgeWeight(forest.bounds, forest.edgeFade, at.X, at.Z)
	density *= heightWeight(forest.minimum, forest.maximum, forest.heightFade, position.Height.InMetres())
	cover := density
	if forest.areaSampled {
		// For a Poisson approximation, no overlapping crown has probability
		// exp(-density * mean area). The seeded cell process is not Poisson;
		// this is an explicit far-field appearance approximation.
		cover = -math.Expm1(-forest.areaDensity * density)
	}
	share := maths.Smooth(forest.fadeFrom, forest.fadeTo, footprint) * cover
	return colour.Scale(1 - share).Add(forest.colour.Scale(share))
}

// chart inverts the same spherical exponential chart that places roots. Its
// coordinates do not drift with radial terrain height or the render origin.
func (forest forestPaint) chart(world maths.Vec) maths.Vec {
	direction := world.Sub(forest.sphere.Centre()).Unit()
	x, z := direction.Dot(forest.space.Orientation.X), direction.Dot(forest.space.Orientation.Z)
	lateral := math.Hypot(x, z)
	if lateral == 0 {
		return maths.Vec{}
	}
	angle := math.Atan2(lateral, direction.Dot(forest.space.Orientation.Y))
	scale := forest.sphere.Radius().InMetres() * angle / lateral
	return maths.V(x*scale, 0, z*scale)
}

// heightColour softens authored boundaries between adjacent palette bands.
// The transition is a chosen radial-height half-width, not a measured ecotone.
// Zero preserves the old steps; the final cutoff still returns the base colour.
func (paint altitudePaint) heightColour(height float64) maths.Light {
	for index, band := range paint.bands {
		if paint.transition > 0 && index+1 < len(paint.bands) && height >= band.Below-paint.transition && height <= band.Below+paint.transition {
			share := maths.Smooth(band.Below-paint.transition, band.Below+paint.transition, height)
			return rgb(band.Colour).Scale(1 - share).Add(rgb(paint.bands[index+1].Colour).Scale(share))
		}
		if height < band.Below {
			return rgb(band.Colour)
		}
	}
	return paint.base
}
