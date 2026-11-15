package smoke

import (
	"fmt"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// mustSampled is trace.Sampled with the same panic as mustRender.
func mustSampled(w *thing.World, eye maths.Subspace, lens film.Sensor,
	t time.Duration, width, height, rays int) *frame.Frame {
	f, err := trace.Sampled(w, eye, lens, t, width, height, rays)
	if err != nil {
		panic(fmt.Sprintf("smoke: Sampled: %v", err))
	}
	return f
}
