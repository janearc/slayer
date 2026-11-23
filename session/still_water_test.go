package session

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// stillLevel is the terrain `sea` the fixture says, in metres.
const stillLevel = 5

// seaOverTerrain is the authored ocean set standing on a terrain that
// says a `sea` level, with its ocean taken out when withOcean is false.
func seaOverTerrain(t *testing.T, withOcean bool) set.Set {
	t.Helper()
	text := strings.Replace(authoredOceanText("1"), "  film:",
		"  terrain:\n    from: fixture\n    file: fixture-plain.tif\n"+
			"    west: 1001\n    north: 8999\n    across: 150\n"+
			"    down: 100\n    step: 2\n"+
			"    sea: {value: 5, unit: m}\n  film:", 1)
	if !withOcean {
		start := strings.Index(text, "    ocean:\n")
		end := strings.Index(text, "    materials:\n")
		text = text[:start] + text[end:]
	}
	read, err := set.Read("sea-over-terrain.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return read
}

// A terrain `sea` with no ocean keeps its still-water globe: it names a
// level and no wind, and nothing else draws water there.
func TestTerrainSeaWithNoOceanStillFloods(t *testing.T) {
	elevation, floods := stillWater(seaOverTerrain(t, false))
	if !floods || elevation != stillLevel {
		t.Fatalf("level %v, floods %t, want the globe at %v", elevation,
			floods, float64(stillLevel))
	}
}

// Anti-test: a set with an ocean and a terrain `sea` stages no globe. The
// ocean is its sea, and a flat sea under it is none; the level is still
// the terrain's, for whatever reads it.
func TestOceanAndTerrainSeaStageNoGlobe(t *testing.T) {
	read := seaOverTerrain(t, true)
	if _, floods := stillWater(read); floods {
		t.Fatal("a set with an ocean was given a flat sea beneath it")
	}
	ground, has := read.Ground()
	if !has {
		t.Fatal("the set has no ground")
	}
	if level, said := ground.Sea(); !said || level != stillLevel {
		t.Fatalf("the terrain's level is %v, said %t", level, said)
	}
}
