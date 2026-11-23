// Package daylight computes the geometric direction of sunlight at a place
// and instant, independently of any scene's lighting or atmosphere.
package daylight

import (
	"fmt"
	"math"
	"time"
)

// SunPosition is the direction to the centre of the sun. Azimuth is in
// degrees clockwise from north and altitude is degrees above the geometric
// horizon. Daylight says the centre is above that horizon; atmospheric
// refraction, twilight and the sun's apparent disc are not included.
type SunPosition struct {
	Azimuth, Altitude float64
	Daylight          bool
}

// Sun estimates solar direction for latitude and longitude in degrees, with
// east longitude positive. at denotes an instant: any Go location, including
// daylight-saving offsets, is converted to UTC before the calculation.
//
// This uses NOAA's fractional-year approximation and produces geometric
// coordinates. Weather, clouds, refraction, moonlight, and actual brightness
// require separate models. At the zenith or nadir, azimuth is undefined and
// the returned value is zero.
//
// https://gml.noaa.gov/grad/solcalc/solareqns.PDF
func Sun(latitude, longitude float64, at time.Time) (SunPosition, error) {
	if math.IsNaN(latitude) || math.IsInf(latitude, 0) || latitude < -90 || latitude > 90 ||
		math.IsNaN(longitude) || math.IsInf(longitude, 0) || longitude < -180 || longitude > 180 {
		return SunPosition{}, fmt.Errorf("daylight: latitude and longitude must be finite and within geographic bounds")
	}
	at = at.UTC()
	year, _, _ := at.Date()
	if year < 1 || year > 9999 {
		return SunPosition{}, fmt.Errorf("daylight: year must be between 1 and 9999")
	}
	days := 365.0
	if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
		days = 366
	}
	hour := float64(at.Hour()) + float64(at.Minute())/60 +
		(float64(at.Second())+float64(at.Nanosecond())/1e9)/3600
	gamma := 2 * math.Pi / days * (float64(at.YearDay()-1) + (hour-12)/24)
	equation := 229.18 * (0.000075 + 0.001868*math.Cos(gamma) -
		0.032077*math.Sin(gamma) - 0.014615*math.Cos(2*gamma) -
		0.040849*math.Sin(2*gamma))
	declination := 0.006918 - 0.399912*math.Cos(gamma) +
		0.070257*math.Sin(gamma) - 0.006758*math.Cos(2*gamma) +
		0.000907*math.Sin(2*gamma) - 0.002697*math.Cos(3*gamma) +
		0.00148*math.Sin(3*gamma)
	solarMinutes := hour*60 + equation + 4*longitude
	hourAngle := (solarMinutes/4 - 180) * math.Pi / 180
	lat := latitude * math.Pi / 180
	east := -math.Cos(declination) * math.Sin(hourAngle)
	north := math.Cos(lat)*math.Sin(declination) -
		math.Sin(lat)*math.Cos(declination)*math.Cos(hourAngle)
	up := math.Sin(lat)*math.Sin(declination) +
		math.Cos(lat)*math.Cos(declination)*math.Cos(hourAngle)
	altitude := math.Asin(math.Max(-1, math.Min(1, up))) * 180 / math.Pi
	azimuth := 0.0
	if math.Hypot(east, north) > 1e-14 {
		azimuth = math.Mod(math.Atan2(east, north)*180/math.Pi+360, 360)
	}
	return SunPosition{Azimuth: azimuth, Altitude: altitude, Daylight: altitude > 0}, nil
}
