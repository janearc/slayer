package named

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/lantern"
	"github.com/janearc/slayer/research"
)

// The real magnetar's numbers. Each measured one is read from the
// research data by its name, where it is held with its source and page;
// the rest are computed here from those by the model each names. A
// printed power of ten is applied here, by a defined constant.

// the powers of ten the research data's mantissas are printed with, and
// the units turned into gauss and kilometres, each defined
const (
	tenToThe9  = 1e9
	tenToThe13 = 1e13
	tenToThe14 = 1e14
	tenToThe46 = 1e46
	tenToThe24 = 1e24
	millionKm  = 1e6 // defined: kilometres in a million kilometres
	gaussTesla = 1e4 // defined: gauss in a tesla
	zappyUnit  = 1e3 // defined: kilometres in zappy's unit
	ergJoule   = 1e7 // defined: ergs in a joule
	cmSquareM  = 1e4 // defined: square centimetres in a square metre
	diameter   = 2   // defined: a diameter is two radii
	cubeRoot   = 1. / 3
)

// the light year and the parsec in kilometres, defined: the IAU's
// parsec, 648000/pi astronomical units of 149,597,870.7 km, and the
// Julian year's light, 365.25 days of 86,400 s at 299,792.458 km/s
const (
	auKm         = 149597870.7
	parsecAU     = 648000 / math.Pi
	lightKmS     = 299792.458
	julianYearS  = 365.25 * 86400
	kpcKm        = 1e3 * parsecAU * auKm
	lightYearKm  = lightKmS * julianYearS
	lightYearKpc = lightYearKm / kpcKm
)

// errField is a field asked of the dipole that it does not have.
var errField = errors.New("named: the magnetar's field")

// star is a real magnetar's measured body: its radius in km, its
// surface dipole field at the equator in gauss, and its spin period in
// seconds.
type star struct {
	radius, field, period float64
}

// realStar is SGR 1806-20 as the research data holds it: the review's
// lower radius, on which the catalogue's spin-inferred field rests, and
// the catalogue's field and period.
func realStar() star {
	return star{radius: research.Value("magnetar-radius-low-km"),
		field:  research.Value("sgr1806-field-1e14-gauss") * tenToThe14,
		period: research.Value("sgr1806-period-seconds")}
}

// fieldAt is the dipole's strength at distance out from the middle, in
// km, at colatitude off the magnetic axis, in radians: the equatorial
// surface field times (radius/distance)^3 times sqrt(1 + 3 cos^2).
func (body star) fieldAt(distance, colatitude float64) float64 {
	fall := math.Pow(body.radius/distance, 3)
	slant := math.Cos(colatitude)
	return body.field * fall * math.Sqrt(1+3*slant*slant)
}

// reach is the distance in the magnetic equator, in km, at which the
// dipole has fallen to field, in gauss. A field the surface does not
// reach above, or none, is refused by name.
func (body star) reach(field float64) (float64, error) {
	if !(field > 0) || field > body.field {
		return 0, fmt.Errorf("%w: %g G is not between nothing and the "+
			"surface's %g G; ask for a weaker field", errField, field,
			body.field)
	}
	return body.radius * math.Pow(body.field/field, cubeRoot), nil
}

// danger is one distance the field's danger is drawn at: its name, the
// field in gauss, and what that field is.
type danger struct {
	name, says string
	field      float64
}

// dangers are the named fields, strongest first, each read from the
// research data.
func dangers() []danger {
	return []danger{
		{"quantum", "the vacuum itself changes how light travels",
			research.Value("quantum-critical-field-1e13-gauss") *
				tenToThe13},
		{"atoms", "atoms are squeezed into thin needles",
			research.Value("atomic-field-1e9-gauss") * tenToThe9},
		{"lab", "the strongest steady magnet ever built",
			research.Value("strongest-steady-field-tesla") * gaussTesla},
	}
}

// lightCylinder is where a point turning with the star would go at the
// speed of light, in km: c P / 2 pi. Inside it the field's lines close
// on the star and the dipole holds; beyond, some lines are open (the
// review, section 5.4.4), so no hoop is drawn by the dipole past it.
func (body star) lightCylinder() float64 {
	return lightKmS * body.period / (2 * math.Pi)
}

// zappyKm is zappy's star's radius in km, as package lantern defines
// it in its own unit of a thousand kilometres.
func zappyKm() float64 {
	return lantern.Zappy().Radius * zappyUnit
}

// earthKm and moonKm are the Earth's diameter and the Moon's mean
// distance, in km, from the research data.
func earthKm() float64 {
	return diameter * research.Value("earth-mean-radius-km")
}

// moonKm is the Moon's mean distance from the Earth, in km, from the
// research data, which holds it in millions of km.
func moonKm() float64 {
	return research.Value("moon-semimajor-1e6-km") * millionKm
}

// spikeEnergy is the 2004 spike's energy in erg at the catalogue's
// distance, from a paper's energy at the distance it took, by the
// inverse square: it shone the same every way, nothing absorbed it.
func spikeEnergy(at15 float64) float64 {
	ratio := research.Value("sgr1806-distance-kpc") /
		research.Value("hurley-flare-distance-kpc")
	return at15 * tenToThe46 * ratio * ratio
}

// sunYears is how many years the Sun takes to give out energy in erg.
func sunYears(energy float64) float64 {
	watts := research.Value("sun-luminosity-1e24-watts") * tenToThe24
	return energy / ergJoule / watts / julianYearS
}

// sunlightMinutes is how many minutes of the sunlight at the Earth carry
// as much energy to a square metre as the spike would have a light year
// from the star, from its fluence at the Earth by the inverse square.
func sunlightMinutes(fluence float64) float64 {
	away := research.Value("sgr1806-distance-kpc") / lightYearKpc
	joules := fluence * away * away / ergJoule * cmSquareM
	return joules / research.Value("solar-irradiance-w-m2") /
		secondsMinute
}

// sunlightAt is how many seconds of the sunlight at the Earth carry as
// much energy to a square metre as the spike would at out km from the
// star, from its fluence at the Earth by the inverse square.
func sunlightAt(fluence, out float64) float64 {
	away := research.Value("sgr1806-distance-kpc") * kpcKm / out
	joules := fluence * away * away / ergJoule * cmSquareM
	return joules / research.Value("solar-irradiance-w-m2")
}

// secondsMinute is the seconds in a minute, defined.
const secondsMinute = 60
