package medium

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Clouds is a layer of cloud held against a body: droplets between Low and High
// above Radius, in patches laid by fractal noise over the sphere. Cover is how
// much is clouded, zero none and one all, but not in proportion: a half clouds
// about two thirds of the sky.
//
// A patch is Thick at its heart, thinning to its edge; up through the layer it
// is whole to a third of the way, then thins to 0.15 of it at the top, where it
// stops. The layer is cut flat at Low, where a patch starts at its full
// thickness. Seed makes a sky's own weather. Nothing below Low or above High.
type Clouds struct {
	Radius    float64
	Low, High float64
	Cover     float64
	Thick     float64
	Seed      uint64
}

// Bounds is the cube the layer's top just fits in.
func (clouds Clouds) Bounds() maths.Box {
	return cube(clouds.Radius + clouds.High)
}

// Span is where a ray crosses the layer's top.
func (clouds Clouds) Span(ray maths.Ray, near, far float64) (float64, float64,
	bool) {
	return sphere(clouds.Radius+clouds.High, ray, near, far)
}

// Density is the patch here, if the point is in the layer.
func (clouds Clouds) Density(point maths.Vec) float64 {
	if clouds.High <= clouds.Low || clouds.Cover <= nought {
		return nought
	}
	above := point.Len() - clouds.Radius
	if above < clouds.Low || above > clouds.High {
		return nought
	}
	// the patches: noise over the sphere's surface, about two dozen
	// patches round at a cover of a half, clouded where it rises above
	// the cover's threshold
	direction := point.Unit()
	frequency := chosen.cloudFrequency
	grain := halfway + noise.Fractal(clouds.Seed, direction.X*frequency,
		direction.Y*frequency, direction.Z*frequency, chosen.cloudOctaves)
	threshold := unity - clouds.Cover
	patch := maths.Smooth(threshold-chosen.cloudEdge,
		threshold+chosen.cloudEdge, grain)
	if patch <= nought {
		return nought
	}
	// puffy: whole to a third of the way up, thinning to the top
	height := (above - clouds.Low) / (clouds.High - clouds.Low)
	puff := unity - maths.Smooth(chosen.cloudWhole, unity, height)
	return clouds.Thick * patch * math.Max(puff, chosen.cloudTop)
}
