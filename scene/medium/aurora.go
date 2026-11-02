package medium

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Aurora is a body's aurora: a glow in a shell between Low and High above
// Radius, in an oval round each magnetic pole at Lat from the pole's equator,
// Width wide, the pole's Axis in the body's own space, up when none is given.
//
// at the oval's heart the density is zero at the shell's bottom, rises to about
// three quarters of Thick a tenth of the way up, and thins by e over every
// third of the shell above that.
//
// Curtains is a dressing: the oval broken into rays and folds along its length,
// as the eye sees them, zero for a smooth band. Seed makes a night's own.
type Aurora struct {
	Radius    float64
	Low, High float64
	Lat       float64
	Width     float64
	Axis      maths.Vec
	Thick     float64
	Curtains  float64
	Seed      uint64
}

// Bounds is the cube the shell's top just fits in.
func (a Aurora) Bounds() maths.Box {
	r := a.Radius + a.High
	return maths.Box{Min: maths.V(-r, -r, -r), Max: maths.V(r, r, r)}
}

// Span is where a ray crosses the shell's top.
func (a Aurora) Span(r maths.Ray, near, far float64) (float64, float64, bool) {
	return sphere(a.Radius+a.High, r, near, far)
}

// Density is the oval here, if the point is in the shell.
func (a Aurora) Density(p maths.Vec) float64 {
	if a.High <= a.Low || a.Width <= 0 {
		return 0
	}
	up := p.Len() - a.Radius
	if up < a.Low || up > a.High {
		return 0
	}
	axis := a.Axis
	if axis == (maths.Vec{}) {
		axis = maths.Up
	}
	axis = axis.Unit()
	d := p.Unit()
	lat := math.Asin(math.Max(-1, math.Min(1, d.Dot(axis))))
	// the oval, either pole
	off := (math.Abs(lat) - a.Lat) / a.Width
	oval := math.Exp(-off * off)
	if oval < 1e-3 {
		return 0
	}
	// brightest near the bottom, rising from nothing over the first
	// tenth of the shell so the lower border is soft, thinning upward
	h := (up - a.Low) / (a.High - a.Low)
	profile := math.Exp(-h*3) * maths.Smooth(0, 0.1, h)
	density := a.Thick * oval * profile
	if a.Curtains > 0 {
		// rays along the oval: a noise fine round the pole and
		// stretched up, and folds coarser than the rays; the rays
		// are no finer than a picture from orbit resolves, or the
		// terminal's rays and smokecheck's land on them differently
		o := maths.Upright(axis)
		x, z := d.Dot(o.X), d.Dot(o.Z)
		phi := math.Atan2(z, x)
		cos, sin := math.Cos(phi), math.Sin(phi)
		rays := noise.Fractal(a.Seed, 24*cos, 24*sin, 2*h, 2)
		folds := noise.Fractal(a.Seed+1, 6*cos, 6*sin, 0, 2)
		density *= math.Max(0, 1+a.Curtains*(2*rays+folds))
	}
	return density
}
