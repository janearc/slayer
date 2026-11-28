// Package relief supplies footprint-filtered slopes for procedural
// bioform surface finishes.
package relief

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/texture"
)

// Pores describes shallow pits on an unwrapped chart. Spacing is the cell
// width. Radius is the largest pit radius, at most half the spacing.
// Depth is the largest pit depth, in the chart's units.
type Pores struct {
	Seed                   uint64
	Spacing, Radius, Depth float64
}

// Slope sums the derivatives of smooth bowl-shaped depressions. Pit
// centres are jittered in three dimensions, so the surface can cut the
// field at any angle. Smaller pits disappear at a finer footprint.
func (pores Pores) Slope(rest maths.Vec, footprint float64) maths.Vec {
	var slope maths.Vec
	radius := pores.Radius / pores.Spacing
	if !(radius > 0 && radius <= .5 && pores.Depth > 0 && pores.Depth < math.Inf(1)) ||
		texture.Detail(footprint, 2*pores.Radius) == 0 {
		return slope
	}
	at := maths.V(rest.X/pores.Spacing, rest.Y/pores.Spacing, rest.Z/pores.Spacing)
	const gridLimit = 1 << 50
	if !(max(math.Abs(at.X), math.Abs(at.Y), math.Abs(at.Z)) < gridLimit) {
		return slope
	}
	for x := cell(at.X - radius); x <= cell(at.X+radius); x++ {
		for y := cell(at.Y - radius); y <= cell(at.Y+radius); y++ {
			for z := cell(at.Z - radius); z <= cell(at.Z+radius); z++ {
				size := radius * (.48 + .52*noise.Hash3(pores.Seed+303, x, y, z))
				visible := texture.Detail(footprint, 2*size*pores.Spacing)
				if visible == 0 {
					continue
				}
				delta := maths.V(
					at.X-float64(x)-noise.Hash3(pores.Seed, x, y, z),
					at.Y-float64(y)-noise.Hash3(pores.Seed+101, x, y, z),
					at.Z-float64(z)-noise.Hash3(pores.Seed+202, x, y, z))
				falloff := 1 - delta.Dot(delta)/(size*size)
				if falloff > 0 {
					depth := pores.Depth * size / radius
					slope = slope.Add(delta.Scale(6 * depth * falloff * falloff *
						visible / (size * size * pores.Spacing)))
				}
			}
		}
	}
	return slope
}

func cell(coordinate float64) int64 { return int64(math.Floor(coordinate)) }

// Furrows describes shallow grooves following the contours of a seeded
// field. Spacing, Width and Depth are in chart units. Width controls the
// distance over which the groove returns to the surrounding surface.
type Furrows struct {
	Seed                  uint64
	Spacing, Width, Depth float64
}

// Slope gives the derivative of the groove height on the chart.
func (furrows Furrows) Slope(rest maths.Vec, footprint float64) maths.Vec {
	visible := texture.Detail(footprint, furrows.Width)
	width := furrows.Width / furrows.Spacing
	if visible == 0 || !(width > 0 && width <= 1 &&
		furrows.Depth > 0 && furrows.Depth < math.Inf(1)) {
		return maths.Vec{}
	}
	at := rest.Scale(1 / furrows.Spacing)
	here := noise.Gradient(furrows.Seed, at.X, at.Y, at.Z)
	away := here / width
	if !(math.Abs(away) < 1) {
		return maths.Vec{}
	}
	const step = .01
	dx := noise.Gradient(furrows.Seed, at.X+step, at.Y, at.Z) - here
	dy := noise.Gradient(furrows.Seed, at.X, at.Y+step, at.Z) - here
	dz := noise.Gradient(furrows.Seed, at.X, at.Y, at.Z+step) - here
	return maths.V(dx, dy, dz).Scale(
		4 * furrows.Depth * visible * away * (1 - away*away) / (furrows.Width * step))
}
