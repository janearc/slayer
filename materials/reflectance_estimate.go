package materials

import "math"

// conductorFromReflectance fits a nonunique effective index to normal air
// reflectance. Its chosen n=1 and derived k are appearance estimates, not
// measured optical constants. Wavelength is in nm; reflectance is in (0,1).
func conductorFromReflectance(wavelength, reflectance float64,
	where Source) (Reading, error) {
	if !(reflectance > 0 && reflectance < 1) {
		return Reading{}, refuse("normal reflectance %v must be "+
			"between zero and one", reflectance)
	}
	where.estimated = true
	k := 2 * math.Sqrt(reflectance/(1-reflectance))
	return newReading(wavelength, 1, k, where)
}

// Estimated reports that this source accompanies derived appearance optics.
// Existing measured table sources return false.
func (where Source) Estimated() bool { return where.estimated }
