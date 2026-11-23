package manufacture

import (
	"math"
	"sync"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// How the skin is bound to the bones. Each point of the trunk follows
// the skull or the vertebrae nearest it along the body, each by a share
// that falls from one at the vertebra's middle to zero a reach away; a
// point on a flipper follows that flipper's humerus, the forearm and
// hand being one stiff plate in a cetacean. The reach and the width of
// the flipper's hold are chosen: wide enough that no crease shows.
const (
	orcaBlendReach  = 1.5  // a vertebra's hold, in its own lengths each side
	orcaFlipperHold = .05  // width over which the flipper's hold fades, in metres
	orcaFlipperLean = .015 // how far the hold's edge lies toward the trunk
)

// orcaSpan is one bone of the axis, the skull or a vertebra, as the skin
// is bound to it: its place, the x of its middle and its length in x.
type orcaSpan struct {
	bone        int
	middle, run float64
}

// orcaAxis is the skull and the column, front to back.
func orcaAxis(bones *skeleton.Skeleton) []orcaSpan {
	spans := []orcaSpan{}
	for i := 0; i < bones.Len(); i++ {
		bone := bones.Bone(i)
		region, _ := orcaVertebraOf(bone.Name())
		if region == "" && bone.Name() != "cranium" {
			continue
		}
		head, tail := bone.Head().X, bone.Tail().X
		spans = append(spans, orcaSpan{bone: i, middle: (head + tail) / 2,
			run: math.Abs(head - tail)})
	}
	return spans
}

// orcaAxialHold is the bones of the axis that move a point at x, each
// by its share. Ahead of the skull's middle the skull holds it alone;
// behind the last vertebra's middle, the last vertebra.
func orcaAxialHold(spans []orcaSpan, x float64) []skeleton.Influence {
	first, last := spans[0], spans[len(spans)-1]
	if x >= first.middle {
		return []skeleton.Influence{skeleton.NewInfluence(first.bone, 1)}
	}
	if x <= last.middle {
		return []skeleton.Influence{skeleton.NewInfluence(last.bone, 1)}
	}
	hold := []skeleton.Influence{}
	for _, span := range spans {
		share := 1 - math.Abs(x-span.middle)/(orcaBlendReach*span.run)
		if share > 0 {
			hold = append(hold, skeleton.NewInfluence(span.bone, share))
		}
	}
	return hold
}

// orcaFlipperShare is how far rest point p belongs to a flipper rather
// than to the trunk: one on the paddle, zero on the body, eased across
// the fillet between.
func orcaFlipperShare(p maths.Vec) float64 {
	lead := orcaFlipper(p) - orcaTrunk(p) + orcaFlipperLean
	return orcaStep(orcaFlipperHold/2, -orcaFlipperHold/2, lead)
}

// orcaBind is the bones that move rest point p and the share of each.
func orcaBind(bones *skeleton.Skeleton, spans []orcaSpan,
	p maths.Vec,
) []skeleton.Influence {
	hold := orcaAxialHold(spans, p.X)
	flipper := orcaFlipperShare(p)
	if flipper == 0 {
		return hold
	}
	side := "right-humerus"
	if p.Z < 0 {
		side = "left-humerus"
	}
	total := 0.
	for _, each := range hold {
		total += each.Weight()
	}
	scaled := []skeleton.Influence{}
	for _, each := range hold {
		scaled = append(scaled, skeleton.NewInfluence(each.Bone(),
			each.Weight()*(1-flipper)/total))
	}
	return append(scaled, skeleton.NewInfluence(orcaMust(bones, side),
		flipper))
}

// orcaBindingOnce binds the rest skin once; it is shared after.
var (
	orcaBindingOnce   sync.Once
	orcaBindingShared [][]skeleton.Influence
	orcaBindingErr    error
)

// orcaBinding is, for every point of the rest skin, the bones that move
// it and the share of each.
func orcaBinding() ([][]skeleton.Influence, error) {
	orcaBindingOnce.Do(func() {
		bones, err := orcaSkeleton()
		if err != nil {
			orcaBindingErr = err
			return
		}
		spans := orcaAxis(bones)
		rest := orcaRest()
		binding := make([][]skeleton.Influence, len(rest.points))
		for i, p := range rest.points {
			binding[i] = orcaBind(bones, spans, p)
		}
		orcaBindingShared = binding
	})
	return orcaBindingShared, orcaBindingErr
}
