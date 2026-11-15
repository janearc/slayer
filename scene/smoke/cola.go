package smoke

import (
	"math"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
)

// A glass of cola, in the studio's own units. Nothing here is measured:
// no table of a cola's optics is in hand, and every number is picked for
// how a glass of it looks.
const (
	fiftyTwoNo = 52 // defined: the number of a glass of cola

	colaFull    = 0.8  // picked: how full the glass is, as a share of its height
	colaIndex   = 1.34 // picked: a sugared drink bends light a little more than water's 1.33
	colaThrough = 0.2  // picked: the share of light a cola's face lets through; it is a dark drink, and the rest shows its brown
	colaRed     = 0.16 // picked: caramel brown, its red,
	colaGreen   = 0.05 // picked: its green,
	colaBlue    = 0.02 // picked: and its blue
	foamRed     = 0.74 // picked: a cola's foam is tan, its red,
	foamGreen   = 0.58 // picked: its green,
	foamBlue    = 0.40 // picked: and its blue

	rising      = 46    // picked: bubbles on their way up through the drink
	risingLeast = 0.006 // picked: the smallest of them, across
	risingMore  = 0.009 // picked: and how much bigger the biggest is
	risingClear = 0.03  // picked: the drink kept clear of bubbles at its foot, its top and its sides

	foam      = 90    // picked: bubbles of foam at the top
	foamLeast = 0.008 // picked: the smallest of them
	foamMore  = 0.008 // picked: and how much bigger the biggest is
	foamRing  = 0.35  // picked: foam gathers at the glass: none nearer the middle than this share of the way out, by area

	spray      = 28    // picked: drops thrown up over the top
	sprayLeast = 0.003 // picked: the smallest drop
	sprayMore  = 0.003 // picked: and how much bigger the biggest is
	sprayHigh  = 0.13  // picked: how far over the drink the highest drop is
	sprayWide  = 0.8   // picked: drops keep to this share of the glass's inside

	colaWide = 22.0 // picked: a long lens, in degrees, close on the glass
	half     = 0.5  // defined: a half

	risingRing = 0.55 // picked: bubbles show where they are near the glass; none nearer the middle than this share of the way out, by area

	// defined: the golden angle, in radians, and the fractions of the
	// square roots of two and of three. A thing placed a golden angle
	// round from the last, each a fixed step further out, never lines up
	// with one before it; and a count times such a fraction, less its
	// whole part, wanders evenly from nought to one.
	//
	// Three numbers with
	// no ratio between them, so a bubble's height and size follow
	// neither each other nor where it is: bubbles spread with no dice.
	goldenTurn = 2.399963229728653
	rootTwo    = 0.41421356237309515
	rootThree  = 0.7320508075688772
)

// colaEye is where the eye stands for the glass of cola: near, a little
// above the drink's top and to the right.
var colaEye = maths.V(1.25, 2.25, 3.3) // picked

// spread is where the i-th of n things lies on a disc of a radius, each
// a golden angle round from the last, with no nearer the middle than a
// share of the disc by area; and two shares from nought to one that are
// its own, for a height and a size.
func spread(i, n int, radius, from float64) (x, z, one, other float64) {
	out := radius * math.Sqrt(from+(unity-from)*(float64(i)+half)/float64(n))
	round := float64(i) * goldenTurn
	_, one = math.Modf(float64(i+unity) * rootTwo)
	_, other = math.Modf(float64(i+unity) * rootThree)
	return out * math.Cos(round), out * math.Sin(round), one, other
}

