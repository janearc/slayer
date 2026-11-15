package island

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/footprint"
	"github.com/janearc/slayer/scene/frame"
)

// Motion retains one terrain representation and at most one pending target.
// Targets remain view-dependent; their changes are spread over scene time.
// Calls must be serial and clocks must not run backwards.
type Motion struct {
	ground  *Terrain
	span    time.Duration
	patches int
	bytes   int64
	current *Prepared
	active  *terrainTransition
	last    time.Duration
	clocked bool
}

// TransitionRecord describes the displayed geometry, not a quality certificate.
// PayloadBytes counts endpoint vertex values; map and mesh storage are separate.
type TransitionRecord struct {
	Share         float64 `json:"share"`
	FromPatches   int     `json:"from-patches"`
	TargetPatches int     `json:"target-patches"`
	PayloadBytes  int64   `json:"endpoint-payload-bytes"`
}

// NewMotion checks explicit transition duration, common-patch and payload limits.
// The terrain's point allowance also applies to common-refinement vertices.
func NewMotion(ground *Terrain, span time.Duration, patches int, bytes int64) (*Motion, error) {
	if ground == nil || ground.from == nil || span <= 0 || patches < 1 || bytes <= 0 {
		return nil, fmt.Errorf("island: invalid terrain transition controls")
	}
	if bytes/transitionValueBytes < int64(ground.detail.Side)*int64(ground.detail.Side) {
		return nil, fmt.Errorf("island: terrain transition payload cannot hold one lattice")
	}
	return &Motion{ground: ground, span: span, patches: patches, bytes: bytes}, nil
}

// Prepare keeps a transition's target fixed until its interval ends. The next
// target is selected using the current view; this bounded delay can lag motion.
// It does not certify the cell-span target or camera clearance.
func (motion *Motion) Prepare(ctx context.Context, view footprint.Perspective, origin frame.Relative, at time.Duration) (Prepared, error) {
	if motion == nil || ctx == nil {
		return Prepared{}, fmt.Errorf("island: no terrain motion owner")
	}
	if err := ctx.Err(); err != nil {
		return Prepared{}, err
	}
	if motion.clocked && at < motion.last {
		return Prepared{}, fmt.Errorf("island: terrain transition clock moved backwards")
	}
	if motion.active != nil && at-motion.active.started >= motion.span {
		current := motion.active.target
		motion.current = &current
		motion.active = nil
	}
	if motion.active == nil {
		selection, err := motion.ground.selectView(ctx, view)
		if err != nil {
			return Prepared{}, err
		}
		if motion.current != nil && slices.Equal(motion.current.Patches, selection.Patches) {
			motion.current.NormalStencils, motion.current.NormalHeightQueries = 0, 0
			motion.current.Limited = selection.Limited
		} else {
			target, err := motion.ground.prepareSelection(ctx, selection, origin)
			if err != nil {
				return Prepared{}, err
			}
			if motion.current == nil || len(motion.current.Patches) == 0 || len(target.Patches) == 0 {
				motion.current = &target
			} else {
				active, err := motion.transition(ctx, *motion.current, target, at)
				if err != nil {
					return Prepared{}, err
				}
				motion.active = active
			}
		}
	}
	var result Prepared
	if motion.active != nil {
		share := float64(at-motion.active.started) / float64(motion.span)
		var err error
		result, err = motion.active.prepare(ctx, origin, maths.Smooth(0, 1, share))
		if err != nil {
			return Prepared{}, err
		}
	} else {
		result = *motion.current
	}
	motion.last, motion.clocked = at, true
	return result, nil
}

// Reset releases endpoint and common-refinement ownership without closing the
// shared source. A subsequent view starts a new terrain track.
func (motion *Motion) Reset() {
	motion.current, motion.active = nil, nil
	motion.clocked = false
}
