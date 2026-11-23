package skeleton

import "github.com/janearc/slayer/maths"

// loftRings is the rings along a lofted bone; chosen, enough for a
// skull's outline to read at a close camera.
const loftRings = 16

// Loft is a bone whose sections are given one by one along its line, a
// form for a skull, a tooth or any bone with an outline of its own. Each
// section is an ellipse, wide square to up and deep along it, its centre
// lifted along up; between sections the sizes run straight.
type Loft struct {
	head, tail, up maths.Vec
	sections       []LoftSection
}

// LoftSection is one section of a loft: a share of the way from head to
// tail, a half width, a half depth and a lift of its centre, in metres.
type LoftSection struct{ share, wide, deep, lift float64 }

// NewLoftSection is a section at share with those sizes and that lift.
func NewLoftSection(share, wide, deep, lift float64) LoftSection {
	return LoftSection{share: share, wide: wide, deep: deep, lift: lift}
}

// NewLoft is a loft from head to tail through sections, in order of
// share, with up the way it lifts and is deep.
func NewLoft(head, tail, up maths.Vec, sections ...LoftSection) Loft {
	return Loft{head: head, tail: tail, up: up, sections: sections}
}

// Middle is the loft's line, head to tail.
func (loft Loft) Middle() []maths.Vec {
	return []maths.Vec{loft.head, loft.tail}
}

// Build adds the loft to the mesh as one swept tube. A loft of no
// sections adds nothing.
func (loft Loft) Build(points []maths.Vec, faces [][3]int,
	place func(maths.Vec) maths.Vec) ([]maths.Vec, [][3]int) {
	if len(loft.sections) == 0 {
		return points, faces
	}
	along := loft.tail.Sub(loft.head)
	way := along.Unit()
	alongWay := way.Dot(loft.up)
	lift := loft.up.Sub(way.Scale(alongWay)).Unit()
	path := func(t float64) maths.Vec {
		raised := loft.at(t).lift
		return loft.head.Add(along.Scale(t)).Add(lift.Scale(raised))
	}
	half := func(t float64) (float64, float64) {
		section := loft.at(t)
		return section.wide, section.deep
	}
	return Sweep(points, faces, loftRings, path, half, loft.up, place)
}

// at is the section at share t: the nearest given one past either end,
// and straight between the two about it within.
func (loft Loft) at(t float64) LoftSection {
	sections := loft.sections
	if t <= sections[0].share {
		return sections[0]
	}
	for index := 1; index < len(sections); index++ {
		low, high := sections[index-1], sections[index]
		if t <= high.share {
			share := (t - low.share) / (high.share - low.share)
			mix := func(a, b float64) float64 { return a + (b-a)*share }
			return LoftSection{share: t, wide: mix(low.wide, high.wide),
				deep: mix(low.deep, high.deep), lift: mix(low.lift, high.lift)}
		}
	}
	return sections[len(sections)-1]
}
