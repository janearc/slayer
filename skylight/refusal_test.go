package skylight

import (
	"errors"
	"math"
	"testing"
)

// offBy is how far outside its range a refused number is put.
const offBy = 0.5 // picked: plainly outside, not a rounding

// a call is what Solve is given.
type call struct {
	column           Column
	grid             Grid
	photons, workers int
}

// goodCall is a call Solve accepts: thin air kept at the ground, the
// boundary between its shells and its top.
func goodCall() call {
	grid := groundGrid()
	grid.Heights = []float64{none, lowTop, airTop}
	return call{thinRayleigh(), grid, fewPhotons, workersOf}
}

// named are every refusal Solve gives.
var named = []error{errPhotons, errWorkers, errRadius, ErrShells, errGap,
	errOverlap, errThickness, errCoefficient, errAlbedo, errAsymmetry,
	errBins, errHeight, errOrder}

// refusals are broken calls, one wrong thing in each, and the refusal
// each must name.
var refusals = []struct {
	name   string
	want   error
	change func(*call)
}{
	{"a gap", errGap, func(c *call) { c.column.Shells[one].Bottom += one }},
	{"a gap at the ground", errGap,
		func(c *call) { c.column.Shells[none].Bottom = one }},
	{"an overlap", errOverlap,
		func(c *call) { c.column.Shells[one].Bottom -= one }},
	{"a shell of no thickness", errThickness,
		func(c *call) { c.column.Shells[none].Top = none }},
	{"an endless top", errThickness,
		func(c *call) { c.column.Shells[one].Top = math.Inf(one) }},
	{"negative gas scattering", errCoefficient,
		func(c *call) { c.column.Shells[none].GasScattering = -offBy }},
	{"negative gas absorption", errCoefficient,
		func(c *call) { c.column.Shells[one].GasAbsorption = -offBy }},
	{"negative particle extinction", errCoefficient,
		func(c *call) { c.column.Shells[none].ParticleExtinction = -offBy }},
	{"a coefficient of no number", errCoefficient,
		func(c *call) { c.column.Shells[none].GasScattering = math.NaN() }},
	{"a particle albedo over one", errAlbedo,
		func(c *call) { c.column.Shells[none].ParticleAlbedo = one + offBy }},
	{"a particle albedo under nought", errAlbedo,
		func(c *call) { c.column.Shells[one].ParticleAlbedo = -offBy }},
	{"a ground albedo over one", errAlbedo,
		func(c *call) { c.column.GroundAlbedo = one + offBy }},
	{"an asymmetry of one", errAsymmetry,
		func(c *call) { c.column.Shells[none].ParticleAsymmetry = one }},
	{"an asymmetry under minus one", errAsymmetry, func(c *call) {
		c.column.Shells[one].ParticleAsymmetry = -one - offBy
	}},
	{"a height above the top", errHeight,
		func(c *call) { c.grid.Heights = []float64{none, airTop + one} }},
	{"a height below the ground", errHeight,
		func(c *call) { c.grid.Heights = []float64{-one, none} }},
	{"no heights", errHeight, func(c *call) { c.grid.Heights = nil }},
	{"heights out of order", errOrder,
		func(c *call) { c.grid.Heights = []float64{lowTop, none} }},
	{"a height twice", errOrder,
		func(c *call) { c.grid.Heights = []float64{lowTop, lowTop} }},
	{"no bins", errBins, func(c *call) { c.grid.Azimuths = none }},
	{"no radius", errRadius, func(c *call) { c.column.Radius = none }},
	{"no shells", ErrShells, func(c *call) { c.column.Shells = nil }},
	{"no photons", errPhotons, func(c *call) { c.photons = none }},
	{"no workers", errWorkers, func(c *call) { c.workers = none }},
}

// solveCall is Solve's refusal of a call, if any.
func solveCall(c call) error {
	_, _, err := Solve(c.column, c.grid, c.photons, seedOf, c.workers)
	return err
}

// each broken call is refused by the name of what is broken, and by no
// other name.
func TestRefusalsByName(t *testing.T) {
	for _, each := range refusals {
		broken := goodCall()
		each.change(&broken)
		err := solveCall(broken)
		if !errors.Is(err, each.want) {
			t.Errorf("%s: %v, want %v", each.name, err, each.want)
		}
		for _, other := range named {
			if other != each.want && errors.Is(err, other) {
				t.Errorf("%s: named %v as well", each.name, other)
			}
		}
	}
}

// the anti-test: the call each broken one was made from is accepted, so
// each refusal is of the one thing changed and not of the call.
func TestTheGoodCallIsAccepted(t *testing.T) {
	if err := solveCall(goodCall()); err != nil {
		t.Errorf("the good call is refused: %v", err)
	}
}
