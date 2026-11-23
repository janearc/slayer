package walk

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"math"
	"strconv"

	"github.com/janearc/slayer/maths"
)

// refusal is an error that is a constant, so nothing can assign it.
type refusal string

// Error is the refusal's words.
func (reason refusal) Error() string { return string(reason) }

// errWalk is every refusal the package makes; each says what was wrong
// and what to do.
const errWalk refusal = "walk"

// Place is a point of a set: metres east and north on the set's map, and
// metres of height above the sea, as hillside's places table writes them.
type Place struct {
	east, north, height float64
}

// NewPlace is a place, refused unless each of its numbers is finite.
func NewPlace(east, north, height float64) (Place, error) {
	for _, each := range []float64{east, north, height} {
		if math.IsNaN(each) || math.IsInf(each, 0) {
			return Place{}, fmt.Errorf("%w: a place of %v e, %v n, %v up; "+
				"give finite metres", errWalk, east, north, height)
		}
	}
	return Place{east: east, north: north, height: height}, nil
}

// world is the place in the world's space as package set makes it: x
// east, y up, and north its minus z. The origin is the map's own; only
// differences are ever taken, so it is never moved.
func (at Place) world() maths.Vec {
	return maths.V(at.east, at.height, 0-at.north)
}

// placeOf is the place a world point is, the inverse of world.
func placeOf(point maths.Vec) Place {
	return Place{east: point.X, north: 0 - point.Z, height: point.Y}
}

// Places is where each named thing of a set stands.
type Places struct {
	named map[string]Place
}

// placesHeader is the header hillside's -places table starts with.
// Defined by the session's scene.go, written.
var placesHeader = []string{"of", "kind", "east", "north", "height"}

// ReadPlaces reads the table hillside -places writes: a header, then a
// row a thing of its name, kind, east, north and height. A table with
// another header, a row with a number that is not one, or a name twice,
// is refused by its row.
func ReadPlaces(table []byte) (Places, error) {
	rows, err := csv.NewReader(bytes.NewReader(table)).ReadAll()
	if err != nil {
		return Places{}, fmt.Errorf("%w: the places table: %w; write it "+
			"with hillside -places", errWalk, err)
	}
	if len(rows) == 0 || !equal(rows[0], placesHeader) {
		return Places{}, fmt.Errorf("%w: the places table does not start "+
			"%v; write it with hillside -places", errWalk, placesHeader)
	}
	read := Places{named: map[string]Place{}}
	for i, row := range rows[1:] {
		if err := read.row(row); err != nil {
			return Places{}, fmt.Errorf("%w: the places table, row %d: %w",
				errWalk, i+2, err)
		}
	}
	return read, nil
}

// row reads one row into the places.
func (read Places) row(row []string) error {
	var metres [3]float64
	for i := range metres {
		number, err := strconv.ParseFloat(row[2+i], 64)
		if err != nil {
			return fmt.Errorf("%s is %q, not a number", placesHeader[2+i],
				row[2+i])
		}
		metres[i] = number
	}
	at, err := NewPlace(metres[0], metres[1], metres[2])
	if err != nil {
		return err
	}
	if _, twice := read.named[row[0]]; twice {
		return fmt.Errorf("%q stands twice", row[0])
	}
	read.named[row[0]] = at
	return nil
}

// equal is whether two rows of words are the same.
func equal(one, other []string) bool {
	if len(one) != len(other) {
		return false
	}
	for i := range one {
		if one[i] != other[i] {
			return false
		}
	}
	return true
}

// of is where a named thing stands, and whether the set has it.
func (read Places) of(name string) (Place, bool) {
	at, found := read.named[name]
	return at, found
}

// where is the places as package set asks for them, in the world's space.
func (read Places) where(name string) (maths.Vec, bool) {
	at, found := read.named[name]
	return at.world(), found
}
