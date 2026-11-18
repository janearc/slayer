//go:build draw && darwin

package gpu

import (
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// splitSide is the samples along a pixel's side in the band-split test:
// four, the count that ships, so a pixel's samples straddle bands of an
// odd height and the sky's looks are shared among them.
const splitSide = 4

// splitHeights are the band heights the test draws at beside the fixed
// most: nought for the trace's own heights, one, and three, which no
// pixel's four rows of samples divide.
var splitHeights = []int{0, 1, 3}

// drawnInBands is the fixture drawn by a function with every band of the
// trace rows tall, or as the trace draws it when rows is nought.
func drawnInBands(t *testing.T, rows int, world *thing.World,
	eye maths.Subspace, by func(*thing.World, maths.Subspace, film.Sensor,
		time.Duration, int, int, int) (*frame.Frame, Profile, error),
) *frame.Frame {
	t.Helper()
	bandRows = rows
	t.Cleanup(func() { bandRows = 0 })
	picture, _ := drawnAt(t, world, eye, splitSide, by)
	return picture
}

// TestAPictureDoesNotDependOnItsBands draws the sky fixture with fixed
// bands of the most rows, and again with the trace's own bands and in
// bands of one and three rows, by the software path
// and by the fast path that shares the sky's looks; every picture is the
// same bit for bit. Nothing here is the cpu's: the gpu is held to itself.
func TestAPictureDoesNotDependOnItsBands(t *testing.T) {
	world, eye := underASky(t)
	drawers := map[string]func(*thing.World, maths.Subspace, film.Sensor,
		time.Duration, int, int, int) (*frame.Frame, Profile, error){
		"software": SampledMeasured, "faster": SampledFaster,
	}
	for name, by := range drawers {
		want := frameHash(drawnInBands(t, rows, world, eye, by))
		for _, height := range splitHeights {
			got := frameHash(drawnInBands(t, height, world, eye, by))
			if got != want {
				t.Errorf("%s: bands of %d rows drew %s; bands of %d "+
					"drew %s", name, height, got, rows, want)
			}
		}
	}
}
