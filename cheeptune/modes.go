package cheeptune

import "math"

// The mode sets: where the partials of a struck object lie, as ratios to
// the note, for the modal engine. Each set says where its ratios come
// from. Three words are used and they mean different things:
//
//	derived   worked out here from an equation, and a test checks the
//	          working. The equation itself is from memory of a text.
//	recalled  written from memory of a text and not checked against it.
//	          Wrong until somebody opens the book.
//	picked    chosen by the writer of the studio, 2026-11-12, with
//	          nobody listening. Every level and every loss below is
//	          picked: no text gave them.
//
// A real instrument is not these numbers. A marimba bar is cut to move
// its modes; a real bell tube has an end plug; a real glass is not a ring.

// modeSet is a named object: its partials, and how each one's ringing
// time and loudness stand against the lowest.
type modeSet struct {
	ratios []float64
	levels []float64
	// loss is how much faster a partial dies for being higher: its time
	// is the lowest's divided by its ratio to this power. Picked.
	loss float64
}

// barRoots are the first seven roots of cos(x)cosh(x) = 1, which is where
// the bending modes of a uniform bar free at both ends lie; a mode's
// frequency goes as its root squared. Derived: modes_test.go checks each
// is a root. The equation is recalled from Fletcher and Rossing, The
// Physics of Musical Instruments, the chapter on bars.
var barRoots = []float64{4.730041, 7.853205, 10.995608, 14.137165,
	17.278760, 20.420352, 23.561945}

// barRatios is each bar mode's frequency over the one counted as the
// note's: the root over that mode's root, squared, times the ratio that
// mode is to have.
func barRatios(count, note int, ratio float64) []float64 {
	out := make([]float64, count)
	for i := range out {
		step := barRoots[i] / barRoots[note]
		out[i] = ratio * step * step
	}
	return out
}

// ringRatios are the bending modes of a thin ring with two, three, four
// and more wavelengths round it, over the lowest: each mode's frequency
// goes as m(m*m-1) over the root of (m*m+1). Derived from that formula,
// which is recalled from Rayleigh, The Theory of Sound, on the flexural
// vibrations of a ring. A bowl or a glass is a ring with a wall and a
// foot, and its real modes are not these.
func ringRatios(count int) []float64 {
	at := func(m float64) float64 {
		return m * (m*m - 1) / math.Sqrt(m*m+1)
	}
	out := make([]float64, count)
	for i := range out {
		out[i] = at(float64(i+2)) / at(2)
	}
	return out
}

// plateRatios are the first modes of a flat circular plate with a free
// edge, over the lowest (two nodal diameters). Recalled, not checked:
// from memory of the table for free circular plates in Fletcher and
// Rossing, The Physics of Musical Instruments, the chapter on plates.
var plateRatios = []float64{1, 1.73, 2.33, 3.91, 4.11, 6.30, 6.71}

// marimbaRatios are what a marimba bar's first three modes are tuned to
// by cutting an arch under it. Recalled, not checked: from memory of
// Fletcher and Rossing on the marimba, a double octave and a major third
// over three octaves above the note.
var marimbaRatios = []float64{1, 4, 10}

// tubeNote is which mode of a tube the heard note sits an octave under,
// counted from nought: the fourth. Recalled, not checked: from memory of
// Fletcher and Rossing on tubular bells, where the strike note is not a
// mode at all but is heard an octave below the fourth, because the
// fourth, fifth and sixth lie close to two, three and four times it.
const tubeNote = 3

// modeSets are the objects the modal engine ships.
//
//	bar      a uniform free bar: a glockenspiel. derived.
//	marimba  a bar cut to a double octave and a tenth. recalled.
//	plate    a free circular plate: a gong-like disc. recalled.
//	bowl     a thin ring: toward a singing bowl or a glass. derived.
//	tube     a free tube, heard an octave under its fourth mode: a
//	         tubular bell. the modes derived, the octave recalled.
var modeSets = map[string]modeSet{
	"bar": {ratios: barRatios(6, 0, 1),
		levels: []float64{1, 0.6, 0.35, 0.2, 0.12, 0.07}, loss: 1.2},
	"marimba": {ratios: marimbaRatios,
		levels: []float64{1, 0.35, 0.12}, loss: 1.6},
	"plate": {ratios: plateRatios,
		levels: []float64{1, 0.7, 0.6, 0.45, 0.4, 0.25, 0.2}, loss: 0.7},
	"bowl": {ratios: ringRatios(5),
		levels: []float64{1, 0.5, 0.28, 0.15, 0.08}, loss: 0.9},
	"tube": {ratios: barRatios(7, tubeNote, 2),
		levels: []float64{0.15, 0.3, 0.6, 1, 0.8, 0.55, 0.3}, loss: 0.8},
}
