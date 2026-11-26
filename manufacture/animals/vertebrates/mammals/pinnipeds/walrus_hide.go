package pinnipeds

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// The colours of the hide. "Most walruses are yellowish brown"; out of the
// water "vaso-dilation causes blood and color to return to the skin", which
// turns pinkish to reddish; "Older males have many lumps or nodules on the
// neck and chest" and "most older males become nearly hairless" (all from
// the OBIS-SEAMAP species profile, as read). The reflectances themselves
// are chosen to match those words; none is sampled from a photograph.
var (
	walrusCinnamon    = maths.RGB(.25, .150, .085)
	walrusPale        = maths.RGB(.36, .25, .16)
	walrusBackSkin    = maths.RGB(.15, .09, .055)
	walrusFlipperSkin = maths.RGB(.16, .105, .07)
	walrusFlush       = maths.RGB(.38, .19, .15)
	walrusPadSkin     = maths.RGB(.33, .23, .17)
	walrusScarSkin    = maths.RGB(.48, .38, .32)
	walrusHairDark    = maths.RGB(.10, .06, .045)
	walrusHairPale    = maths.RGB(.62, .50, .36)
	walrusFollicleDot = maths.RGB(.07, .035, .03)
)

// walrusScar is a thin healed wound: the curve where the hide crosses a
// slab of the given width, as far as length from centre. The slab passes
// through centre, or offset from it along normal: two scars with one
// centre and different offsets lie side by side, as a pair of tusks
// leaves them.
type walrusScar struct {
	centre, normal        maths.Vec
	length, width, offset float64
}

// walrusScars are the scars: short ones, the first two a pair. Their
// places and sizes are chosen.
var walrusScars = []walrusScar{
	{maths.V(.72, .36, .10), maths.V(.5, .2, .84), .07, .0026, 0},
	{maths.V(.72, .36, .10), maths.V(.5, .2, .84), .07, .0026, .012},
	{maths.V(.55, .30, -.30), maths.V(.8, .1, -.6), .08, .0030, 0},
	{maths.V(.30, .05, .43), maths.V(.2, .9, .4), .09, .0026, 0},
	{maths.V(-.40, .20, -.40), maths.V(.3, .5, .8), .08, .0024, 0},
	{maths.V(.90, .30, .25), maths.V(.6, .6, -.5), .06, .0024, 0},
	{maths.V(-.90, .10, .43), maths.V(.9, .3, .3), .06, .0028, 0},
}

// walrusScarShare is how much of a scar's colour shows at p, from nought
// to one.
func walrusScarShare(p maths.Vec) float64 {
	share := 0.
	for _, scar := range walrusScars {
		offset := p.Sub(scar.centre)
		if offset.Len() > scar.length {
			continue
		}
		distance := offset.Dot(scar.normal.Unit()) - scar.offset
		share = math.Max(share, 1-maths.Smooth(0, scar.width, math.Abs(distance)))
	}
	return share
}

// walrusMix is the light between two, from the first at nought to the
// second at one.
func walrusMix(first, second maths.Light, share float64) maths.Light {
	return maths.RGB(first.R+(second.R-first.R)*share,
		first.G+(second.G-first.G)*share, first.B+(second.B-first.B)*share)
}

// walrusHair is the short sparse hair at p: nought where the hide is bare,
// minus one for a dark hair, one for a pale hair.
func walrusHair(p maths.Vec) float64 {
	// density is the share of cells with no hair: a bull is nearly
	// hairless.
	const (
		cell      = .011
		density   = .95
		tuftWidth = .0022
	)
	ix, iy, iz := int(math.Floor(p.X/cell)), int(math.Floor(p.Y/cell)),
		int(math.Floor(p.Z/cell))
	if walrusHash(ix, iy, iz, 6) < density {
		return 0
	}
	centre := maths.V((float64(ix)+walrusHash(ix, iy, iz, 7))*cell,
		(float64(iy)+walrusHash(ix, iy, iz, 8))*cell,
		(float64(iz)+walrusHash(ix, iy, iz, 9))*cell)
	if p.Sub(centre).Len() > tuftWidth {
		return 0
	}
	if walrusHash(ix, iy, iz, 10) < .5 {
		return -1
	}
	return 1
}

