package generate

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/celestial/measure"
)

// A planet is tidally locked when its day is its year: it turns once for
// each time round its sun, so the same face stays toward it. The tides
// the sun raises have slowed every close planet to this, as the Earth has
// slowed the Moon. The body is given Locked, and its Day follows from its
// orbit; a Day given with it must agree with the orbit's year to
// keplerSlack, as a Year given with a Distance does.

// Year is the time the planet takes to go once round its sun, by Kepler's
// third law for the two masses together, or the refusal of an orbit that
// has no sun's mass or no distance to work it from.
func (planet Planet) Year() (time.Duration, error) {
	metres, err := distance(planet)
	if err != nil {
		return 0, err
	}
	if !planet.Sun.Mass.Positive() || !(metres > 0) {
		return 0, fmt.Errorf("generate: %s has no sun's mass and "+
			"distance to work a year from", planet.Name)
	}
	heavy := measure.Kilograms(planet.Sun.Mass.InKilograms() +
		planet.Mass.InKilograms())
	pull := measure.GM(heavy).InMetres() * speedOfLight * speedOfLight
	seconds := 2 * math.Pi * math.Sqrt(metres*metres*metres/pull)
	return time.Duration(seconds * float64(time.Second)), nil
}

// Turn is the time the planet takes to turn once: the Day it is given,
// or, where it is Locked, its year. A locked planet given a Day that is
// not its year is refused, as is one with no orbit to take a year from.
func Turn(planet Planet) (time.Duration, error) {
	if !planet.Locked {
		return planet.Day, nil
	}
	year, err := planet.Year()
	if err != nil {
		return 0, err
	}
	given := math.Abs(float64(planet.Day))
	if given != 0 && math.Abs(given-float64(year)) >
		keplerSlack*float64(year) {
		return 0, fmt.Errorf("generate: %s is locked, so its day is its "+
			"year of %.2f hours, not the %.2f given", planet.Name,
			year.Hours(), planet.Day.Hours())
	}
	return year, nil
}
