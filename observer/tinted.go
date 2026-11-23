package observer

import (
	"fmt"
	"math"
	"sync"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// tintLines are the wavelengths, in nanometres, of the three lines a tinted
// light is made of: a red at 630, a green at 532 and a blue at 467.
//
// They are the monochromatic primaries of Recommendation ITU-R BT.2020,
// whose gamut holds the srgb lamps' gamut, so that light of every colour
// the three srgb lamps can make, pure red, green and blue among them, is a
// sum of the three with no amount below nought. The observer is linear in
// power, so the sum is exact.
//
// The test beside this file checks, with this observer, that each corner
// of the srgb gamut needs no negative amount.
var tintLines = [3]float64{630, 532, 467}

// lineDuals are, for each line, the three numbers by which the amount of
// that line is the colour's dot product with them: the rows of the inverse
// of the matrix whose columns are the three lines as the observer sees each
// alone, in srgb light. The lines are constants, so they are worked out
// once.
var lineDuals = sync.OnceValue(func() [3][3]float64 {
	var seen [3][3]float64
	for index, wave := range tintLines {
		red, green, blue := srgb.Light(Swatch(bands.Lines(
			map[float64]float64{wave: 1})))
		seen[index] = [3]float64{red, green, blue}
	}
	whole := dot(seen[0], cross(seen[1], seen[2]))
	var duals [3][3]float64
	for index := range duals {
		next, after := seen[(index+1)%3], seen[(index+2)%3]
		for axis, value := range cross(next, after) {
			duals[index][axis] = value / whole
		}
	}
	return duals
})

// Tinted is a light whose colour to the observer is the amounts of light
// given from the three srgb lamps, in linear light: spectral light made of
// three lines, so that it can go where any light of the kit goes, and have
// the colour it is asked for when it gets there.
//
// A neon tube is not a black body, and a lamp given only a heat cannot be
// magenta or green; this is the light of a lamp that can. It is as bright
// as the colour is, in the units of the lamps: 1, 1, 1 is white. A colour
// that is no light, or has an amount below nought or that is not a number,
// is refused.
func Tinted(red, green, blue float64) (bands.Light, error) {
	colour := [3]float64{red, green, blue}
	total := 0.0
	for _, channel := range colour {
		if math.IsNaN(channel) || math.IsInf(channel, 0) || channel < 0 {
			return bands.Light{}, fmt.Errorf("observer: a tint of %v is not "+
				"amounts of light", colour)
		}
		total += channel
	}
	if total == 0 {
		return bands.Light{}, fmt.Errorf("observer: a tint of %v is no light",
			colour)
	}
	lines := map[float64]float64{}
	for index, dual := range lineDuals() {
		if amount := dot(dual, colour); amount > 0 {
			lines[tintLines[index]] = amount
		}
	}
	return bands.Lines(lines), nil
}

// dot is the scalar product of two vectors of three.
func dot(one, other [3]float64) float64 {
	return one[0]*other[0] + one[1]*other[1] + one[2]*other[2]
}

// cross is the vector product of two vectors of three.
func cross(one, other [3]float64) [3]float64 {
	return [3]float64{one[1]*other[2] - one[2]*other[1],
		one[2]*other[0] - one[0]*other[2], one[0]*other[1] - one[1]*other[0]}
}
