package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// arcology stacks inhabited rings and sky parks around a through courtyard.
// Columns bridge each open gallery gap into the overlapping floor below.
func (row macrostructureMaker) arcology(a *instrumentAssembly, from seed, detail string) {
	w, d, h := row.spec.Width.InMetres(), row.spec.Depth.InMetres(), row.spec.Height.InMetres()
	tiers := int(macroValue("arcology-tiers"))
	pitch := h / float64(tiers)
	holeW, holeD := w*macroValue("courtyard-share"), d*macroValue("courtyard-share")
	for tier := range tiers {
		t := float64(tier) / float64(tiers-1)
		width, depth := w*(1-macroValue("arcology-taper-share")*t),
			d*(1-macroValue("arcology-taper-share")*t)
		base := float64(tier) * pitch
		floor := pitch * macroValue("occupied-height-share")
		if tier == tiers-1 {
			floor = pitch
		}
		centre := maths.V(from.swing(0, 0, 0)*w*.035*t, base, from.swing(0, 0, 1)*d*.025*t)
		macroRing(a, width, depth, holeW, holeD, floor, centre, macroStone)
		if tier > 0 {
			gap := pitch * (1 - macroValue("occupied-height-share"))
			for _, x := range []float64{-1, 1} {
				for _, z := range []float64{-1, 1} {
					at := centre.Add(maths.V(x*(width+holeW)/4, -gap, z*(depth+holeD)/4))
					macroBox(a, w*.025, gap+pitch*.02, d*.025, at, macroMetal)
				}
			}
		}
		if tier < tiers-1 {
			parkHeight := h * macroValue("park-thickness-share")
			macroRing(a, width*.98, depth*.98, holeW*1.03, holeD*1.03,
				parkHeight, centre.Add(maths.V(0, floor, 0)), macroPark)
		}
		if detail != "hull" && tier > 0 {
			macroBox(a, width*.65, floor*.38, d*.002,
				centre.Add(maths.V(0, floor*.3, depth/2)), macroGlass)
		}
	}
}

// spires gives three unequal hollow towers real helical ribs and connected
// elevated galleries. Taper contracts their rotated rectangular sections.
func (row macrostructureMaker) spires(a *instrumentAssembly, from seed, segments int,
	detail string) {
	w, d, h := row.spec.Width.InMetres(), row.spec.Depth.InMetres(), row.spec.Height.InMetres()
	foundation := h * macroValue("foundation-height-share")
	macroBox(a, w, foundation, d, maths.Vec{}, macroStone)
	centres := []maths.Vec{maths.V(-w*.23, foundation, -d*.12),
		maths.V(w*.23, foundation, -d*.08), maths.V(from.swing(1, 0, 0)*w*.045, foundation, d*.25)}
	shares := []float64{1, .78 + from.draw(1, 0, 1)*.08, .6 + from.draw(1, 0, 2)*.08}
	for i, centre := range centres {
		height := (h - foundation) * shares[i]
		phase := from.swing(1, i, 3) * .18
		twist := row.spec.Twist.InRadians() * (1 - float64(i)*.15)
		loft := macroLoft{width: w * macroValue("spire-width-share"),
			depth:      d * macroValue("spire-width-share"),
			innerWidth: w * macroValue("spire-width-share") * .5,
			innerDepth: d * macroValue("spire-width-share") * .5,
			height:     height, taper: macroValue("spire-taper-share"), twist: twist,
			phase: phase, segments: segments}
		a.add(loft, maths.NewSubspace(centre, maths.Identity, 1), macroGlass)
		for corner := range 4 {
			for step := range segments {
				low := loft.corners(float64(step)/float64(segments), false)[corner].Add(centre)
				high := loft.corners(float64(step+1)/float64(segments), false)[corner].Add(centre)
				radius := math.Min(w, d) * macroValue("rib-radius-share")
				if step == segments-1 {
					high.Y -= radius
				}
				a.rod(low, high, radius, macroMetal)
			}
		}
	}
	for i, pair := range [][2]int{{0, 1}, {1, 2}} {
		elevation := h * (.34 + float64(i)*.16)
		first, last := centres[pair[0]], centres[pair[1]]
		first.Y, last.Y = elevation, elevation
		row.gallery(a, first, last, math.Min(w, d)*.055, h*.018, from.draw(1, i, 4))
	}
}

