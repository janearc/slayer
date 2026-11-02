// Package observer converts spectral light to colour using the CIE 1931
// observer. It is the boundary between celestial’s spectral physics and
// libtheme’s colour representation.
package observer

import (
	"math"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/radiation"
	theme "github.com/janearc/libtheme-css/primitives/bands"
	"github.com/janearc/libtheme-css/primitives/swatch"
)

// Visible is the eye's range as celestial's: the 1931 observer's table,
// from its first row to its last, as libtheme reads it.
func Visible() radiation.Range {
	lo, hi := swatch.Observed()
	return rangeOf(float64(lo), float64(hi))
}

// FalseColour is the colour a wavelength of a range stands for: the range
// laid across the eye's, evenly in the logarithm of wavelength, by
// celestial's profile, and the whole nanometre it lands nearest as
// libtheme's monochrome swatch.
//
// a wavelength outside the range, or a range that is none, is black.
func FalseColour(span radiation.Range, nm float64) swatch.Swatch {
	shown, err := radiation.NewProfile(span.String(), span, Visible())
	if err != nil {
		return swatch.Black
	}
	onto, ok := shown.Onto(nm)
	if !ok {
		return swatch.Black
	}
	return swatch.Monochrome(int(math.Round(onto)))
}

// rangeOf is a range from its short end to its long, or the zero range,
// which holds nothing, for two ends that are no range.
func rangeOf(short, long float64) radiation.Range {
	span, err := radiation.NewRange(short, long)
	if err != nil {
		return radiation.Range{}
	}
	return *span
}

// Swatch is the colour of a light to the 1931 observer.
//
// celestial and libtheme number their bands alike, each band the floor
// of twenty-four times the logarithm of its wavelength, two-based.
//
// each band's power is handed to libtheme as a line at the band's
// middle, in the same band. libtheme colours a band by the observer over
// all of it, so where in its band a line is does not move its colour.
func Swatch(light bands.Light) swatch.Swatch {
	first, last := light.Span()
	lines := map[float64]float64{}
	for band := first; band <= last; band++ {
		if power := light.Power(band); power != 0 {
			short, long := theme.Band(band).Edges()
			lines[math.Sqrt(short*long)] = power
		}
	}
	return theme.Lines(lines).Swatch()
}
