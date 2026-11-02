package manufacture

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// inScene is a length as the scene holds one. the scene's unit has no name;
// here it is a metre, and this is the one place that says so.
func inScene(length measure.Length) float64 { return length.InMetres() }

// positive is a number greater than zero that is finite.
func positive(number float64) bool {
	return number > 0 && !math.IsInf(number, 0)
}

// cut is how finely a round form is cut into flat sides: how many go
// round it. made only by cutInto, which refuses fewer than can close a
// ring and more than is ever wanted.
type cut struct{ around int }

const (
	// leastAround is the fewest flat sides that close a ring.
	leastAround = 3
	// mostAround is the most sides a ring is cut into. it is tuning,
	// not measurement: the same bound the conifer door in gpu puts on
	// a cone's sides, so a form grown in the kernel and the same form
	// flattened here can be cut alike.
	mostAround = 64
)

var errCut = fmt.Errorf("manufacture: a ring is cut into %d to %d sides",
	leastAround, mostAround)

// cutInto is a cut of so many sides round, or why there can be none.
func cutInto(around int) (cut, error) {
	if around < leastAround || around > mostAround {
		return cut{}, errCut
	}
	return cut{around: around}, nil
}

// tube is a straight run from a ring at its foot to a ring at its tip.
//
// with no radius at the tip it is a cone. flat, its rings are two corners
// each and it is a blade: one strip, which a ray meets from either side.
// the trunk of a tree, a limb, a log, a can, a needle and a leaflet are all
// this. doc/manufactory.md says why there is no more to it.
//
// made only by newTube and newBlade.
type tube struct {
	length, foot, tip measure.Length
	flat              bool
}

// what a tube refuses
var (
	errTubeLength = errors.New("manufacture: a tube needs a length " +
		"above nought, with an end")
	errTubeFoot = errors.New("manufacture: a tube needs a radius at " +
		"its foot above nought, with an end; a cone narrows to its tip")
	errTubeTip = errors.New("manufacture: a tube's radius at its tip " +
		"is nought or above, with an end")
)

// newTube is a round tube of a length, with a radius at each end, or
// why there can be none.
func newTube(length, foot, tip measure.Length) (tube, error) {
	if err := refuseTube(length, foot, tip); err != nil {
		return tube{}, err
	}
	return tube{length: length, foot: foot, tip: tip}, nil
}

// newBlade is a flat tube: a strip of a length, half as wide as foot at
// one end and as tip at the other, or why there can be none.
func newBlade(length, foot, tip measure.Length) (tube, error) {
	if err := refuseTube(length, foot, tip); err != nil {
		return tube{}, err
	}
	return tube{length: length, foot: foot, tip: tip, flat: true}, nil
}

// refuseTube is why these three cannot be a tube, or nothing.
func refuseTube(length, foot, tip measure.Length) error {
	if !positive(inScene(length)) {
		return errTubeLength
	}
	if !positive(inScene(foot)) {
		return errTubeFoot
	}
	if end := inScene(tip); end != 0 && !positive(end) {
		return errTubeTip
	}
	return nil
}

// pointed is whether the tube comes to a point at its tip.
func (made tube) pointed() bool { return inScene(made.tip) == 0 }

// ring is the corners of one ring of the tube in its own space, at a height up
// y and a radius out from it. a round ring starts on x and goes round toward
// minus z, which is counterclockwise seen from the tip; a flat one is its two
// ends on x. a ring of no radius is one point, as many times over.
func (made tube) ring(radius, height float64, fine cut) []maths.Vec {
	if made.flat {
		return []maths.Vec{maths.V(radius, height, 0),
			maths.V(-radius, height, 0)}
	}
	corners := make([]maths.Vec, fine.around)
	for side := range corners {
		angle := fullTurn * float64(side) / float64(fine.around)
		corners[side] = maths.V(math.Cos(angle)*radius, height,
			-math.Sin(angle)*radius)
	}
	return corners
}

// rings is the tube's two rings, foot and tip, corner for corner.
func (made tube) rings(fine cut) ([]maths.Vec, []maths.Vec) {
	return made.ring(inScene(made.foot), 0, fine),
		made.ring(inScene(made.tip), inScene(made.length), fine)
}

// wall is the tube's sides, in its own space: the faces ninja's orientedTube
// makes, in its order. each ring corner is worked out once and used by both
// faces that meet at it, so the wall closes on itself exactly.
//
// a tube that comes to a point has one face a side. a blade has one side: wound
// back the other way as well it would be a closed piece holding nothing, which
// slayer's mesh calls inside out as soon as it is placed and rounding tips the
// nothing less than zero.
func (made tube) wall(fine cut) surface {
	foot, _ := made.rings(fine)
	count := len(foot)
	if made.flat {
		count = 1
	}
	var sides surface
	for side := range count {
		sides = append(sides, made.side(side, fine)...)
	}
	return sides
}

// side is one side of the wall: the faces between one ring corner and
// the next, a quad as two, or one where the tube comes to a point.
func (made tube) side(index int, fine cut) surface {
	foot, tip := made.rings(fine)
	next := (index + 1) % len(foot)
	first := face{foot[index], foot[next], tip[index]}
	if made.pointed() {
		return surface{first}
	}
	return surface{first, {foot[next], tip[next], tip[index]}}
}

// caps is the tube's two ends, each a fan about the middle of its
// ring. a point needs no cap, and a blade has no ends to close.
func (made tube) caps(fine cut) surface {
	if made.flat {
		return nil
	}
	foot, tip := made.rings(fine)
	var ends surface
	for side := range foot {
		next := (side + 1) % len(foot)
		ends = append(ends, face{{}, foot[next], foot[side]})
		if !made.pointed() {
			top := maths.V(0, inScene(made.length), 0)
			ends = append(ends, face{top, tip[side], tip[next]})
		}
	}
	return ends
}

// skin is the whole tube in its own space: its wall, then its caps.
func (made tube) skin(fine cut) surface {
	return append(made.wall(fine), made.caps(fine)...)
}

// nearlyAlong is how much of a run may lie along x before x stops
// being a good side to stand a ring on. it is orientedTube's number,
// and the kernel's tube must use the same or its rings turn.
const nearlyAlong = 0.9

var errNoRun = errors.New("manufacture: a run needs a way to go, with " +
	"an end")

// spaceAlong is the space of a tube that starts at a place and runs a way: its
// y is the run, and its x is square to it by orientedTube's rule, the world's x
// unless the run is nearly along x, and then the world's minus z (ninja's y, in
// the world's space).
//
// a tube's skin placed in it is the tube orientedTube makes between the two
// points.
func spaceAlong(at, run maths.Vec) (maths.Subspace, error) {
	if !positive(run.Len()) {
		return maths.Subspace{}, errNoRun
	}
	axis := run.Unit()
	side := maths.XAxis
	if math.Abs(axis.X) >= nearlyAlong {
		side = maths.ZAxis.Neg()
	}
	side = side.Sub(axis.Scale(side.Dot(axis))).Unit()
	return maths.Subspace{Origin: at, Orientation: maths.Orientation{
		X: side, Y: axis, Z: side.Cross(axis)}}, nil
}
