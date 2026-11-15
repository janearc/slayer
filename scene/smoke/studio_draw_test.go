//go:build draw

package smoke_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/smoke"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// Smoke forty-nine and the things in it, in the order its builder adds
// them: the cove, the table's top and four legs, the glass, and the
// water in it.
const (
	smokeFortyNine = 49 // defined: the number of a glass of water on a wooden table in an infinity cove
	studioGlass    = 6  // defined: the glass is added after the cove and the table's five pieces
	studioWater    = 7  // defined: and the water after the glass
	studioThings   = 8  // defined: how many things the smoke has
)

// Where the water is in the glass, as shares of the picture, measured off
// it: a column down the glass's middle, a row in the water and a row
// above its level, in the glass's empty top.
const (
	waterColumn = 0.515 // measured: the glass's middle, across
	waterRow    = 0.50  // measured: a row through the body of the water
	emptyRow    = 0.36  // measured: a row above the water's level, where the glass is empty
)

// Where things are in smoke forty-nine's picture, as shares of its width
// and height, measured off the picture as its builder's numbers draw it.
const (
	glassLeft, glassRight = 0.47, 0.56 // measured: the glass's outline, across
	glassTop, glassFoot   = 0.33, 0.60 // measured: the glass's outline, down
	coveColumn            = 0.90       // measured: a column right of the table's end, where there is only cove
	coveRows              = 0.75       // measured: the cove runs unshadowed down that column this far
	grainRow              = 0.60       // measured: a row across the table's top
	grainFrom, grainTo    = 0.10, 0.40 // measured: the stretch of that row that is the top's face, left of the glass
	coveStep              = 0.03       // picked: no row of the cove is this much brighter or darker than the one above; a line where floor meets wall would be
	grainSpread           = 0.03       // picked: the top's early and late wood differ by more than this in a row
	bentAtLeast           = 20         // picked: pixels inside the glass's outline that its bending must change
)

// studioRig is smoke forty-nine, its eye and its lens.
type studioRig struct {
	made smoke.Smoke
	eye  maths.Subspace
	lens film.Pinhole
	shot *frame.Frame
}

// draw is a world through the smoke's eye.
func (rig *studioRig) draw(t *testing.T, stage *thing.World) *frame.Frame {
	t.Helper()
	shot, err := trace.Render(stage, rig.eye, rig.lens, nought, undrawnW, undrawnH)
	if err != nil {
		t.Fatal(err)
	}
	return shot
}

// part is a share of a whole number of pixels, as a pixel's place.
func part(share float64, whole int) int { return int(share * float64(whole)) }

// lit is how bright a picture is at a share of the way across and down.
func lit(shot *frame.Frame, across, down float64) float64 {
	return shot.At(part(across, shot.W), part(down, shot.H)).Light().Luminance()
}

// glassBends checks the glass bends what is behind it: with the glass
// taken away, pixels inside its outline change, and a corner far from it
// does not.
func (rig *studioRig) glassBends(t *testing.T) {
	stage, _ := rig.made.Scene()
	stage.Things = stage.Things[:studioGlass]
	bare := rig.draw(t, stage)
	changed := nought
	for y := part(glassTop, undrawnH); y < part(glassFoot, undrawnH); y++ {
		for x := part(glassLeft, undrawnW); x < part(glassRight, undrawnW); x++ {
			if rig.shot.At(x, y) != bare.At(x, y) {
				changed++
			}
		}
	}
	if changed < bentAtLeast {
		t.Errorf("taking the glass away changes %d pixels inside its outline", changed)
	}
	if rig.shot.At(nought, nought) != bare.At(nought, nought) {
		t.Errorf("taking the glass away changes the far corner: %+v, %+v",
			rig.shot.At(nought, nought), bare.At(nought, nought))
	}
}