// gallery is an inhabited deck between two tower frames, with a sky park.
func (row macrostructureMaker) gallery(a *instrumentAssembly, from, to maths.Vec,
	width, thickness, parkShare float64) {
	run := to.Sub(from)
	length := run.Len()
	angle := -math.Atan2(run.Z, run.X)
	centre := from.Add(to).Scale(.5)
	space := maths.NewSubspace(centre, maths.About(maths.YAxis, angle), 1)
	a.add(boxShape(length, thickness, width), space, macroStone)
	park := space
	park.Origin.Y += thickness
	a.add(boxShape(length*(.5+.25*parkShare), thickness*.16, width*.8), park, macroPark)
	brace := math.Min(row.spec.Width.InMetres(), row.spec.Depth.InMetres()) * .008
	lowFrom, lowTo := from, to
	lowFrom.Y -= thickness * 3
	lowTo.Y -= thickness * 3
	a.rod(lowFrom, to, brace, macroMetal)
	a.rod(lowTo, from, brace, macroMetal)
}

// bridge carries courtyard galleries above a pair of parabolic arches.
// Vertical spandrels join every arch station to the deck's underside.
func (row macrostructureMaker) bridge(a *instrumentAssembly, from seed, segments int,
	detail string) {
	w, d, h := row.spec.Width.InMetres(), row.spec.Depth.InMetres(), row.spec.Height.InMetres()
	span := w * macroValue("bridge-span-share")
	deck := h * macroValue("bridge-deck-height-share")
	rise := deck * (.87 + .08*from.draw(2, 0, 1))
	radius := math.Min(d*macroValue("arch-radius-share"), h*.015)
	point := func(x, z float64) maths.Vec {
		return maths.V(x,
			rise*(1-4*x*x/(span*span))+radius, z)
	}
	for _, z := range []float64{-d * .26, d * .26} {
		a.add(macroArch{span: span, rise: rise, radius: radius, segments: segments},
			maths.NewSubspace(maths.V(0, radius, z), maths.Identity, 1), macroMetal)
		for step := range segments {
			x0 := -span/2 + span*float64(step)/float64(segments)
			low := point(x0, z)
			a.rod(low, maths.V(x0, deck, z), radius*.45, macroStone)
		}
		a.rod(point(span/2, z), maths.V(span/2, deck, z), radius*.45, macroStone)
		for _, x := range []float64{-w * .43, w * .43} {
			macroBox(a, w*.14, radius*2, d*.3, maths.V(x, 0, math.Copysign(d*.35, z)), macroStone)
		}
	}
	deckThickness := h * macroValue("bridge-deck-thickness-share")
	macroRing(a, w*.96, d*.95, w*.62, d*.35, deckThickness, maths.V(0, deck, 0), macroStone)
	macroRing(a, w*.94, d*.93, w*.64, d*.37, h*macroValue("park-thickness-share"),
		maths.V(0, deck+deckThickness, 0), macroPark)
	// An unequal end pavilion breaks the bridge's otherwise bilateral plan.
	side := 1.0
	if from.draw(2, 0, 0) < .5 {
		side = -1
	}
	base := deck + deckThickness
	macroBox(a, w*.1, h-base, d*.22, maths.V(side*w*.39, base, d*.29), macroGlass)
	if detail != "hull" {
		for _, z := range []float64{-d * .475, d * .475} {
			macroBox(a, w*.9, deckThickness*.5, d*.002, maths.V(0, deck+deckThickness*.2, z),
				macroGlass)
		}
	}
}
