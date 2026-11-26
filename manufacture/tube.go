package manufacture

import (
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/manufacture/branching"
	"github.com/janearc/slayer/manufacture/geometry"
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

var errCut = geometry.ErrCut

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

// Shared dimensional refusals retain their identities across constructors.
var (
	errTubeLength = geometry.ErrTubeLength
	errTubeFoot   = geometry.ErrTubeFoot
	errTubeTip    = geometry.ErrTubeTip
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

// refuseTube is why these dimensions cannot form a tube.
func refuseTube(length, foot, tip measure.Length) error {
	return geometry.CheckTube(inScene(length), inScene(foot), inScene(tip))
}

func (made tube) ring(radius, height float64, fine cut) []maths.Vec {
	return geometry.TubeRing(radius, height, fine.around, made.flat)
}

func (made tube) wall(fine cut) surface {
	return geometry.TubeWall[face](inScene(made.length), inScene(made.foot), inScene(made.tip), fine.around, made.flat)
}

func (made tube) caps(fine cut) surface {
	return geometry.TubeCaps[face](inScene(made.length), inScene(made.foot), inScene(made.tip), fine.around, made.flat)
}

func (made tube) skin(fine cut) surface { return append(made.wall(fine), made.caps(fine)...) }

var errNoRun = branching.ErrNoRun

// spaceAlong returns the shared right-handed frame of a run.
func spaceAlong(at, run maths.Vec) (maths.Subspace, error) {
	return branching.SpaceAlong(at, run)
}
