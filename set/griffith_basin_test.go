package set

import "testing"

func TestGriffithBasinReadsPlacedApproximation(t *testing.T) {
	source := []byte(`"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: offshore test
  film: {width: 64, height: 36, rays: 1, exposure: 1}
  cameras:
    view:
      title: view
      orientation: {look_at: [0, 0, 0]}
      location: {at: [0, -5, 2]}
  world:
    griffith-basin:
      at: {east: {value: -1300, unit: m}, north: {value: 346, unit: m}, up: {value: 0, unit: m}}
      heading: {value: 30, unit: deg}
      scale: 0.5
      relief-scale: 0.25
      city-forward: {value: -800, unit: m}
      ridge-forward: {value: 3000, unit: m}
      city-detail: 0.25
      terrain-samples: 257
      city-light-scale: 6
      city-light-footprint: 4
`)
	made, err := Read("griffith-basin.yaml", source)
	if err != nil {
		t.Fatal(err)
	}
	world, has := made.World()
	if !has {
		t.Fatal("world not read")
	}
	basin, has := world.Griffith()
	if !has {
		t.Fatal("Griffith basin not read")
	}
	if at := basin.At(); at.East() != -1300 || at.North() != 346 || at.Up() != 0 {
		t.Errorf("basin position = %#v", at)
	}
	if basin.Heading() != 30 {
		t.Errorf("heading = %g", basin.Heading())
	}
	if basin.Scale() != 0.5 || basin.ReliefScale() != 0.25 {
		t.Errorf("landscape scale/relief = %g/%g", basin.Scale(), basin.ReliefScale())
	}
	if basin.CityForward() != -800 || basin.RidgeForward() != 3000 || basin.CityDetail() != .25 || basin.TerrainSamples() != 257 || basin.LightScale() != 6 || basin.LightFootprint() != 4 {
		t.Errorf("shot detail options = city forward %g, ridge forward %g, density %g, samples %d, light %g, footprint %g", basin.CityForward(), basin.RidgeForward(), basin.CityDetail(), basin.TerrainSamples(), basin.LightScale(), basin.LightFootprint())
	}
}

func TestGriffithBasinReadsSubtleNightLightScale(t *testing.T) {
	for _, value := range []string{"1e-9", "1e-7", "0.001"} {
		source := []byte(`"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: dim city lights
  film: {width: 64, height: 36, rays: 1, exposure: 1}
  cameras:
    view:
      title: view
      orientation: {look_at: [0, 0, 0]}
      location: {at: [0, -5, 2]}
  world:
    griffith-basin:
      at: {east: {value: 0, unit: m}, north: {value: 0, unit: m}, up: {value: 0, unit: m}}
      heading: {value: 0, unit: deg}
      scale: 1
      relief-scale: 1
      city-light-scale: ` + value + `
`)
		if _, err := Read("dim-lights.yaml", source); err != nil {
			t.Errorf("city-light-scale %s was refused: %v", value, err)
		}
	}
}

func TestGriffithBasinRejectsNightLightScaleBelowMinimum(t *testing.T) {
	source := []byte(`"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: below minimum city lights
  film: {width: 64, height: 36, rays: 1, exposure: 1}
  cameras:
    view:
      title: view
      orientation: {look_at: [0, 0, 0]}
      location: {at: [0, -5, 2]}
  world:
    griffith-basin:
      at: {east: {value: 0, unit: m}, north: {value: 0, unit: m}, up: {value: 0, unit: m}}
      heading: {value: 0, unit: deg}
      scale: 1
      relief-scale: 1
      city-light-scale: 1e-10
`)
	if _, err := Read("dim-lights-too-low.yaml", source); err == nil {
		t.Fatal("city-light-scale below 1e-9 was accepted")
	}
}
