package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

const groundlessAirScene = `scene:
  name: groundless earth air
  light: sun
  air: earth
  groundless-air:
    coordinate-origin: earth-mean-radius
    reference-altitude: {value: 12, unit: m}
    boundary: earth-standard-land
  sun: {azimuth: 120, altitude: 35}
  film: {width: 1, height: 1, rays: 1}
  cameras:
    view:
      title: view
      orientation: {look_at: [0, 0, 0]}
      location: {at: [0, 12, 50]}
`

func TestGroundlessAirRequiresExplicitEarthModelReference(t *testing.T) {
	for _, schema := range []string{"", set.EcosystemSchema,
		"https://github.com/janearc/slayer/schemas/ecosystem-v2.json"} {
		text := groundlessAirScene
		if schema != "" {
			text = "$schema: " + schema + "\n" + text
		}
		read, err := set.Read("groundless.yaml", []byte(text))
		if err != nil {
			t.Fatalf("schema %q: %v", schema, err)
		}
		altitude, has := read.GroundlessAir()
		if !has || altitude.ReferenceAltitude() != 12 ||
			altitude.Boundary() != "earth-standard-land" ||
			altitude.CoordinateOrigin() != "earth-mean-radius" {
			t.Fatalf("schema %q: groundless air = (%g, %q, %q, %v)", schema,
				altitude.ReferenceAltitude(), altitude.Boundary(),
				altitude.CoordinateOrigin(), has)
		}
	}
}

func TestGroundlessAirRequiresMatchingEarthAir(t *testing.T) {
	for _, replacement := range []struct {
		name, air string
	}{
		{name: "missing air", air: ""},
		{name: "other air", air: "  air: venus\n"},
	} {
		t.Run(replacement.name, func(t *testing.T) {
			text := strings.Replace(groundlessAirScene, "  air: earth\n",
				replacement.air, 1)
			if _, err := set.Read("groundless.yaml", []byte(text)); err == nil ||
				!strings.Contains(err.Error(), "air: earth") {
				t.Fatalf("mismatched air accepted: %v", err)
			}
		})
	}
}

func TestGroundlessAirAllowsExistingMoonlightPathOnlyWithMoon(t *testing.T) {
	text := strings.Replace(groundlessAirScene, "light: sun", "light: moon", 1)
	text = strings.Replace(text, "  sun: {azimuth: 120, altitude: 35}\n",
		"  moon: {azimuth: 56, altitude: 10}\n", 1)
	if _, err := set.Read("groundless-moon.yaml", []byte(text)); err != nil {
		t.Fatalf("explicit groundless Earth Moonlight scene: %v", err)
	}
	withoutMoon := strings.Replace(text, "  moon: {azimuth: 56, altitude: 10}\n", "", 1)
	if _, err := set.Read("groundless-moon.yaml", []byte(withoutMoon)); err == nil ||
		!strings.Contains(err.Error(), "needs moon") {
		t.Fatalf("groundless Moonlight without a Moon was accepted: %v", err)
	}
}

func TestTerrainBackedAirDoesNotUseGroundlessMoonlightException(t *testing.T) {
	text := strings.Replace(groundlessAirScene, "light: sun", "light: moon", 1)
	text = strings.Replace(text, "  sun: {azimuth: 120, altitude: 35}\n",
		"  moon: {azimuth: 56, altitude: 10}\n", 1)
	text = strings.Replace(text, "  groundless-air:\n    coordinate-origin: earth-mean-radius\n    reference-altitude: {value: 12, unit: m}\n    boundary: earth-standard-land\n", "", 1)
	if _, err := set.Read("terrain-backed-moon.yaml", []byte(text)); err == nil ||
		!strings.Contains(err.Error(), "light: sun") {
		t.Fatalf("terrain-backed Earth air accepted the groundless Moonlight route: %v", err)
	}
}

func TestGroundlessAirRejectsUnknownApproximation(t *testing.T) {
	text := strings.Replace(groundlessAirScene, "earth-standard-land",
		"open-ocean", 1)
	if _, err := set.Read("groundless.yaml", []byte(text)); err == nil {
		t.Fatal("unknown groundless boundary approximation was accepted")
	}
}

func TestGroundlessAirAllowsWorldOceanButRefusesWorldSkyAndNonMetreUnits(t *testing.T) {
	world := `  world:
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
`
	text := strings.Replace(groundlessAirScene, "  film:", world+"  film:", 1)
	if _, err := set.Read("groundless-ocean.yaml", []byte(text)); err != nil {
		t.Fatalf("world ocean with groundless air: %v", err)
	}

	withSky := strings.Replace(text, "    materials:", `    sky:
      horizon: {srgb_linear: [0.1, 0.1, 0.1]}
      zenith: {srgb_linear: [0.1, 0.1, 0.1]}
      ground: {srgb_linear: [0.1, 0.1, 0.1]}
      source: "synthetic validation"
    materials:`, 1)
	if _, err := set.Read("groundless-sky.yaml", []byte(withSky)); err == nil ||
		!strings.Contains(err.Error(), "world sky") {
		t.Fatalf("conflicting world sky accepted: %v", err)
	}

	withUnit := strings.Replace(text, "  world:\n", "  world:\n    unit: 2\n", 1)
	if _, err := set.Read("groundless-unit.yaml", []byte(withUnit)); err == nil ||
		!strings.Contains(err.Error(), "world.unit 1") {
		t.Fatalf("non-metre world accepted: %v", err)
	}
}

func TestGroundlessAirRejectsAltitudeAtOrAboveModelTop(t *testing.T) {
	text := strings.Replace(groundlessAirScene, "value: 12, unit: m",
		"value: 86000, unit: m", 1)
	if _, err := set.Read("groundless-top.yaml", []byte(text)); err == nil {
		t.Fatal("reference altitude at the model top was accepted")
	}
}
