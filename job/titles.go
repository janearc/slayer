package job

import "time"

// Titles is what is drawn over the frames: the colourway its roles come
// from, how many rows of cells the picture is laid out in, and its
// elements, in the order written, each over the ones before.
type Titles struct {
	wire titlesWire
}

// Titles is the job's titles, and whether it says any.
func (job Job) Titles() (Titles, bool) {
	if job.wire.Titles == nil {
		return Titles{}, false
	}
	return Titles{*job.wire.Titles}, true
}

// Theme is the colourway file, absolute, and whether the job said one;
// unsaid, the hud's own default is used.
func (words Titles) Theme() (string, bool) {
	return words.wire.Theme, words.wire.Theme != ""
}

// Rows is how many rows of cells the picture is laid out in, and whether
// the job said; unsaid, the hud's own default is used.
func (words Titles) Rows() (int, bool) { return said(words.wire.Rows) }

// Elements is each element, in the order written.
func (words Titles) Elements() []Title {
	elements := make([]Title, len(words.wire.Elements))
	for i, each := range words.wire.Elements {
		elements[i] = Title{name: each.name, wire: each.title}
	}
	return elements
}

// Title is one element drawn over the frames, by its name and its kind:
// a text, a frame or a bar, as a set's hud element is, and the span of
// film time it is drawn in. A part its kind does not take is zero.
type Title struct {
	name string
	wire titleWire
}

// Name is the element's name.
func (made Title) Name() string { return made.name }

// Kind is text, frame or bar.
func (made Title) Kind() string { return made.wire.Kind }

// Words is what a text says: its words, or, when counted, a count
// written out by the format, as fmt writes a float64.
func (made Title) Words() (words, format string, counted bool,
	count Ramp) {
	counted = made.wire.Count != nil
	if counted {
		count = Ramp{*made.wire.Count}
	}
	return made.wire.Words, made.wire.Format, counted, count
}

// Value is a bar's value, from zero to one, and whether it says one.
func (made Title) Value() (Ramp, bool) {
	if made.wire.Value == nil {
		return Ramp{}, false
	}
	return Ramp{*made.wire.Value}, true
}

// Box is a frame's or a bar's part of the picture: its left, top, right
// and bottom as shares of the picture's width and height.
func (made Title) Box() (left, top, right, bottom float64) {
	return share(made.wire.Left), share(made.wire.Top),
		share(made.wire.Right), share(made.wire.Bottom)
}

// At is where a text stands, as shares across and down, and how it sits
// there: left, centre or right, left where it says none.
func (made Title) At() (across, down float64, align string) {
	align = made.wire.Align
	if align == "" {
		align = defaultAlign
	}
	return share(made.wire.Across), share(made.wire.Down), align
}

// share is a share the element says, or zero.
func share(value *float64) float64 {
	number, _ := pointed(value)
	return number
}

// Roles is the theme's roles the element is drawn in: ink and ground for
// a text, ink for a frame, fill and empty for a bar. No ground is clear.
func (made Title) Roles() (ink, ground, fill, empty string) {
	return made.wire.Ink, made.wire.Ground, made.wire.Fill,
		made.wire.Empty
}

// From is when the element appears, in film time, and whether it says;
// unsaid, it is drawn from the first frame.
func (made Title) From() (time.Duration, bool) {
	return spanOf(made.wire.From)
}

// Until is when the element goes, in film time, and whether it says;
// unsaid, it is drawn to the last frame.
func (made Title) Until() (time.Duration, bool) {
	return spanOf(made.wire.Until)
}

// Ramp is a number that may change across the film: from, until start
// seconds, then evenly to to over so many seconds. A number written
// alone is from and to both.
type Ramp struct {
	wire ramp
}

// Values is the ramp's from and to, and its start and over in seconds.
func (value Ramp) Values() (from, to, start, over float64) {
	return value.wire.From, value.wire.To, value.wire.Start,
		value.wire.Over
}
