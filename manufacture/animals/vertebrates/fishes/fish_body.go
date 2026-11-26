package fishes

import (
	"github.com/janearc/slayer/manufacture/branching"
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// A fish's body is lofted: rings across it, from the snout to the base of
// the tail, each as deep and as wide as the species' proportions put it
// there, closed at either end. Its depth rises from the snout to where the
// species is deepest, falls by a cosine to the tail stalk, and flares a
// little into the tail. A ring is a rounded ellipse; a keel is a ridge
// along the stalk's side where the species has keels.

// How the body's surface is shaped. Each is picked, to read as a fish.
const (
	fishRound      = 2.3  // picked: a ring's power; two is an ellipse
	fishFlare      = .35  // picked: the stalk's widening into the tail
	fishSpineHigh  = .12  // picked: the column's height, of a half depth
	fishEnd        = .004 // picked: the closing point beyond the tail base
	fishKeelRun    = .07  // picked: the median keel's run before the tail
	fishKeelSize   = .25  // picked: its height, of the stalk's depth
	fishKeelSpread = .2   // picked: its spread, of a half depth
	fishSmallKeel  = .08  // picked: a small keel's height, of the stalk
	fishSmallRun   = .02  // picked: a small keel's run
	fishSmallAt    = .45  // picked: its height, of a half depth
	fishHalf       = .5   // defined: the middle of the length
)

// The loft's cut, at a fish alone and at a fish of a school, seen far.
// Chosen: enough rings to bend smoothly at the joints of its column.
const (
	fishRings        = 72
	fishAround       = 40
	fishSchoolRings  = 36
	fishSchoolAround = 18
)

// fishOutline is a fish's proportions as lengths: how long it is, how deep
// and how wide at its deepest and at its stalk, where those fall, how
// blunt its snout is, and its keels.
type fishOutline struct {
	length, depth, width     float64
	stalk, stalkWidth        float64
	deepest, stalkAt, tailAt float64
	snout                    float64
	keels                    int
}

// outline is the fish's proportions, picked from its row.
func (plan fishPlan) outline() fishOutline {
	length := plan.length()
	depth := plan.figure("depth") * length
	stalk := plan.figure("stalk") * length
	return fishOutline{length: length, depth: depth,
		width: plan.figure("width") * depth, stalk: stalk,
		stalkWidth: plan.figure("stalk-width") * stalk,
		deepest:    plan.figure("deepest"), stalkAt: plan.figure("stalk-at"),
		tailAt: plan.figure("tail-base"), snout: plan.figure("snout"),
		keels: plan.whole("keels")}
}

// x is where along the fish a share of its length from the snout lies.
func (outline fishOutline) x(s float64) float64 {
	return outline.length * (fishHalf - s)
}

// shareAt is the share of the length from the snout at which x lies.
func (outline fishOutline) shareAt(x float64) float64 {
	return fishHalf - x/outline.length
}

// section is a size along the body: nothing at the snout, rising to most
// at the deepest, falling to least at the stalk, flaring into the tail.
func (outline fishOutline) section(most, least, s float64) float64 {
	switch {
	case s <= 0:
		return 0
	case s <= outline.deepest:
		u := 1 - s/outline.deepest
		return most * math.Pow(1-u*u, outline.snout)
	case s <= outline.stalkAt:
		u := (s - outline.deepest) / (outline.stalkAt - outline.deepest)
		return least + (most-least)*(1+math.Cos(math.Pi*u))/2
	}
	u := math.Min(1, (s-outline.stalkAt)/(outline.tailAt-outline.stalkAt))
	return least * (1 + fishFlare*u*u)
}

// halfDepth is half the body's depth at share s.
func (outline fishOutline) halfDepth(s float64) float64 {
	return outline.section(outline.depth, outline.stalk, s) / 2
}

// halfWidth is half the body's width at share s, keels aside.
func (outline fishOutline) halfWidth(s float64) float64 {
	return outline.section(outline.width, outline.stalkWidth, s) / 2
}

// spineY is the height of the column at share s.
func (outline fishOutline) spineY(s float64) float64 {
	return fishSpineHigh * outline.halfDepth(s)
}

// keel is how far the keels stand out from the side at share s and
// height y: the median keel along the stalk, and the two small ones above
// and below it at the tail's base, where the species has three.
func (outline fishOutline) keel(s, y float64) float64 {
	if outline.keels == 0 {
		return 0
	}
	half := math.Max(outline.halfDepth(s), outline.stalk/2)
	ridge := func(from, to, at, size float64) float64 {
		if s < from || s > to {
			return 0
		}
		spread := fishKeelSpread * half
		off := (math.Abs(y) - at) / spread
		return size * math.Sin(math.Pi*(s-from)/(to-from)) * math.Exp(-off*off)
	}
	out := ridge(outline.stalkAt-fishKeelRun, outline.tailAt, 0,
		fishKeelSize*outline.stalk)
	if outline.keels >= 3 {
		out += ridge(outline.tailAt-fishSmallRun, outline.tailAt,
			fishSmallAt*half, fishSmallKeel*outline.stalk)
	}
	return out
}

// surface is the point of the body's ring at share s and angle round it,
// the angle nought at the left side and a quarter turn at the back.
func (outline fishOutline) surface(s, angle float64) maths.Vec {
	power := 2 / fishRound
	along, up := math.Cos(angle), math.Sin(angle)
	y := outline.halfDepth(s) * math.Copysign(math.Pow(math.Abs(up), power), up)
	z := outline.halfWidth(s) * math.Copysign(math.Pow(math.Abs(along),
		power), along)
	z += math.Copysign(outline.keel(s, y), along)
	return maths.V(outline.x(s), y, z)
}

// side is how far out from the middle the left side is at share s and
// height y, keels aside: where the ring at s passes y.
func (outline fishOutline) side(s, y float64) float64 {
	half := outline.halfDepth(s)
	if half <= 0 || math.Abs(y) >= half {
		return 0
	}
	up := math.Pow(math.Abs(y)/half, fishRound/2)
	return outline.halfWidth(s) * math.Pow(1-up*up, 1/fishRound)
}

// loft is the body's surface at rest, rings across and around many points
// round each, closed by a point at the snout and one past the tail's base.
func (outline fishOutline) loft(rings, around int) ([]maths.Vec, [][3]int) {
	points := []maths.Vec{maths.V(outline.x(0), 0, 0)}
	for ring := 1; ring <= rings; ring++ {
		s := outline.tailAt * (1 - math.Cos(math.Pi*float64(ring)/
			float64(rings))) / 2
		for k := 0; k < around; k++ {
			angle := 2 * math.Pi * float64(k) / float64(around)
			points = append(points, outline.surface(s, angle))
		}
	}
	points = append(points, maths.V(outline.x(outline.tailAt+fishEnd),
		outline.spineY(outline.tailAt), 0))
	return points, fishOutward(points, fishLoftFaces(rings, around))
}

// fishLoftFaces are the triangles of a loft of rings rings of around
// points between a first and a last point.
func fishLoftFaces(rings, around int) [][3]int {
	at := func(ring, k int) int { return 1 + (ring-1)*around + k%around }
	last := 1 + rings*around
	faces := [][3]int{}
	for k := 0; k < around; k++ {
		faces = append(faces, [3]int{0, at(1, k+1), at(1, k)},
			[3]int{last, at(rings, k), at(rings, k+1)})
		for ring := 1; ring < rings; ring++ {
			faces = append(faces,
				[3]int{at(ring, k), at(ring, k+1), at(ring+1, k+1)},
				[3]int{at(ring, k), at(ring+1, k+1), at(ring+1, k)})
		}
	}
	return faces
}

// fishOutward is faces turned inside out if, together, they enclose a
// negative volume, so that a closed surface faces outward.
func fishOutward(points []maths.Vec, faces [][3]int) [][3]int {
	if geometry.SignedVolume6(points, faces) >= 0 {
		return faces
	}
	geometry.ReverseFaces(faces)
	return faces
}

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
	seed                           branching.Seed
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
		barFrom:      plan.figure("bar-from"), seed: plan.from}
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
		paint.seed.Swing(row, k, 0))*paint.spotPitch
	if along > out.stalkAt || along < paint.head {
		return false
	}
	up := height + fishSpotJitter*paint.spotSize*paint.seed.Swing(row, k, 1)
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
