// Package maths supplies typed power in watts, temperature in kelvin, and area
// in square metres, together with Planck's law.
//
// Constructors validate each quantity. Distinct types keep units explicit at
// package boundaries and prevent a temperature from being passed where a power
// is required.
package maths

import (
	"errors"
	"math"
)

// none is no power, no heat and no area at all. It is defined as zero.
const none = 0

// upward is math.IsInf's code for infinity above all numbers. It is
// defined by that function.
const upward = 1

// The SI's defining constants follow, exact by definition. libtheme keeps
// the same ones where nothing outside libtheme can reach them. They are
// definitions and not measurements. So writing them here again copies
// no one's work and can never drift from libtheme's. Both are the SI's.

// planck is the Planck constant in joule seconds. It is exact since the
// SI of 2019 and defined by it.
const planck = 6.62607015e-34

// speed is the speed of light in metres a second. It is exact since
// 1983 and defined by it.
const speed = 299792458.0

// boltzmann is the Boltzmann constant in joules a kelvin. It is exact
// since the SI of 2019 and defined by it.
const boltzmann = 1.380649e-23

// sunLuminosity is the sun's luminosity in watts. It is defined as the
// IAU fixes it for reference by IAU 2015 resolution B3, the nominal
// solar luminosity.
const sunLuminosity = 3.828e26

var (
	errPower = errors.New("ninjafilm: a power is a finite number of " +
		"watts, not below nought")
	errTemperature = errors.New("ninjafilm: a temperature is a finite " +
		"number of kelvin, not below nought")
)

// Power is how much energy leaves something each second.
type Power struct {
	watts float64
}

// Watts is a power given in watts. It refuses a power that is negative,
// endless or not a number.
func Watts(amount float64) (made Power, err error) {
	if !(amount >= none) || math.IsInf(amount, upward) {
		return made, errPower
	}
	return Power{amount}, nil
}

// SolarLuminosities is a power given in suns: how astronomers state a
// star's output. One sun is the IAU's nominal solar luminosity.
//
//ninja:vow practice slayer's stars take their power here; tests only so far
func SolarLuminosities(amount float64) (Power, error) {
	return Watts(amount * sunLuminosity)
}

// Watts is the power in watts.
func (power Power) Watts() float64 {
	return power.watts
}

// Temperature is how hot something is, from absolute zero.
type Temperature struct {
	kelvin float64
}

// Kelvin is a temperature given in kelvin. It refuses one below absolute
// zero. It refuses one that is endless or not a number.
//
//ninja:vow practice slayer's lamps take their heat here; tests only so far
func Kelvin(amount float64) (made Temperature, err error) {
	if !(amount >= none) || math.IsInf(amount, upward) {
		return made, errTemperature
	}
	return Temperature{amount}, nil
}
