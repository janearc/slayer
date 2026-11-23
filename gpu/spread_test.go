//go:build draw && darwin

package gpu

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/thing"
	slayer "github.com/janearc/slayer/trace"
)

// The picture the spread checks draw: the exposure checks' own, at four
// samples a side so a pixel's sixteen are what a set's film has.
const (
	spreadSide   = 4
	spreadWidth  = openWidth / spreadSide
	spreadHeight = openHeight / spreadSide
	// placeSlack is how far a kernel's spread ray may lie from film's,
	// as the length of the difference of two unit vectors. Picked: the
	// two start from the same shares and differ in a float's square
	// root and its sums, so this is the exposure checks' own slack for
	// a way, with ten times the room.
	placeSlack = 10 * waySlack
	// tentRamp is defined as the width of a tent's slope, in pixels: one.
	tentRamp = 1
	// placeMoved is how far most spread rays lie from their samples'
	// middles at least, the same way. Picked: a tenth of a sample's
	// own angle in this picture.
	placeMoved = 1e-3
)

// spreadWorld is the hall with its rays spread under the tent.
func spreadWorld(t *testing.T) (*thing.World, maths.Subspace) {
	world, eye := hall(t)
	world.Filter = thing.TentFilter
	return world, eye
}

// spreadView is the header's fields for the hall's eye through a
// pinhole, its samples spread at the test's side.
func spreadView(t *testing.T) (values, maths.Subspace) {
	world, eye := spreadWorld(t)
	view := viewValues(eye, film.Pinhole{}, shutter{}, openWidth,
		openHeight)
	view["spread"] = spreadRow(world, spreadSide)
	return view, eye
}

// placedApart is how far the kernel's rays are from film's placed rays
// at most, and how many lie further than placeMoved from their samples'
// middles.
func placedApart(got []float32, eye maths.Subspace) (float64, int) {
	var furthest float64
	moved := 0
	for y := range openHeight {
		for x := range openWidth {
			ray := got[(y*openWidth+x)*probeWords:]
			toward := maths.V(float64(ray[3]), float64(ray[4]),
				float64(ray[5]))
			across, down := film.Within(x, y, spreadSide, tentRamp)
			want := film.LookThrough(eye, film.Pinhole{}, x, y,
				across, down, openWidth, openHeight,
				maths.Band{})
			middle := film.Look(eye, film.Pinhole{}, x, y,
				openWidth, openHeight, maths.Band{})
			apart := toward.Sub(want.Dir).Len()
			furthest = math.Max(furthest, apart)
			if toward.Sub(middle.Dir).Len() > placeMoved {
				moved++
			}
		}
	}
	return furthest, moved
}

// The header says nothing of a spread unless the world spreads its rays
// and a pixel has more than one sample: the kernel then starts every
// sample at its middle, as it always did.
func TestTheHeaderSpreadsOnlyWhereThereIsSomethingToSpread(t *testing.T) {
	spread := &thing.World{Filter: thing.TentFilter}
	if row := spreadRow(&thing.World{}, spreadSide); row != nil {
		t.Errorf("a world that does not spread has the row %v", row)
	}
	if row := spreadRow(spread, wholeOne); row != nil {
		t.Errorf("a pixel of one sample has the row %v", row)
	}
	row := spreadRow(spread, spreadSide)
	centring := float64(film.Stratum(spreadSide * spreadSide))
	if len(row) != rampAt+wholeOne || row[0] != wholeOne ||
		row[1] != spreadSide || row[2] != centring ||
		row[rampAt] != tentRamp {
		t.Errorf("a spread at %d a side has the row %v", spreadSide,
			row)
	}
}

// The kernel's spread rays are film's own for the same samples, to a
// float's precision: one definition of a place, in both tracers.
func TestTheKernelsSpreadRaysAreFilmsOwn(t *testing.T) {
	view, eye := spreadView(t)
	got := probedRays(t, view, openWidth, openHeight)
	furthest, moved := placedApart(got, eye)
	t.Logf("ways %.3g apart at most; %d of %d off their middles",
		furthest, moved, openWidth*openHeight)
	if !(furthest <= placeSlack) {
		t.Errorf("the kernel's spread rays are %g from film's; want "+
			"%g or less", furthest, placeSlack)
	}
}

// Anti: and they are not the samples' middles. If the header's row were
// not read, every sample would start where it always did.
func TestTheKernelsSpreadRaysLeaveTheirMiddles(t *testing.T) {
	view, eye := spreadView(t)
	got := probedRays(t, view, openWidth, openHeight)
	_, moved := placedApart(got, eye)
	if half := openWidth * openHeight / wholeTwo; moved < half {
		t.Errorf("%d of %d spread rays left their middles; want more "+
			"than half", moved, openWidth*openHeight)
	}
}

// A spread picture on the gpu is the tracer's spread picture, at the
// package's exact tolerance: the two obey one definition at an edge.
func TestASpreadPictureIsTheTracers(t *testing.T) {
	baking := bakeSide
	bakeSide = 0
	defer func() { bakeSide = baking }()
	world, eye := spreadWorld(t)
	drawn, err := Sampled(world, eye, film.Pinhole{}, 0, spreadWidth,
		spreadHeight, spreadSide)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	truth, err := slayer.Sampled(world, eye, film.Pinhole{}, 0,
		spreadWidth, spreadHeight, spreadSide)
	if err != nil {
		t.Fatal(err)
	}
	found := compare(truth, drawn, spreadWidth, spreadHeight)
	t.Logf("spread, apart: %+v", found)
	for _, problem := range findings(found, exact) {
		t.Errorf("spread: %s", problem)
	}
}
