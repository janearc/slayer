//go:build !darwin

package gpu

import (
	"errors"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// ErrNoMetal is every draw off a mac: the kernel is Metal, and this is
// not Apple's. A WebGPU kernel is the way to the Steam Deck.
var ErrNoMetal = errors.New("gpu: slayer's kernel is Metal, mac only")

// Sampled is refused off a mac, so the commands that use it still build
// everywhere and say why they cannot draw.
func Sampled(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int) (*frame.Frame, error) {
	return nil, ErrNoMetal
}

// SampledMeasured reports the same unsupported backend as Sampled.
func SampledMeasured(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int,
) (*frame.Frame, Profile, error) {
	return nil, Profile{}, ErrNoMetal
}

// SampledWithInitialMedia is unavailable off a Mac. It never silently draws
// in air when the caller explicitly supplies a camera medium.
func SampledWithInitialMedia(mode Mode, world *thing.World,
	eye maths.Subspace, lens film.Sensor, at time.Duration, width, height,
	side int, media []trace.InitialMedium) (*frame.Frame, Profile, error) {
	return nil, Profile{}, ErrNoMetal
}

// SampledNative is unavailable on platforms without Metal.
func SampledNative(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int,
) (*frame.Frame, Profile, error) {
	return nil, Profile{}, ErrNoMetal
}

// SampledAccelerated is unavailable on platforms without Metal.
func SampledAccelerated(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int,
) (*frame.Frame, Profile, error) {
	return nil, Profile{}, ErrNoMetal
}

// SampledFast is unavailable on platforms without Metal.
func SampledFast(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int) (*frame.Frame, Profile, error) {
	return nil, Profile{}, ErrNoMetal
}

// SampledFaster is unavailable on platforms without Metal.
func SampledFaster(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int) (*frame.Frame, Profile, error) {
	return nil, Profile{}, ErrNoMetal
}

// SampledCached is unavailable on platforms without Metal.
func SampledCached(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int) (*frame.Frame, Profile, error) {
	return nil, Profile{}, ErrNoMetal
}
