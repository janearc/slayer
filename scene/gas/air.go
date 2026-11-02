package gas

import (
	"math"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/scene/thing"
)

// Air is the earth's air at sea level, built from the real windows.
// Every number names its source; the figures are per metre at sea
// level density, and a volume's own density scales them.
//
// The visible is mostly through, the short end scattered by the
// molecules themselves, which is the blue sky and the limb: rayleigh
// scattering at 1.16e-5 a metre at 550 nm, falling as the fourth power
// of the wavelength.
//
// That is bodhaine, wood, dutton and slusser, 1999, journal of
// atmospheric and oceanic technology 16, 1854: the sea-level
// coefficient; the exponent is near 4.05 with dispersion, taken as 4.
//
// Ozone's chappuis band absorbs a little across the visible, most
// near 600 nm, about 3e-7 a metre for a column of 300 dobson units
// spread over the first eight kilometres (burrows et al., 1999, the
// gome cross sections).
//
// that does not follow: 300 dobson units are 8.06e18 molecules a square
// centimetre, and at a peak cross section near 5e-21 (from memory, not
// checked here) the column is 0.04 deep, 5e-6 a metre over 8 km, about
// seventeen times this. kept as it was, as ruled.
//
// Ultraviolet below about 300 nm is stopped by ozone's hartley band,
// a cross section near 1e-17 square centimetres a molecule at 255 nm
// (molina and molina, 1986), which over the column is some eighty to
// ninety optical depths: here 1e-2 a metre.
//
// That is opaque within hundreds of metres of the column's worth.
// Between 300 and 400 nm most comes through, scattered more than the
// visible.
//
// Infrared has windows near 8 to 14 microns and walls where water
// vapour and carbon dioxide absorb: the 6.3 micron water band, the 15
// micron carbon dioxide band (goody and yung, atmospheric radiation,
// 1989).
//
// One number for the range cannot hold both, so the range is given
// the window's number, 5e-5 a metre of absorption, and the walls are
// a thread until the table is by wavelength there too.
//
// Microwave comes through between oxygen's 60 gigahertz band and water's
// near 22 (liebe, 1989, the mpm model): 1e-6 a metre, the window's. in
// the oxygen band itself sea-level air takes near 15 dB a km, 3.4e-3 a
// metre, which one number for the range cannot hold. radio, 1e-7.
//
// X-rays are stopped by photoabsorption: at 10 keV air's mass
// attenuation is about 5 square centimetres a gram (nist xcom, hubbell
// and seltzer), which at 1.2e-3 grams a cubic centimetre is 6e-3 a
// centimetre, 0.6 a metre: opaque within metres.
//
// Gamma rays are stopped by the column's twenty-six radiation lengths:
// air's radiation length is 36.6 grams a square centimetre (the
// particle data group's atomic and nuclear properties), and a metre of
// air at sea level is 0.12 grams a square centimetre.
//
// So 3.3e-3 radiation lengths a metre, taken as the extinction, and
// the whole column of a thousand grams a square centimetre is
// twenty-seven: the ground sees nothing from space in gamma.
func Air() Gas {
	return Gas{
		Name: "air",
		Sources: []string{
			"bodhaine, wood, dutton and slusser 1999, j. atmos. " +
				"oceanic technol. 16, 1854 (rayleigh)",
			"burrows et al. 1999, gome ozone cross sections " +
				"(chappuis)",
			"molina and molina 1986, j. geophys. res. 91, 14501 " +
				"(hartley)",
			"goody and yung 1989, atmospheric radiation " +
				"(infrared windows)",
			"liebe 1989, int. j. infrared millimeter waves 10, " +
				"631 (mpm: microwave)",
			"nist xcom, hubbell and seltzer (x-ray attenuation)",
			"particle data group, atomic and nuclear properties " +
				"of materials (radiation length)",
		},
		Ranges: map[radiation.Range]Optics{
			radiation.Gamma():       {Absorb: 3.3e-3},
			radiation.XRay():        {Absorb: 0.6},
			radiation.Ultraviolet(): {Absorb: 1e-2, Scatter: 6e-5},
			thing.Visible: {Absorb: 3e-7,
				Scatter: 1.16e-5},
			radiation.Infrared():  {Absorb: 5e-5, Scatter: 1e-7},
			radiation.Microwave(): {Absorb: 1e-6},
			radiation.Radio():     {Absorb: 1e-7},
		},
		Visible: func(nm float64) (absorb, scatter float64) {
			if nm < 300 {
				return 1e-2, 6e-5
			}
			// rayleigh, by the fourth power from 550 nm
			ratio := 550 / nm
			scatter = 1.16e-5 * ratio * ratio * ratio * ratio
			// the chappuis band, a hump centred near 600 nm
			absorb = 3e-7 * math.Exp(-(nm-600)*(nm-600)/(2*60*60))
			return absorb, scatter
		},
		Phase: 0,
	}
}

