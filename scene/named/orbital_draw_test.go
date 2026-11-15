//go:build draw

package named_test

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/scene/named"
)

// the orbital's sizes the tests hold it to, in thousands of km, from
// cmd/orbital as it stood: the outer band's radius, and the near eye's
// height above its floor
const (
	outerRadius = 1840.0
	eyeAbove    = 0.5
)

// a far view of the orbital, its lantern held still, with neon and argon
// in its gas, from further out than the command's
var farOrbital = map[string]any{"far": true, "still": true,
	"neon": fraction / (few * few), "argon": fraction / (few * few),
	"distance": float64(4000), "gas": fraction, "day": "12h"}

// The near view builds: the pair under a far sun, the eye just above the
// outer floor, nothing keeping its own time; the scene says what each
// band's floor holds; and it draws at a tiny size.
func TestTheOrbitalsNearViewBuildsAndDraws(t *testing.T) {
	var said bytes.Buffer
	built, err := mustGet(t, "orbital").Build(nil, &said)
	if err != nil {
		t.Fatal(err)
	}
	if built.Timed() {
		t.Error("the near view keeps time, and has no lantern")
	}
	if above := outerRadius - built.Eye().Origin.Z; math.Abs(
		above-eyeAbove) > closeEnough {
		t.Errorf("the eye is %v above the floor, want %v", above,
			eyeAbove)
	}
	for _, band := range []string{"ring 0:", "ring 1:"} {
		if !strings.Contains(said.String(), band) {
			t.Errorf("the orbital did not say %q: %q", band, said.String())
		}
	}
	drawsTiny(t, "orbital", built)
}

// The far view builds round the lantern, whose beams keep their own time,
// with its eye as far out as asked; it is told a time and draws.
func TestTheOrbitalsFarViewKeepsTime(t *testing.T) {
	built := mustBuild(t, "orbital", farOrbital)
	if !built.Timed() {
		t.Error("the far view does not keep time; its beams would not")
	}
	away := built.Eye().Origin.Len()
	if math.Abs(away-farOrbital["distance"].(float64)) > closeEnough*away {
		t.Errorf("the eye is %v out, want %v", away,
			farOrbital["distance"])
	}
	built.Set(aSpell)
	drawsTiny(t, "orbital", built)
}

// anti-tests: the near view's eye is not where the far view's is, and the
// far view is lit by the lantern alone, with no flood to fill it.
func TestTheOrbitalsViewsAreTheirOwn(t *testing.T) {
	near := mustBuild(t, "orbital", map[string]any{"day": aSpell})
	far := mustBuild(t, "orbital", map[string]any{"far": true})
	if near.Eye().Origin == far.Eye().Origin {
		t.Error("the near and far views share an eye")
	}
	if floods(near) != once || floods(far) != none {
		t.Errorf("the near view has %d floods and the far %d; only the "+
			"near is filled", floods(near), floods(far))
	}
}

// floods is how many of a built scene's lamps are a flood.
func floods(built *named.Built) int {
	count := none
	for _, lamp := range built.World().Lamps {
		if _, flood := lamp.(film.Flood); flood {
			count++
		}
	}
	return count
}
