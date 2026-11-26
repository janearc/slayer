//go:build !darwin

package gpu

import (
	"image"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// sampledRegion refuses the Metal backend on unsupported systems.
func sampledRegion(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int, region image.Rectangle,
) (*frame.Frame, Profile, error) {
	return nil, Profile{}, ErrNoMetal
}
