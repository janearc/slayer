//go:build draw

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

// mustRender is trace.Render for a test that has no way to go on past an
// error: it panics with the error, which fails the test that called it.
func mustRender(w *thing.World, eye maths.Subspace, lens film.Sensor,
	t time.Duration, width, height int) *frame.Frame {
	f, err := trace.Render(w, eye, lens, t, width, height)
	if err != nil {
		panic(fmt.Sprintf("smoke: Render: %v", err))
	}
	return f
}

// mustRenderIn is trace.RenderIn with the same panic as mustRender.
func mustRenderIn(w *thing.World, eye maths.Subspace, lens film.Sensor,
	t time.Duration, width, height int, band maths.Band) *frame.Frame {
	f, err := trace.RenderIn(w, eye, lens, t, width, height, band)
	if err != nil {
		panic(fmt.Sprintf("smoke: RenderIn: %v", err))
	}
	return f
}