// waterBends checks the water is in the glass and bends what is behind
// it more than the glass alone: with the water poured out, the picture
// changes in the body of the water and not above its level, where there
// was only glass and air.
func (rig *studioRig) waterBends(t *testing.T) {
	stage, _ := rig.made.Scene()
	stage.Things = stage.Things[:studioWater]
	empty := rig.draw(t, stage)
	across := part(waterColumn, undrawnW)
	in, above := part(waterRow, undrawnH), part(emptyRow, undrawnH)
	if rig.shot.At(across, in) == empty.At(across, in) {
		t.Errorf("pouring the water out changes nothing in the body of it: %+v",
			empty.At(across, in))
	}
	if rig.shot.At(across, above) != empty.At(across, above) {
		t.Errorf("pouring the water out changes the glass above its level: %+v, %+v",
			rig.shot.At(across, above), empty.At(across, above))
	}
}

// coveHasNoLine checks the cove turns from floor to wall with no line:
// down a column of nothing but cove, no row is much brighter or darker
// than the one above it.
func (rig *studioRig) coveHasNoLine(t *testing.T) {
	last := lit(rig.shot, coveColumn, nought)
	for y := unity; y < part(coveRows, undrawnH); y++ {
		here := rig.shot.At(part(coveColumn, undrawnW), y).Light().Luminance()
		if math.Abs(here-last) > coveStep {
			t.Fatalf("the cove steps from %v to %v at row %d: a line", last, here, y)
		}
		last = here
	}
}

// topHasGrain checks the table's top is wood: along a row of its face
// there is early wood and late wood, not one colour.
func (rig *studioRig) topHasGrain(t *testing.T) {
	least, most := math.Inf(unity), math.Inf(-unity)
	for x := part(grainFrom, undrawnW); x < part(grainTo, undrawnW); x++ {
		here := rig.shot.At(x, part(grainRow, undrawnH)).Light().Luminance()
		least, most = math.Min(least, here), math.Max(most, here)
	}
	if most-least < grainSpread {
		t.Errorf("the table's top runs from %v to %v along a row: one colour, no grain",
			least, most)
	}
}

// darkWithoutLamps is the anti-test of all three: with its lamps out the
// studio is black all over, so what the others see is the lamps' light
// on the things, and nothing that glows by itself.
func (rig *studioRig) darkWithoutLamps(t *testing.T) {
	stage, _ := rig.made.Scene()
	stage.Lamps = nil
	for i, got := range rig.draw(t, stage).Pix {
		if got.R != nought || got.G != nought || got.B != nought {
			t.Fatalf("pixel %d is %+v with every lamp out", i, got)
		}
	}
}

// Smoke forty-nine, a glass of water on a wooden table in an infinity
// cove: the glass bends what is behind it and the water bends it more,
// the cove has no line where the floor turns up, and the table's top
// has a grain. The anti-tests: a corner far from the glass does not
// change when the glass is taken away; above the water's level nothing
// changes when the water is poured out; and with its lamps out the
// studio is black, so none of it glows.
func TestSmoke49AGlassOfWaterOnAWoodenTableInAnInfinityCove(t *testing.T) {
	made, ok := smoke.Get(smokeFortyNine)
	if !ok {
		t.Fatal("there is no smoke 49")
	}
	stage, eye := made.Scene()
	if len(stage.Things) != studioThings {
		t.Fatalf("smoke 49 has %d things, want a cove, a table of five pieces, "+
			"a glass and its water", len(stage.Things))
	}
	rig := &studioRig{made: made, eye: eye, lens: lensOf(t, smokeFortyNine)}
	rig.shot = rig.draw(t, stage)
	t.Run("the glass bends what is behind it", rig.glassBends)
	t.Run("the water bends it more, below its level and not above", rig.waterBends)
	t.Run("the cove has no line where the floor turns up", rig.coveHasNoLine)
	t.Run("the table's top has a grain", rig.topHasGrain)
	t.Run("with its lamps out the studio is black", rig.darkWithoutLamps)
}
