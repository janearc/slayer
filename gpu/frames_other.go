//go:build !darwin

package gpu

import (
	"fmt"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// Frames is refused off a mac, as Sampled is: it is made, so a caller
// builds everywhere, and every draw says why it cannot.
type Frames struct{ mode Mode }

// NewFrames is a drawer that refuses every draw with ErrNoMetal.
func NewFrames(mode Mode) (*Frames, error) {
	if !mode.known() {
		return nil, fmt.Errorf("%w: %d", errNoMode, mode)
	}
	return &Frames{mode: mode}, nil
}

// NewCompositeFrames validates the selector but cannot draw off a Mac.
func NewCompositeFrames(mode Mode, pass CompositePass) (*Frames, error) {
	if err := validPass(pass); err != nil {
		return nil, err
	}
	return NewFrames(mode)
}

// Transmission has no frame off a Mac.
func (frames *Frames) Transmission() *frame.Frame { return nil }

// Draw is refused off a mac.
func (frames *Frames) Draw(world *thing.World, generation uint64,
	eye maths.Subspace, lens film.Sensor, at time.Duration, width,
	height, side int) (*frame.Frame, Profile, error) {
	return nil, Profile{}, ErrNoMetal
}

// DrawOpen is refused off a mac.
func (frames *Frames) DrawOpen(world *thing.World, generation uint64,
	eye, closing maths.Subspace, lens film.Sensor, at time.Duration,
	width, height, side int) (*frame.Frame, Profile, error) {
	return nil, Profile{}, ErrNoMetal
}

// Close has nothing to let go.
func (frames *Frames) Close() {}
