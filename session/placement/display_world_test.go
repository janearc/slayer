package placement

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/stage"
)

func compactWorldSource(east, camera int) string {
	return fmt.Sprintf(`$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Compact distant world
  setting: space
  seed: 3
  light: none
  film: {width: 641, height: 361, rays: 1, wide: 60}
  cameras:
    view:
      location: {at: [%d, 0, 0]}
      orientation: {look_at: [%d, 50, 0]}
  set:
    things:
      visitor: {kind: can, seed: 4, east: %d, north: 20}
  world:
    materials:
      first: {colour: {srgb_linear: [0.4, 0.5, 0.6]}, source: eye}
      second: {colour: {srgb_linear: [0.4, 0.5, 0.6]}, source: eye}
    spheres:
      a: {middle: [%d, 30, 0], radius: 0.5, material: first, source: eye}
      b: {middle: [%d, 30, 0], radius: 0.5, material: second, source: eye}
`, camera, camera, east+3, east, east)
}

func TestCompactWorldKeepsLocalVerticesAndWorldPicks(t *testing.T) {
	const farEast = 400000000
	cache := NewDisplayCache()
	open := func(source string) *DisplayScene {
		t.Helper()
		scene, err := cache.Scene(readSelection(t, source))
		if err != nil {
			t.Fatal(err)
		}
		return scene
	}
	near := open(compactWorldSource(0, 0))
	distant := open(compactWorldSource(farEast, farEast))
	localWorld := objectNamed(t, near, "world")
	distantWorld := objectNamed(t, distant, "world")
	if near.SplitWorldVertices || distant.SplitWorldVertices {
		t.Fatal("local or compact world requested a widened vertex layout")
	}
	if localWorld.At != maths.Local() || distantWorld.At.Origin != maths.V(farEast, 0, -30) {
		t.Fatalf("near and distant world frames: %v, %v", localWorld.At, distantWorld.At)
	}
	low, high, ok := distantWorld.Mesh.Bounds()
	if !ok || float32(low.X) != -.5 || float32(high.X) != .5 {
		t.Fatalf("float32 upload would lose sphere width: %v to %v", low, high)
	}
	for _, scene := range []*DisplayScene{near, distant} {
		hit, found := scene.PickAt(.5, .5)
		if !found || hit.ID != "scene.world" || hit.Part != "first" || math.Abs(hit.Distance-29.5) > 1e-9 || scene.Pick(.5, .5) != "" {
			t.Fatalf("changed world occlusion or material order: %+v", hit)
		}
		visitor := objectNamed(t, scene, "visitor")
		low, high, ok := visitor.Mesh.Bounds()
		if !ok {
			t.Fatal("visitor mesh has no bounds")
		}
		candidate := visitor.At.Out(low.Add(high).Scale(.5))
		x, y := scene.project(scene.Eye.In(candidate))
		if got := scene.Pick(x, y); got != visitor.ID {
			t.Fatalf("lost nearby authored pick: %q", got)
		}
		seen := scene.InView()
		if len(seen) != 2 || seen[0].ID != "scene.set.things/visitor" || seen[1].ID != "scene.world" {
			t.Fatalf("lost the authored object or world report: %+v", seen)
		}
	}
	panned := open(compactWorldSource(farEast, farEast+1))
	if objectNamed(t, panned, "world").Mesh != distantWorld.Mesh || panned.Pick(.5, .5) != "" {
		t.Fatal("camera translation rebuilt or made the world selectable")
	}
	if hit, found := panned.PickAt(.5, .5); found {
		t.Fatalf("panned centre still hits the old world: %+v", hit)
	}
	renamed := open(strings.ReplaceAll(compactWorldSource(farEast, farEast), "first", "renamed"))
	if hit, found := renamed.PickAt(.5, .5); !found || hit.Part != "renamed" || objectNamed(t, renamed, "world").Mesh == distantWorld.Mesh {
		t.Fatalf("cache retained old materials after source rename: %+v", hit)
	}
}

func TestDispersedWorldRetainsItsAggregateFrame(t *testing.T) {
	source := strings.Replace(compactWorldSource(400000000, 400000000), "b: {middle: [400000000,", "b: {middle: [0,", 1)
	scene, err := NewDisplayCache().Scene(readSelection(t, source))
	if err != nil {
		t.Fatal(err)
	}
	if !scene.SplitWorldVertices {
		t.Fatal("dispersed world did not advertise its baked coordinates")
	}
	if world := objectNamed(t, scene, "world"); world.At != maths.Local() {
		t.Fatalf("dispersed world used a local anchor: %v", world.At)
	}
}

