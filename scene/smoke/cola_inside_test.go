package smoke

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// What fizzes in the glass of cola is where it should be, read off the
// world the smoke builds and not off the sums that placed it: every
// bubble is whole inside the drink, every ball of foam is inside the
// glass and at the drink's top, and every drop is over the drink, over
// the glass's mouth, and no higher than drops are thrown. And there is
// as much of each as the smoke says. The anti-tests: no ball is through
// the glass, no drop is in the drink, and nothing round is in the glass
// that is not a bubble, foam or a drop.
func TestTheColasFizzIsWhereFizzIs(t *testing.T) {
	stage, _ := fiftyTwo()
	foot := maths.V(glassAtX, tableHigh+glassHair, glassAtZ)
	inside := glassRadius - glassWall
	level := glassHigh * colaFull
	seen := map[string]int{}
	for _, each := range stage.Things {
		ball, round := each.Shape.(shape.Sphere)
		if !round {
			continue
		}
		at := each.At(nought).Origin.Sub(foot)
		out, r := math.Hypot(at.X, at.Z), ball.Radius
		name := each.Material.Name
		seen[name]++
		if out+r > inside {
			t.Errorf("a ball of %s %v out and %v across is through the glass",
				name, out, r)
		}
		switch name {
		case "bubble":
			if at.Y-r < glassBase || at.Y+r > level {
				t.Errorf("a bubble at %v, %v across, is not whole in the drink",
					at.Y, r)
			}
		case "foam":
			if math.Abs(at.Y-level) > r {
				t.Errorf("a ball of foam at %v is not at the drink's top, %v",
					at.Y, level)
			}
		case "cola":
			// a drop may fly over the rim; it may not be in the drink
			if at.Y-r < level || at.Y > level+foamLeast+sprayHigh {
				t.Errorf("a drop at %v is not over the drink, or is thrown "+
					"higher than drops are", at.Y)
			}
		default:
			t.Errorf("a ball of %s in the glass of cola", name)
		}
	}
	if seen["bubble"] != rising || seen["foam"] != foam || seen["cola"] != spray {
		t.Errorf("the glass holds %v, want %d bubbles, %d of foam and %d drops",
			seen, rising, foam, spray)
	}
}