// Haze is a thin aerosol haze: forward scattering, a little absorption,
// one number across the visible, by the usual continental aerosol
// (hess, koepke and schult 1998, the opac data set): 1e-4 a metre of
// scattering, asymmetry 0.7.
//
// its visibility is about thirty-five kilometres, by koschmieder's 3.912
// over its extinction of 1.1e-4. it carries air's x-ray and gamma figures
// under its own name, and on air counts them twice.
func Haze() Gas {
	return Gas{
		Name: "haze",
		Sources: []string{
			"hess, koepke and schult 1998, bull. amer. meteor. " +
				"soc. 79, 831 (opac)",
		},
		Ranges: map[radiation.Range]Optics{
			thing.Visible: {Absorb: 1e-5, Scatter: 1e-4},
			radiation.Ultraviolet(): {Absorb: 2e-5,
				Scatter: 1.5e-4},
			radiation.Infrared():  {Absorb: 5e-6, Scatter: 3e-5},
			radiation.Microwave(): {}, radiation.Radio(): {},
			radiation.XRay():  {Absorb: 0.6},
			radiation.Gamma(): {Absorb: 3.3e-3},
		},
		Phase: 0.7,
	}
}

// Venus is venus's air at its surface, carbon dioxide at ninety-two
// bars and 735 kelvin under a deck of sulphuric acid cloud: per metre
// at the surface, for a shell whose density is a share of the surface's
// and falls by the scale height, 15.9 km (seiff et al. 1985, vira).
//
// The visible is rayleigh scattering by carbon dioxide, said to be
// derived: the surface holding 65 times the earth's sea-level number of
// molecules, and carbon dioxide's cross section about 2.6 times
// nitrogen's at 550 nm (sneep and ubachs 2005).
//
// so 1.16e-5 times 65 times 2.6, about 2e-3 a metre at 550 nm, falling
// as the fourth power. but 92 bar at 735 K against 1 bar at 288 K is 36
// times the molecules, not 65, which is venus's surface density in
// kilograms a cubic metre: worked, 1.1e-3. kept as it was, as ruled.
//
// The cloud deck is not here: a shell that falls by one scale height
// cannot hold a layer at fifty kilometres, and its optical depth of
// about thirty (esposito et al. 1983) waits for a medium with a deck.
// So this venus is clear to the ground and yellower than the real one.
//
// Ultraviolet below 320 nm is taken by sulphur dioxide and the unknown
// absorber: 1e-2 a metre. Infrared is carbon dioxide's wall, 1e-2 a
// metre (the 4.3 and 15 micron bands; the near-infrared windows at 1
// to 2.5 microns, through which the surface was mapped, are a thread).
//
// Microwave and radio come through, which is how radar mapped the
// ground, pioneer venus first: 1e-7 a metre (pettengill et al. 1980).
//
// X-ray and gamma are air's figures times the 65: 39 and 0.21 a metre.
// by mass, which is what stops them, the ratio is near 55, 36 times
// carbon dioxide's 44 over air's 29.
func Venus() Gas {
	return Gas{
		Name: "venus",
		Sources: []string{
			"seiff et al. 1985, adv. space res. 5, 3 " +
				"(vira: the profile)",
			"sneep and ubachs 2005, j. quant. spectrosc. " +
				"radiat. transfer 92, 293 (co2 rayleigh)",
			"esposito et al. 1983, in venus, u. arizona " +
				"press (the cloud deck)",
			"pettengill et al. 1980, j. geophys. res. 85, " +
				"8261 (radio through the air)",
			"nist xcom, hubbell and seltzer (x-ray attenuation); " +
				"particle data group (radiation length)",
		},
		Ranges: map[radiation.Range]Optics{
			radiation.Gamma():       {Absorb: 0.21},
			radiation.XRay():        {Absorb: 39},
			radiation.Ultraviolet(): {Absorb: 1e-2, Scatter: 1e-2},
			thing.Visible:           {Absorb: 1e-6, Scatter: 2e-3},
			radiation.Infrared():    {Absorb: 1e-2, Scatter: 1e-5},
			radiation.Microwave():   {Absorb: 1e-7},
			radiation.Radio():       {Absorb: 1e-7},
		},
		Visible: func(nm float64) (absorb, scatter float64) {
			if nm < 320 {
				return 1e-2, 1e-2
			}
			ratio := 550 / nm
			scatter = 2e-3 * ratio * ratio * ratio * ratio
			// the unknown absorber's tail into the blue, which is
			// what makes the planet yellow: a hump near 400 nm
			absorb = 1e-6 +
				2e-4*math.Exp(-(nm-400)*(nm-400)/(2*40*40))
			return absorb, scatter
		},
		Phase: 0,
	}
}

