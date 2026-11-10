//go:build !darwin

package gpu

import (
	"errors"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
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

// SampledCached is unavailable on platforms without Metal.
func SampledCached(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int) (*frame.Frame, Profile, error) {
	return nil, Profile{}, ErrNoMetal
}
