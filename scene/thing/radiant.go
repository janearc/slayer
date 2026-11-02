package thing

import (
	"math"
	"sort"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
)

// Radiant is what a material does with radiation outside the visible: how much
// of a band passes one unit of the thing, from zero for opaque to one for
// clear, and how much of a band the thing gives off.
//
// A material without one is opaque to everything and dark in everything but
// visible light, as a wall is.
type Radiant interface {
	Through(band maths.Band) float64
	Emits(band maths.Band) float64
	// Scatters is how much of a band that lands on the thing it sends
	// back: the way it came, for a radar to see, and every way alike,
	// as a matte surface does, under a lamp that lights in the band.
	Scatters(band maths.Band) float64
}

// Spectrum is a Radiant by range, each range one of celestial's,
// radiation.XRay(), radiation.Gamma() and the rest, and Visible for the
// eye's. A range not declared is opaque and dark.
//
// Passes is how much of a range passes one unit of a thing, or of a
// volume at unit density; Gives how much of a range the thing gives
// off; Returns how much of a range that lands is sent back, toward a
// radar or, under a lamp, every way alike.
//
// A range not in Passes passes nothing: a volume that gives off a band
// must say it passes it too, or only its skin shows.
//
// Where the ranges overlap, the extreme ultraviolet inside the
// ultraviolet, the microwave inside the radio, the visible's ends inside
// the ultraviolet and the infrared, and the infrared's long end inside
// the radio, a band takes the narrowest range the spectrum declares.
type Spectrum struct {
	Passes  map[radiation.Range]float64
	Gives   map[radiation.Range]float64
	Returns map[radiation.Range]float64
}

// Visible is the eye's range, the 1931 observer's; a thing says what it
// does with the visible under it.
var Visible = observer.Visible()

// ranges is every range a band may fall in, narrowest first: celestial's
// named ranges and the visible. Fermi is left out. It is an
// instrument's sensitivity, the sensor's side of film, and not a range
// of the spectrum, so a thing cannot pass or give off "fermi".
var ranges = func() []radiation.Range {
	rs := []radiation.Range{Visible, radiation.Gamma(), radiation.XRay(),
		radiation.ExtremeUltraviolet(), radiation.Ultraviolet(),
		radiation.Infrared(), radiation.Radio(), radiation.Microwave()}
	sort.SliceStable(rs, func(i, j int) bool {
		return width(rs[i]) < width(rs[j])
	})
	return rs
}()

// width is how far a range runs, in nanometres.
func width(span radiation.Range) float64 {
	short, long := span.Edges()
	return long - short
}

// holds is whether a range holds the band.
func holds(span radiation.Range, band maths.Band) bool {
	return span.Holds(band.Nanometres)
}

// Range is the narrowest of the ranges a band falls in, or the zero
// range for none. The zero band, every band at once, is Visible.
func Range(band maths.Band) radiation.Range {
	if band.Nanometres == 0 {
		return Visible
	}
	for _, span := range ranges {
		if holds(span, band) {
			return span
		}
	}
	return radiation.Range{}
}

// Seen is whether a band is visible light, or the zero band, which is
// all of it at once.
func Seen(band maths.Band) bool {
	return band.Nanometres == 0 || holds(Visible, band)
}

// Lookup is what a table by range says for a band: the entry for the
// narrowest range the table declares that holds the band, and whether
// there is one. The zero band looks up the visible.
func Lookup[V any](table map[radiation.Range]V, band maths.Band) (V, bool) {
	if band.Nanometres == 0 {
		v, ok := table[Visible]
		return v, ok
	}
	for _, p := range ranges {
		if v, ok := table[p]; ok && holds(p, band) {
			return v, true
		}
	}
	var none V
	return none, false
}

// Through is how much of the band passes one unit.
func (s Spectrum) Through(band maths.Band) float64 {
	v, _ := Lookup(s.Passes, band)
	return v
}

// Emits is how much of the band the thing gives off.
func (s Spectrum) Emits(band maths.Band) float64 {
	v, _ := Lookup(s.Gives, band)
	return v
}

// Scatters is how much of the band that lands is sent back.
func (s Spectrum) Scatters(band maths.Band) float64 {
	v, _ := Lookup(s.Returns, band)
	return v
}

// Lit is a Radiant that gives off only where the lamps reach it: an
// ionisation front, gas lit into glowing by a star. The picture scales
// what it Emits by the lamplight reaching each point of it, to the
// power of the band's Excitation.
//
// An excitation of one is a line that glows as it is lit; more is one
// that wants a star close; less is one that glows on in the shade.
type Lit interface {
	Radiant
	Excitation(band maths.Band) float64
}

// Excited is gas that glows in lines where a star's light reaches it:
// Lines are the lines it gives off, strength by wavelength in
// nanometres, each Width nanometres wide, and Passes how much of any
// band passes a unit of it.
//
// Excitations, by the same wavelengths, are how strongly each line
// wants the star's light; a line with none is one.
//
// The eagle's pillars glow in sulphur where the light is weak,
// hydrogen where it lands, and oxygen only near the star, which is why
// the hubble palette's three filters each show a different place.
//
// the lines are summed shortest first, and a band as near two lines takes
// the shorter's excitation, so the same scene gives the same light every
// time; a map's own order is chosen at random by the language.
type Excited struct {
	Lines       map[float64]float64
	Excitations map[float64]float64
	Width       float64
	Passes      float64
}

// Through is Passes, for any band.
func (e Excited) Through(maths.Band) float64 { return e.Passes }

// Emits is the lines' strength at the band, each a gaussian Width wide.
func (e Excited) Emits(band maths.Band) float64 {
	width := e.Width
	if width <= 0 {
		width = 1
	}
	sum := 0.0
	for _, line := range e.lines() {
		strength := e.Lines[line]
		d := (band.Nanometres - line) / width
		sum += strength * math.Exp(-d*d)
	}
	return sum
}

// lines is the lines' wavelengths, shortest first.
func (e Excited) lines() []float64 {
	out := make([]float64, 0, len(e.Lines))
	for line := range e.Lines {
		out = append(out, line)
	}
	sort.Float64s(out)
	return out
}

// Scatters is zero: excited gas returns no radar.
func (Excited) Scatters(maths.Band) float64 { return 0 }

// Excitation is the nearest line's, or one.
func (e Excited) Excitation(band maths.Band) float64 {
	nearest, apart := 0.0, math.Inf(1)
	for _, line := range e.lines() {
		if d := math.Abs(band.Nanometres - line); d < apart {
			nearest, apart = line, d
		}
	}
	if x, is := e.Excitations[nearest]; is && x > 0 {
		return x
	}
	return 1
}
