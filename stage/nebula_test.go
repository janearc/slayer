package stage_test

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies/template"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

// nebula is the nebula's world, built, as written or changed by each
// pair of was and now.
func nebula(t *testing.T, changes ...string) *thing.World {
	t.Helper()
	source, err := os.ReadFile("testdata/nebula.yaml")
	if err != nil {
		t.Fatal(err)
	}
	written := string(source)
	for i := 0; i+1 < len(changes); i += 2 {
		if !strings.Contains(written, changes[i]) {
			t.Fatalf("%q is not in the nebula", changes[i])
		}
		written = strings.Replace(written, changes[i], changes[i+1], 1)
	}
	read, err := set.Read("nebula.yaml", []byte(written))
	if err != nil {
		t.Fatal(err)
	}
	said, _ := read.World()
	world, err := stage.World(said)
	if err != nil {
		t.Fatal(err)
	}
	return world
}

// pulsar is the nebula's pulsar placed alone at the world's middle: its
// template as it is, at the scale given, its ground tiled four deep.
func pulsar(t *testing.T, scale float64) *thing.World {
	t.Helper()
	body, _ := template.Named("pulsar")
	alone := thing.NewWorld()
	if err := body.Place(alone, maths.Vec{}, scale, 4); err != nil {
		t.Fatal(err)
	}
	return alone
}

// the nebula goes into the world as its file writes it: the pulsar's
// parts, then the two spheres; the pulsar's gas, then the nebula; the
// pulsar's lamps, then the spot, then the flood.
func TestAWorldIsBuiltInTheOrderItIsWritten(t *testing.T) {
	body, _ := template.Named("pulsar")
	world, alone := nebula(t), pulsar(t, 0.001/body.Radius)
	if len(world.Things) != len(alone.Things)+2 ||
		len(world.Volumes) != len(alone.Volumes)+1 ||
		len(world.Lamps) != len(alone.Lamps)+2 {
		t.Fatalf("%d things, %d volumes, %d lamps; the pulsar alone has "+
			"%d, %d, %d", len(world.Things), len(world.Volumes),
			len(world.Lamps), len(alone.Things), len(alone.Volumes),
			len(alone.Lamps))
	}
	if len(alone.Things) > 0 &&
		world.Things[0].At(0) != alone.Things[0].At(0) {
		t.Errorf("the pulsar stands at %+v, not %+v", world.Things[0].At(0),
			alone.Things[0].At(0))
	}
	if _, isSpot := world.Lamps[len(world.Lamps)-2].(film.Spot); !isSpot {
		t.Errorf("the spot is not second last: %+v", world.Lamps)
	}
	if _, isFlood := world.Lamps[len(world.Lamps)-1].(film.Flood); !isFlood {
		t.Errorf("the flood is not last: %+v", world.Lamps)
	}
}

// a body the set gives no radius is its true size in the world's unit:
// its kilometres a ten thousandth of a unit each, in units of ten
// thousand kilometres.
func TestABodyIsItsTrueSizeInTheWorldsUnit(t *testing.T) {
	world := nebula(t, "        radius: 0.001\n", "")
	alone := pulsar(t, 1e3/1e7)
	if len(alone.Things) == 0 || len(world.Things) != len(alone.Things)+2 ||
		!reflect.DeepEqual(world.Things[0].Shape, alone.Things[0].Shape) {
		t.Errorf("the pulsar at its true size is %+v, not %+v",
			world.Things[0].Shape, alone.Things[0].Shape)
	}
}

// a sphere carried by a motion runs through its frames from its middle,
// in the material it wears; one that is carried by none stands still at
// its middle.
func TestASphereRidesItsMotion(t *testing.T) {
	world := nebula(t)
	companion := world.Things[len(world.Things)-2]
	run, keyed := companion.Motion.(thing.Keyed)
	percent := 4.1667
	want := thing.Keyed{Length: 7200 * time.Second, Ease: "linear",
		Hold: true, Frames: []thing.Keyframe{
			{Share: 0, At: maths.V(64, 0, 0)},
			{Share: percent / 100, At: maths.V(61.82, 0, 16.56)},
			{Share: 1, At: maths.V(64, 0, 0)}}}
	if !keyed || !reflect.DeepEqual(run, want) ||
		companion.Shape != (shape.Sphere{Radius: 10.5}) ||
		companion.Material.Name != "cool-star" ||
		companion.Material.Glow != swatches.Light(swatches.Tungsten) ||
		companion.Material.Colour != (maths.Light{}) {
		t.Errorf("the companion is %+v on %+v", companion, companion.Motion)
	}
	still := world.Things[len(world.Things)-1]
	if still.At(time.Hour).Origin != maths.V(5, 1, 0) {
		t.Errorf("the still sphere is at %v", still.At(time.Hour).Origin)
	}
}

// a sky of stars is the stars its set says, a wisp the ragged ball of
// gas, a spot libtheme's colour of its black body so many times over,
// its angles in degrees, and a flood a swatch so strong.
func TestTheNebulasPiecesAreWhatTheySay(t *testing.T) {
	world := nebula(t)
	if world.Sky != (sky.Stars{Seed: 29, Share: 0.08, Size: 0.5,
		Bright: 0.6}) {
		t.Errorf("the sky is %+v", world.Sky)
	}
	gas := world.Volumes[len(world.Volumes)-1]
	if gas.Medium != (medium.Wispy{Of: medium.Puff{Radius: 600,
		Thick: 0.002}, Seed: 7, Scale: 200, Amount: 0.6}) ||
		gas.Material.Name != "nebula" ||
		gas.Material.Colour != swatches.Light(swatches.Red) ||
		gas.Motion.At(0).Origin != (maths.Vec{}) {
		t.Errorf("the nebula is %+v", gas)
	}
	wind := world.Lamps[len(world.Lamps)-2]
	if wind != (film.Spot{Aim: maths.V(1, 0, 0),
		Strength: swatches.Light(swatch.Planckian(9000)).Scale(9000),
		Cone:     12 * maths.Degree, Soft: 3 * maths.Degree}) {
		t.Errorf("the wind is %+v", wind)
	}
	glow := world.Lamps[len(world.Lamps)-1]
	if glow != (film.Flood{
		Strength: swatches.Light(swatches.Rust).Scale(0.25)}) {
		t.Errorf("the nebula's light is %+v", glow)
	}
}
