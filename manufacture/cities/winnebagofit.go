package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The two ends of a run along the vehicle, as the sign of z at that end.
const (
	behindEnd = -1.0
	aheadEnd  = 1.0
)

// mirrors are a mirror on each side of the cab, ahead of the body: an arm
// out of the cab's wall, and a flat of glass at its end.
func (dress winnebagoDress) mirrors() []placedElement {
	body, mirror := dress.sizes.body, dress.fine.mirror
	arm, glass := dress.clothes.of(roleFitting), dress.clothes.of(roleGlazing)
	along := body.length/half + mirror.forward
	wall := body.cabWidth / half
	face := Box{mirror.width, mirror.height, mirror.depth}
	var mirrors []placedElement
	for _, side := range []float64{kerbSide, offSide} {
		end := side * (wall + mirror.arm)
		mirrors = append(mirrors,
			rod("arm", maths.V(side*(wall-mirror.thickness), mirror.at, along),
				maths.V(end, mirror.at, along), mirror.thickness, arm),
			part("mirror", face, glass, standingAt(maths.V(
				end-mirror.width/half, mirror.at-mirror.height/half,
				along-mirror.depth/half))))
	}
	return mirrors
}

// ladder is the ladder up the rear of the body to the roof: two rails, and
// a rung every pitch between them, against the rear face.
func (dress winnebagoDress) ladder() []placedElement {
	body, ladder := dress.sizes.body, dress.fine.ladder
	steel := dress.clothes.of(roleFitting)
	top := body.roof - ladder.topGap
	rails := Box{ladder.rail, top - ladder.foot, ladder.rail}
	rung := Box{ladder.width, ladder.rail, ladder.rail}
	behind := -body.length/half - ladder.rail
	left := ladder.offset - ladder.width/half
	parts := []placedElement{
		part("rail", rails, steel, standingAt(maths.V(left-ladder.rail/half,
			ladder.foot, behind))),
		part("rail", rails, steel, standingAt(maths.V(
			left+ladder.width-ladder.rail/half, ladder.foot, behind)))}
	for index := range int((top - ladder.foot) / ladder.pitch) {
		up := ladder.foot + (float64(index)+1/float64(half))*ladder.pitch
		parts = append(parts, part("rung", rung, steel,
			standingAt(maths.V(left, up, behind))))
	}
	return parts
}

// conditioner is the air-conditioner on the roof: a plate, the shroud on
// it, a fan housing on that, and two bars crossing over the fan.
func (dress winnebagoDress) conditioner() []placedElement {
	body, plant := dress.sizes.body, dress.fine.conditioner
	shell, bars := dress.clothes.of(rolePlant), dress.clothes.of(roleFitting)
	at := func(height float64) maths.Subspace {
		return standingAt(maths.V(0, body.roof+height, plant.place))
	}
	plate := Box{plant.width + bothSides*plant.bar, plant.base,
		plant.length + bothSides*plant.bar}
	shroud := Box{plant.width, plant.height, plant.length}
	round := outlineCircle{Radius: plant.fan}
	crown := plant.base + plant.height + plant.base
	across := Box{bothSides * plant.fan, plant.bar, plant.bar}
	along := Box{plant.bar, plant.bar, bothSides * plant.fan}
	return []placedElement{
		part("plate", plate, shell, seated(plate, at(0))),
		part("shroud", shroud, shell, seated(shroud, at(plant.base))),
		part("fan", prism{Bottom: round, Top: round, Height: plant.base},
			shell, at(plant.base+plant.height)),
		part("grille", across, bars, seated(across, at(crown))),
		part("grille", along, bars, seated(along, at(crown)))}
}

// awning is the roll-out awning on the kerb side: the roller in its case
// at the wall, the fabric sloping down and out from it, and an arm at
// each end from the wall to the fabric's outer edge.
func (dress winnebagoDress) awning() []placedElement {
	body, awning := dress.sizes.body, dress.fine.awning
	steel := dress.clothes.of(roleFitting)
	wall := kerbSide * body.width / half
	slope := Box{math.Hypot(awning.reach, awning.drop), awning.leaf,
		awning.length}
	turn := maths.Rotation(maths.ZAxis, math.Atan2(-awning.drop,
		kerbSide*awning.reach))
	middle := maths.V(wall+kerbSide*awning.reach/half, awning.top-
		awning.drop/half, awning.place)
	round := outlineCircle{Radius: awning.caseRadius}
	parts := []placedElement{
		part("fabric", slope, dress.clothes.of(roleAwning),
			mounted(slope, turn, standingAt(middle))),
		part("roller", prism{Bottom: round, Top: round, Height: awning.length},
			steel, maths.NewSubspace(maths.V(wall+kerbSide*awning.caseRadius,
				awning.top, awning.place-awning.length/half),
				maths.Rotation(maths.XAxis, quarterTurn), 0))}
	return append(parts, dress.awningArms(wall, steel)...)
}

// awningArms are the awning's two arms, one at each end: from the wall,
// out and up to the fabric's outer edge.
func (dress winnebagoDress) awningArms(wall float64,
	steel *thing.Material) []placedElement {
	awning := dress.fine.awning
	outer := maths.V(wall+kerbSide*awning.reach, awning.top-awning.drop, 0)
	var arms []placedElement
	for _, end := range []float64{behindEnd, aheadEnd} {
		outer.Z = awning.place + end*(awning.length/half-awning.arm)
		arms = append(arms, rod("arm", maths.V(wall, awning.armFoot, outer.Z),
			outer, awning.arm, steel))
	}
	return arms
}

// lampAt is a lamp of a kind that shines, standing on a vehicle's face at
// a distance from the middle and a place along its length, half its
// depth out of the face.
func (dress winnebagoDress) lampAt(name string, role partRole, kind lampSizes,
	side, along float64) placedElement {
	size := Box{kind.width, kind.height, kind.depth}
	lens := solid{Name: name, Form: size, Material: dress.clothes.of(role),
		Lamp: &Lamp{Heat: kind.heat, Luminance: kind.strength}}
	return placedElement{Element: lens, Place: standingAt(maths.V(
		side*kind.spread-kind.width/half, kind.at, along-kind.depth/half))}
}

// lamps are the front bumper and grille, a headlamp each side of the
// grille, and a tail lamp each side of the rear.
func (dress winnebagoDress) lamps() []placedElement {
	sizes, ends := dress.sizes, dress.fine.ends
	front := sizes.frontFace()
	rear := -sizes.body.length / half
	bumper := Box{sizes.body.cabWidth, ends.bumperHeight,
		bothSides * ends.bumperProud}
	grille := Box{ends.grilleWidth, ends.grilleHeight,
		bothSides * dress.fine.conditioner.bar}
	parts := []placedElement{
		part("bumper", bumper, dress.clothes.of(roleFitting), standingAt(
			maths.V(-bumper.Width/half, ends.bumperAt, front-ends.bumperProud))),
		part("grille", grille, dress.clothes.of(roleFitting), standingAt(
			maths.V(-grille.Width/half, ends.head.at, front-grille.Depth/half)))}
	for _, side := range []float64{kerbSide, offSide} {
		parts = append(parts,
			dress.lampAt("headlamp", roleHeadlamp, ends.head, side, front),
			dress.lampAt("taillamp", roleTaillamp, ends.tail, side, rear))
	}
	return parts
}
