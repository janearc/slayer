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

	// held is the lines sorted, set by Prepared.
	held *heldLines
}

// line is one of an Excited's lines: its wavelength, its strength, and the
// excitation it wants.
type line struct {
	nm, strength, excitation float64
}

// heldLines are an Excited's lines, shortest first, with what Excitation
// answers where no line is the nearest. They are read, never changed, after
// Prepared makes them, so render workers may share them.
type heldLines struct {
	lines []line
	none  float64
}

// Prepared is the same gas with its lines sorted once, so that Emits and
// Excitation, asked at every step of a march, do not sort them again. An
// Excited built by a literal gives the same answers without it, only slower.
// The prepared value does not see a change made to Lines or Excitations after
// this call.
func (e Excited) Prepared() Excited {
	e.held = nil
	held := &heldLines{none: e.excitationOf(0)}
	for _, nm := range e.sortedLines(nil) {
		held.lines = append(held.lines, line{nm, e.Lines[nm], e.excitationOf(nm)})
	}
	e.held = held
	return e
}

// excitationOf is what Excitations says of a line: its entry where it has
// one above zero, else one.
func (e Excited) excitationOf(nm float64) float64 {
	if x, is := e.Excitations[nm]; is && x > 0 {
		return x
	}
	return 1
}

// Through is Passes, for any band.
func (e Excited) Through(maths.Band) float64 { return e.Passes }

// Emits is the lines' strength at the band, each a gaussian Width wide.
func (e Excited) Emits(band maths.Band) float64 {
	width := e.Width
	if width <= 0 {
		width = 1
	}
	if e.held != nil {
		return emitted(e.held.lines, band.Nanometres, width)
	}
	var buf [8]line
	lines := buf[:0]
	var nms [8]float64
	for _, nm := range e.sortedLines(nms[:0]) {
		lines = append(lines, line{nm: nm, strength: e.Lines[nm]})
	}
	return emitted(lines, band.Nanometres, width)
}

// emitted is the sum of the lines' gaussians at a wavelength, in the order
// the lines are given.
func emitted(lines []line, nm, width float64) float64 {
	sum := 0.0
	for _, l := range lines {
		strength := l.strength
		d := (nm - l.nm) / width
		sum += strength * math.Exp(-d*d)
	}
	return sum
}

// sortedLines is the lines' wavelengths, shortest first, appended to out.
func (e Excited) sortedLines(out []float64) []float64 {
	for nm := range e.Lines {
		out = append(out, nm)
	}
	sort.Float64s(out)
	return out
}

// Scatters is zero: excited gas returns no radar.
func (Excited) Scatters(maths.Band) float64 { return 0 }

// Excitation is the nearest line's, or one.
func (e Excited) Excitation(band maths.Band) float64 {
	if e.held != nil {
		nearest, apart := -1, math.Inf(1)
		for i, l := range e.held.lines {
			if d := math.Abs(band.Nanometres - l.nm); d < apart {
				nearest, apart = i, d
			}
		}
		if nearest < 0 {
			return e.held.none
		}
		return e.held.lines[nearest].excitation
	}
	nearest, apart := 0.0, math.Inf(1)
	var nms [8]float64
	for _, nm := range e.sortedLines(nms[:0]) {
		if d := math.Abs(band.Nanometres - nm); d < apart {
			nearest, apart = nm, d
		}
	}
	return e.excitationOf(nearest)
}
