package trace

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// eyeBands is the three wavelengths a gas is asked at for the eye's
// three channels, red, green and blue, since maths.Light is three numbers
// and not a spectrum: the sRGB primaries' dominant wavelengths, near
// enough.
//
// Rayleigh's fourth power across them is a blue sky and a red limb.
// When light is per band this goes.
var eyeBands = [3]maths.Band{{Nanometres: 610}, {Nanometres: 550},
	{Nanometres: 465}}

// byEye is a light from a gas's number at each of the eye's bands.
func byEye(ask func(band maths.Band) float64) maths.Light {
	return maths.RGB(ask(eyeBands[0]), ask(eyeBands[1]), ask(eyeBands[2]))
}

// gasStep is one step of a ray through a gas: each of the eye's
// channels keeps what beer's law leaves of it over the step, and
// gathers what the gas scatters toward the eye of every lamp's light
// that reaches the point.
//
// That is the gas's scattering at the band, its phase at the lamp's
// angle to the eye, and the density there. thick is the step's density
// times its length in the medium's own units.
func gasStep(scene staged, crossed *crossing, step stride, ray maths.Ray,
	gain float64, passed passage) passage {
	point, dir, thick := ray.Point(step.along), ray.Dir, step.thick
	air := crossed.volume.gas
	// what the gas itself says gets through the step, at each of the
	// eye's bands
	kept := byEye(func(band maths.Band) float64 {
		return air.Transmit(band, thick)
	})
	scatter := crossed.volume.scatter
	var in maths.Light
	for _, one := range scene.asked(point, dir.Neg(), byLight) {
		got := scene.shadowed(one, point)
		if one.ambient() {
			// from everywhere at once: as a painted volume takes it
			in = in.Add(got.Scale(evenly))
			continue
		}
		// the angle scattered through: from the lamp's way of
		// travel, minus toward, to the eye's, minus dir
		cosine := one.toward.Unit().Neg().Dot(dir.Neg())
		in = in.Add(got.Scale(air.PhaseAt(cosine)))
	}
	// what this step sends on: the light gathered times what the gas
	// scatters of it, over the step's thickness, of what still shows
	gathered := in.Times(scatter).Scale(thick * gain)
	passed.added = passed.added.Add(gathered.Times(passed.keep))
	passed.keep = passed.keep.Times(kept)
	return passed
}

// gasClear is how much of each channel of a lamp's light reaches a
// point through every gas volume between: e to the minus each gas's
// own extinction at that channel's band times its optical depth
// toward the lamp. It is one where there is no gas.
//
// The extinction is the one stage found once for the picture, where a
// gas step asks the gas's Transmit. Asking here too was measured: the
// gas picture's benchmark went from 26.5 to 29.4 milliseconds.
func gasClear(volumes []placedVolume, point, toward maths.Vec,
	far float64) maths.Light {
	var depth maths.Light
	for _, volume := range volumes {
		if volume.gas == nil {
			continue
		}
		thick := volume.opticalDepth(point, toward, far)
		depth = depth.Add(volume.extinction.Scale(thick))
	}
	passed := maths.RGB(math.Exp(-depth.R), math.Exp(-depth.G),
		math.Exp(-depth.B))
	return passed
}

// hasGas is whether any volume of the scene is a gas.
func hasGas(volumes []placedVolume) bool {
	for _, volume := range volumes {
		if volume.gas != nil {
			return true
		}
	}
	return false
}
