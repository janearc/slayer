package geometry

// CatmullRom interpolates low to high using neighbouring values for the
// slopes of a uniform Catmull-Rom spline.
func CatmullRom(before, low, high, after, share float64) float64 {
	rest := 1 - share
	lowSlope := (high - before) * .5
	highSlope := (after - low) * .5
	return (1+2*share)*rest*rest*low +
		share*rest*rest*lowSlope +
		share*share*(3-2*share)*high -
		share*share*rest*highSlope
}