// Mars is mars's air at its surface, carbon dioxide at six millibars
// and 210 kelvin, per metre, for a shell falling by the scale height,
// 11.1 km (the viking and pathfinder profiles, seiff and kirk 1977).
//
// The gas itself is nearly clear, derived: 0.6 percent of the earth's
// pressure at 210 K is about 0.8 percent of its molecules, times
// carbon dioxide's 2.6, so 1.16e-5 times 0.021, about 2.4e-7 a metre
// at 550 nm.
//
// What colours the sky is dust: an optical depth of about 0.5 over the
// scale height (smith 2004, the thermal emission spectrometer record)
// is 4.5e-5 a metre of extinction, scattering forward, asymmetry 0.7.
//
// It absorbs the blue more than the red (ockert-bell et al. 1997),
// which is the butterscotch sky and the blue sunset.
//
// The dust is folded into the visible rule here as one gas; a dust
// medium of its own is a thread.
//
// Ultraviolet reaches the ground, scattered more: 1e-6 a metre.
// Infrared has carbon dioxide's 15 micron band: 1e-4 a metre. Microwave
// and radio are clear: 1e-9.
//
// X-ray and gamma are air's figures times the 0.008 of molecules: 5e-3
// and 3e-5 a metre. by mass the ratio is near 0.012, carbon dioxide
// being half again as heavy as air.
func Mars() Gas {
	return Gas{
		Name: "mars",
		Sources: []string{
			"seiff and kirk 1977, j. geophys. res. 82, " +
				"4364 (the viking profile)",
			"sneep and ubachs 2005, j. quant. spectrosc. " +
				"radiat. transfer 92, 293 (co2 rayleigh)",
			"smith 2004, icarus 167, 148 (dust opacity, tes)",
			"ockert-bell et al. 1997, j. geophys. res. " +
				"102, 9039 (dust optical properties)",
			"nist xcom, hubbell and seltzer (x-ray attenuation); " +
				"particle data group (radiation length)",
		},
		Ranges: map[radiation.Range]Optics{
			radiation.Gamma():       {Absorb: 3e-5},
			radiation.XRay():        {Absorb: 5e-3},
			radiation.Ultraviolet(): {Absorb: 1e-6, Scatter: 6e-5},
			thing.Visible: {Absorb: 1e-5,
				Scatter: 3.5e-5},
			radiation.Infrared():  {Absorb: 1e-4, Scatter: 1e-5},
			radiation.Microwave(): {Absorb: 1e-9},
			radiation.Radio():     {Absorb: 1e-9},
		},
		Visible: func(nm float64) (absorb, scatter float64) {
			ratio := 550 / nm
			gas := 2.4e-7 * ratio * ratio * ratio * ratio
			// the dust: scattering flat across the visible, and an
			// absorption that climbs toward the blue
			blue := math.Max(0, (600-nm)/200)
			return 5e-6 + 2e-5*blue, gas + 3.5e-5
		},
		Phase: 0.7,
	}
}
