package cities

import (
	"cmp"
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Lashings are cables strung across a street between two walls that face
// each other, to hold the buildings of the street together.
//
// They are a
// decoration over a group of buildings, as a habit is a way of laying
// them out: given the volumes of the buildings along both sides of a
// street, they find the walls that face each other and string cables
// across at varied heights, from plates on each wall, each hanging as a
// real cable does, in a catenary.
//
// On a cable may hang a string of round paper lanterns, each a glowing
// part and some of them lights too, a banner across the middle of the
// street, a bundle of utility cable, or a tarpaulin.
//
// Density is the one
// knob, from nought to one: at a fifth there are a few wires and a
// lantern here and there, and at one the street is so hung with lanterns
// and signs that it is hard to see down it. There are more cables high up
// than low, and few at the retail level, which they clear.
//
// A banner is a place for a sign, a Slot, and is drawn as a plain glowing
// panel unless the ask names a kind to stand in the slot. The package
// knows nothing of kinds; the manufactory takes the slots and stands the
// kind it was named in each. Every number is in the lashings culture,
// picked, and the draws are the seed's.

// lashingsCulture is the culture lashings are drawn from.
const lashingsCulture = "lashings"

// The names of what a cable carries, as the culture's options say them.
const (
	dressLanterns  = "lanterns"
	dressBanner    = "banner"
	dressBundle    = "bundle"
	dressTarpaulin = "tarpaulin"
)

// The styles a street is hung in. The festival is the mixed dress above,
// by density; lantern rows are tidy rows of one kind of lantern, level
// across the street, one every so many metres along it.
const (
	styleFestival    = "festival"
	styleLanternRows = "lantern-rows"
)

// Styles are the names of the styles a street may be hung in.
func Styles() []string { return []string{styleFestival, styleLanternRows} }

// LashingsAsk is what a set asks of the lashings over a group: how thickly
// it is hung, from nought to one, the name of the kind that stands in
// each sign's slot, or nothing for plain glowing panels, and the style,
// or nothing for the festival.
//
// Lantern rows are hung by their own numbers
// in the culture and not by density, which they do not read. Culture is
// the culture they are drawn from, a variant of the lashings culture for
// narrower gaps, or nothing for the lashings culture itself.
type LashingsAsk struct {
	Density float64
	Sign    string
	Style   string
	Culture string
}

// Slot is a place for a sign hung from a cable: its middle in the group's
// frame, the horizontal direction its width runs along, and its width and
// height in metres. It faces along the street, and so down it.
type Slot struct {
	Middle        maths.Vec
	Along         maths.Vec
	Width, Height float64
}

// Lashings are the cables over a group, and what hangs from them.
type Lashings struct {
	// Streets are the stretches of the group the cables cross.
	Streets []street
	// Slots are the places for signs, one for each banner.
	Slots []Slot
	// Cables and Lanterns are how many of each there are.
	Cables, Lanterns int

	ask    LashingsAsk
	parts  []placedElement
	panels []placedElement
}

// NewLashings is the lashings over a group of buildings, drawn from the
// seed. It refuses a density that is not from nought to one, and a
// culture that does not say what it needs.
func NewLashings(ask LashingsAsk, group []Volume,
	seed Seed) (*Lashings, error) {
	if !(ask.Density >= 0 && ask.Density <= 1) {
		return nil, fmt.Errorf("cities: lashings are hung at a density "+
			"from nought to one, not %v", ask.Density)
	}
	if ask.Style != "" && !slices.Contains(Styles(), ask.Style) {
		return nil, fmt.Errorf("cities: lashings are hung in one of %v, "+
			"not %q", Styles(), ask.Style)
	}
	drawing, err := newDrawer(cmp.Or(ask.Culture, lashingsCulture), seed)
	if err != nil {
		return nil, err
	}
	lashings := &Lashings{ask: ask, Streets: streets(group, drawing)}
	for index, street := range lashings.Streets {
		job := stringing{drawing: drawing, out: lashings, street: street,
			index: index, density: ask.Density}
		if ask.Style == styleLanternRows {
			job.hangRows()
		} else {
			job.hang()
		}
	}
	return lashings, drawing.trouble
}

// Design is the lashings as a design to stand in the group's frame: the
// plates, the cables and what hangs from them. The banners are plain
// panels unless the ask named a kind for the slots, and then they are
// left to it.
func (lashings *Lashings) Design() Design {
	design := Design{Name: "lashings",
		Elements: slices.Clone(lashings.parts)}
	if lashings.ask.Sign == "" {
		design.Elements = append(design.Elements, lashings.panels...)
	}
	return design
}

// stringing is the job of hanging one street: the drawing it takes its
// numbers from, what it adds to, the street, and its number among them.
type stringing struct {
	drawing *drawer
	out     *Lashings
	street  street
	index   int
	density float64
}

// numberBetween is the number a share of the way from one to another.
func numberBetween(least, most, share float64) float64 {
	return least + (most-least)*share
}

// hang strings the street: as many cables as its density and length ask
// for, each in a slot of its own along the street, so that they spread.
func (job *stringing) hang() {
	drawing := job.drawing
	rate := numberBetween(drawing.fixed("cable-rate-least"),
		drawing.fixed("cable-rate-most"),
		math.Pow(job.density, drawing.fixed("density-power")))
	long := job.street.length()
	count := int(rate*long + drawing.number("cable-round",
		place{Row: job.index}))
	margin := drawing.fixed("edge-margin")
	slot := (long - 2*margin) / float64(max(count, 1))
	for cable := range count {
		at := place{Row: job.index, Bay: cable}
		job.cable(at, job.street.Start+margin+
			(float64(cable)+drawing.number("cable-jitter", at))*slot)
	}
}

// cable is one cable across the street at a place along it, with its
// plates and what it carries.
func (job *stringing) cable(at place, along float64) {
	drawing := job.drawing
	hung, from, to, err := job.hanging(at, along)
	if err != nil {
		drawing.fail(err)
		return
	}
	dress := job.dressed(at)
	radius, look := drawing.fixed("cable-radius"), job.steel()
	if dress == dressBundle {
		radius = drawing.fixed("bundle-radius")
		look = paint("lashing bundle", drawing.pick("bundle-colour",
			drawing.options("bundle-colour"), at))
	}
	job.lay(hung, along, from.Y, to.Y, radius, look)
	job.carry(dress, at, hung)
}

// lay is a cable laid across the street as it hangs, of a radius and a
// look, with its plates at the heights of its two ends, a distance along
// the street. It counts the cable.
func (job *stringing) lay(hung hung, along, nearHeight, farHeight,
	radius float64, look *thing.Material) {
	links := int(job.drawing.fixed("cable-links"))
	job.out.parts = append(job.out.parts, placedElement{Element: solid{
		Name: "lashing cable", Form: lashingCable{hung.points(links), radius},
		Material: look}, Place: standingAt(maths.Vec{})})
	job.out.parts = append(job.out.parts,
		job.plates(along, nearHeight, farHeight)...)
	job.out.Cables++
}

// hanging is the cable at a place along the street: where its anchors are
// and how it hangs between them. Its anchors favour the upper storeys, the
// second differs from the first by the cable's offset, and it is lifted if
// it would hang under the floor.
func (job *stringing) hanging(at place, along float64) (hung hung,
	from, to maths.Vec, err error) {
	drawing, street := job.drawing, job.street
	sag := drawing.number("sag-share", at) * street.width()
	offset := drawing.number("end-offset", at)
	low := street.Floor + sag
	high := math.Max(street.Top-math.Abs(offset), low)
	share := math.Pow(drawing.number("height-draw", at),
		1/drawing.fixed("height-bias"))
	height := numberBetween(low, high, share)
	thick := drawing.fixed("plate-thick")
	from = street.point(street.Near+thick, along, height)
	to = street.point(street.Far-thick, along, height+offset)
	hung, err = hang(from, to, sag)
	if lift := street.Floor - hung.lowest(); err == nil && lift > 0 {
		from.Y, to.Y = from.Y+lift, to.Y+lift
		hung, err = hang(from, to, sag)
	}
	return hung, from, to, err
}

// point is a place from how far across the street it is, how far along
// the street and how high, in the group's frame.
func (street street) point(across, along, height float64) maths.Vec {
	if street.Across == crossingX {
		return maths.V(across, height, along)
	}
	return maths.V(along, height, across)
}

// steel is the material of cables and plates.
func (job *stringing) steel() *thing.Material {
	return paint("lashing steel", job.drawing.pick("steel-colour",
		job.drawing.options("steel-colour"), place{}))
}

// plates are the two plates a cable is anchored to, one on each wall, at
// the height of each end of the cable.
func (job *stringing) plates(along, nearHeight, farHeight float64) []placedElement {
	drawing, street := job.drawing, job.street
	wide, high, thick := drawing.fixed("plate-width"),
		drawing.fixed("plate-height"), drawing.fixed("plate-thick")
	plate := func(across, height float64) placedElement {
		size := Box{thick, high, wide}
		if street.Across == crossingZ {
			size = Box{wide, high, thick}
		}
		return placedElement{Element: solid{Name: "anchor plate", Form: size,
			Material: job.steel()}, Place: standingAt(street.point(across,
			along-wide/2, height-high/2))}
	}
	return []placedElement{plate(street.Near, nearHeight),
		plate(street.Far-thick, farHeight)}
}

// dressed is what a cable carries, or nothing: whether it carries
// anything is a chance that grows with density, and what, by weight.
func (job *stringing) dressed(at place) string {
	drawing := job.drawing
	share := numberBetween(drawing.fixed("dress-share-least"),
		drawing.fixed("dress-share-most"), job.density)
	if !(drawing.number("dress-draw", at) < share) {
		return ""
	}
	return drawing.pick("dress", drawing.options("dress"), at).Name
}

// carry hangs what a cable carries from it. A bundle is the cable itself,
// thickened, and nothing more.
func (job *stringing) carry(dress string, at place, hung hung) {
	switch dress {
	case dressLanterns:
		job.lanterns(at, hung)
	case dressBanner:
		job.banner(at, hung)
	case dressTarpaulin:
		job.tarpaulin(at, hung)
	}
}