func channelWorldSource(east, camera int) string {
	return fmt.Sprintf(`$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Working channel precision
  setting: space
  seed: 3
  light: none
  film: {width: 641, height: 361, rays: 1, wide: 60}
  cameras:
    view:
      location: {at: [%d, -10, 5]}
      orientation: {look_at: [%d, 12, 0]}
  world:
    materials:
      brick: {colour: {srgb_linear: [0.3, 0.1, 0.05]}, source: eye}
      stone: {colour: {srgb_linear: [0.4, 0.4, 0.4]}, source: eye}
      paving: {colour: {srgb_linear: [0.1, 0.1, 0.1]}, source: eye}
      silt: {colour: {srgb_linear: [0.02, 0.02, 0.01]}, source: eye}
    channels:
      canal:
        line: [[%d, 0], [%d, 20]]
        width: {value: 12, unit: m}
        depth: {value: 3, unit: m}
        top: {value: 0.5, unit: m}
        reach: {value: 20, unit: m}
        ground: paving
        bed: silt
        wall: {material: brick, thickness: {value: 0.5, unit: m}}
        coping:
          material: stone
          height: {value: 0.2, unit: m}
          overhang: {value: 0.1, unit: m}
        source: eye
`, camera, camera, east, east)
}

func TestCompactWorldRebasesBakedProducerChannel(t *testing.T) {
	const east = 400000000
	read := readSelection(t, channelWorldSource(east, east))
	world, _ := read.World()
	producer := thing.NewWorld()
	if err := stage.Add(producer, world, maths.Vec{}); err != nil {
		t.Fatal(err)
	}
	if len(producer.Things) == 0 {
		t.Fatal("producer channel made no surfaces")
	}
	part := producer.Things[0]
	bounds := part.Shape.Bounds()
	if part.At(0) != maths.Local() || bounds.Min.X < east-40 || bounds.Max.X <= bounds.Min.X ||
		float32(bounds.Max.X) != float32(bounds.Min.X) {
		t.Fatalf("expected world coordinates baked into a producer mesh: %v, %+v", part.At(0), bounds)
	}

	cache := NewDisplayCache()
	control, err := cache.Scene(readSelection(t, channelWorldSource(0, 0)))
	if err != nil {
		t.Fatal(err)
	}
	distant, err := cache.Scene(read)
	if err != nil {
		t.Fatal(err)
	}
	reference := objectNamed(t, control, "world")
	anchored := objectNamed(t, distant, "world")
	if reference.At != maths.Local() || anchored.At.Origin.X <= DisplayFar || len(distant.Objects) != 1 {
		t.Fatalf("small world or baked aggregate changed representation: %v, %v", reference.At, anchored.At)
	}
	for i, vertex := range anchored.Mesh.Vertices {
		if worldCoordinateMagnitude(vertex.Position) > 40 {
			t.Fatalf("vertex %d has unanchored coordinates: %v", i, vertex.Position)
		}
	}
	materials := make(map[string]bool)
	for _, x := range []float64{.20, .35, .50, .65, .80} {
		for _, y := range []float64{.35, .50, .65, .80} {
			first, hit := control.PickAt(x, y)
			second, hitFar := distant.PickAt(x, y)
			if hit != hitFar {
				t.Fatalf("channel hit changed at %g,%g", x, y)
			}
			if !hit {
				continue
			}
			if first.ID != "scene.world" || first.ID != second.ID || first.Part != second.Part ||
				math.Abs(first.Distance-second.Distance) > 1e-6 {
				t.Fatalf("channel occlusion changed: %+v vs %+v", first, second)
			}
			if selected := distant.Pick(x, y); selected != "" {
				t.Fatalf("world blocker became editable: %q", selected)
			}
			materials[first.Part] = true
		}
	}
	if len(materials) < 2 {
		t.Fatalf("expected rays through multiple channel materials: %v", materials)
	}
	panned, err := cache.Scene(readSelection(t, channelWorldSource(east, east+1)))
	if err != nil {
		t.Fatal(err)
	}
	reused := objectNamed(t, panned, "world")
	if reused.Mesh != anchored.Mesh || reused.At != anchored.At {
		t.Fatal("moving the camera rebuilt the anchored channel mesh")
	}
}
