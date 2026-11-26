// Package mammals contains construction shared by existing mammal families.
package mammals

import (
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
)

// Span is an axial bone as the skin is bound to it: its index,
// the x of its middle and its length in x.
type Span struct {
	Bone        int
	Middle, Run float64
}

// AxialHold gives the bones that carry a point and their shares.
// An initial slice preserves the caller's nil or empty result for a gap.
func AxialHold(spans []Span, x, reach float64, hold []skeleton.Influence) []skeleton.Influence {
	first, last := spans[0], spans[len(spans)-1]
	if x >= first.Middle {
		return []skeleton.Influence{skeleton.NewInfluence(first.Bone, 1)}
	}
	if x <= last.Middle {
		return []skeleton.Influence{skeleton.NewInfluence(last.Bone, 1)}
	}
	for _, span := range spans {
		share := 1 - math.Abs(x-span.Middle)/(reach*span.Run)
		if share > 0 {
			hold = append(hold, skeleton.NewInfluence(span.Bone, share))
		}
	}
	return hold
}
