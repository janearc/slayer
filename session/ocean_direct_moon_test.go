package session

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/trace"
)

const directMoonFixture = `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: direct lunar water source
  setting: space
  light: moon
  moon: {azimuth: 230, altitude: 40, disc: false}
  film: {width: 1, height: 1, rays: 1, exposure: 1, wide: 40}
  world:
    unit: 1
    materials:
      seawater:
        colour: {srgb_linear: [0.01, 0.04, 0.06]}
        clear:
          through: 1
          index: 1.333
          gloss: 1
          rank: 0
          extinction-per-metre: [0.03, 0.008, 0.004]
          scatter:
            coefficient-per-metre: [0.005, 0.003, 0.002]
            phase-g: 0
            steps: 4
            max-distance: {value: 80, unit: m}
        source: "authored synthetic test-water coefficients"
    ocean:
      material: seawater
      direct-moon: mean-plane
      diffuse-sky-samples: 8
      middle: [0, 0, 0]
      extent: {value: 16, unit: m}
      samples: 9
      sea:
        wind: {value: 0, unit: m/s}
        fetch: {value: 0, unit: m}
        depth: {value: 0, unit: m}
        toward: {value: 0, unit: rad}
      seed: 1
      time: {value: 0, unit: s}
      flat-mean:
        mean-level: {value: 0, unit: m}
        collar-width: {value: 4, unit: m}
        extends-to-infinity: true
      source: "authored flat test interface"
  cameras:
    underwater:
      location: {at: [0, -10, -10]}
      orientation: {look_at: [0, 0, 0]}
      initial-media: [seawater]
`

func TestOceanDirectMoonStagesItsRealSource(t *testing.T) {
	read, err := set.Read("moon.yaml", []byte(directMoonFixture))
	if err != nil {
		t.Fatal(err)
	}
	drawn := stagedWorld(read, asked{set: "moon.yaml"})
	if drawn.err != nil {
		t.Fatal(drawn.err)
	}
	ocean := drawn.world.OceanSurface
	if ocean == nil || ocean.SunRoute == thing.OceanSunRouteUnspecified ||
		ocean.SunLampIndex != 0 || ocean.FillLampIndex != -1 ||
		len(drawn.world.Lamps) != 1 {
		t.Fatalf("Moon source not tagged: ocean=%+v", ocean)
	}
	if err := trace.ValidateClearState(drawn.world,
		drawn.given.initialMedia); err != nil {
		t.Fatal(err)
	}
}

func TestOceanDirectMoonRefusesWrongLight(t *testing.T) {
	for _, row := range []struct {
		name, text string
		gpu        bool
	}{
		{"wrong light", strings.Replace(directMoonFixture,
			"  light: moon\n", "  light: sun\n", 1), false},
	} {
		t.Run(row.name, func(t *testing.T) {
			read, err := set.Read("moon.yaml", []byte(row.text))
			if err != nil {
				t.Fatal(err)
			}
			if got := stagedWorld(read, asked{set: "moon.yaml",
				onGPU: row.gpu}); got.err == nil {
				t.Fatal("accepted unsupported direct Moon path")
			}
		})
	}
}

func TestOceanDirectMoonRefusesSimultaneousSun(t *testing.T) {
	text := strings.Replace(directMoonFixture,
		"      direct-moon: mean-plane\n",
		"      direct-moon: mean-plane\n      direct-sun: mean-plane\n", 1)
	if _, err := set.Read("both.yaml", []byte(text)); err == nil {
		t.Fatal("accepted conflicting directional source declarations")
	}
}

func TestOceanDirectMoonLeavesExistingMoonStagingUntagged(t *testing.T) {
	text := strings.Replace(directMoonFixture,
		"      direct-moon: mean-plane\n", "", 1)
	read, err := set.Read("moon-default.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	drawn := stagedWorld(read, asked{set: "moon-default.yaml"})
	if drawn.err != nil {
		t.Fatal(drawn.err)
	}
	ocean := drawn.world.OceanSurface
	if ocean.MeanPlaneSource() || ocean.SunLampIndex != -1 ||
		ocean.SunRoute != thing.OceanSunRouteUnspecified {
		t.Fatalf("default Moon acquired a new transport route: %+v", ocean)
	}
}