// walrusBody is the colour of the body by region before any detail: the
// yellowish brown back and flanks, the paler belly, the flush of blood in
// broad patches over the neck, chest and shoulders where the skin is
// thin and the blood comes back out of the water, and the darker bare
// flippers, all broken by slow mottling, a back darker than the flanks,
// and broad darker blotches. Scales and shares chosen.
func (rig *walrusRig) walrusBody(p maths.Vec) maths.Light {
	const (
		mottleScale, mottleShare = 2.5, .85
		patchScale, patchShare   = 9., .22
		flushScale, flushShare   = 3.5, .30
		bellyHigh, bellyLow      = -.12, -.40
		backLow, backHigh        = .15, .50
		neckStart, neckEnd       = .50, 1.00
		neckFade, neckGone       = 1.30, 1.45
		bellyShare, backShare    = .70, .45
		blotchScale, blotchShare = 1.6, .50
		blotchLow, blotchHigh    = .45, .75
	)
	pale := walrusMix(walrusCinnamon, walrusPale, mottleShare*walrusNoise(p.Scale(mottleScale)))
	neck := maths.Smooth(neckStart, neckEnd, p.X) * (1 - maths.Smooth(neckFade, neckGone, p.X))
	belly := 1 - maths.Smooth(bellyLow, bellyHigh, p.Y)
	colour := walrusMix(walrusCinnamon, pale, .8)
	colour = walrusMix(colour, walrusPale, bellyShare*belly)
	colour = walrusMix(colour, walrusBackSkin,
		backShare*maths.Smooth(backLow, backHigh, p.Y))
	blotch := maths.Smooth(blotchLow, blotchHigh,
		walrusNoise(p.Scale(blotchScale).Add(maths.V(3, 11, 5))))
	colour = walrusMix(colour, walrusBackSkin, blotchShare*blotch*(1-belly))
	flush := maths.Smooth(.35, .75, walrusNoise(p.Scale(flushScale).Add(maths.V(7, 3, 5))))
	colour = walrusMix(colour, walrusFlush, flushShare*flush*(.35+.65*neck))
	colour = walrusMix(colour, walrusFlipperSkin, rig.walrusLimbShare(p))
	tone := 1 - patchShare/2 + patchShare*walrusNoise(p.Scale(patchScale))
	return maths.RGB(colour.R*tone, colour.G*tone, colour.B*tone)
}

// walrusPadShare is how much of the muzzle pad's skin there is at rest
// point p, from nought to one: in front of the skull's snout, between
// the nostrils and the mouth, as far out as the pad reaches. Chosen.
func (rig *walrusRig) walrusPadShare(p maths.Vec) float64 {
	const (
		start, top, bottom, width = .96, -.11, .18, .165
		edge                      = .03
	)
	t, across, drop := rig.frame.local(p)
	return maths.Smooth(start, start+edge, t) *
		maths.Smooth(top-edge, top, drop) * (1 - maths.Smooth(bottom, bottom+edge, drop)) *
		(1 - maths.Smooth(width, width+edge, math.Abs(across)))
}

// walrusHideTexture is the walrus hide as a texture: a colour for every
// point in the rest pose, by region, read at the rest place the hide's
// mesh carries. It adds scars, the follicle dots on the muzzle pad, and
// sparse short hair. Creases and bosses are not coloured: the hide has no
// fold cut in it.
type walrusHideTexture struct{ rig *walrusRig }

// At is the hide's reflectance at p.
func (texture walrusHideTexture) At(p maths.Vec) maths.Light {
	rig := texture.rig
	if rig == nil {
		rig = defaultWalrusRig()
	}
	const (
		dotWidth, padShare   = .0030, .75
		scarShare, hairShare = .12, .22
	)
	colour := rig.walrusBody(p)
	pad := rig.walrusPadShare(p)
	colour = walrusMix(colour, walrusPadSkin, padShare*pad)
	if pad > .5 && rig.walrusFollicleDistance(p) < dotWidth {
		colour = walrusMix(colour, walrusFollicleDot, 1)
	}
	colour = walrusMix(colour, walrusScarSkin, scarShare*walrusScarShare(p))
	switch hair := walrusHair(p); {
	case hair < 0:
		colour = walrusMix(colour, walrusHairDark, hairShare)
	case hair > 0:
		colour = walrusMix(colour, walrusHairPale, hairShare)
	}
	return colour
}

