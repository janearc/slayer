package medium

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Aurora is a body's aurora: a glow in a shell between Low and High above
// Radius, in an oval round each magnetic pole at Lat from the pole's equator,
// Width wide, the pole's Axis in the body's own space, up when none is given.
// Lat and Width are angles in radians.
//
// at the oval's heart the density is zero at the shell's bottom, rises to about
// three quarters of Thick a tenth of the way up, and thins by e over every
// third of the shell above that. The shell is cut flat at High, where about a
// twentieth of the peak remains.
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
func (aurora Aurora) Bounds() maths.Box {
	return cube(aurora.Radius + aurora.High)
}

// Span is where a ray crosses the shell's top.
func (aurora Aurora) Span(ray maths.Ray, near, far float64) (float64, float64,
	bool) {
	return sphere(aurora.Radius+aurora.High, ray, near, far)
}

// Density is the oval here, if the point is in the shell.
func (aurora Aurora) Density(point maths.Vec) float64 {
	if aurora.High <= aurora.Low || aurora.Width <= nought {
		return nought
	}
	above := point.Len() - aurora.Radius
	if above < aurora.Low || above > aurora.High {
		return nought
	}
	axis := aurora.Axis
	if axis == (maths.Vec{}) {
		axis = maths.Up
	}
	axis = axis.Unit()
	direction := point.Unit()
	toAxis := direction.Dot(axis)
	sine := math.Max(-unity, math.Min(unity, toAxis))
	latitude := math.Asin(sine)
	// the oval, either pole
	off := (math.Abs(latitude) - aurora.Lat) / aurora.Width
	oval := math.Exp(-off * off)
	if oval < chosen.auroraFloor {
		return nought
	}
	// brightest near the bottom, rising from nothing over the first
	// tenth of the shell so the lower border is soft, thinning upward
	height := (above - aurora.Low) / (aurora.High - aurora.Low)
	profile := math.Exp(-height*chosen.auroraFall) *
		maths.Smooth(nought, chosen.auroraSoft, height)
	density := aurora.Thick * oval * profile
	if aurora.Curtains > nought {
		// rays along the oval: a noise fine round the pole and stretched
		// up, and folds coarser than the rays
		upright := maths.Upright(axis)
		x, z := direction.Dot(upright.X), direction.Dot(upright.Z)
		phi := math.Atan2(z, x)
		cos, sin := math.Cos(phi), math.Sin(phi)
		rays := noise.Fractal(aurora.Seed, chosen.auroraRays*cos,
			chosen.auroraRays*sin, chosen.auroraStretch*height,
			chosen.auroraOctaves)
		folds := noise.Fractal(aurora.Seed+chosen.auroraFoldSeed,
			chosen.auroraFolds*cos, chosen.auroraFolds*sin, nought,
			chosen.auroraOctaves)
		density *= math.Max(nought, unity+aurora.Curtains*
			(chosen.auroraRayWeight*rays+folds))
	}
	return density
}
