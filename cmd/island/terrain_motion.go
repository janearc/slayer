package main

import (
	"context"
	"time"

	"github.com/janearc/slayer/scene/footprint"
	"github.com/janearc/slayer/scene/frame"
	"github.com/janearc/slayer/scene/island"
)

// terrainTransitionPlan gives temporal refinement explicit time and work bounds.
type terrainTransitionPlan struct {
	Seconds float64 `yaml:"seconds" json:"seconds"`
	Patches int     `yaml:"patches" json:"patches"`
	Bytes   int64   `yaml:"payload-bytes" json:"payload-bytes"`
}

// prepareGround uses the optional scene-time transition owner. Omission keeps
// the stateless per-view preparation path.
func (session *sceneSession) prepareGround(ctx context.Context, plan *plan, view footprint.Perspective, origin frame.Relative) (island.Prepared, error) {
	if session.terrainMotion == nil {
		return session.ground.Prepare(ctx, view, origin)
	}
	return session.terrainMotion.Prepare(ctx, view, origin, time.Duration(plan.Time*float64(time.Second)))
}