// walrusIvoryTexture is the colour of a tusk by how far down it a point
// lies: stained and darker where it comes out of the jaw, paler at the
// worn end, with fine lines round it at the growth rings.
type walrusIvoryTexture struct{}

// At is the ivory's reflectance at p.
func (walrusIvoryTexture) At(p maths.Vec) maths.Light {
	const (
		stainTop, stainBottom = .14, -.25
		ringsPerMetre         = 90.
		ringShare             = .06
	)
	stained, worn := maths.RGB(.52, .40, .24), maths.RGB(.80, .74, .60)
	down := maths.Smooth(stainBottom, stainTop, p.Y)
	colour := walrusMix(worn, stained, down)
	ring := ringShare * math.Sin(ringsPerMetre*p.Y)
	return maths.RGB(colour.R*(1+ring), colour.G*(1+ring), colour.B*(1+ring))
}

// The loose skin of a bull's neck and shoulders lies in creases. Each
// here is short and uneven: it runs part of the way round the body,
// leans back as it goes down the flank, wanders, and lies differently on
// the two sides. None closes into a ring. Every place, depth and width
// is picked; a depth is in the density's units, and walrusCutDepth says
// how far in metres it moves the hide.

// walrusCreaseLine is one crease. At is where along the body it crosses
// the middle of the back, in rest metres, and lean how far back it lies
// for each radian round the body from there. It runs from the angle from
// to the angle to, in radians round the body from the top of the back.
//
// Depth is how deep it cuts at its deepest and width its half width
// along the body. Wander is how far it strays from its line, and phase
// where in its straying it begins.
type walrusCreaseLine struct {
	at, lean      float64
	from, to      float64
	depth, width  float64
	wander, phase float64
}

// walrusCreaseLines are the creases, from behind the head back to the
// fold behind the fore flipper.
var walrusCreaseLines = []walrusCreaseLine{
	{at: 1.26, lean: .05, from: 0, to: 2.2, depth: .09, width: .022,
		wander: .015, phase: .3},
	{at: 1.13, lean: .09, from: .3, to: 1.9, depth: .11, width: .028,
		wander: .020, phase: 1.7},
	{at: .99, lean: .12, from: 0, to: 1.5, depth: .10, width: .030,
		wander: .025, phase: 4.1},
	{at: .84, lean: .15, from: .6, to: 2.0, depth: .09, width: .030,
		wander: .020, phase: 2.6},
	{at: .58, lean: .10, from: 1.0, to: 2.1, depth: .07, width: .035,
		wander: .030, phase: 5.2},
}

const (
	// walrusCreaseBack and walrusCreaseFront are the ends of the stretch
	// of the body that creases, behind the fore flippers and at the back
	// of the head, in rest metres.
	walrusCreaseBack  = .20
	walrusCreaseFront = 1.40
	// walrusCreaseFade is the angle over which a crease shallows out at
	// each of its ends, in radians.
	walrusCreaseFade = .30
	// walrusCreaseTurns is how many times a crease strays back and forth
	// in one radian round the body.
	walrusCreaseTurns = 3.
)

// cut is how deep the crease cuts at x along the body and the angle round
// from the top of the back, on the side given, one for the right and
// minus one for the left. The two sides stray alike at the top of the
// back, where they meet, and apart from there down.
func (line walrusCreaseLine) cut(x, round, side float64) float64 {
	apart := side * math.Min(round, 1)
	middle := line.at - line.lean*round +
		line.wander*math.Sin(walrusCreaseTurns*round+line.phase+apart)
	off := (x - middle) / line.width
	ends := maths.Smooth(line.from-walrusCreaseFade, line.from, round) *
		(1 - maths.Smooth(line.to, line.to+walrusCreaseFade, round))
	return line.depth * ends * math.Exp(-off*off)
}

// walrusCreasesOf is the creases as a cut into the outline given: how
// deep they cut at a point, nought off the neck and shoulders.
func walrusCreasesOf(outline *walrusOutline) func(maths.Vec) float64 {
	return func(p maths.Vec) float64 {
		if p.X < walrusCreaseBack || p.X > walrusCreaseFront {
			return 0
		}
		round := math.Atan2(math.Abs(p.Z), p.Y-outline.at(p.X).centre)
		side := 1.
		if p.Z < 0 {
			side = -1
		}
		total := 0.
		for _, line := range walrusCreaseLines {
			total += line.cut(p.X, round, side)
		}
		return total
	}
}

