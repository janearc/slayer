package facts

import (
	"math"

	"github.com/janearc/slayer/research"
)

// The defined values the arithmetic needs.
const (
	// lightSpeed is the speed of light in metres a second: defined by
	// the SI.
	lightSpeed = 299792458.0
	// astronomicalUnit is the au in metres: defined by IAU 2012
	// Resolution B2, as Prsa and others restate (iau-2015-b3-nominal,
	// page 8).
	astronomicalUnit = 149597870700.0
	// gmScale is the power of ten the registered mantissa of the sun's
	// GM is written under: defined by how the paper writes it.
	gmScale = 1e20
	// billion is defined: a thousand million.
	billion = 1e9
	// ten is defined: the base a logarithm of a mass is in.
	ten = 10
	// two is defined: the schwarzschild radius is two GM over c squared.
	two = 2
)

// SubringPercent is how bright, and how wide, each photon subring of a
// still hole is as a share of the one outside it, in percent: e to the
// minus pi, about four (johnson-2020-photon-ring, page 4).
func SubringPercent() Number {
	return registered("photon-subring-dimming-schwarzschild-percent", none)
}

// M87Ring is the diameter of the ring the EHT imaged round M87's hole,
// in microarcseconds (eht-2019-m87-paper-one, page 1).
func M87Ring() Number {
	return registered("m87-ring-diameter-microarcsec", none)
}

// SgrARing is the diameter of the ring the EHT imaged round our galaxy's
// central hole, in microarcseconds (eht-2022-sgra-paper-one, page 1).
func SgrARing() Number {
	return registered("sgra-ring-diameter-microarcsec", one)
}

// The two papers' masses for TON 618's hole, as the logarithm of suns.
const (
	ton618Lighter = "ton618-hole-mass-log-suns-ge"
	ton618Heavier = "ton618-hole-mass-log-suns-shemmer"
)

// Ton618Billions is TON 618's hole's mass in billions of suns, by one
// paper's estimate: the lighter, Ge and others 2019, or the heavier,
// Shemmer and others 2004. Ten to the registered logarithm, over a
// billion.
func Ton618Billions(heavier bool) Number {
	name := ton618Lighter
	if heavier {
		name = ton618Heavier
	}
	suns := math.Pow(ten, research.Value(name))
	return computed(suns/billion, none, "ten to research "+name+
		", over a billion")
}

// HorizonAU is the schwarzschild radius of a hole of so many suns, in
// au: two GM over c squared, with the sun's nominal GM from IAU 2015
// Resolution B3.
//
// For the lighter TON 618, ten to the 10.61 suns:
// 2 * 1.3271244e20 * 4.074e10 / 299792458^2 = 1.203e14 m = 804 au.
func HorizonAU(suns float64) float64 {
	gm := research.Value("sun-gm-nominal-mantissa") * gmScale * suns
	return two * gm / (lightSpeed * lightSpeed) / astronomicalUnit
}

// Ton618Horizon is TON 618's horizon radius in au, by one paper's mass.
func Ton618Horizon(heavier bool) Number {
	name := ton618Lighter
	if heavier {
		name = ton618Heavier
	}
	suns := math.Pow(ten, research.Value(name))
	return computed(HorizonAU(suns), none, "HorizonAU of ten to "+
		"research "+name)
}

// Ton618Neptunes is how many times Neptune's distance from the sun
// TON 618's horizon radius is, by one paper's mass: the horizon in au
// over Neptune's semimajor axis in au.
func Ton618Neptunes(heavier bool) Number {
	horizon := Ton618Horizon(heavier)
	neptune := research.Value("neptune-semimajor-axis-au")
	return computed(horizon.value/neptune, none, horizon.from+
		", over research neptune-semimajor-axis-au")
}
