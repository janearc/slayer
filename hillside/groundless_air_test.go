package hillside

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/moonlight"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/sources"
)

const groundlessAirFixture = `scene:
  name: groundless
  light: sun
  air: earth
  groundless-air:
    coordinate-origin: earth-mean-radius
    reference-altitude: {value: 12, unit: m}
    boundary: earth-standard-land
  sun: {azimuth: 120, altitude: 35}
  world:
    materials:
      interface:
        clear: {through: 1, index: 1.333, gloss: 1, rank: 0}
        source: "synthetic validation"
    ocean:
      material: interface
      middle: [0, 0, 0]
      extent: {value: 20, unit: m}
      samples: 3
      sea:
        wind: {value: 2, unit: m/s}
        fetch: {value: 100, unit: m}
        depth: {value: 30, unit: m}
        toward: {value: 0, unit: rad}
      seed: 1
      time: {value: 0, unit: s}
      source: "synthetic validation"
  film: {width: 1, height: 1, rays: 1}
  cameras:
    view:
      title: view
      orientation: {look_at: [0, 0, 0]}
      location: {at: [0, 12, 50]}
  set:
    things:
      target:
        kind: spruce
        seed: 1
        position:
          east: {value: 10, unit: m}
          north: {value: 20, unit: m}
          up: {value: 12, unit: m}
`

func groundlessAirRead(t *testing.T, text string) set.Set {
	t.Helper()
	read, err := set.Read("groundless-air.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return read
}

func TestGroundlessAirRefusesTerrainRelativeManufacturedThings(t *testing.T) {
	text := strings.Replace(groundlessAirFixture,
		"        position:\n          east: {value: 10, unit: m}\n          north: {value: 20, unit: m}\n          up: {value: 12, unit: m}\n", "        east: 10\n        north: 20\n", 1)
	read := groundlessAirRead(t, text)
	stage := testStaging()
	if err := stage.placeGroundlessAir(read, mustGroundlessAir(t, read)); err == nil || !strings.Contains(err.Error(), "needs an absolute position") {
		t.Fatalf("terrain-relative thing was accepted: %v", err)
	}
}

func TestGroundlessAirAllowsWorldOceanWithoutManufacturedThings(t *testing.T) {
	text := groundlessAirFixture[:strings.Index(groundlessAirFixture, "  set:\n")]
	read := groundlessAirRead(t, text)
	t.Setenv(sources.Placed, t.TempDir())
	stage := testStaging()
	stage.given.set = "aired.yaml"
	if worldOnly(read) {
		t.Fatal("explicit groundless air was routed through world-only staging")
	}
	err := stage.placeGroundlessAir(read, mustGroundlessAir(t, read))
	if err == nil || strings.Contains(err.Error(), "absolute position") {
		t.Fatalf("no-thing ocean was refused before pinned-shelf validation: %v", err)
	}
	if !stage.groundlessAir || stage.airAltitude != 12 {
		t.Fatalf("no-thing scene did not select explicit air frame: %+v", stage)
	}
	if err := stage.halved(read); err != nil {
		t.Fatalf("no-thing world ocean on groundless set: %v", err)
	}
	if len(stage.world.Things) == 0 {
		t.Fatal("no-thing authored ocean did not add its interface")
	}
}

func TestGroundlessEarthAirStagesAbsoluteThingAndWorldOcean(t *testing.T) {
	if os.Getenv(sources.Placed) == "" {
		t.Skip("no shelf named in SLAYER_SHELF")
	}
	read := groundlessAirRead(t, groundlessAirFixture)
	stage := testStaging()
	stage.given.set = "../groundless-air.yaml"
	if err := stage.place(read, set.Planet{}, false); err != nil {
		t.Fatal(err)
	}
	if got, want := stage.places["target"], maths.V(10, 12, -20); got != want {
		t.Fatalf("staged absolute thing position = %v, want %v", got, want)
	}
	if stage.airAltitude != 12 || !stage.groundlessAir || stage.lowest != 0 {
		t.Fatalf("air frame: altitude %g, groundless %v, terrain offset %g",
			stage.airAltitude, stage.groundlessAir, stage.lowest)
	}
	if err := stage.halved(read); err != nil {
		t.Fatalf("groundless world ocean: %v", err)
	}
	if len(stage.world.Things) < 2 {
		t.Fatalf("staged %d world things; want manufactured thing and ocean interface",
			len(stage.world.Things))
	}
}

func TestGroundlessEarthAirStagesPhysicalMoonlightWithoutTerrain(t *testing.T) {
	text := strings.Replace(groundlessAirFixture, "light: sun", "light: moon", 1)
	text = strings.Replace(text, "  sun: {azimuth: 120, altitude: 35}\n",
		"  moon: {azimuth: 56, altitude: 10}\n", 1)
	read := groundlessAirRead(t, text)
	t.Setenv(sources.Placed, t.TempDir())
	stage := testStaging()
	stage.given.set = "groundless-moon.yaml"
	stage.given.exposure = 1
	stage.given.moonScale = 1
	stage.given.moonGlow = 1
	if err := stage.place(read, set.Planet{}, false); err != nil {
		t.Fatalf("stage explicit Moonlight set: %v", err)
	}
	if !stage.groundlessAir || stage.airAltitude != 12 || stage.lowest != 0 {
		t.Fatalf("Moonlight air frame: altitude %g, groundless %v, lowest %g",
			stage.airAltitude, stage.groundlessAir, stage.lowest)
	}
	if len(stage.world.Lamps) != 1 {
		t.Fatalf("Moonlight lamps = %d, want one lunar distant source", len(stage.world.Lamps))
	}
	if _, ok := stage.world.Lamps[0].(emit.Distant); !ok {
		t.Fatalf("Moonlight source has type %T, want emit.Distant", stage.world.Lamps[0])
	}
	if skies, ok := stage.world.Sky.(moonlight.Skies); !ok || len(skies) != 1 {
		t.Fatalf("Moonlight sky = %T (%v), want one physical Moonlight sky", stage.world.Sky, ok)
	}
	if err := stage.halved(read); err != nil {
		t.Fatalf("stage Moonlight world ocean: %v", err)
	}
	if stage.world.OceanSurface == nil {
		t.Fatal("Moonlight scene lost its ocean interface")
	}
}

func TestGroundlessAirRequiresPinnedSunSpectrum(t *testing.T) {
	t.Setenv(sources.Placed, t.TempDir())
	stage := &staging{given: asked{set: "aired.yaml"}}
	if _, _, _, err := stage.airFields("earth"); err == nil ||
		!strings.Contains(err.Error(), "shelf") {
		t.Fatalf("Earth atmosphere without pinned shelf spectrum: %v", err)
	}
}

func mustGroundlessAir(t *testing.T, read set.Set) set.GroundlessAir {
	t.Helper()
	air, has := read.GroundlessAir()
	if !has {
		t.Fatal("fixture lost its groundless-air reference")
	}
	return air
}
