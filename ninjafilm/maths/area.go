package maths

import (
	"errors"
	"math"
)

var errArea = errors.New("ninjafilm: an area is a finite number of " +
	"square metres, above nought")

// Area is how much surface something has to gather light with. Lens
// openings, mirrors and dishes have one.
type Area struct {
	squareMetres float64
}

// SquareMetres is an area given in square metres. It refuses an area of
// nothing. It refuses one that is negative, endless or not a number.
// Collectors with no area gather nothing and are not collectors.
//
//ninja:vow practice slayer's camera sizes its collector here; tests only
func SquareMetres(amount float64) (made Area, err error) {
	if !(amount > none) || math.IsInf(amount, upward) {
		return made, errArea
	}
	return Area{amount}, nil
}

// SquareMetres is the area in square metres.
func (area Area) SquareMetres() float64 {
	return area.squareMetres
}
