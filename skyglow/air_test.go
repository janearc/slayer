package skyglow

import (
	"math"
	"testing"
)

// cityAir is the air the neon street is worked in: three kilometres of
// visibility by Koschmieder, 3.912 over the visibility, four hundred
// metres deep, and haze that scatters nine tenths of what it meets.
func cityAir() Air {
	return Air{Extinction: 1.3e-3, Depth: 400, Albedo: 0.9}
}

// simpsonSlices is how many pieces the tests cut a path into to add up
// the density along it. Even, for Simpson's rule.
const simpsonSlices = 2000

// columnTolerance is how far a closed form may be from the sum a test
// makes: Simpson's rule is exact for the cubic the thinning is, so only
// the rounding of the sums is left.
const columnTolerance = 1e-9

// integrated is the density of the air added up by Simpson's rule along
// a straight path that rises from one height to another, per metre of
// the path's length times its length.
func integrated(air Air, from, to, length float64) float64 {
	width := 1.0 / simpsonSlices
	sum := air.density(from) + air.density(to)
	for slice := 1; slice < simpsonSlices; slice++ {
		weight := 2.0
		if slice%2 == 1 {
			weight = 4.0
		}
		sum += weight * air.density(from+float64(slice)*width*(to-from))
	}
	return sum * width / 3 * length
}

// The column is the integral of the density: the closed form of a
// smooth step's thinning against the sum a test makes.
func TestColumnIsTheIntegralOfDensity(t *testing.T) {
	air := cityAir()
	for _, height := range []float64{10, 100, 250, 399} {
		want := integrated(air, 0, height, height)
		if got := air.column(height); math.Abs(got-want) > columnTolerance {
			t.Errorf("column to %g m is %g, the sum is %g", height, got, want)
		}
	}
}

// Above the depth the air has thinned to nothing, so the column holds: it
// is the extinction times the depth, halved, the mean of a smooth step.
func TestColumnHoldsAboveTheDepth(t *testing.T) {
	air := cityAir()
	whole := air.Extinction * air.Depth / 2
	for _, height := range []float64{400, 500, 5000} {
		if got := air.column(height); math.Abs(got-whole) > columnTolerance {
			t.Errorf("column to %g m is %g, wanted %g", height, got, whole)
		}
	}
}

// The optical depth of a path is the change in the column over the change
// in height, times its length: the sum along a slanted path agrees.
func TestOpticalIsTheDensityAddedAlongThePath(t *testing.T) {
	air := cityAir()
	for _, path := range [][3]float64{{0, 300, 500}, {300, 20, 900},
		{50, 380, 400}, {0, 0.5e-6, 700}} {
		want := integrated(air, path[0], path[1], path[2])
		got := air.optical(path[0], path[1], path[2])
		if math.Abs(got-want) > columnTolerance {
			t.Errorf("path %v: optical depth %g, the sum is %g", path, got,
				want)
		}
	}
}

// A path along one height has the density of that height times its
// length, with no division by a rise of nothing.
func TestOpticalOfALevelPathIsDensityTimesLength(t *testing.T) {
	air := cityAir()
	got := air.optical(100, 100, 250)
	if want := air.density(100) * 250; math.Abs(got-want) > columnTolerance {
		t.Errorf("level path: %g, wanted %g", got, want)
	}
}

// Anti-test: air has nothing below its base or above its depth, and a
// path through nothing has no optical depth.
func TestThereIsNoAirBelowTheBaseOrAboveTheDepth(t *testing.T) {
	air := Air{Extinction: 1e-3, Depth: 300, Albedo: 0.9, Base: 50}
	for _, height := range []float64{-10, 0, 49.9, 350, 1000} {
		if density := air.density(height); density != 0 {
			t.Errorf("density at %g m is %g, wanted none", height, density)
		}
	}
	if tau := air.optical(-80, -10, 300); tau != 0 {
		t.Errorf("a path under the base has optical depth %g", tau)
	}
	if tau := air.optical(400, 900, 700); tau != 0 {
		t.Errorf("a path over the depth has optical depth %g", tau)
	}
}

// Anti-test: air that is not air is not valid, each by what is wrong.
func TestAirThatIsNotAirIsNotValid(t *testing.T) {
	good := cityAir()
	bad := map[string]Air{
		"no extinction":   {Depth: 400, Albedo: 0.9},
		"no depth":        {Extinction: 1e-3, Albedo: 0.9},
		"no albedo":       {Extinction: 1e-3, Depth: 400},
		"albedo over one": {Extinction: 1e-3, Depth: 400, Albedo: 1.1},
		"not a number": {Extinction: math.NaN(), Depth: 400,
			Albedo: 0.9},
		"infinite depth": {Extinction: 1e-3, Depth: math.Inf(1),
			Albedo: 0.9},
		"infinite base": {Extinction: 1e-3, Depth: 400, Albedo: 0.9,
			Base: math.Inf(-1)},
	}
	if !good.Valid() {
		t.Fatal("the city's air is not valid")
	}
	for name, air := range bad {
		if air.Valid() {
			t.Errorf("%s: %+v is valid", name, air)
		}
	}
}
