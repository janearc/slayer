//go:build draw && darwin

package gpu

import (
	"errors"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// the size the dark lamp pictures are drawn at, and the rays along each
// side of a pixel: small, since every pixel is compared exactly
const (
	darkWidth  = 48
	darkHeight = 32
	darkSide   = 2
)

// darkStrength is the dark lamp's light at one unit: bright enough that
// any light of it that landed would show in every channel
const darkStrength = 60

// darkWidthAngle and darkSoftAngle are the dark lamp's cone and its soft
// rim, narrow, so that pointed up it sends nothing down
const (
	darkWidthAngle = 12 * maths.Degree
	darkSoftAngle  = 4 * maths.Degree
)

// spot is a lamp kept to a narrow cone, six units over the origin,
// pointed along aim
func spot(aim maths.Vec) emit.Lit {
	return emit.Lit{At: maths.V(0, 6, 0), Strength: maths.Grey(darkStrength),
		Toward: aim.Unit(), Width: darkWidthAngle, Soft: darkSoftAngle}
}

// gassed is a gas over ground, under a sun, with the spot first among
// the lamps when there is one, so the sun's index moves with it
func gassed(t *testing.T, withSpot bool,
	aim maths.Vec) (*testing.T, *thing.World, maths.Subspace) {
	world := thing.NewWorld()
	world.Add(ground())
	world.AddVolume(puffOf(dusk(0), 4, 1))
	if withSpot {
		world.Lamp(spot(aim))
	}
	world.Lamp(sunlit(maths.V(0.4, 1, 0.3), maths.RGB(5, 4.5, 4)))
	from := maths.V(0, 2, 12)
	return t, world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, -1, 0))}
}

// drawnSmall draws a world, seen from eye, on the gpu at the dark lamp
// pictures' size
func drawnSmall(t *testing.T, world *thing.World,
	eye maths.Subspace) *frame.Frame {
	t.Helper()
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	drawn, err := Sampled(world, eye, lens, 0, darkWidth, darkHeight,
		darkSide)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

// differing is how many pixels of two pictures are not the same
func differing(one, other *frame.Frame) int {
	count := 0
	for row := range darkHeight {
		for column := range darkWidth {
			if one.At(column, row) != other.At(column, row) {
				count++
			}
		}
	}
	return count
}

// a spot pointed up, away from the gas and the ground, sends nothing to
// any point seen: the picture with it is the picture without it, pixel
// for pixel, through the path that asks every lamp
func TestALampDarkAtEveryPointSeenChangesNoPixel(t *testing.T) {
	without := drawnSmall(gassed(t, false, maths.V(0, 1, 0)))
	with := drawnSmall(gassed(t, true, maths.V(0, 1, 0)))
	if count := differing(without, with); count != 0 {
		t.Errorf("a spot pointed away changed %d pixels; want none", count)
	}
	// the spot really is dark where the gas is, or the test proves nothing
	middle := maths.V(0, 0, 0)
	if _, _, light := spot(maths.V(0, 1, 0)).Light(middle,
		maths.Vec{}); light != (maths.Light{}) {
		t.Fatalf("the spot pointed up sends %v to the gas", light)
	}
}

// the anti-test: the same spot pointed down lights the gas and the
// ground, and is not left out with the dark ones
func TestALampPointedAtTheSceneIsStillLit(t *testing.T) {
	without := drawnSmall(gassed(t, false, maths.V(0, -1, 0)))
	with := drawnSmall(gassed(t, true, maths.V(0, -1, 0)))
	if count := differing(without, with); count == 0 {
		t.Error("a spot pointed down at the ground changed no pixel")
	}
}

// past lampsEvery a point picks among its lamps by worth, and a dark
// lamp is worth nothing, so it is never picked and moves no pick: the
// picture with it, put last, is the picture without it
func TestADarkLampMovesNoPickAmongManyLamps(t *testing.T) {
	ringed := func(withSpot bool) (*testing.T, *thing.World,
		maths.Subspace) {
		world := thing.NewWorld()
		world.Add(thing.Thing{Shape: floor(t, 12), Material: &thing.Material{
			Name: "floor", Colour: maths.Grey(0.6)}, Motion: at(0, -1.5, 0)})
		many(world, false)
		if withSpot {
			world.Lamp(spot(maths.V(0, 1, 0)))
		}
		from := maths.V(0, 4, 10)
		return t, world, maths.Subspace{Origin: from,
			Orientation: maths.LookAt(from, maths.V(0, -1, 0))}
	}
	without := drawnSmall(ringed(false))
	with := drawnSmall(ringed(true))
	if count := differing(without, with); count != 0 {
		t.Errorf("a dark lamp among many changed %d pixels; want none", count)
	}
}
