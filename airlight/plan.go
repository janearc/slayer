package airlight

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/janearc/slayer/diskcache"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/skylight"
)

// Plan is everything that decides an air's fields: the air's name, its
// column at each wavelength cut at the set's ground, the sun at its top
// there, the wavelengths and the band each stands for, the grid, the
// span of the sun's zenith walked for, the photons, the crossings
// wanted in the bins read and the most photons walked to have them,
// the seed, the workers and the solver's version.
type Plan struct {
	name        string
	ground      float64
	wavelengths []int
	band        float64
	columns     []skylight.Column
	sun         []float64
	grid        skylight.Grid
	suns        skylight.Suns
	photons     int
	wanted      float64
	most        int
	seed        uint64
	workers     int
	version     string
}

// For is the plan walked for a sun along a way alone: for the span of
// the two zenith bins a reading of that sun is drawn between, so its
// photons start only where their light reaches the ground under it.
//
// At the same photons the bins it reads hold so many times the
// crossings as the planet's disc is to the band they start on: about
// ten for a sun two degrees up, four at seventeen, two at forty-seven.
//
// It walks as many photons as bring those bins to the crossings that
// leave the colour noise of one 8-bit step, from a first walk of its
// middle wavelength; never fewer than the air's, nor more than
// mostPhotons.
func (plan Plan) For(way maths.Vec) Plan {
	plan.suns = readSpan(zenithOf(way), plan.grid.Zeniths)
	step := colourNoiseAtOne / colourStep
	plan.wanted, plan.most = step*step, mostPhotons
	return plan
}

// enough is the photons each wavelength is walked with: the plan's,
// where it wants no crossings; and otherwise the plan's scaled by the
// crossings it wants over those a walk of its middle wavelength with
// its photons held, from the plan's photons to its most. That first
// walk is from a seed of its own and is not kept: the count is chosen
// from it, so counting it too would let the count lean on its sample.
func (plan Plan) enough() (int, error) {
	if !(plan.wanted > none) {
		return plan.photons, nil
	}
	middle := len(plan.columns) / two
	first, spent, err := skylight.SolveFor(plan.columns[middle], plan.grid,
		plan.suns, plan.photons, plan.seed+pilotSeedStep, plan.workers)
	if err != nil {
		return none, fmt.Errorf("airlight: %s at %d nm: %w", plan.name,
			plan.wavelengths[middle], err)
	}
	pilot := newFields(plan)
	pilot.keep(middle, first, spent)
	held := pilot.held(plan.suns)
	if !(held > none) {
		return plan.most, nil
	}
	scaled := math.Ceil(float64(plan.photons) * plan.wanted / held)
	return int(min(max(scaled, float64(plan.photons)),
		float64(plan.most))), nil
}

// readSpan is the span of the sun's zenith, in radians, of the two bins
// of zeniths equal bins that a sun at a zenith is read between.
func readSpan(sunZenith float64, zeniths int) skylight.Suns {
	at := axisOf(sunZenith/math.Pi, zeniths)
	width := math.Pi / float64(zeniths)
	return skylight.Suns{From: float64(at.low) * width,
		To: float64(at.high+one) * width}
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
		suns:    skylight.Suns{From: none, To: math.Pi},
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
func worked(air air, nanometres, ground float64) (skylight.Column,
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
// air, its wavelengths, the photons each and the suns they are for.
func (plan Plan) Said() string {
	photons := fmt.Sprintf("%d photons each", plan.photons)
	if plan.wanted > none {
		photons = fmt.Sprintf("%d to %d photons each, as many as bring the "+
			"bins read to %.0f crossings", plan.photons, plan.most,
			plan.wanted)
	}
	return fmt.Sprintf("%s's air at %d wavelengths from %d to %d nm, %s, "+
		"for a sun %.0f to %.0f degrees from up", plan.name,
		len(plan.wavelengths), firstNanometre, lastNanometre, photons,
		plan.suns.From/maths.Degree, plan.suns.To/maths.Degree)
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
		Text("suns", exactly(plan.suns.From)+" to "+
			exactly(plan.suns.To)).
		Text("photons", strconv.Itoa(plan.photons)).
		Text("crossings", exactly(plan.wanted)+" to at most "+
			strconv.Itoa(plan.most)).
		Text("seed", strconv.FormatUint(plan.seed, decimal)).
		Text("workers", strconv.Itoa(plan.workers)).
		Text("solver", plan.version)
}

// Solve walks each wavelength's column and keeps its field: the
// radiance in every bin, the counts behind them, the direct sun's share
// at each step of the sun's zenith, and the share of the sunlight on
// the part of the planet walked that its ground absorbed.
func (plan Plan) Solve() (*Fields, error) {
	photons, err := plan.enough()
	if err != nil {
		return nil, err
	}
	fields := newFields(plan)
	for wave, column := range plan.columns {
		field, spent, err := skylight.SolveFor(column, plan.grid, plan.suns,
			photons, plan.seed, plan.workers)
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
