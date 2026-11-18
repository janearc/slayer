package stage_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

const wakeFoamSet = `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: wake material units
  film: {width: 1, height: 1, rays: 1, exposure: 1}
  world:
    unit: 2
    materials:
      seawater:
        colour: {srgb_linear: [0.01, 0.04, 0.06]}
        clear: {through: 1, index: 1.333, gloss: 1, rank: 0}
        source: "authored synthetic test water"
    ocean:
      material: seawater
      middle: [10, 20, 0]
      extent: {value: 64, unit: m}
      samples: 9
      sea:
        wind: {value: 0, unit: m/s}
        fetch: {value: 0, unit: m}
        depth: {value: 0, unit: m}
        toward: {value: 0, unit: rad}
      seed: 1
      time: {value: 2, unit: s}
      surface-wake:
        origin: [12, 23, 0]
        heading: {value: 0, unit: rad}
        speed: {value: 6, unit: m/s}
        amplitude: {value: 0.1, unit: m}
        half-width: {value: 5, unit: m}
        length: {value: 48, unit: m}
        wavelength: {value: 16, unit: m}
        add-to-spectral-shading: false
        foam:
          coverage: 0.8
          band: 0.2
          reflectance: [0.9, 0.92, 0.94]
          satin: 0.8
          pattern-wavelength: {value: 3, unit: m}
          width-scale: 0.4
      source: "authored test wave and material"
  cameras:
    view:
      location: {at: [0, -10, 5]}
      orientation: {look_at: [0, 0, 0]}
`

func TestWakeFoamSharesOceanMetresAndSnapshotTime(t *testing.T) {
	read, err := set.Read("foam-units.yaml", []byte(wakeFoamSet))
	if err != nil {
		t.Fatal(err)
	}
	said, _ := read.World()
	world, err := stage.World(said)
	if err != nil {
		t.Fatal(err)
	}
	foam := world.OceanSurface.Boundary.Material.WakeFoam
	if foam == nil || foam.OriginEast != 4 || foam.OriginNorth != 6 ||
		foam.Speed != 6 || foam.HalfWidth != 2 || foam.Length != 48 ||
		foam.Wavelength != 3 || !foam.FrozenTime || foam.Seconds != 2 {
		t.Fatalf("staged foam=%+v; want physical metres and frozen authored time", foam)
	}
}

func TestWakeFoamZeroCoverageKeepsOriginalMaterial(t *testing.T) {
	text := strings.Replace(wakeFoamSet, "coverage: 0.8", "coverage: 0", 1)
	read, err := set.Read("disabled-foam.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	said, _ := read.World()
	world, err := stage.World(said)
	if err != nil {
		t.Fatal(err)
	}
	if material := world.OceanSurface.Boundary.Material; material.WakeFoam != nil || material.Through != 1 {
		t.Fatalf("disabled foam changed material: %+v", material)
	}
}

func TestWakeFoamRefusesUnsupportedMeanPlaneSource(t *testing.T) {
	for _, source := range []string{"direct-sun", "direct-moon"} {
		t.Run(source, func(t *testing.T) {
			text := strings.Replace(wakeFoamSet, "      material: seawater\n",
				"      material: seawater\n      "+source+": mean-plane\n", 1)
			read, err := set.Read("unsupported-source.yaml", []byte(text))
			if err != nil {
				t.Fatal(err)
			}
			said, _ := read.World()
			if _, err := stage.World(said); err == nil ||
				!strings.Contains(err.Error(), "participating source approximations") {
				t.Fatalf("foam with unsampled incoming source coverage: %v", err)
			}
		})
	}
}
