package airlight

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/janearc/slayer/diskcache"
	"github.com/janearc/slayer/skylight"
)

// Plan is everything that decides an air's fields: the air's name, its
// column at each wavelength cut at the set's ground, the sun at its top
// there, the wavelengths and the band each stands for, the grid, the
// photons, the seed, the workers and the solver's version.
type Plan struct {
	name        string
	ground      float64
	wavelengths []int
	band        float64
	columns     []skylight.Column
	sun         []float64
	grid        skylight.Grid
	photons     int
	seed        uint64
	workers     int
	version     string
}

// NewPlan is the plan for a known air seen from a ground so many metres
// above the planet's mean radius, each wavelength walked with its
// photons, on its bins. It refuses an odd count of view bins, whose
// middle bin would hold the light from above and below the horizon in
// one; what the air refuses; and a ground at or above its top.
func NewPlan(known Known, ground float64) (Plan, error) {
	if known.Views%two != none {
		return Plan{}, fmt.Errorf("airlight: %d view bins: the horizon "+
			"falls between two only when the count is even", known.Views)
	}
	air := known.Air
	plan := Plan{name: air.Name(), ground: ground,
		wavelengths: wavelengths(), band: everyNanometres,
		grid: skylight.Grid{Heights: []float64{none}, Zeniths: zenithBins,
			Views: known.Views, Azimuths: known.Azimuths},
		photons: known.Photons, seed: seed, workers: workers,
		version: solverVersion}
	for _, nanometres := range plan.wavelengths {
		column, sun, err := worked(air, float64(nanometres), ground)
		if err != nil {
			return Plan{}, err
		}
		plan.columns = append(plan.columns, column)
		plan.sun = append(plan.sun, sun)
	}
	return plan, nil
}

// wavelengths is the span the fields are worked over, every step.
func wavelengths() []int {
	var made []int
	for nanometres := firstNanometre; nanometres <= lastNanometre; {
		made = append(made, nanometres)
		nanometres += everyNanometres
	}
	return made
}

// worked is an air's column at a wavelength, cut at the ground, and the
// sun at its top there.
func worked(air Air, nanometres, ground float64) (skylight.Column,
	float64, error) {
	column, err := air.Column(nanometres)
	if err == nil {
		column, err = grounded(column, ground)
	}
	if err != nil {
		return column, none, fmt.Errorf("airlight: %s at %g nm: %w",
			air.Name(), nanometres, err)
	}
	sun, err := air.Sun(nanometres)
	return column, sun, err
}

// Said is the plan in words, for a run to say what it worked out: the
// air, its wavelengths and the photons each.
func (plan Plan) Said() string {
	return fmt.Sprintf("%s's air at %d wavelengths from %d to %d nm, %d "+
		"photons each", plan.name, len(plan.wavelengths), firstNanometre,
		lastNanometre, plan.photons)
}

// Keyed is a key with every part of the plan said: what a cache keeps
// the fields under. The columns and the sun are kept as the bytes of
// every number in them, so a change to the air's data or to the code
// that works its columns moves the key.
func (plan Plan) Keyed(key diskcache.Key) diskcache.Key {
	grid := []int{plan.grid.Zeniths, plan.grid.Views, plan.grid.Azimuths}
	heights := make([]string, len(plan.grid.Heights))
	for index, height := range plan.grid.Heights {
		heights[index] = exactly(height)
	}
	return key.Text("air", plan.name).
		Bytes("columns", columnBytes(plan.columns)).
		Bytes("sun", numberBytes(plan.sun)).
		Text("ground", exactly(plan.ground)).
		Text("wavelengths", wholes(plan.wavelengths)+" every "+
			exactly(plan.band)).
		Text("grid", wholes(grid)+" at "+strings.Join(heights, " ")).
		Text("photons", strconv.Itoa(plan.photons)).
		Text("seed", strconv.FormatUint(plan.seed, decimal)).
		Text("workers", strconv.Itoa(plan.workers)).
		Text("solver", plan.version)
}

// Solve walks each wavelength's column and keeps its field: the
// radiance in every bin, the counts behind them, the direct sun's share
// at each step of the sun's zenith, and the share of the sunlight on
// the planet that its ground absorbed.
func (plan Plan) Solve() (*Fields, error) {
	fields := newFields(plan)
	for wave, column := range plan.columns {
		field, spent, err := skylight.Solve(column, plan.grid, plan.photons,
			plan.seed, plan.workers)
		if err != nil {
			return nil, fmt.Errorf("airlight: %s at %d nm: %w", plan.name,
				plan.wavelengths[wave], err)
		}
		fields.keep(wave, field, spent)
	}
	return fields, nil
}

// exactly is a float64 as a key says it: the shortest decimal that reads
// back as the same float64.
func exactly(number float64) string {
	return strconv.FormatFloat(number, 'g', shortest, float64Bits)
}

// wholes is whole numbers as words, a space between.
func wholes(numbers []int) string {
	words := make([]string, len(numbers))
	for index, number := range numbers {
		words[index] = strconv.Itoa(number)
	}
	return strings.Join(words, " ")
}

// columnBytes is every number of the columns, in order: each column's
// radius, ground albedo and count of shells, then each shell's seven.
func columnBytes(columns []skylight.Column) []byte {
	var numbers []float64
	for _, column := range columns {
		numbers = append(numbers, column.Radius, column.GroundAlbedo,
			float64(len(column.Shells)))
		for _, shell := range column.Shells {
			numbers = append(numbers, shell.Bottom, shell.Top,
				shell.GasScattering, shell.GasAbsorption,
				shell.ParticleExtinction, shell.ParticleAlbedo,
				shell.ParticleAsymmetry)
		}
	}
	return numberBytes(numbers)
}

// numberBytes is numbers as bytes: each a float64's bits, little-endian.
func numberBytes(numbers []float64) []byte {
	made := make([]byte, none, len(numbers)*float64Bytes)
	for _, number := range numbers {
		made = binary.LittleEndian.AppendUint64(made,
			math.Float64bits(number))
	}
	return made
}
