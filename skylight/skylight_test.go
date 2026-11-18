package skylight

import (
	"fmt"
	"math"
)

// The numbers the tests share.
const (
	earthRadius = 6.371e6  // picked: Earth's mean radius, 6,371 km
	airTop      = 1e5      // picked: a hundred kilometres of air
	lowTop      = 8e3      // picked: about one scale height of Earth's air
	farRadius   = 1e9      // brief: a planet so large its air is a plane
	thinTop     = 1e3      // brief: one shell a kilometre thick
	seedOf      = 20261115 // picked: the day the package was written
	workersOf   = 4        // brief: four workers, the machine's share
	fewPhotons  = 20000    // picked: enough for a budget in a second
	sigmas      = 5        // picked: by chance, fails once in millions
	coarseBins  = 4        // picked: enough bins to hold a tally
)

// degree is a degree, in radians.
const degree = math.Pi / 180 // defined: a half turn has 180

// uniformColumn is a column of one shell from the ground to top.
func uniformColumn(radius, top float64, shell Shell,
	ground float64) Column {
	shell.Bottom, shell.Top = none, top
	return Column{Radius: radius, Shells: []Shell{shell},
		GroundAlbedo: ground}
}

// groundGrid keeps radiance at the ground only, coarsely.
func groundGrid() Grid {
	return Grid{Heights: []float64{none}, Zeniths: coarseBins,
		Views: coarseBins, Azimuths: coarseBins}
}

// keptGrid keeps radiance at the ground, halfway up and at the top of a
// column airTop high.
func keptGrid() Grid {
	grid := groundGrid()
	grid.Heights = []float64{none, airTop / two, airTop}
	return grid
}

// within is nil when got is want to within sigmas standard errors, for
// a mean of draws samples whose variance is spread.
func within(got, want, spread float64, draws int) error {
	allowed := sigmas * math.Sqrt(spread/float64(draws))
	if !(math.Abs(got-want) <= allowed) {
		return fmt.Errorf("%.6g, want %.6g within %.3g", got, want,
			allowed)
	}
	return nil
}
