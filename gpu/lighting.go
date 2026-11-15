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

// SampledFast is SampledAccelerated with fast light: every boxed medium
// that is not a gas is lit from the lamp store of reach.go, an
// approximation whose difference from SampledAccelerated reach_test.go
// measures. Surfaces and every other medium are lit as before.
func SampledFast(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int) (*frame.Frame, Profile, error) {
	return measuredBackend(world, eye, lens, at, width, height, side,
		fastTraversal)
}

// SampledFaster is SampledFast with the sky's looks shared among a
// pixel's samples (sky_share.go): each sample takes its share of the sixteen
// looks where the exact tracer takes all of them at every sample.
func SampledFaster(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int) (*frame.Frame, Profile, error) {
	return measuredBackend(world, eye, lens, at, width, height, side,
		fasterTraversal)
}
