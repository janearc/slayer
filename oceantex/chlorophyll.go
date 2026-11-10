package oceantex

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"time"

	"github.com/janearc/libtheme-css/primitives/swatch"
)

// Chlorophyll says how much chlorophyll is usually at the surface of the
// sea at a place in a month, in milligrams a cubic metre: the one number
// the water's colour is worked out from. Latitude is degrees north,
// longitude degrees east. It reports false where it has nothing, such as
// on land.
//
// This package carries the step from chlorophyll to colour. It does not
// carry a map of the world's chlorophyll; see README.md for why, and for
// what Grid reads.
type Chlorophyll interface {
	At(latitude, longitude float64, month time.Month) (float64, bool)
}

// Even is the same chlorophyll everywhere, all year.
type Even float64

// At is the one value.
func (even Even) At(float64, float64, time.Month) (float64, bool) {
	return float64(even), true
}

// Grid is a chlorophyll map as cells of latitude and longitude, one layer
// a month.
type Grid struct {
	cells map[cell]float64
	step  float64
}

type cell struct {
	lat, lon int
	month    time.Month
}

// ReadGrid reads a map from comma-separated rows of
//
//	latitude, longitude, month, chlorophyll
//
// where latitude and longitude are the middle of a cell in degrees, month
// is 1 to 12, and every cell is step degrees across. A first row that is
// not numbers is taken for a heading, and lines starting with # are
// skipped. Rows may come in any order, and cells may be missing.
func ReadGrid(rows io.Reader, step float64) (*Grid, error) {
	if !(step > 0) {
		return nil, errors.New("oceantex: a grid's step must be positive")
	}
	grid := &Grid{cells: map[cell]float64{}, step: step}
	reader := csv.NewReader(rows)
	reader.FieldsPerRecord = 4
	reader.Comment = '#'
	reader.TrimLeadingSpace = true
	for line := 1; ; line++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		var values [4]float64
		bad := false
		for i, field := range row {
			if values[i], err = strconv.ParseFloat(field, 64); err != nil {
				bad = true
			}
		}
		if bad && line == 1 {
			continue
		}
		month := time.Month(values[2])
		if bad || month < time.January || month > time.December ||
			values[3] < 0 {
			return nil, fmt.Errorf("oceantex: grid row %d is not "+
				"latitude, longitude, month, chlorophyll", line)
		}
		grid.cells[grid.cell(values[0], values[1], month)] = values[3]
	}
	return grid, nil
}

// cell is the cell a place and month fall in; longitude is wrapped to
// [0, 360) so either convention lands in the same cell.
func (grid *Grid) cell(latitude, longitude float64, month time.Month) cell {
	longitude = math.Mod(math.Mod(longitude, 360)+360, 360)
	return cell{int(math.Floor(latitude / grid.step)),
		int(math.Floor(longitude / grid.step)), month}
}

// At is the value of the cell a place falls in.
func (grid *Grid) At(
	latitude, longitude float64, month time.Month,
) (float64, bool) {
	value, found := grid.cells[grid.cell(latitude, longitude, month)]
	return value, found
}

// ColourAt is the likely colour of the sea at a place on a date: the
// colour of deep water holding the chlorophyll the map gives for that
// place and month, under libtheme's daylight, D65, seen from straight
// above with the surface's reflections left out. Its luminance is against
// a white matte surface in the same light, so it is a dark colour: the
// sea sends back a few percent of what falls on it.
//
// It is the colour of the water, not of a photograph. Sky reflected in
// the surface, glint, foam, the bottom showing through shallows, and mud
// or bloom the model has no term for all change what a camera sees.
func ColourAt(
	latitude, longitude float64, date time.Time, chart Chlorophyll,
) (swatch.Swatch, error) {
	amount, found := chart.At(latitude, longitude, date.Month())
	if !found {
		return swatch.NoLight(), fmt.Errorf("oceantex: no chlorophyll "+
			"for %.2f, %.2f in %s", latitude, longitude, date.Month())
	}
	return Water(swatch.D65(), amount), nil
}
