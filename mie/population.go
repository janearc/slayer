package mie

import (
	"errors"
	"fmt"
	"math"
)

// ErrPopulation is a population that cannot be made or asked: a radius
// not above nought, a spread not above one, a wavelength not above
// nought, or any of them not a number.
var ErrPopulation = errors.New("mie: the population is refused")

// The quadrature over a population's radii.
const (
	// quadraturePoints is how many radii Average takes. Picked: odd, so
	// the mean radius is itself one; at a spread of 1.3 neighbours are
	// under 2 percent apart in radius, a step in x of about 0.1 at x of
	// 6, a good deal finer than the interference ripple, whose period in
	// x is about pi over (m - 1), and so the sum follows the ripple's
	// average rather than aliasing it.
	quadraturePoints = 257
	// quadratureSpan is how far out, in standard deviations of the log
	// radius, the radii go, either side. Defined: the brief's four,
	// where the log-normal's weight is 3e-4 of its peak's.
	quadratureSpan = 4.0
)

// Population is a log-normal distribution of sphere radii: the logarithm
// of the radius is normally distributed, about the log of the geometric
// mean radius, with the log of the geometric standard deviation.
type Population struct {
	meanRadius, spread float64
}

// NewPopulation is the population whose geometric mean radius is the
// radius, in metres, and whose geometric standard deviation is the
// spread, a number above one: one is a single size. It refuses a radius
// not above nought and a spread not above one.
func NewPopulation(radius, spread float64) (Population, error) {
	if !(radius > 0) || !(spread > 1) {
		return Population{}, fmt.Errorf("%w: a radius of %g and a "+
			"spread of %g are not above nought and above one",
			ErrPopulation, radius, spread)
	}
	return Population{meanRadius: radius, spread: spread}, nil
}

// Averages is what a population does to light of one wavelength, a
// particle's worth.
type Averages struct {
	// Extinction is the mean extinction cross section of a particle,
	// in square metres: Q_ext times the area, averaged by number.
	Extinction float64
	// Albedo is the single-scattering albedo: the mean scattering
	// cross section over the mean extinction's.
	Albedo float64
	// Asymmetry is g averaged with the scattering cross section as its
	// weight.
	Asymmetry float64
	// Geometric is the mean geometric cross section of a particle, in
	// square metres, so that an optical depth can be turned into a
	// number density, and back: extinction per metre is the density
	// times Extinction, and the density is that over Extinction.
	Geometric float64
}

// moments are the weighted sums an average is made of.
type moments struct {
	extinction, scattering, product, geometric, weight float64
}

// add takes in one radius: its weight, its geometric cross section and
// its efficiencies.
//
// Each term is rounded by a conversion before it is summed. Without it
// the compiler may fuse a product into the sum it feeds, and the sum of
// scattering can then come out more than the sum of extinction, an albedo
// over one by a rounding.
func (total *moments) add(weight, area float64, each Efficiencies) {
	held := weight * area
	extinction := float64(held * each.Extinction)
	scattering := float64(held * each.Scattering)
	total.extinction += extinction
	total.scattering += scattering
	total.product += float64(scattering * each.Asymmetry)
	total.geometric += held
	total.weight += weight
}

// averages are the sums made means.
func (total moments) averages() Averages {
	return Averages{Extinction: total.extinction / total.weight,
		Albedo:    total.scattering / total.extinction,
		Asymmetry: total.product / total.scattering,
		Geometric: total.geometric / total.weight}
}

// Average is the population's effect on light of a wavelength in
// metres, in a medium that has the sphere's index relative to it, by a
// fixed sum over quadraturePoints radii, evenly spaced in the log of
// the radius over quadratureSpan standard deviations either side, each
// weighted by the log-normal's density there. It refuses a wavelength
// not above nought, and what Sphere refuses of any radius in the sum.
func (population Population) Average(wavelength float64,
	index complex128) (Averages, error) {
	if !(wavelength > 0) {
		return Averages{}, fmt.Errorf("%w: a wavelength of %g is not "+
			"above nought", ErrPopulation, wavelength)
	}
	var total moments
	width := math.Log(population.spread)
	for point := range quadraturePoints {
		fraction := float64(point) / (quadraturePoints - 1)
		offset := quadratureSpan * (2*fraction - 1) // defined: from minus the span to plus it
		radius := population.meanRadius * math.Exp(offset*width)
		each, err := Sphere(2*math.Pi*radius/wavelength, index) // defined: x = 2 pi r / lambda
		if err != nil {
			return Averages{}, err
		}
		total.add(math.Exp(-offset*offset/2), math.Pi*radius*radius, each) // defined: the normal density's shape, and a circle's area
	}
	return total.averages(), nil
}
