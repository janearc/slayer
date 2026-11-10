package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/celestial/atmosphere"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/plume"
	"github.com/janearc/slayer/scene/thing"
)

// burningHill is a small hillside at night with one spruce on fire and
// nothing else lit but the moon, exposed for the fire.
func burningHill() []string {
	return append([]string{"-w", "40", "-h", "30", "-n", "1", "-night",
		"-alight", "1", "-exposure", "100000"},
		only(map[string]string{"spruce": "1", "boulder": "1"})...)
}

// -alight sets that many spruces on fire, each a flame's body as wide
// as the tree and half again as tall with its smoke above it, and no
// more spruces than there are; the gpu, which has no flame yet, is
// refused it.
func TestAlightSetsSprucesOnFire(t *testing.T) {
	world, made, err := hillside(mustParse(t, burningHill()...))
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

// a fire lights what is near it: the burning hillside drawn as made,
// and again with the fire's lamps taken out and only the moon's two
// left. the flame glows alike in both, so what the first has more of is
// the fire's light on what is near it.
//
// a behaviour, not a law: the picture's light, the flame's own glow
// in it, comes to more with the fire's light, and the test asks for a
// tenth more. measured 464 against 395 once a matte surface took its
// colour over pi of what lands on it. the bar was a quarter, picked
// from 618 against 400 measured before that, by a tracer pi too
// bright: the fire's added light fell from 218 to 69, and 218 over pi
// is 69.4. a moved bar, listed to be judged.
func TestAFireLightsTheGround(t *testing.T) {
	given := mustParse(t, burningHill()...)
	world, made, err := hillside(given)
	if err != nil {
		t.Fatal(err)
	}
	const moons = 2
	if made.lamps <= moons || len(world.Lamps) != made.lamps {
		t.Fatalf("%d lamps counted, %d in the world", made.lamps,
			len(world.Lamps))
	}
	lit, err := draw(world, given, made.across)
	if err != nil {
		t.Fatal(err)
	}
	world.Lamps = world.Lamps[:moons]
	unlit, err := draw(world, given, made.across)
	if err != nil {
		t.Fatal(err)
	}
	if !(light(lit) > 1.1*light(unlit)) {
		t.Errorf("the hillside is %v with the fire's light and %v "+
			"without it", light(lit), light(unlit))
	}
}

// fermat proves a burning hillside: a pixel through the fire, named, and more
// picked, each made again bit for bit, the named one's page telling the steps
// through the flame and how hot it was at each.
func TestABurningHillsideIsProved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "burning.pfm")
	drawing := append(burningHill(), "-out", filepath.Join(dir, "b.png"),
		"-raw", path)
	var wrong bytes.Buffer
	if code := run(drawing, io.Discard, &wrong); code != 0 {
		t.Fatalf("drawing gave %d: %s", code, wrong.String())
	}
	var said bytes.Buffer
	proving := append(burningHill(), "-prove", path, "-pixel", "20,12",
		"-many", "8")
	if code := run(proving, &said, &wrong); code != 0 ||
		!strings.Contains(said.String(), "holds: 9 of 9") {
		t.Fatalf("gave %d: %s %s", code, said.String(), wrong.String())
	}
	page, err := os.ReadFile(filepath.Join(dir, "burning-proof",
		"pixel-20-12.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), "a step through the flame") ||
		!strings.Contains(string(page), "kelvin=") {
		t.Errorf("the page through the fire tells no flame:\n%s", page)
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
