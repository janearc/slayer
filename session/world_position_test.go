package session

import (
	"strconv"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

func positionedSetText(setting, first, second string) string {
	return `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: positions
  setting: ` + setting + `
  light: none
  film: {width: 1, height: 1, rays: 1}
  cameras:
    between:
      title: between
      orientation: {look_at: [0, 0, 0]}
      location: {at: [0, -5, 2]}
  set:
    things:
      near:
        kind: spruce
        seed: 1
` + first + `
      far:
        kind: spruce
        seed: 2
` + second + `
`
}

func readPositionSet(t *testing.T, text string) set.Set {
	t.Helper()
	read, err := set.Read("positions.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return read
}

func testStaging() *staging {
	return &staging{given: asked{set: "positions.yaml"},
		world: thing.NewWorld(), have: map[stocked]kept{},
		places: map[string]maths.Vec{}, radii: map[string]float64{}}
}

func numberString(value float64) string { return strconv.FormatFloat(value, 'f', -1, 64) }

func TestAbsolutePositionRebasesAndNamedCameraUsesStagedPlace(t *testing.T) {
	first := `        position:
          east: {value: 120, unit: m}
          north: {value: 180, unit: m}
          up: {value: 50, unit: m}`
	second := `        position:
          east: {value: 130, unit: m}
          north: {value: 180, unit: m}
          up: {value: 50, unit: m}`
	text := `scene:
  name: positions
  film: {width: 1, height: 1, rays: 1}
  cameras:
    between:
      title: between
      location:
        loc_relative: {object: near}
      orientation:
        point_to: far
  set:
    things:
      near:
        kind: spruce
        seed: 1
` + first + `
      far:
        kind: spruce
        seed: 2
` + second + `
`
	read := readPositionSet(t, text)
	stage := testStaging()
	stage.west, stage.north, stage.lowest = 100, 200, 30
	if err := stage.stand(read); err != nil {
		t.Fatal(err)
	}
	if got, want := stage.places["near"], maths.V(20, 20, 20); got != want {
		t.Fatalf("rebased absolute position = %v, want %v", got, want)
	}
	shot, err := read.Shot("between", func(name string) (maths.Vec, bool) {
		at, has := stage.places[name]
		return at, has
	})
	if err != nil {
		t.Fatal(err)
	}
	if shot.Eye().Origin != maths.V(20, 20, 20) {
		t.Fatalf("camera reference starts at %v, want staged named origin", shot.Eye().Origin)
	}
}

func TestGroundlessEarthAirKeepsAbsoluteManufacturedWorldCoordinates(t *testing.T) {
	first := `        position:
          east: {value: 120, unit: m}
          north: {value: 180, unit: m}
          up: {value: 50, unit: m}`
	second := `        position:
          east: {value: 130, unit: m}
          north: {value: 180, unit: m}
          up: {value: 50, unit: m}`
	read := readPositionSet(t, positionedSetText("ground", first, second))
	stage := testStaging()
	stage.groundlessAir = true
	if err := stage.stand(read); err != nil {
		t.Fatal(err)
	}
	if got, want := stage.places["near"], maths.V(120, 50, -180); got != want {
		t.Fatalf("absolute groundless position = %v, want %v", got, want)
	}
	if got, want := stage.places["far"], maths.V(130, 50, -180); got != want {
		t.Fatalf("second absolute groundless position = %v, want %v", got, want)
	}
}

func TestAbsolutePositionAndSpacePlacementShareTheNamedMetreCoordinate(t *testing.T) {
	first := `        position:
          east: {value: -12, unit: m}
          north: {value: 450, unit: m}
          up: {value: -3, unit: m}`
	second := `        position:
          east: {value: 4, unit: m}
          north: {value: 0, unit: m}
          up: {value: 0, unit: m}`
	read := readPositionSet(t, positionedSetText("space", first, second))
	stage := testStaging()
	if err := stage.afloat(read); err != nil {
		t.Fatal(err)
	}
	want := maths.V(-12, -3, -450)
	if got := stage.places["near"]; got != want {
		t.Fatalf("space placement = %v, want %v", got, want)
	}
	if got := placement.SpaceAt(read.Things()[0]).Origin; got != want {
		t.Fatalf("shared space placement = %v, want %v", got, want)
	}
}

func TestLegacySpaceCoordinatesAndTerrainHeightKeepTheirOldMeaning(t *testing.T) {
	legacy := func(east, north, up float64) string {
		return "        east: " + numberString(east) + "\n" +
			"        north: " + numberString(north) + "\n" +
			"        height: {value: " + numberString(up) + ", unit: m}"
	}
	near := legacy(3, 4, 5)
	far := legacy(-2, 1, 8)
	read := readPositionSet(t, positionedSetText("space",
		near, far))
	stage := testStaging()
	if err := stage.afloat(read); err != nil {
		t.Fatal(err)
	}
	if got, want := stage.places["near"], maths.V(3, 5, -4); got != want {
		t.Fatalf("legacy space place = %v, want %v", got, want)
	}
	if _, has := read.Things()[0].Position(); has {
		t.Fatal("legacy space placement was reinterpreted as absolute position")
	}
}
