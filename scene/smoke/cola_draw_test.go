//go:build draw

package smoke_test

import (
	"testing"

	"github.com/janearc/slayer/scene/smoke"
)

// Smoke fifty-two and the things in it, in the order its builder adds
// them: the cove, the table's five pieces, the glass and the drink, and
// then everything that fizzes.
const (
	smokeFiftyTwo = 52  // defined: the number of a glass of cola
	colaStill     = 8   // defined: the cove, the table's five pieces, the glass and the drink
	fizzChanges   = 400 // picked: pixels the fizz must change; the foam alone covers more
	drinkChanges  = 400 // picked: pixels the drink must change in an empty glass
)

// differing is how many pixels two pictures of one size differ in.
func (rig *studioRig) differing(t *testing.T, things int) int {
	t.Helper()
	fewer, _ := rig.made.Scene()
	fewer.Things = fewer.Things[:things]
	without := rig.draw(t, fewer)
	changed := nought
	for i, got := range rig.shot.Pix {
		if got != without.Pix[i] {
			changed++
		}
	}
	return changed
}

// Smoke fifty-two, a glass of cola: the drink is in the glass and the
// fizz is in and on the drink. The anti-tests: the picture with the
// fizz taken out, and the one with the cola poured out, are each
// another picture; and with its lamps out the studio is black, so
// nothing in it glows.
func TestSmoke52AGlassOfCola(t *testing.T) {
	made, ok := smoke.Get(smokeFiftyTwo)
	if !ok {
		t.Fatal("there is no smoke 52")
	}
	stage, eye := made.Scene()
	if len(stage.Things) <= colaStill {
		t.Fatalf("smoke 52 has %d things: a glass of cola with nothing "+
			"fizzing in it", len(stage.Things))
	}
	rig := &studioRig{made: made, eye: eye, lens: lensOf(t, smokeFiftyTwo)}
	rig.shot = rig.draw(t, stage)

	t.Run("the fizz shows", func(t *testing.T) {
		if changed := rig.differing(t, colaStill); changed < fizzChanges {
			t.Errorf("taking the fizz out of the cola changes %d pixels", changed)
		}
	})
	t.Run("the drink shows", func(t *testing.T) {
		// against the still drink, so the fizz is out of both
		still, _ := made.Scene()
		still.Things = still.Things[:colaStill]
		full := rig.draw(t, still)
		still.Things = still.Things[:colaStill-unity]
		empty := rig.draw(t, still)
		changed := nought
		for i, got := range full.Pix {
			if got != empty.Pix[i] {
				changed++
			}
		}
		if changed < drinkChanges {
			t.Errorf("pouring the cola out changes %d pixels", changed)
		}
	})
	t.Run("with its lamps out the studio is black", rig.darkWithoutLamps)
}
