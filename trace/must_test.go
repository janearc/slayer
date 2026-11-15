package trace

import (
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// must is a picture a test knows can be drawn: the frame, or a panic
// with why not, so a test's own scene is never quietly missing.
func must(picture *frame.Frame, err error) *frame.Frame {
	if err != nil {
		panic(err)
	}
	return picture
}

// draw is Render of arguments a test knows are sound.
func draw(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height int) *frame.Frame {
	return must(Render(world, eye, lens, at, width, height))
}

// drawIn is RenderIn of arguments a test knows are sound.
func drawIn(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height int, band maths.Band) *frame.Frame {
	return must(RenderIn(world, eye, lens, at, width, height, band))
}

// sampled is Sampled of arguments a test knows are sound.
func sampled(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, rays int) *frame.Frame {
	return must(Sampled(world, eye, lens, at, width, height, rays))
}

// lone is what one ray brings back from a world at a moment: a lone
// ray, with no pixel round it. The ray's direction is a unit long.
func lone(world *thing.World, ray maths.Ray,
	at time.Duration) frame.Pixel {
	return shade(stage(world, at), look{ray: ray})
}