// walrusFloor is the height of the ground the belly lies on.
const walrusFloor = -.41

// The bristles of the muzzle pads. The sources, as read:
//   - "There can be 400 to 700 vibrissae in 13 to 15 rows" (Kastelein,
//     Stevens and Mosterd 1990, Aquatic Mammals, on Wikipedia's Walrus page)
//   - "some 480-odd vibrissae roughly the texture of uncooked spaghetti"
//     (California Wild, California Academy of Sciences, fall 2002)
//   - "Each whisker averages 8 centimetres ... and about 3 millimetres
//     thick" (Guinness World Records, most whiskers on a pinniped)
//   - "large, fleshy, forward-facing mystacial pads" sprouting "short,
//     stiff, whitish" bristles (OBIS-SEAMAP species profile)
//
// So: fourteen rows of seventeen a side, 476 in all, each about 8 cm and
// 3 mm at its root, stiff and straight. The rows lie on the pad in the
// skull's frame, staggered and a little uneven; their spacing is chosen to
// fill the pad, which shows between them.
const (
	// walrusFollicleRows is the number of rows down the pad.
	walrusFollicleRows = 14
	// walrusFollicleColumns is the number of follicles in a row.
	walrusFollicleColumns = 17
	// walrusFollicleTop is the top row's drop below the skull's axis:
	// above it, so less than nought. Chosen, with the two steps, so the
	// rows cover the face of the pads from under the nostrils to the lip.
	walrusFollicleTop = -.06
	// walrusFollicleRowStep is the distance between rows.
	walrusFollicleRowStep = .0165
	// walrusFollicleColumnStep is the distance between follicles in a row.
	walrusFollicleColumnStep = .0088
	// walrusFollicleInner is how far from the middle the first column sits.
	walrusFollicleInner = .012
	// walrusFollicleJitter is how far a follicle strays from the lattice,
	// as a share of a step. Chosen.
	walrusFollicleJitter = .25
	// walrusStrandSegments is the straight pieces in each bristle and
	// walrusStrandSides the flat faces round it.
	walrusStrandSegments = 6
	walrusStrandSides    = 6
	// walrusStrandRadius is a bristle's radius at its root: 3 mm thick.
	walrusStrandRadius = .0015
	// walrusStrandLength is a bristle's average length: 8 cm.
	walrusStrandLength = .08
)

// walrusFollicle is the drop below the skull's axis and the signed offset
// to the side of the follicle at a row and column on one side of the pad.
// Odd rows are shifted half a step, and every follicle strays a little.
func walrusFollicle(row, column int, side float64) (drop, across float64) {
	sign := int(side)
	shift := .5*float64(row%2) +
		walrusFollicleJitter*(2*walrusHash(row, column, sign, 11)-1)
	lift := walrusFollicleJitter * (2*walrusHash(row, column, sign, 12) - 1)
	across = walrusFollicleInner + (float64(column)+shift)*walrusFollicleColumnStep
	drop = walrusFollicleTop + (float64(row)+lift)*walrusFollicleRowStep
	return drop, side * across
}

// walrusFollicleDistance is how far, across the pad, rest point p lies from
// the nearest follicle, measured in the skull's frame.
func (rig *walrusRig) walrusFollicleDistance(p maths.Vec) float64 {
	_, across, drop := rig.frame.local(p)
	side := 1.
	if across < 0 {
		side = -1
	}
	row := int(math.Round((drop - walrusFollicleTop) / walrusFollicleRowStep))
	column := int(math.Round((math.Abs(across) - walrusFollicleInner) /
		walrusFollicleColumnStep))
	nearest := math.Inf(1)
	for r := max(0, row-1); r <= min(walrusFollicleRows-1, row+1); r++ {
		for c := max(0, column-1); c <= min(walrusFollicleColumns-1, column+1); c++ {
			d, a := walrusFollicle(r, c, side)
			nearest = math.Min(nearest, math.Hypot(drop-d, across-a))
		}
	}
	return nearest
}

