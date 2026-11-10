//go:build darwin

package gpu

import (
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
	"time"
)

// SampledCached uses native mesh queries and an approximate smoke-light grid.
// Direct sampling remains available through SampledAccelerated.
func SampledCached(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int) (*frame.Frame, Profile, error) {
	return measuredBackend(world, eye, lens, at, width, height, side, cachedTraversal)
}
