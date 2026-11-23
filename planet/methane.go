package planet

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/janearc/celestial/bands"
)

// metresPerKilometre is defined: a kilometre is a thousand metres.
const metresPerKilometre = 1e3

// kilometreAmagat is how many molecules stand over a square metre in a
// kilometre of gas at zero celsius and 101,325 pascals: the unit methane's
// absorption is measured against.
var kilometreAmagat = metresPerKilometre * standard

// halfBand is how far a libtheme band reaches either side of its
// middle, as a factor of the wavelength: half a band of the bands an octave
// celestial holds.
var halfBand = halfBandOf(bands.PerOctave())

// halfBandOf is half a band's reach, as a factor of the wavelength, for so
// many bands an octave: the octave's doubling over twice the bands.
func halfBandOf(perOctave bands.Band) float64 {
	return math.Pow(double, whole/(double*float64(perOctave)))
}

// The layout of the table MethaneAbsorbing reads, by the table.
const (
	// methaneColumns is defined as the columns a row of the table has: four.
	methaneColumns = 4
	// vacuumColumn is defined as the place of the vacuum wavelength in a
	// row: the first.
	vacuumColumn = 0
	// coefficientColumn is defined as the place of the coefficient in a row:
	// the third.
	coefficientColumn = 2
)

// A methaneRow is a row of the table: the vacuum wavelength, in nanometres,
// and the coefficient, in reciprocal kilometre-amagats.
type methaneRow struct{ vacuum, coefficient float64 }

// absorption is methane's absorption by the table's rows.
type absorption struct{ rows []methaneRow }

// at is the absorption cross-section, square metres a molecule, at a
// wavelength in nanometres: the mean of the coefficients of the rows within
// a libtheme band about it, none outside the table.
func (table absorption) at(nm float64) float64 {
	low, high := nm/halfBand, nm*halfBand
	var sum float64
	var count int
	for _, row := range table.rows {
		if row.vacuum >= low && row.vacuum < high {
			sum += row.coefficient
			count++
		}
	}
	if count == nothing {
		return nothing
	}
	return sum / float64(count) / kilometreAmagat
}

// MethaneAbsorbing is methane's absorption cross-section, square metres
// a molecule, at a wavelength in nanometres, read from karkoschka's 1994
// table (karkoschka-1994-methane): rows of vacuum and air wavelength,
// the coefficient in reciprocal kilometre-amagats, and an albedo.
//
// a wavelength is taken as a whole libtheme band about it, and the
// coefficient is the mean of the table's rows in that band. averaging
// the coefficient, not the light it lets through, makes a band of deep,
// narrow lines absorb more evenly than it does: modelled.
//
// outside the table, 300.4 to 1000 nm, methane absorbs nothing here. a row
// whose coefficient is below zero or is not a finite number is refused.
func MethaneAbsorbing(table io.Reader) (func(nm float64) float64, error) {
	rows, err := methaneRows(table)
	if err != nil {
		return nil, err
	}
	return absorption{rows: rows}.at, nil
}

// methaneRows is each row's vacuum wavelength and coefficient, in the
// table's order. lines that are not four numbers are its header.
func methaneRows(table io.Reader) ([]methaneRow, error) {
	var rows []methaneRow
	lines := bufio.NewScanner(table)
	for lines.Scan() {
		fields := strings.Fields(lines.Text())
		if len(fields) != methaneColumns {
			continue
		}
		vacuum, err := strconv.ParseFloat(fields[vacuumColumn], 64)
		if err != nil {
			continue
		}
		coefficient, err := strconv.ParseFloat(fields[coefficientColumn], 64)
		if err != nil || !(coefficient >= nothing) || endless(coefficient) {
			return nil, fmt.Errorf("methane at %s nm: %q", fields[vacuumColumn],
				fields[coefficientColumn])
		}
		rows = append(rows, methaneRow{vacuum: vacuum,
			coefficient: coefficient})
	}
	if err := lines.Err(); err != nil {
		return nil, err
	}
	if len(rows) == nothing {
		return nil, fmt.Errorf("a methane table with no rows")
	}
	return rows, nil
}
