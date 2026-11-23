package medium

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Plume is a volcanic plume: a column of gas and dust rising from the
// origin along y to Height, and an umbrella of the same arcing over
// from the top and falling back to the ground Width across, as io's
// plumes do.
//
// Thick is the density at the column's heart on the ground. The column thins
// to half of Thick at the top. The umbrella is a thin shell, a fifth of Thick
// at the ground and half of it at the top, so at the top the two together are
// Thick, and the plume is cut flat at Height. The umbrella's gaussian is cut
// three shell thicknesses from the shell, where it is under a ten thousandth
// of Thick; the shapes are chosen in choices.yaml.
type Plume struct {
	Height, Width float64
	Thick         float64
}

// reach is how far from the y axis the plume's gas goes: the shell at the
// ground, and the skirt of its gaussian beyond.
func (plume Plume) reach() float64 {
	skirt := chosen.plumeSkirt * chosen.plumeSkin
	return plume.Width / double * (unity + skirt)
}

// Bounds is the box the plume's gas is in, the umbrella's skirt included.
func (plume Plume) Bounds() maths.Box {
	across := plume.reach()
	return corners(maths.V(-across, nought, -across),
		maths.V(across, plume.Height, across))
}

// Density is the column plus the umbrella's shell, and zero beyond the
// skirt.
func (plume Plume) Density(point maths.Vec) float64 {
	if plume.Height <= nought || plume.Width <= nought {
		return nought
	}
	climb := point.Y / plume.Height
	if climb <= nought || climb >= unity {
		return nought
	}
	half := plume.Width / double
	rho := math.Hypot(point.X, point.Z)
	if rho > plume.reach() {
		return nought
	}
	// the column, thinning as it rises
	core := chosen.plumeCore * half
	column := math.Exp(-rho*rho/(core*core)) *
		(unity - chosen.plumeThin*climb)
	// the umbrella, a dome: a shell closing from the width at the
	// ground to the column's top, a skin thick, thinner as it falls
	shell := half * math.Sqrt(unity-climb*climb)
	skin := chosen.plumeSkin * half
	across := (rho - shell) / skin
	umbrella := chosen.plumePeak *
		(chosen.plumeFoot + chosen.plumeRise*climb) * math.Exp(-across*across)
	return plume.Thick * (column + umbrella)
}
