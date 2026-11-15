package smoke

import (
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
)

// A ball of water with bubbles in it, in the studio's own units.
const (
	fiftyOneNo = 51 // defined: the number of a ball of water with bubbles in it

	ballRadius  = 0.26 // picked: the ball of water, a little wider than the tumbler is tall
	ballAbove   = 0.42 // picked: the ball's middle above the table's top; it floats clear of it
	airIndex    = 1.0  // defined: air's index, as the tracer takes the air to be
	bubbleRank  = 1    // defined: a bubble outranks the water it is in, so inside a bubble there is air
	bubblesWide = 24.0 // picked: a long lens, in degrees, so the ball fills the picture
)

// bubblesEye is where the eye stands for the ball: nearer than the
// studio's, a little above the ball and to the right.
var bubblesEye = maths.V(1.3, 2.0, 3.6) // picked

// bubbles are the air in the ball: where each is from the ball's middle,
// and how big. Picked, every one whole inside the ball. A look through
// one bends four times, air to water, water to air, air to water and
// water to air; through two, one behind the other, six. The tracer
// followed four when this smoke was made, and where bubbles lined up it
// drew a dark dot; it follows eight now and draws the bubble behind.
var bubbles = [][4]float64{
	{-0.11, 0.08, 0.05, 0.045},
	{0.09, 0.12, -0.02, 0.03},
	{0.03, -0.09, 0.10, 0.05},
	{-0.06, -0.12, -0.06, 0.025},
	{0.14, -0.03, 0.04, 0.035},
	{-0.02, 0.17, 0.08, 0.02},
	{0.0, 0.02, -0.12, 0.04},
}

// fiftyOne is the fifty-first milestone: a ball of water with bubbles
// in it. In the studio of smoke forty-nine, over the oak table, floats a
// ball of water, as water does in orbit, and in the water are seven
// bubbles of air.
//
// A bubble is a clear thing inside a clear thing. Its face has water on
// one side and air on the other, so it bends light the other way from
// the ball it is in, and past the critical angle it mirrors: a bubble
// under water has a bright rim. Until 2026-11-12 the tracer took every
// clear face to have air outside it and could not draw this.
//
// The ball floats because nothing here says it may not: the studio has
// no gravity. What is not right: a clear thing blocks a lamp whole, so
// the ball's shadow on the table is solid, where real water would gather
// light into the middle of it.
func fiftyOne() (*thing.World, maths.Subspace) {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: sure(cove()),
		Material: painted("cove", swatches.Light(swatches.Paint)),
		Motion:   thing.Still(maths.Local())})
	table(w)
	middle := maths.V(glassAtX, tableHigh+ballAbove, glassAtZ)
	w.Add(thing.Thing{Shape: shape.Sphere{Radius: ballRadius},
		Material: &thing.Material{Name: "water",
			Colour: maths.Grey(glassGrey), Gloss: unity,
			Index: waterIndex, Through: unity, Rank: waterRank},
		Motion: thing.Still(maths.Subspace{Origin: middle})})
	for _, each := range bubbles {
		at := middle.Add(maths.V(each[0], each[1], each[2]))
		w.Add(thing.Thing{Shape: shape.Sphere{Radius: each[3]},
			Material: &thing.Material{Name: "air",
				Colour: maths.Grey(glassGrey), Gloss: unity,
				Index: airIndex, Through: unity, Rank: bubbleRank},
			Motion: thing.Still(maths.Subspace{Origin: at})})
	}
	w.Lamp(film.Spot{At: studioKeyAt, Aim: middle.Sub(studioKeyAt),
		Strength: matte(swatches.Light(swatches.Daylight).Scale(studioSpot)),
		Cone:     studioCone * maths.Degree, Soft: studioSoft * maths.Degree})
	w.Lamp(film.Flood{
		Strength: swatches.Light(swatches.Daylight).Scale(studioFill)})
	return w, maths.Subspace{Origin: bubblesEye,
		Orientation: maths.LookAt(bubblesEye, middle)}
}
