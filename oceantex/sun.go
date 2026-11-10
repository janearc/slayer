package oceantex

import (
	"math"
	"time"

	"github.com/janearc/libtheme-css/primitives/swatch"
)

var (
	anomalyEpoch   = num("sun.anomaly.epoch")
	anomalyRate    = num("sun.anomaly.rate")
	longitudeEpoch = num("sun.longitude.epoch")
	longitudeRate  = num("sun.longitude.rate")
	centreFirst    = num("sun.centre.first")
	centreSecond   = num("sun.centre.second")
	obliquityEpoch = num("sun.obliquity.epoch")
	obliquityRate  = num("sun.obliquity.rate")
	distanceMean   = num("sun.distance.mean")
	distanceFirst  = num("sun.distance.first")
	distanceSecond = num("sun.distance.second")
	siderealEpoch  = num("sidereal.epoch")
	siderealRate   = num("sidereal.rate")
	sunKelvin      = num("sun.temperature")
	airScale       = num("airmass.scale")
	airAngle       = num("airmass.angle")
	airPower       = num("airmass.power")
	rayleighScale  = num("rayleigh.scale")
	rayleighPower  = num("rayleigh.power")
	rayleighSecond = num("rayleigh.second")
	rayleighFourth = num("rayleigh.fourth")
)

// j2000 is the instant the Naval Observatory's formulas count days from:
// 2000 January 1.5, noon.
var j2000 = time.Date(2000, time.January, 1, 12, 0, 0, 0, time.UTC)

const degrees = math.Pi / 180

// Sun is where the sun is in the sky of a place at a moment: a unit
// vector with x east, y north and z up, and the sun's distance in
// astronomical units. Latitude is degrees north, longitude degrees east.
//
// The sun's place among the stars is the Naval Observatory's approximate
// formulas [USNO], good to about an arcminute; the turn of the Earth is
// Greenwich sidereal time [Me]. Refraction by the air is left out, so a
// sun on the horizon is drawn about half a degree low.
func Sun(latitude, longitude float64, when time.Time) (toward Vec, au float64) {
	days := when.Sub(j2000).Hours() / 24
	anomaly := (anomalyEpoch + anomalyRate*days) * degrees
	mean := longitudeEpoch + longitudeRate*days
	ecliptic := (mean + centreFirst*math.Sin(anomaly) +
		centreSecond*math.Sin(2*anomaly)) * degrees
	obliquity := (obliquityEpoch - obliquityRate*days) * degrees
	au = distanceMean - distanceFirst*math.Cos(anomaly) -
		distanceSecond*math.Cos(2*anomaly)

	ascension := math.Atan2(math.Cos(obliquity)*math.Sin(ecliptic),
		math.Cos(ecliptic))
	declination := math.Asin(math.Sin(obliquity) * math.Sin(ecliptic))

	sidereal := (siderealEpoch + siderealRate*days + longitude) * degrees
	hour := sidereal - ascension
	lat := latitude * degrees
	sinDec, cosDec := math.Sincos(declination)
	sinLat, cosLat := math.Sincos(lat)
	return Vec{
		X: -cosDec * math.Sin(hour),
		Y: sinDec*cosLat - cosDec*sinLat*math.Cos(hour),
		Z: sinDec*sinLat + cosDec*cosLat*math.Cos(hour),
	}, au
}

// AirMass is how much air sunlight crosses on its way down, as a multiple
// of the air straight overhead, given the cosine of the sun's angle from
// overhead [KY89]. It is 1 with the sun overhead and about 38 at the
// horizon.
func AirMass(cosine float64) float64 {
	cosine = math.Min(1, math.Max(0, cosine))
	zenith := math.Acos(cosine) / degrees
	return 1 / (cosine + airScale*math.Pow(airAngle-zenith, airPower))
}

// Haze is the optical depth of clear, clean air straight overhead at a
// wavelength in nanometres: the scattering by its molecules alone [HT74].
func Haze(nm float64) float64 {
	um := nm / 1000
	return rayleighScale / math.Pow(um, rayleighPower) *
		(1 + rayleighSecond/(um*um) + rayleighFourth/(um*um*um*um))
}

// Sunlight is the sun's light at the top of the air and at the sea, for a
// sun whose direction has this cosine from overhead, at a distance in
// astronomical units: two spectra, power by wavelength in nanometres, in
// the same units, so that the second measured against the first is how
// much the air let through.
//
// GLUE: above the air the sun is libtheme's black body at the IAU's
// nominal solar temperature. On the way down it loses only what clean
// air's molecules scatter, by Beer's law over AirMass. Ozone, water
// vapour and dust are left out, so a low sun is drawn brighter and less
// red than a real one. The light the sky scatters back down is not added.
func Sunlight(cosine, au float64) (above, below map[int]float64) {
	above = swatch.Blackbody(sunKelvin)
	below = make(map[int]float64, len(above))
	mass := AirMass(cosine)
	for nm, power := range above {
		above[nm] = power / (au * au)
		below[nm] = above[nm] * math.Exp(-Haze(float64(nm))*mass)
	}
	return above, below
}

// SunAt sets a scene's sun for a place and a moment: its direction, and
// its light after the air. The light is measured against the sun above
// the air at one astronomical unit, so a low sun is dimmer as well as
// redder. It reports whether the sun is up.
func (scene *Scene) SunAt(latitude, longitude float64, when time.Time) bool {
	toward, au := Sun(latitude, longitude, when)
	scene.Sun = toward
	_, scene.Lamp = Sunlight(toward.Z, au)
	scene.White = swatch.Blackbody(sunKelvin)
	return toward.Z > 0
}
