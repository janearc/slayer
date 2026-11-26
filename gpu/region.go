package gpu

import (
	"errors"
	"image"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// ErrRegionUnavailable says that Metal region drawing is not yet available.
var ErrRegionUnavailable = errors.New("gpu: region drawing is not yet implemented")

// SampledRegion experimentally draws a half-open rectangle of a full picture.
// Only software traversal through a pinhole is implemented.
func SampledRegion(mode Mode, world *thing.World, eye maths.Subspace,
	lens film.Sensor, at time.Duration, width, height, side int,
	region image.Rectangle) (*frame.Frame, Profile, error) {
	if !mode.known() {
		return nil, Profile{}, errNoMode
	}
	if mode != Software {
		return nil, Profile{}, ErrRegionUnavailable
	}
	if world == nil {
		return nil, Profile{}, errNoWorld
	}
	if width <= 0 || height <= 0 || side <= 0 || region.Empty() ||
		!region.In(image.Rect(0, 0, width, height)) {
		return nil, Profile{}, errors.New("gpu: invalid picture or region")
	}
	return sampledRegion(world, eye, lens, at, width, height, side, region)
}
