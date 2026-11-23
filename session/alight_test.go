package session

import (
	"io"
	"strings"
	"testing"

	"github.com/janearc/celestial/atmosphere"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/plume"
	"github.com/janearc/slayer/scene/thing"
)

// the burning hillside's own numbers: so wide and so tall in pixels, so
// many rays along each side of a pixel, and an exposure for the fire
const (
	burningWide, burningTall, burningRays = 40, 30, 1
	burningExposure                       = 100000.0
)

// burningHillAsked is the request burningHill's arguments make, said as
// the fields it fills, for a test that stages the hillside itself. A test
// that runs the command takes the arguments.
func burningHillAsked() asked {
	given := usualAsked()
	given.width, given.height, given.samples = burningWide, burningTall,
		burningRays
	given.night, given.alight, given.exposure = true, 1, burningExposure
	given.counts = onlyCounts(map[string]int{"spruce": 1, "boulder": 1})
	return given
}

// -alight sets that many spruces on fire, each a flame's body as wide
// as the tree and half again as tall with its smoke above it, and no
// more spruces than there are; the gpu, which has no flame yet, is
// refused it.
func TestAlightSetsSprucesOnFire(t *testing.T) {
	given := burningHillAsked()
	world, made, err := hillside(given)
	if err != nil {
		t.Fatal(err)
	}
	bodies, smokes := 0, 0
	for _, volume := range world.Volumes {
		if _, ok := volume.Medium.(*flame.Body); ok && volume.Material.Name == "flame" {
			bodies++
			box := volume.Medium.Bounds()
			if !(box.Max.Y > 0) || !(box.Max.X > 0) {
				t.Errorf("a flame with no size: %v", box)
			}
		}
		if _, ok := volume.Medium.(*plume.Column); ok &&
			volume.Material.Name == "smoke" {
			smokes++
		}
	}
	if bodies != 1 || smokes != 1 || made.burning != 1 ||
		!strings.Contains(made.String(), "1 alight") {
		t.Errorf("%d flames, %d smokes, %d burning: %s", bodies, smokes,
			made.burning, made)
	}
	for _, arguments := range [][]string{{"-spruce", "1", "-alight", "2"},
		{"-alight", "-1"}} {
		_, err := parse(arguments, io.Discard)
		if err == nil || !strings.Contains(err.Error(), "alight") {
			t.Errorf("%v: %v, not refused by -alight", arguments, err)
		}
	}
}

// the smoke over a tree burns in the air at the tree's height above the
// sea: a set's world counts heights from its lowest ground, so a tree 2
// m up on a floor 1100 m above the sea has the standard atmosphere's
// air at 1102 m, warmth and density both, not the air at 2 m.
func TestSmokeBurnsInTheAirAtItsHeight(t *testing.T) {
	world := thing.NewWorld()
	at := maths.Subspace{Origin: maths.V(0, 2, 0)}
	if err := smoke(world, at, 2, 1100, 7); err != nil {
		t.Fatal(err)
	}
	air, err := atmosphere.Standard(measure.Metres(1102))
	if err != nil {
		t.Fatal(err)
	}
	want, err := plume.New(plume.Fire{Heat: treeHeat,
		Convective: treeConvective, Base: 4, Yield: treeSoot,
		Combustion: treeCombustion, Extinction: sootAt550,
		Air:     air.Temperature().InKelvin(),
		Density: air.Density().InKilogramsPerCubicMetre(),
		Top:     smokeTop, Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	made := world.Volumes[len(world.Volumes)-1].Medium.(*plume.Column)
	if got := made.Rise(40); got != want.Rise(40) {
		t.Errorf("at 40 m the smoke is %g k warm, in the air at 1102 m %g",
			got, want.Rise(40))
	}
}
