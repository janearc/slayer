package maths

import (
	"math"

	"github.com/janearc/slayer/ninjafilm/data"
)

// slices is how many pieces Exitance cuts a stretch of wavelengths into
// for Simpson's rule. It is picked. It is even, as the rule needs.
//
// It is enough that a band of libtheme's about three percent wide comes
// out within a millionth of the exact integral where Planck's curve is
// smooth across the band.
var slices = data.Whole("simpson-slices")

// nanometre is a nanometre in metres. It is defined by the nano prefix
// as 1e-9.
const nanometre = 1e-9

// interior is the place of the first of the points Simpson's rule
// weighs between its two ends. It is defined by the rule.
const interior = 1

// oddPoint is what an odd point of Simpson's rule leaves when its place
// is divided by two. It is defined by the rule.
const oddPoint = 1

// pair is a count of two: the two in two pi in Planck's law, and the
// two of dividing a place to see if it is odd. It is defined as two.
const pair = 2

// oddWeight is what Simpson's rule weighs an odd point by. It is
// defined by the rule.
const oddWeight = 4

// evenWeight is what Simpson's rule weighs an even point by. It is
// defined by the rule.
const evenWeight = 2

// third is the share of the step that Simpson's rule multiplies the
// weighed sum by. It is defined by the rule.
const third = 3

// fifth is the power of the wavelength in Planck's law. It is defined
// by the law.
const fifth = 5

// Exitance is the power a black body at this temperature gives off
// every way from each square metre of its surface between two
// wavelengths in nanometres. It is Planck's law in absolute watts
// integrated across the stretch.
//
// This is the physical black body at any wavelength at all, from gamma
// rays to radio. libtheme's swatch.Blackbody is a different answer to a
// different question. It says what colour a black body is to an eye.
//
// It is relative, and it stops where the table of the eye stops. That is
// right for colour and not for the watts of a lamp. Most of them leave
// as infrared an eye never sees.
//
// Past about thirty in Planck's exponent a band holds too many e-folds
// for the pieces to resolve. The exponent is hc over wavelength k T. At
// 700 the answer is off by a factor of two in a power of 1e-304 watts.
// That matters to no one.
func (heat Temperature) Exitance(shortest, longest float64) float64 {
	if heat.kelvin == none || !(longest > shortest) || !(shortest > none) {
		return none
	}
	short, long := shortest*nanometre, longest*nanometre
	step := (long - short) / float64(slices)
	total := heat.spectral(short) + heat.spectral(long)
	for place := range slices - interior {
		index := place + interior
		weight := float64(evenWeight)
		if index%pair == oddPoint {
			weight = oddWeight
		}
		total += weight * heat.spectral(short+float64(index)*step)
	}
	return total * step / third
}

// spectral is Planck's law for the power a black body gives off from a square
// metre every way at one wavelength in metres. It is watts a square metre per
// metre of wavelength.
//
// Far into the short wavelengths the exponential overflows to infinity. The
// power then comes out zero, which it truly is there.
func (heat Temperature) spectral(wavelength float64) float64 {
	exponent := planck * speed / (wavelength * boltzmann * heat.kelvin)
	scale := pair * math.Pi * planck * speed * speed
	return scale / (math.Pow(wavelength, fifth) * math.Expm1(exponent))
}
