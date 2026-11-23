package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// A fish's skin is painted by its species' rule, on the body at rest, so
// the paint moves with the bending body: the dark back shading into the
// silver belly at the height the row gives, a band along the flank where
// the species has one, rows of spots placed by the fish's own seed where
// it has them, the line of the mouth and the edge of the gill cover. The
// eye is a ball with a pupil and an iris under a glossy cornea.

// How the skin's paint is drawn. Picked, to read as a fish's.
const (
	fishShadeBlend = .12   // picked: the back's blend into the belly
	fishLipLine    = .0025 // picked: the mouth's line, of the length
	fishLipShade   = .35   // picked: the mouth's line's darkness
	fishCoverCurve = .06   // picked: the gill cover's edge's curve
	fishCoverShade = .7    // picked: the gill cover's edge's darkness
	fishCoverSide  = .45   // picked: the edge only on the flank, not over
	fishBarLow     = -.6   // picked: how far down a bar runs, in half depths
	fishSpotRowGap = 2.5   // picked: rows of spots apart, in spots
	fishSpotJitter = .25   // picked: a spot's stray, of its pitch
	fishSpotFlank  = .9    // picked: the flank spots keep to
	fishSpotSides  = 64    // defined: a draw's place for the right side
	fishPupil      = .9    // picked: the pupil's edge, as a cosine
	fishIris       = .64   // picked: the iris's edge, as a cosine
	fishSclera     = .3    // picked: the hidden ball, of the pupil's
)

// fishColour is a look's colour as slayer's light.
func fishColour(look fishLook) maths.Light {
	return maths.RGB(look.Colour[0], look.Colour[1], look.Colour[2])
}

// fishMix is a, a share t of the way to b.
func fishMix(a, b maths.Light, t float64) maths.Light {
	return a.Scale(1 - t).Add(b.Scale(t))
}

// fishPaint is the body's paint: the fish's proportions, its colours,
// its markings' figures, and its seed, which places its spots.
type fishPaint struct {
	outline                        fishOutline
	back, belly, stripe, spot      maths.Light
	head, gape, backLine           float64
	stripeHeight, stripeWidth      float64
	spotRows                       int
	spotPitch, spotSize, spotRaise float64
	barCount                       int
	barWidth, barFrom              float64
	number                         uint64
}

// paint is the fish's body paint from its plan.
func (plan fishPlan) paint() fishPaint {
	looks := plan.species.Looks
	out := plan.outline()
	return fishPaint{outline: out, back: fishColour(looks["back"]),
		belly: fishColour(looks["belly"]), stripe: fishColour(looks["stripe"]),
		spot: fishColour(looks["spot"]), head: plan.figure("head"),
		gape:         plan.figure("gape") * plan.figure("head"),
		backLine:     plan.figure("back-line"),
		stripeHeight: plan.figure("stripe-height"),
		stripeWidth:  plan.figure("stripe-width"),
		spotRows:     plan.whole("spot-rows"),
		spotPitch:    plan.figure("spot-pitch"),
		spotSize:     plan.figure("spot-size") * out.length,
		spotRaise:    plan.figure("spot-height"),
		barCount:     plan.whole("bar-count"),
		barWidth:     plan.figure("bar-width"),
		barFrom:      plan.figure("bar-from"), number: plan.from.number}
}

// barred is whether a point at share s, v of its half depth up, lies in
// one of the fish's upright bars: evenly spaced from the first share to
// the stalk, each so wide, down from the back to fishBarLow.
func (paint fishPaint) barred(s, v float64) bool {
	if paint.barCount == 0 || v < fishBarLow {
		return false
	}
	pitch := 0.
	if paint.barCount > 1 {
		pitch = (paint.outline.stalkAt - paint.barFrom) /
			float64(paint.barCount-1)
	}
	for k := 0; k < paint.barCount; k++ {
		if math.Abs(s-paint.barFrom-float64(k)*pitch) < paint.barWidth/2 {
			return true
		}
	}
	return false
}

// At is the body's colour at a point of it at rest.
func (paint fishPaint) At(p maths.Vec) maths.Light {
	s := paint.outline.shareAt(p.X)
	half := paint.outline.halfDepth(s)
	if half <= 0 {
		return paint.back
	}
	v := p.Y / half
	shade := maths.Smooth(paint.backLine-fishShadeBlend,
		paint.backLine+fishShadeBlend, v)
	colour := fishMix(paint.belly, paint.back, shade)
	trunk := s > paint.head && s < paint.outline.stalkAt
	if paint.stripeWidth > 0 && trunk {
		off := (v - paint.stripeHeight) / paint.stripeWidth
		colour = fishMix(colour, paint.stripe, math.Exp(-off*off))
	}
	if trunk && paint.spotted(p, s, half) || paint.barred(s, v) {
		colour = paint.spot
	}
	return paint.lines(colour, p, s, v)
}

