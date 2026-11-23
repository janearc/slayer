package trace

import (
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The size of the pictures the spread draws are held to.
const (
	spreadSide = 16 // defined: small enough to draw at once
	spreadRays = 4  // defined: sixteen rays a pixel, as a set's film has
)

// placedLens is a pinhole whose every fine pixel looks through the place
// film.Within gives it, for a picture of so many rays along a pixel's
// side. It has no places of its own to ask for, so a world that does not
// spread its rays draws through it what a world that does draws through
// the pinhole.
type placedLens struct {
	lens film.Pinhole
	side int
	ramp float64
}

// Ray is the pinhole's ray through the fine pixel's place.
func (placed placedLens) Ray(x, y, w, h int, band maths.Band) maths.Ray {
	across, down := film.Within(x, y, placed.side, placed.ramp)
	return placed.lens.RayThrough(x, y, across, down, w, h, band)
}

// spreading is the smoking scene with its rays spread under the tent.
func spreading() *thing.World {
	world := smoking()
	world.Filter = thing.TentFilter
	return world
}

// A world that spreads its rays is drawn, to the bit, as the plain mean
// of rays that start at film's places for its filter: nothing else of
// the draw moves. So for each of the three filters that spread.
func TestASpreadDrawIsTheMeanOfItsPlacedRays(t *testing.T) {
	for _, filter := range []thing.Filter{thing.BoxFilter,
		thing.TrapezoidFilter, thing.TentFilter} {
		world := smoking()
		world.Filter = filter
		ramp, _ := filter.Ramp()
		got := sampled(world, overTheScene, film.Pinhole{}, 0,
			spreadSide, spreadSide, spreadRays)
		want := sampled(smoking(), overTheScene,
			placedLens{side: spreadRays, ramp: ramp}, 0, spreadSide,
			spreadSide, spreadRays)
		samePixels(t, got, want, "spread,")
	}
}

// At one ray a pixel there is nothing to spread: the picture is Render's.
func TestASpreadDrawOfOneRayIsRender(t *testing.T) {
	once := sampled(spreading(), overTheScene, film.Pinhole{}, 0,
		spreadSide, spreadSide, oneRay)
	plain := draw(smoking(), overTheScene, film.Pinhole{}, 0, spreadSide,
		spreadSide)
	samePixels(t, once, plain, "spread at one ray,")
}

// Anti: spread, the picture is not the regular grid's. If the world's
// word were dropped on the way to the rays the two would be one.
func TestASpreadDrawIsNotTheGrids(t *testing.T) {
	spread := sampled(spreading(), overTheScene, film.Pinhole{}, 0,
		spreadSide, spreadSide, spreadRays)
	grid := sampled(smoking(), overTheScene, film.Pinhole{}, 0,
		spreadSide, spreadSide, spreadRays)
	moved := 0
	for index := range grid.Pix {
		if spread.Pix[index] != grid.Pix[index] {
			moved++
		}
	}
	if moved == 0 {
		t.Error("no pixel moved when the rays were spread")
	}
}

// A spread draw by rows is Sampled's to the bit: a pixel is still the
// sum of its own rays, so a row can be kept and taken up.
func TestAKeptSpreadDrawIsSampled(t *testing.T) {
	want := sampled(spreading(), overTheScene, film.Pinhole{}, 0,
		spreadSide, spreadSide, spreadRays)
	spreadOut := spreading()
	got := must(SampledKept(spreadOut, overTheScene, film.Pinhole{}, 0,
		spreadSide, spreadSide, spreadRays, nil, Keeping{}))
	samePixels(t, got, want, "spread and drawn by rows,")
}
