// Package span represents inclusive wavelength intervals in whole nanometres
// and determines which approximated RGB-channel wavelengths lie within them.
package span

import (
	"cmp"
	"errors"
	"slices"

	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
)

// inclusive is one on, so that both ends of a span are in it. It is
// defined as one.
const inclusive = 1

// shining is what a lamp that shines inside a span counts. It is
// defined as one.
const shining = 1

// dark is what a lamp that does not shine inside a span counts. It is defined
// as zero.
const dark = 0

// errSpan is a span that runs backwards, or past what the observer sees.
var errSpan = errors.New("ninjafilm: a span runs forwards, and inside " +
	"what libtheme's observer sees")

// Span is a stretch of the spectrum in whole nanometres, both ends in
// it. Records are sensitive across one, and their images block one.
type Span struct{ from, to int }

// seen is the whole of what libtheme's observer sees.
func seen() Span {
	shortest, longest := swatch.Observed()
	return Span{shortest, longest}
}

// New is a span from one wavelength to another, or why not.
func New(from, to int) (made Span, err error) {
	whole := seen()
	if from > to || from < whole.from || to > whole.to {
		return made, errSpan
	}
	return Span{from, to}, nil
}

// each is every whole nanometre of the span, in order.
func (stretch Span) each() []int {
	all := make([]int, stretch.to-stretch.from+inclusive)
	for index := range all {
		all[index] = stretch.from + index
	}
	return all
}

// toward is the direction a colour lies from white on libtheme's
// horseshoe. That is its hue, with how strong it is taken out.
func toward(colour swatch.Swatch) maths.Vec {
	white, at := swatch.White.XY(), colour.XY()
	return maths.Vec{X: at.X - white.X, Y: at.Y - white.Y}.Unit()
}

// hue is a direction on the horseshoe that wavelengths are judged by.
type hue struct{ aim maths.Vec }

// along is how well a wavelength points the way of the hue.
func (judge hue) along(nanometre int) float64 {
	return toward(swatch.Monochrome(nanometre)).Dot(judge.aim)
}

// compare is which of two wavelengths points better the way of the hue.
func (judge hue) compare(one, other int) int {
	return cmp.Compare(judge.along(one), judge.along(other))
}

// wavelength is the one wavelength that has a colour's hue. It is where
// the line from white through the colour meets the edge of what can be
// seen. That makes it the colour's dominant wavelength.
func wavelength(colour swatch.Swatch) int {
	judge := hue{toward(colour)}
	return slices.MaxFunc(seen().each(), judge.compare)
}

// lit is the wavelength each of the three lamps of a light shines at.
type lit struct{ red, green, blue int }

// lines is the wavelength each lamp shines at. Lights are three lamps
// and not a spectrum. So each lamp stands for one wavelength, which is
// its own hue's.
var lines = lampLines()

// lampLines is the wavelength of each lamp.
func lampLines() lit {
	return lit{red: dominant(maths.XAxis), green: dominant(maths.YAxis),
		blue: dominant(maths.ZAxis)}
}

// dominant is the wavelength of the colour a lamp gives lit by itself.
func dominant(lamp maths.Vec) int {
	colour := srgb.FromLight(lamp.X, lamp.Y, lamp.Z)
	return wavelength(colour)
}

// has is one if a wavelength is inside the span and zero if it is not.
func (stretch Span) has(nanometre int) float64 {
	if stretch.from <= nanometre && nanometre <= stretch.to {
		return shining
	}
	return dark
}

// Holds is which of the three lamps shine inside the span. Each that does is
// one and each that does not is zero. Spans that split the spectrum between
// them split the lamps between them.
func (stretch Span) Holds() maths.Light {
	return maths.RGB(stretch.has(lines.red), stretch.has(lines.green),
		stretch.has(lines.blue))
}
