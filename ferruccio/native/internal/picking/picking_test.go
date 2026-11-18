package picking

import (
	"math"
	"os"
	"strings"
	"testing"
	"time"
)

const specimen = `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Native picking
  seed: 19
  setting: space
  light: none
  set:
    things:
      visitor:
        kind: human-scale-2m
        seed: 7
        complexity: quick
        east: 0
        north: 0
        height: {value: 0, unit: m}
        heading: {value: 0, unit: deg}
  film: {width: 640, height: 400, wide: 40, rays: 1, exposure: 1}
  cameras:
    view:
      location: {at: [0, -6, 1.3]}
      orientation: {look_at: [0, 0, 1.3]}
`

func TestPickReturnsStableDraftIdentityFromActualSurface(t *testing.T) {
	scene, err := New(specimen, 1.6)
	if err != nil {
		t.Fatal(err)
	}
	if got := scene.Pick(.5, .5); got != "scene.set.things/visitor" {
		t.Fatalf("picked %q", got)
	}
}

func TestPickMissesOutsideHumanAndInvalidCoordinates(t *testing.T) {
	scene, err := New(specimen, 1.6)
	if err != nil {
		t.Fatal(err)
	}
	for _, point := range [][2]float64{{0, 0}, {1, 1}, {-.01, .5}, {1.01, .5}, {.5, math.NaN()}, {math.Inf(1), .5}} {
		if got := scene.Pick(point[0], point[1]); got != "" {
			t.Fatalf("false pick %q at %v", got, point)
		}
	}
}

func TestDragUsesObjectHorizontalPlaneAndMetres(t *testing.T) {
	scene, err := New(specimen, 1.6)
	if err != nil {
		t.Fatal(err)
	}
	east, north, err := scene.Drag("scene.set.things/visitor", .5, .65, .55, .65)
	if err != nil || !(east > 0) || math.Abs(north) > 1e-12 {
		t.Fatalf("drag = %g,%g,%v", east, north, err)
	}
	backEast, backNorth, err := scene.Drag("scene.set.things/visitor", .55, .65, .5, .65)
	if err != nil || math.Abs(backEast+east) > 1e-12 || math.Abs(backNorth+north) > 1e-12 {
		t.Fatal("drag not reversible")
	}
	east, north, err = scene.Drag("scene.set.things/visitor", .55, .65, .55, .65)
	if err != nil || east != 0 || north != 0 {
		t.Fatal("zero gesture moved object")
	}
}

func TestDragRefusesBehindCameraUnknownAndInvalidCoordinates(t *testing.T) {
	scene, err := New(specimen, 1.6)
	if err != nil {
		t.Fatal(err)
	}
	for _, gesture := range []struct {
		id             string
		x0, y0, x1, y1 float64
	}{
		{"unknown", .5, .65, .6, .65},
		{"scene.set.things/visitor", .5, .3, .6, .3},
		{"scene.set.things/visitor", math.NaN(), .65, .6, .65},
		{"scene.set.things/visitor", .5, .65, 2, .65},
	} {
		if _, _, err := scene.Drag(gesture.id, gesture.x0, gesture.y0, gesture.x1, gesture.y1); err == nil {
			t.Fatal("accepted invalid drag", gesture)
		}
	}
}

func TestNewRefusesInvalidSourceAspectAndUnsupportedPose(t *testing.T) {
	for _, aspect := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := New(specimen, aspect); err == nil {
			t.Fatal("accepted invalid aspect")
		}
	}
	for _, source := range []string{"scene: [unfinished", strings.Replace(specimen, "setting: space", "setting: ground", 1), strings.Replace(specimen, "wide: 40, ", "", 1), strings.Replace(specimen, "location: {at: [0, -6, 1.3]}", "location: {at: [0, 0, 1.3]}", 1)} {
		if _, err := New(source, 1.6); err == nil {
			t.Fatal("accepted invalid scene")
		}
	}
}

func TestDragRefusesExactlyParallelRayAndUnavailableScene(t *testing.T) {
	scene, err := New(strings.Replace(specimen, "height: 400", "height: 401", 1), 640.0/401.0)
	if err != nil {
		t.Fatal(err)
	}
	// An odd-height film has one native pixel exactly on its horizontal axis.
	parallelY := 0.5
	if _, _, err := scene.Drag("scene.set.things/visitor", .5, parallelY, .6, parallelY); err == nil {
		t.Fatal("parallel drag accepted")
	}
	var absent *Scene
	if absent.Pick(.5, .5) != "" {
		t.Fatal("absent scene selected")
	}
	if _, _, err := absent.Drag("any", .5, .6, .6, .6); err == nil {
		t.Fatal("absent scene dragged")
	}
}

func TestActualPlaygroundSelectsCenterAndDragsInMetres(t *testing.T) {
	source, err := os.ReadFile("../../../../sets/ferruccio-playground.yaml")
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	scene, err := New(string(source), 1.6)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("native playground selection construction: %s", time.Since(started))
	id := scene.Pick(.5, .5)
	if id != "scene.set.things/larson" {
		t.Fatalf("actual playground center picked %q", id)
	}
	east, north, err := scene.Drag(id, .5, .5, .55, .5)
	if err != nil || east <= 0 || !finite(north) {
		t.Fatalf("actual playground drag = %g,%g,%v", east, north, err)
	}
}
