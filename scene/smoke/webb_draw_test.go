//go:build draw

package smoke_test

import (
	"testing"

	"github.com/janearc/slayer/scene/smoke"
)

// Smoke fifty, and how much of its picture the Larson's lamps must reach.
const (
	smokeFifty  = 50  // defined: the number of Webb lit by the Larson
	lampsReach  = 200 // picked: pixels the Larson's own lamps must change; they light the Larson's hull and Webb's struts, far more than this
	onlyTheSun  = 1   // defined: the sun is one lamp, and the last the smoke adds
	fewestLamps = 3   // defined: the sun, and at least the Larson's cabin and belly lamps

	// picked: ten times the Larson's porthole glow, which is 0.0001 of
	// its running lights' colour (saucer-choices.yaml, porthole-glow)
	// and the only thing in the smoke that shines with no lamp
	glowAtMost = 0.001
	glowReach  = 2000 // picked: more pixels than twelve portholes cover in the picture; Webb's struts lit would be more
)

// Smoke fifty, Webb lit by the Larson: the Larson's lamps are what show
// Webb's cold side. The anti-tests: with only the sun the picture loses
// what they lit; and with every lamp out all that is left is the faint glow of the
// Larson's own porthole glass, far too dim to see: nothing else in the
// smoke shines by itself, and nothing is lit that a lamp does not light.
func TestSmoke50WebbIsLitByTheLarson(t *testing.T) {
	made, ok := smoke.Get(smokeFifty)
	if !ok {
		t.Fatal("there is no smoke 50")
	}
	stage, eye := made.Scene()
	if len(stage.Lamps) < fewestLamps {
		t.Fatalf("smoke 50 has %d lamps, want the sun and the Larson's",
			len(stage.Lamps))
	}
	rig := &studioRig{made: made, eye: eye, lens: lensOf(t, smokeFifty)}
	rig.shot = rig.draw(t, stage)

	t.Run("the Larson's lamps light what the sun does not", func(t *testing.T) {
		sunlit, _ := made.Scene()
		sunlit.Lamps = sunlit.Lamps[len(sunlit.Lamps)-onlyTheSun:]
		without := rig.draw(t, sunlit)
		changed := nought
		for i, got := range rig.shot.Pix {
			if got != without.Pix[i] {
				changed++
			}
		}
		if changed < lampsReach {
			t.Errorf("putting out the Larson's lamps changes %d pixels", changed)
		}
	})
	t.Run("with every lamp out only the portholes' glow is left", func(t *testing.T) {
		dark, _ := made.Scene()
		dark.Lamps = nil
		glowing := nought
		for i, got := range rig.draw(t, dark).Pix {
			if got.R > glowAtMost || got.G > glowAtMost || got.B > glowAtMost {
				t.Fatalf("pixel %d is %+v with every lamp out", i, got)
			}
			if got.R > nought {
				glowing++
			}
		}
		if glowing > glowReach {
			t.Errorf("%d pixels glow with every lamp out: more than portholes",
				glowing)
		}
	})
}