// fizz is everything that fizzes in the glass, added to the world at the
// glass's foot: the bubbles rising in the drink, the foam at its top,
// and the drops thrown over it.
func fizz(w *thing.World, foot maths.Vec, drink *thing.Material) {
	inside := glassRadius - glassWall
	level := glassHigh * colaFull
	air := &thing.Material{Name: "bubble", Colour: maths.Grey(glassGrey),
		Gloss: unity, Index: airIndex, Through: unity, Rank: bubbleRank}
	for i := range rising {
		x, z, high, big := spread(i, rising, inside-risingClear, risingRing)
		low := glassBase + risingClear
		at := maths.V(x, low+high*(level-risingClear-low), z)
		w.Add(thing.Thing{Shape: shape.Sphere{Radius: risingLeast + big*risingMore},
			Material: air, Motion: thing.Still(maths.Subspace{Origin: foot.Add(at)})})
	}
	froth := painted("foam", maths.RGB(foamRed, foamGreen, foamBlue))
	for i := range foam {
		across := foamLeast + foamMore*float64(i%pair)
		x, z, high, _ := spread(i, foam, inside-across, foamRing)
		at := maths.V(x, level+(high-half)*across, z)
		w.Add(thing.Thing{Shape: shape.Sphere{Radius: across},
			Material: froth, Motion: thing.Still(maths.Subspace{Origin: foot.Add(at)})})
	}
	for i := range spray {
		x, z, high, big := spread(i, spray, inside*sprayWide, nought)
		at := maths.V(x, level+foamLeast+high*sprayHigh, z)
		w.Add(thing.Thing{Shape: shape.Sphere{Radius: sprayLeast + big*sprayMore},
			Material: drink, Motion: thing.Still(maths.Subspace{Origin: foot.Add(at)})})
	}
}

// fiftyTwo is the fifty-second milestone: a glass of cola. In smoke
// forty-nine's studio, on the oak table, stands the same tumbler, filled
// with a dark brown drink. Bubbles rise through it, each a ball of air
// in the drink, bright at its rim where the drink mirrors; at the top is
// a ring of tan foam against the glass; and over that are the drops a
// fizzing drink throws up.
//
// A bubble seen through the glass is six bends of light: air to glass,
// glass to drink, drink to bubble, and the three back out. The tracer
// followed four until this smoke and follows eight now.
//
// Not right: the drink's brown is at its faces and not through its
// depth, so a thick glass of it is no darker than a thin one; the foam
// is tan balls and not films; nothing moves; and a clear thing blocks a
// lamp whole, so the glass's shadow is solid.
func fiftyTwo() (*thing.World, maths.Subspace) {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: sure(cove()),
		Material: painted("cove", swatches.Light(swatches.Paint)),
		Motion:   thing.Still(maths.Local())})
	table(w)
	foot := maths.V(glassAtX, tableHigh+glassHair, glassAtZ)
	w.Add(thing.Thing{Shape: sure(tumbler()),
		Material: &thing.Material{Name: "glass",
			Colour: maths.Grey(glassGrey), Gloss: unity,
			Index: glassIndex, Through: unity, Rank: glassRank},
		Motion: thing.Still(maths.Subspace{Origin: foot})})
	drink := &thing.Material{Name: "cola",
		Colour: maths.RGB(colaRed, colaGreen, colaBlue), Gloss: unity,
		Index: colaIndex, Through: colaThrough, Rank: waterRank}
	w.Add(thing.Thing{Shape: sure(pouredTo(colaFull)), Material: drink,
		Motion: thing.Still(maths.Subspace{Origin: foot})})
	fizz(w, foot, drink)
	top := foot.Add(maths.V(nought, glassHigh*colaFull, nought))
	w.Lamp(film.Spot{At: studioKeyAt, Aim: top.Sub(studioKeyAt),
		Strength: matte(swatches.Light(swatches.Daylight).Scale(studioSpot)),
		Cone:     studioCone * maths.Degree, Soft: studioSoft * maths.Degree})
	w.Lamp(film.Flood{
		Strength: swatches.Light(swatches.Daylight).Scale(studioFill)})
	look := foot.Add(maths.V(nought, glassHigh*half, nought))
	return w, maths.Subspace{Origin: colaEye,
		Orientation: maths.LookAt(colaEye, look)}
}