// lines is the colour darkened along the mouth and the gill cover's edge.
func (paint fishPaint) lines(colour maths.Light, p maths.Vec,
	s, v float64) maths.Light {
	length := paint.outline.length
	if s < paint.gape {
		mouth := -fishJawChin * paint.outline.halfDepth(paint.gape) *
			s / paint.gape
		if math.Abs(p.Y-mouth) < fishLipLine*length {
			return colour.Scale(fishLipShade)
		}
	}
	edge := paint.head * (fishCoverBack - fishCoverCurve*v*v)
	flank := math.Abs(p.Z) > fishCoverSide*paint.outline.halfWidth(s)
	if math.Abs(s-edge)*length < fishLipLine*length && flank {
		return colour.Scale(fishCoverShade)
	}
	return colour
}

// spotted is whether a point of the flank at share s, the body half deep
// there, lies in one of the fish's spots: rows of them below its middle,
// each spot strayed a little from its place by the fish's seed, the two
// sides strayed apart.
func (paint fishPaint) spotted(p maths.Vec, s, half float64) bool {
	if paint.spotRows == 0 || math.Abs(p.Y) > fishSpotFlank*half {
		return false
	}
	side := 0
	if p.Z < 0 {
		side = fishSpotSides
	}
	cell := int(math.Floor((s - paint.head) / paint.spotPitch))
	for row := 0; row < paint.spotRows; row++ {
		height := paint.spotRaise*half - float64(row)*fishSpotRowGap*
			paint.spotSize
		for k := cell - 1; k <= cell+1; k++ {
			if paint.inSpot(p, row+side, k, height) {
				return true
			}
		}
	}
	return false
}

// inSpot is whether a point lies in the spot of a row and a place along
// it, at a height.
func (paint fishPaint) inSpot(p maths.Vec, row, k int, height float64) bool {
	out := paint.outline
	along := paint.head + (float64(k)+fishHalf+fishSpotJitter*
		across(paint.number, row, k, 0))*paint.spotPitch
	if along > out.stalkAt || along < paint.head {
		return false
	}
	up := height + fishSpotJitter*paint.spotSize*across(paint.number, row, k, 1)
	off := math.Hypot(p.X-out.x(along), p.Y-up)
	return off < paint.spotSize/2
}

// fishEyePaint is an eye's paint: the left eye's middle, the right being
// its mirror, and the pupil's and the iris's colours.
type fishEyePaint struct {
	middle      maths.Vec
	pupil, iris maths.Light
}

// At is the eye's colour at a point of it at rest: the pupil looking out
// from the side, the iris round it, the hidden ball dark.
func (paint fishEyePaint) At(p maths.Vec) maths.Light {
	off := maths.V(p.X, p.Y, math.Abs(p.Z)).Sub(paint.middle)
	if off.Len() == 0 {
		return paint.pupil
	}
	out := off.Z / off.Len()
	switch {
	case out > fishPupil:
		return paint.pupil
	case out > fishIris:
		return paint.iris
	}
	return paint.iris.Scale(fishSclera)
}

// looks is what each surface of the fish is made of, by name.
func (body *fishBody) looks() map[string]*thing.Material {
	plan := body.plan
	looks := plan.species.Looks
	plain := func(name string) *thing.Material {
		return &thing.Material{Name: plan.name + " " + name,
			Colour: fishColour(looks[name]), Gloss: looks[name].Gloss}
	}
	middle, _ := body.eye()
	eye := &thing.Material{Name: plan.name + " eye",
		Colour: fishColour(looks["iris"]), Gloss: looks["iris"].Gloss,
		Index: fishCornea, Texture: fishEyePaint{middle: middle,
			pupil: fishColour(looks["pupil"]), iris: fishColour(looks["iris"])}}
	skin := plain("belly")
	skin.Name, skin.Texture = plan.name+" skin", plan.paint()
	return map[string]*thing.Material{"body": skin, "fin": plain("fin"),
		"low-fin": plain("low-fin"), "tail": plain("tail"),
		"finlet": plain("finlet"), "eye": eye,
		"bone": plain("bone")}
}