// walrusPadSurface is how far forward along the skull's axis, as a share
// of the skull's length, the hide lies at that offset to the side and drop
// below the axis, looking from behind the pad, or false if no hide lies on
// that line.
func (rig *walrusRig) walrusPadSurface(field func(maths.Vec) float64, across,
	drop float64) (float64, bool) {
	const (
		inside, outside = .70, 1.45
		steps           = 26
	)
	at := func(t float64) maths.Vec { return rig.frame.at(t, across, drop) }
	if field(at(inside)) >= 0 || field(at(outside)) < 0 {
		return 0, false
	}
	low, high := inside, outside
	low, high = maths.Bisect(low, high, steps, func(t float64) bool { return field(at(t)) < 0 })
	return (low + high) / 2, true
}

// walrusAddStrand adds one closed six-sided bristle to the mesh being
// built: from a little below the hide, thick at its root and tapering to a
// blunt tip, stiff, so nearly straight.
func walrusAddStrand(vs []maths.Vec, fs [][3]int, root, grow maths.Vec, length float64,
	skin func(maths.Vec) maths.Vec) ([]maths.Vec, [][3]int) {
	const (
		buried       = .004
		droop        = .06
		tip          = .45
		sides, steps = walrusStrandSides, walrusStrandSegments
	)
	up := maths.V(0, 1, 0)
	across := up.Sub(grow.Scale(up.Dot(grow))).Unit()
	around := grow.Cross(across)
	first := len(vs)
	start := root.Sub(grow.Scale(buried))
	for i := 0; i <= steps; i++ {
		t := float64(i) / steps
		centre := start.Add(grow.Scale(length * t)).Add(maths.V(0, -droop*length*t*t, 0))
		radius := walrusStrandRadius * (1 - (1-tip)*t)
		for j := 0; j < sides; j++ {
			angle := 2 * math.Pi * float64(j) / sides
			offset := across.Scale(radius * math.Cos(angle)).Add(around.Scale(radius * math.Sin(angle)))
			vs = append(vs, skin(centre.Add(offset)))
		}
	}
	fs = geometry.AppendIndexedLoftSides(fs, first, steps+1, sides)
	rootMiddle := len(vs)
	vs = append(vs, skin(start))
	tipMiddle := len(vs)
	vs = append(vs, skin(start.Add(grow.Scale(length)).Add(maths.V(0, -droop*length, 0))))
	for j := 0; j < sides; j++ {
		k := (j + 1) % sides
		fs = append(fs, [3]int{rootMiddle, first + k, first + j},
			[3]int{tipMiddle, first + steps*sides + j, first + steps*sides + k})
	}
	return vs, fs
}

// walrusVibrissae is every bristle of one muzzle pad as one mesh: a stiff
// strand rooted in each follicle, leaving the pad along its own normal
// and tipped steeply down, so the rows hang over one another as a
// moustache does, splayed a little outward toward the pad's edge and a
// little at random. Lengths spread round 8 cm. The strands move with the head by skin. A follicle with no
// hide under it is refused, not left out.
func (rig *walrusRig) walrusVibrissae(side float64, field func(maths.Vec) float64,
	skin func(maths.Vec) maths.Vec) (*shape.Mesh, error) {
	const (
		forward, downward = 0., 1.6
		outward, edge     = .20, .15
		wander, spread    = .04, .3
	)
	var vs []maths.Vec
	var fs [][3]int
	sign := int(side)
	for row := 0; row < walrusFollicleRows; row++ {
		for column := 0; column < walrusFollicleColumns; column++ {
			drop, across := walrusFollicle(row, column, side)
			t, ok := rig.walrusPadSurface(field, across, drop)
			if !ok {
				return nil, fmt.Errorf("%w: the follicle in row %d, column "+
					"%d on side %g", errWalrusLandmark, row, column, side)
			}
			root := rig.frame.at(t, across, drop)
			stray := maths.V(walrusHash(row, column, sign, 13)-.5,
				walrusHash(row, column, sign, 14)-.5,
				walrusHash(row, column, sign, 15)-.5).Scale(2 * wander)
			grow := walrusGradient(field, root).
				Add(rig.frame.axis.Scale(forward)).
				Add(rig.frame.down.Scale(downward)).
				Add(maths.V(0, 0, side*outward*math.Abs(across)/edge)).
				Add(stray).Unit()
			length := walrusStrandLength *
				(1 - spread/2 + spread*walrusHash(row, column, sign, 5))
			vs, fs = walrusAddStrand(vs, fs, root, grow, length, skin)
		}
	}
	return shape.NewMesh(vs, fs, nil, false)
}
