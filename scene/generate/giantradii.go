package generate

import (
	_ "embed"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// A giant planet's radius is read from the tables of Fortney, Marley &
// Barnes 2007 (ApJ 659, 1661): their tables 2, 3 and 4 give it for twelve
// masses, five core masses, five distances from the Sun and three ages
// (300 Myr, 1 Gyr and 4.5 Gyr). The rows are in fortney2007.txt, as the
// paper prints them. A planet between the rows is interpolated: in the
// logarithm of mass, distance and age, and linearly in core mass.

// fortneyText is the tables as the paper prints them.
//
//go:embed fortney2007.txt
var fortneyText string

// jupiterEquatorial is Jupiter's equatorial radius at one bar in metres,
// 71492 km, the unit the tables' radii are in (Fortney et al. 2007,
// page 11).
const jupiterEquatorial = 71492e3

// giantMasses are the twelve masses of the tables' columns in Earth
// masses (Fortney et al. 2007, table 2, second header row).
var giantMasses = [...]float64{
	17, 28, 46, 77, 129, 215, 318, 464, 774, 1292, 2154, 3594,
}

// giantGrid is the tables: the ages in gigayears, the distances in
// astronomical units, the core masses in Earth masses, and the radius in
// Jupiter radii by age, distance, core and mass column. An empty cell of
// the paper is not a number.
type giantGrid struct {
	ages, distances, cores []float64
	radius                 [][][][]float64
}

// number is a field of the tables as a number: not a number for the
// star and the dash the paper leaves in an empty cell.
func number(field string) float64 {
	value, err := strconv.ParseFloat(field, 64)
	if err != nil {
		return math.NaN()
	}
	return value
}

// parseGiantGrid reads the tables' text. A line that starts with age,
// distance or core opens an age, a distance, or a row of one core's
// twelve radii; every other line is the file's own words.
func parseGiantGrid(text string) giantGrid {
	var grid giantGrid
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value := number(fields[1])
		switch fields[0] {
		case "age":
			grid.ages = append(grid.ages, value)
			grid.radius = append(grid.radius, nil)
		case "distance":
			age := len(grid.ages) - 1
			if age == 0 {
				grid.distances = append(grid.distances, value)
			}
			grid.radius[age] = append(grid.radius[age], nil)
		case "core":
			grid.addRow(value, fields[2:])
		}
	}
	return grid
}

// addRow adds a row of radii for a core mass to the distance last opened,
// and notes the core mass the first time round.
func (grid *giantGrid) addRow(core float64, cells []string) {
	age := len(grid.ages) - 1
	distance := len(grid.radius[age]) - 1
	if age == 0 && distance == 0 {
		grid.cores = append(grid.cores, core)
	}
	row := make([]float64, len(cells))
	for each, cell := range cells {
		row[each] = number(cell)
	}
	grid.radius[age][distance] = append(grid.radius[age][distance], row)
}

// fortney is the tables, read once.
var fortney = sync.OnceValue(func() giantGrid {
	return parseGiantGrid(fortneyText)
})

// identity is a scale that leaves a number as it is.
func identity(value float64) float64 { return value }

// bracket is the node at or below a value, and how far the value is
// from it toward the next, as a share on a scale. A value off either
// end is held at that end.
func bracket(nodes []float64, value float64,
	scale func(float64) float64) (int, float64) {
	last := len(nodes) - 1
	switch {
	case value <= nodes[0]:
		return 0, 0
	case value >= nodes[last]:
		return last - 1, 1
	}
	below := sort.SearchFloat64s(nodes, value) - 1
	span := scale(nodes[below+1]) - scale(nodes[below])
	return below, (scale(value) - scale(nodes[below])) / span
}

// mix is the point a share of the way from one number to another.
func mix(from, to, share float64) float64 {
	return from + (to-from)*share
}

// coreRadius is the radius for one core mass in the cells of a column
// that the paper fills: between the two nearest filled cores, and held at
// the nearest when the core is past them all, as it is for a core that
// outweighs the planet.
func (grid giantGrid) coreRadius(age, distance, column int,
	core float64) float64 {
	var cores, radii []float64
	for row, each := range grid.cores {
		cell := grid.radius[age][distance][row][column]
		if !math.IsNaN(cell) {
			cores = append(cores, each)
			radii = append(radii, cell)
		}
	}
	if len(cores) == 1 {
		return radii[0]
	}
	below, share := bracket(cores, core, identity)
	return mix(radii[below], radii[below+1], share)
}

// columnRadius is the radius at one mass column for a core mass, a
// distance and an age: the four tables' corners, mixed.
func (grid giantGrid) columnRadius(column int,
	core, au, gigayears float64) float64 {
	age, ageShare := bracket(grid.ages, gigayears, math.Log)
	near, nearShare := bracket(grid.distances, au, math.Log)
	var corner [2][2]float64
	for later := range corner {
		for farther := range corner[later] {
			corner[later][farther] = grid.coreRadius(age+later,
				near+farther, column, core)
		}
	}
	young := mix(corner[0][0], corner[0][1], nearShare)
	old := mix(corner[1][0], corner[1][1], nearShare)
	return mix(young, old, ageShare)
}

