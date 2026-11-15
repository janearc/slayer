//go:build draw

package smoke_test

import (
	"testing"

	"github.com/janearc/slayer/scene/smoke"
)

// Smoke fifty-one and the things in it, in the order its builder adds
// them: the cove, the table's top and four legs, the ball of water, and
// its seven bubbles.
const (
	smokeFiftyOne = 51  // defined: the number of a ball of water with bubbles in it
	ballThings    = 7   // defined: the cove, the table's five pieces and the ball
	ballAndAir    = 14  // defined: and seven bubbles
	bubblesChange = 300 // picked: pixels the seven bubbles must change; each covers more than a hundred
)

// Smoke fifty-one, a ball of water with bubbles in it: the bubbles are
// seen through the water, and nothing in the studio glows.
func TestSmoke51ABallOfWaterWithBubblesInIt(t *testing.T) {
	made, ok := smoke.Get(smokeFiftyOne)
	if !ok {
		t.Fatal("there is no smoke 51")
	}
	stage, eye := made.Scene()
	if len(stage.Things) != ballAndAir {
		t.Fatalf("smoke 51 has %d things, want a cove, a table of five "+
			"pieces, a ball of water and seven bubbles", len(stage.Things))
	}
	rig := &studioRig{made: made, eye: eye, lens: lensOf(t, smokeFiftyOne)}
	rig.shot = rig.draw(t, stage)

	t.Run("the bubbles show through the water", func(t *testing.T) {
		still, _ := made.Scene()
		still.Things = still.Things[:ballThings]
		without := rig.draw(t, still)
		changed := nought
		for i, got := range rig.shot.Pix {
			if got != without.Pix[i] {
				changed++
			}
		}
		if changed < bubblesChange {
			t.Errorf("taking the bubbles out of the water changes %d pixels",
				changed)
		}
		// the anti-test: the far corner is cove, and no bubble reaches it
		if rig.shot.At(nought, nought) != without.At(nought, nought) {
			t.Errorf("taking the bubbles out changes the far corner: %+v, %+v",
				rig.shot.At(nought, nought), without.At(nought, nought))
		}
	})
	t.Run("the water bends what is behind it", func(t *testing.T) {
		dry, _ := made.Scene()
		dry.Things = dry.Things[:ballThings-unity]
		without := rig.draw(t, dry)
		middle := rig.shot.At(undrawnW/pair, undrawnH/pair)
		if middle == without.At(undrawnW/pair, undrawnH/pair) {
			t.Errorf("taking the ball away changes nothing at the picture's "+
				"middle: %+v", middle)
		}
	})
	t.Run("with its lamps out the studio is black", rig.darkWithoutLamps)
}
