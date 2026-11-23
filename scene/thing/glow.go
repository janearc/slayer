package thing

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
)

// Glow is the light a bounded stretch of air holds that comes from what
// shines in the world, worked out beforehand: package skyglow's field is
// one. It is the mean radiance of that light at a place, which the
// camera's march adds, times the albedo, as the source of each step it
// takes through the air the glow is of.
//
// A world with no Glow draws as it always did.
type Glow interface {
	// Mean is the mean radiance, per channel, over all directions at a
	// place in the world's space.
	Mean(at maths.Vec) maths.Light
	// Lights says whether a medium is the air the glow is of. A march
	// through any other volume takes nothing from it.
	Lights(made medium.Medium) bool
}
