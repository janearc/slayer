package planet

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// kilometreAmagat is how many molecules stand over a square metre in a
// kilometre of gas at zero celsius and 101,325 pascals: the unit methane's
// absorption is measured against.
var kilometreAmagat = 1e3 * standard

// halfBand is how far a libtheme band reaches either side of its
// middle, as a factor of the wavelength: half of 24 bands an octave.
var halfBand = math.Pow(2, 1.0/48)

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
// outside the table, 300.4 to 1000 nm, methane absorbs nothing here.
func MethaneAbsorbing(table io.Reader) (func(nm float64) float64, error) {
	rows, err := methaneRows(table)
	if err != nil {
		return nil, err
	}
	return func(nm float64) float64 {
		low, high := nm/halfBand, nm*halfBand
		sum, count := 0.0, 0
		for _, row := range rows {
			if row[0] >= low && row[0] < high {
				sum += row[1]
				count++
			}
		}
		if count == 0 {
			return 0
		}
		return sum / float64(count) / kilometreAmagat
	}, nil
}

// methaneRows is each row's vacuum wavelength and coefficient, in the
// table's order. lines that are not four numbers are its header.
func methaneRows(table io.Reader) ([][2]float64, error) {
	var rows [][2]float64
	lines := bufio.NewScanner(table)
	for lines.Scan() {
		fields := strings.Fields(lines.Text())
		if len(fields) != 4 {
			continue
		}
		vacuum, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			continue
		}
		coefficient, err := strconv.ParseFloat(fields[2], 64)
		if err != nil || coefficient < 0 {
			return nil, fmt.Errorf("methane at %s nm: %q", fields[0],
				fields[2])
		}
		rows = append(rows, [2]float64{vacuum, coefficient})
	}
	if err := lines.Err(); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("a methane table with no rows")
	}
	return rows, nil
}
