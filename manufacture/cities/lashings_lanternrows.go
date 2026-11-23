package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Lantern rows: a street hung as streets of shops are for a festival, a
// cable across it every so many metres, all at one height, each level but
// for a slight sag, and on each a string of one kind of lantern evenly
// spaced, with small bulbs on the cable between them. Seen end-on down
// the street the rows stack into a dense band, from many tidy rows and not
// from scatter. Every number is in the lashings culture.

// capSink is how far a lantern's cap goes into the globe, as a share of
// the cap's own height: the globe is narrower than the cap's foot at its
// pole, so a cap standing on the pole would float.
const capSink = 0.5

// hangRows strings the street in rows: one every row-pitch metres along
// it, the whole run of them centred on the stretch the walls face.
func (job *stringing) hangRows() {
	drawing := job.drawing
	pitch, margin := drawing.fixed("row-pitch"), drawing.fixed("edge-margin")
	room := job.street.length() - 2*margin
	rows := int(room/pitch) + 1
	first := job.street.Start + margin + (room-float64(rows-1)*pitch)/2
	for row := range rows {
		job.row(place{Row: job.index, Bay: row}, first+float64(row)*pitch)
	}
}

// row is one row across the street at a distance along it: a cable between
// plates at one height, sagging slightly, with its lanterns and bulbs.
func (job *stringing) row(at place, along float64) {
	drawing, street := job.drawing, job.street
	height := math.Min(drawing.fixed("row-height"), street.Top)
	thick := drawing.fixed("plate-thick")
	from := street.point(street.Near+thick, along, height)
	to := street.point(street.Far-thick, along, height)
	hung, err := hang(from, to, drawing.fixed("row-sag"))
	if err != nil {
		drawing.fail(err)
		return
	}
	job.lay(hung, along, from.Y, to.Y, drawing.fixed("cable-radius"),
		job.steel())
	job.strung(at, hung)
}

// strung hangs the lanterns of a row evenly along its cable, and the bulbs
// between each pair of them on the cable.
func (job *stringing) strung(at place, hung hung) {
	drawing := job.drawing
	count := int(hung.run / drawing.fixed("row-lantern-pitch"))
	bulbs := int(drawing.fixed("row-bulbs"))
	for index := range count {
		whole := float64(index) + hangMiddle
		job.redLantern(place{Row: at.Row, Bay: at.Bay, Storey: index},
			hung.at(whole/float64(count)), index)
		if index+1 == count {
			break
		}
		for bulb := 1; bulb <= bulbs; bulb++ {
			job.bulb(hung.at((whole + float64(bulb)/float64(bulbs+1)) /
				float64(count)))
		}
	}
}

// redLantern hangs one lantern from a point of a cable by a cord: a squat
// ribbed globe of glowing red paper with a lamp in it, a dark cap on its
// top and its bottom, and a gold tassel under it. Every row-lamp-every
// lanterns, counted along the row, hold the lamp.
func (job *stringing) redLantern(at place, top maths.Vec, number int) {
	drawing := job.drawing
	radius := drawing.fixed("row-lantern-radius")
	stretch := drawing.fixed("row-lantern-stretch")
	drop := drawing.fixed("row-lantern-drop")
	half := radius * stretch
	middle := top.Sub(maths.V(0, drop+half, 0))
	paper := drawing.pick("row-lantern-colour",
		drawing.options("row-lantern-colour"), at)
	shell := &thing.Material{Name: "cities lantern " + paper.Name,
		Colour: colourOf(paper).Scale(paperShare),
		Glow:   colourOf(paper).Scale(drawing.fixed("row-lantern-glow"))}
	var lamp *Lamp
	if number%max(int(drawing.fixed("row-lamp-every")), 1) == 0 {
		lamp = &Lamp{Heat: drawing.fixed("row-lantern-heat"),
			Luminance: drawing.fixed("row-lantern-luminance"),
			Beyond:    drawing.fixed("row-lantern-beyond")}
	}
	job.out.parts = append(job.out.parts, placedElement{Element: solid{
		Name: "lantern", Material: shell, Lamp: lamp,
		Form: ribbedLantern{Radius: radius, Stretch: stretch,
			Ribs:  int(drawing.fixed("row-ribs")),
			Depth: drawing.fixed("row-rib-depth")}},
		Place: standingAt(middle)}, job.cord(top, drop))
	job.out.parts = append(job.out.parts, job.caps(at, middle, half)...)
	job.out.parts = append(job.out.parts, job.tassel(at, middle, half)...)
	job.out.Lanterns++
}

// caps are the dark caps on the top and the bottom of a globe of a height
// above and below its middle: a short taper, narrow toward the globe's
// outside and wide at the globe, standing a little way into it.
func (job *stringing) caps(at place, middle maths.Vec, half float64) []placedElement {
	drawing := job.drawing
	wide := drawing.fixed("row-lantern-radius") * drawing.fixed("row-cap-share")
	high := drawing.fixed("row-cap-height")
	dark := paint("lantern cap", drawing.pick("row-cap-colour",
		drawing.options("row-cap-colour"), at))
	sunk := high * capSink
	return []placedElement{
		{Element: solid{Name: "lantern cap", Material: dark, Form: prism{
			Bottom: outlineCircle{wide}, Top: outlineCircle{wide * drawing.fixed(
				"row-cap-narrow")}, Height: high}},
			Place: standingAt(middle.Add(maths.V(0, half-sunk, 0)))},
		{Element: solid{Name: "lantern cap", Material: dark, Form: prism{
			Bottom: outlineCircle{wide * drawing.fixed("row-cap-narrow")},
			Top:    outlineCircle{wide}, Height: high}},
			Place: standingAt(middle.Sub(maths.V(0, half+high-sunk, 0)))}}
}

// tassel is the gold tassel under a globe: a cord from the bottom cap and
// a skirt of strands that is wide at its foot.
func (job *stringing) tassel(at place, middle maths.Vec, half float64) []placedElement {
	drawing := job.drawing
	gold := paint("lantern tassel", drawing.pick("row-tassel-colour",
		drawing.options("row-tassel-colour"), at))
	long := drawing.fixed("row-tassel-length")
	thin := drawing.fixed("row-tassel-radius")
	start := middle.Y - half - drawing.fixed("row-cap-height")*(1-capSink)
	foot := maths.V(middle.X, start-long, middle.Z)
	return []placedElement{{Element: solid{Name: "lantern tassel", Material: gold,
		Form: prism{Bottom: outlineCircle{thin * drawing.fixed("row-tassel-spread")},
			Top: outlineCircle{thin}, Height: long}}, Place: standingAt(foot)}}
}

// bulb hangs one small white bulb on a cable at a point: a glowing ball
// that is no light, hung a radius under the cable so that it hangs on it.
func (job *stringing) bulb(on maths.Vec) {
	drawing := job.drawing
	radius := drawing.fixed("row-bulb-radius")
	white := drawing.pick("row-bulb-colour", drawing.options("row-bulb-colour"),
		place{})
	glass := &thing.Material{Name: "cities bulb " + white.Name,
		Colour: colourOf(white).Scale(paperShare),
		Glow:   colourOf(white).Scale(drawing.fixed("row-bulb-glow"))}
	job.out.parts = append(job.out.parts, placedElement{Element: solid{Name: "bulb",
		Form: lashingLantern{radius, 1}, Material: glass},
		Place: standingAt(on.Sub(maths.V(0, radius, 0)))})
}
