package gpu

import (
	"testing"

	thin "github.com/janearc/slayer/ninjafilm/lens"
)

// thinLens is a 50 mm lens wide open at f/1.4 on a full frame, focused on
// the hall's middle, thirteen metres off, so the ball in front and the
// awning behind are soft: every ray of a pixel starts somewhere else on
// the aperture
func thinLens(t *testing.T) thin.Thin {
	t.Helper()
	full, err := thin.FrameOf(0.036, 0.024)
	if err != nil {
		t.Fatal(err)
	}
	glass, err := thin.NewThin(full, 50, 1.4, 13)
	if err != nil {
		t.Fatal(err)
	}
	return glass
}

// the gpu draws the hall through a thin lens as slayer's tracer does:
// the rays are the lens's own, made on the cpu by film.Look, so its
// aperture and its focus are the tracer's
func TestTheGPUDrawsThroughALensAsSlayerDoes(t *testing.T) {
	world, eye := hall(t)
	baking := bakeSide
	bakeSide = 0
	defer func() { bakeSide = baking }()
	withinThrough(t, world, eye, thinLens(t), 96, 64, 2, exact)
}
