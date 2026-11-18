// Package skylight works out the light in a planet's air at one
// wavelength: the sunlight that air and ground have scattered, by
// height, by the sun's angle from the local vertical, and by the way an
// observer looks, for a planet lit by the sun's parallel rays.
//
// It is the core of slayer's computed sky (doc/sky-from-the-air.md). A
// caller gives the air as spherical shells, each uniform within, and
// the ground as a Lambertian reflector. Solve follows photons through
// them, scattering each as many times as the air asks.
//
// Earth's thin air, where most skylight has scattered once, and the air
// under Venus's cloud, where it has scattered hundreds of times, go
// through the same code. The Field Solve returns reads the radiance
// back, with the direct sun's share at any height and angle.
//
// What it leaves out: polarisation, since Rayleigh's light is polarised
// and this radiance is not; light the air or the ground give off;
// refraction; and air that is not layered, which is the volume
// tracer's. The sun is a point: its disc's half degree is not modelled.
package skylight

import "sync"

// Shell is one layer of air between two heights above the ground, in
// metres, uniform within, at one wavelength. Its gas scatters by
// Rayleigh's phase, three quarters of one and the turn's cosine
// squared, which sums to four pi over every direction.
//
// Real molecules depolarise light a little, which changes Rayleigh's
// phase by a percent or so; that is ignored here. The particles scatter
// by the Henyey-Greenstein phase of their asymmetry.
type Shell struct {
	// Bottom and Top are heights above the ground, in metres.
	Bottom, Top float64
	// GasScattering is the gas's Rayleigh scattering, per metre.
	GasScattering float64
	// GasAbsorption is what the gas takes and keeps, per metre.
	GasAbsorption float64
	// ParticleExtinction is what the particles scatter and absorb
	// together, per metre.
	ParticleExtinction float64
	// ParticleAlbedo is the share of the particles' extinction that
	// they scatter, from nought to one.
	ParticleAlbedo float64
	// ParticleAsymmetry is the mean cosine of the particles' turn, the
	// Henyey-Greenstein g, strictly between minus one and one.
	ParticleAsymmetry float64
}

// Column is a planet's air at one wavelength: the planet's radius in
// metres, its shells from the ground up, and its ground's albedo.
type Column struct {
	// Radius is the planet's, to its ground, in metres.
	Radius float64
	// Shells run from the ground up, each beginning where the one
	// below ends.
	Shells []Shell
	// GroundAlbedo is the share of light the ground sends back, alike
	// in every direction, from nought to one.
	GroundAlbedo float64
}

// Grid says where Solve keeps radiance and how finely. Each kept height
// lies in the column, from the ground to the top; one inside a shell
// splits the shell there, which changes nothing in the air.
//
// The bins are uniform: the local solar zenith from nought to pi, the
// view's cosine from local up from minus one to one, and the azimuth
// between the view and the sun from nought to pi.
type Grid struct {
	// Heights are where radiance is kept, in metres, rising.
	Heights []float64
	// Zeniths, Views and Azimuths are the counts of bins.
	Zeniths, Views, Azimuths int
}

// Budget is where the sunlight went, as power for unit irradiance
// perpendicular to the sun's rays, so in square metres. Escaped, Air,
// Ground and Capped add to Launched within Russian roulette's noise.
type Budget struct {
	// Launched is the light that crossed the top sphere's disc.
	Launched float64
	// Escaped is what left the top of the air.
	Escaped float64
	// Air and Ground are what each absorbed.
	Air, Ground float64
	// Capped is what photons still carried when the cap on events
	// stopped them.
	Capped float64
}

// Solve walks photons through the column and keeps the light that
// crosses the grid's heights, with where all the light went. The same
// seed and number of workers give the same field and budget: each
// worker draws from its own generator, from the seed and its index.
//
// Each worker walks a fixed share of the photons, and their tallies are
// added in the workers' order. Photons is how many cross the disc of
// the top sphere, facing the sun, for the whole planet.
func Solve(column Column, grid Grid, photons int, seed uint64,
	workers int) (*Field, Budget, error) {
	field, spent, _, err := solve(column, grid, photons, seed, workers,
		mostEvents)
	return field, spent, err
}

// solve is Solve with the cap on events per photon given, and with what
// the walk counted besides power.
func solve(column Column, grid Grid, photons int, seed uint64,
	workers, most int) (*Field, Budget, census, error) {
	if err := check(column, grid, photons, workers); err != nil {
		return nil, Budget{}, census{}, err
	}
	made := layered(column, grid.Heights)
	walkers := make([]*walker, workers)
	var group sync.WaitGroup
	for index := range walkers {
		walkers[index] = newWalker(&made, grid, seed, index, most)
		share := shareOf(photons, workers, index)
		group.Go(func() { walkers[index].run(share) })
	}
	group.Wait()
	field, spent, counted := gather(&made, grid, walkers, photons)
	return field, spent, counted, nil
}

// shareOf is how many of the photons the worker at index walks: an
// even share, and one more for the first workers while a remainder
// lasts.
func shareOf(photons, workers, index int) int {
	share := photons / workers
	if index < photons%workers {
		share++
	}
	return share
}
