package texture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// the numbers a cover is read with
const (
	// coverMiddle is where in a blend, as a share of it, a range's edge
	// stands: a blend is centred on the edge it softens.
	coverMiddle = 0.5 // defined: half
	// coverLevel is the slope of ground that lies flat: rise over run.
	coverLevel = 0 // defined: no rise
)

// Range is a stretch of one number, a height or a slope, with soft ends:
// its weight is one inside it, nought well outside it, and eased across a
// blend centred on each end it has. A range has either end, or both.
type Range struct {
	from, until, blend float64
}

// NewRange is the range from one number to another, each end softened over
// a blend, in the number's own units. An end at minus or plus infinity is
// not there: a range with neither end holds every number with weight one.
func NewRange(from, until, blend float64) Range {
	return Range{from: from, until: until, blend: blend}
}

// Weight is how much of the range a number holds, nought to one.
func (stretch Range) Weight(value float64) float64 {
	weight := 1.0
	if !math.IsInf(stretch.from, -1) {
		weight *= maths.Smooth(stretch.from-stretch.blend*coverMiddle,
			stretch.from+stretch.blend*coverMiddle, value)
	}
	if !math.IsInf(stretch.until, 1) {
		weight *= 1 - maths.Smooth(stretch.until-stretch.blend*coverMiddle,
			stretch.until+stretch.blend*coverMiddle, value)
	}
	return weight
}

// coverLayer is one colour laid over the layers before it, where its two
// ranges hold.
type coverLayer struct {
	colour        maths.Light
	height, slope Range
}

// Cover is ground coloured by rule from where a point stands and which way
// the ground leans there: a base colour everywhere, and over it each layer
// in turn, laid as far as its height and its slope hold. Height is metres
// above the sea, slope is rise over run, so level ground has none and a
// wall a great deal.
//
// It is read by the CPU tracer through Facing, which hands it the way the
// surface faces. Asked by At alone, with no way, it takes the ground to be
// level.
type Cover struct {
	sea    float64
	base   maths.Light
	layers []coverLayer
}

// compile-time check that a Cover is read by its way.
var _ Oriented = Cover{}

// NewCover is a cover of one colour, over ground whose sea stands at the
// height in the thing's own space that sea says.
func NewCover(base maths.Light, sea float64) Cover {
	return Cover{sea: sea, base: base}
}

// Over is the cover with one more layer laid over those it has: a colour,
// where the height holds and where the slope holds. It leaves the cover it
// is called on as it was.
func (cover Cover) Over(colour maths.Light, height, slope Range) Cover {
	layers := append(append([]coverLayer(nil), cover.layers...),
		coverLayer{colour: colour, height: height, slope: slope})
	cover.layers = layers
	return cover
}

// At is the cover at a point in the thing's own space, taking the ground
// there to lie level.
func (cover Cover) At(point maths.Vec) maths.Light {
	return cover.paint(point.Y-cover.sea, coverLevel)
}

// Facing is the cover at a point: its place in its own space, in the
// world, and the unit normal of the ground there. The way is the
// normal's lean from up, so a normal that turns toward the eye from
// under a ledge reads as the ground does.
func (cover Cover) Facing(local, _, normal maths.Vec) maths.Light {
	return cover.paint(local.Y-cover.sea, leanOf(normal))
}

// leanOf is a normal's slope, rise over run: how far the ground it stands
// on rises for each metre it runs. A normal with no up is a wall.
func leanOf(normal maths.Vec) float64 {
	return math.Hypot(normal.X, normal.Z) / math.Abs(normal.Y)
}

// paint is the cover at a height above the sea and a slope.
func (cover Cover) paint(height, slope float64) maths.Light {
	colour := cover.base
	for _, layer := range cover.layers {
		share := layer.height.Weight(height) * layer.slope.Weight(slope)
		colour = colour.Scale(1 - share).Add(layer.colour.Scale(share))
	}
	return colour
}
