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
// stops. Seed makes a sky's own weather. Nothing below Low or above High.
type Clouds struct {
	Radius    float64
	Low, High float64
	Cover     float64
	Thick     float64
	Seed      uint64
}

// Bounds is the cube the layer's top just fits in.
func (c Clouds) Bounds() maths.Box {
	r := c.Radius + c.High
	return maths.Box{Min: maths.V(-r, -r, -r), Max: maths.V(r, r, r)}
}

// Span is where a ray crosses the layer's top.
func (c Clouds) Span(r maths.Ray, near, far float64) (float64, float64, bool) {
	return sphere(c.Radius+c.High, r, near, far)
}

// Density is the patch here, if the point is in the layer.
func (c Clouds) Density(p maths.Vec) float64 {
	if c.High <= c.Low || c.Cover <= 0 {
		return 0
	}
	up := p.Len() - c.Radius
	if up < c.Low || up > c.High {
		return 0
	}
	// the patches: noise over the sphere's surface, about two dozen
	// patches round at a cover of a half, clouded where it rises above
	// the cover's threshold
	d := p.Unit()
	n := 0.5 + noise.Fractal(c.Seed, d.X*6, d.Y*6, d.Z*6, 4)
	edge := 1 - c.Cover
	patch := maths.Smooth(edge-0.08, edge+0.08, n)
	if patch <= 0 {
		return 0
	}
	// puffy: whole to a third of the way up, thinning to the top
	h := (up - c.Low) / (c.High - c.Low)
	puff := 1 - maths.Smooth(0.33, 1, h)
	return c.Thick * patch * math.Max(puff, 0.15)
}