// at is the radius in Jupiter radii for a mass and core mass in Earth
// masses, a distance in astronomical units from a Sun and an age in
// gigayears, with the mass held inside the tables' masses.
func (grid giantGrid) at(earthMasses, core, au, gigayears float64) float64 {
	column, share := bracket(giantMasses[:], earthMasses, math.Log)
	return mix(grid.columnRadius(column, core, au, gigayears),
		grid.columnRadius(column+1, core, au, gigayears), share)
}

// The hottest giants are larger than the tables give. Sestovic, Demory &
// Queloz 2018 (A&A 616, A76) fit the radius against mass and incident
// flux for 0.1 Jupiter masses and up: below a flux of Fs it is flat,
// above it the radius rises with the logarithm of the flux, by A for each
// power of ten (their equation 3; the values are in their table 1).
const (
	// jupiterMass is Jupiter's mass in kilograms (NASA's Jupiter fact
	// sheet, Williams, NSSDCA), the unit their mass bins are in.
	jupiterMass = 1.8982e27
	// solarLuminosity is the Sun's in watts (IAU 2015 Resolution B3),
	// to turn a sun's luminosity into the flux the fit is in.
	solarLuminosity = 3.828e26
)

// inflationBin is one of the fit's mass bins: the mass in Jupiter masses
// where it ends, the rise A in Jupiter radii for each power of ten of
// flux, the threshold's log10 of the flux in watts per square metre, and
// the width in log10 of mass of the blend into the next bin, which ends
// at this bin's top.
type inflationBin struct {
	top, rise, threshold, blend float64
}

// inflationBins are the paper's table 1 bins, whose boundaries are 0.37,
// 0.98 and 2.5 Jupiter masses. The lightest bin's rise in the paper is
// negative and poorly known, and it says such planets may not inflate
// (Enoch et al. 2012, as it cites), so its rise is none, with the next
// bin's threshold.
//
// Inside a bin, A and the threshold are the paper's. Below each edge, over
// the blend, they are mixed into the next bin's by a smoothstep of the
// logarithm of mass, and at the edge they are the next bin's: so the
// planets at and over an edge, Jupiter's among them, are the paper's, and
// inflation fades in across the 0.37 edge and does not switch on. That
// smooth transition is specific to the generator. The paper's bins step
// at each edge: a step of a hundredth in mass would otherwise
// jump a radius by near half a Jupiter's at the strongest light.
//
// The blends are 0.32, 0.06 and 0.09 in log10 of mass, near the narrowest
// that keep a hundredth step in mass to three per cent of the radius at
// fluxes to 1e9 watts per square metre (found by trying, with a little
// room): the jump in inflation at the edge, over the blend, bounds the
// step. The 0.37 edge jumps from none to the whole of its bin's, so it
// needs the widest; the others jump by less than a Jupiter radius.
var inflationBins = []inflationBin{
	{top: 0.37, rise: 0, threshold: 5.52, blend: 0.32},
	{top: 0.98, rise: 0.70, threshold: 5.52, blend: 0.06},
	{top: 2.50, rise: 0.52, threshold: 5.82, blend: 0.09},
	{top: math.Inf(1), rise: 0.22, threshold: 5.2},
}

// inflationAt is the rise and the threshold for a mass in Jupiter
// masses: its bin's, or, in the blend below the bin's top, mixed toward
// the next bin's.
func inflationAt(jupiterMasses float64) (rise, threshold float64) {
	for each, bin := range inflationBins {
		if jupiterMasses >= bin.top {
			continue
		}
		through := 0.0
		if bin.blend > 0 {
			through = min(1, max(0,
				math.Log10(jupiterMasses/bin.top)/bin.blend+1))
		}
		fade := through * through * (3 - 2*through)
		next := inflationBins[min(each+1, len(inflationBins)-1)]
		return mix(bin.rise, next.rise, fade),
			mix(bin.threshold, next.threshold, fade)
	}
	return 0, 0
}

// inflation is how much larger, in Jupiter radii, a giant of a mass in
// Jupiter masses is than its cold radius for an incident flux in watts
// per square metre: the paper's mean radius less its cold one (their
// equations 1 and 3), nothing below the threshold and rising from zero
// above it, so the two meet without a step.
func inflation(jupiterMasses, flux float64) float64 {
	if flux <= 0 {
		return 0
	}
	rise, threshold := inflationAt(jupiterMasses)
	return rise * max(0, math.Log10(flux)-threshold)
}

// coldGiantRadius is a giant's radius in Jupiter radii from the tables,
// for a planet whose mass and core mass are in Earth masses, at a
// distance in astronomical units from a Sun's worth of light, and an age
// in gigayears. Past the tables' lightest and heaviest masses it holds
// the table's radius at that end and follows Chen & Kipping's mean
// relation from there, so that it stays joined.
func coldGiantRadius(earthMasses, core, au, gigayears float64) float64 {
	last := len(giantMasses) - 1
	edge := min(max(earthMasses, giantMasses[0]), giantMasses[last])
	radius := fortney().at(edge, core, au, gigayears)
	return radius * forecast(earthMasses) / forecast(edge)
}
