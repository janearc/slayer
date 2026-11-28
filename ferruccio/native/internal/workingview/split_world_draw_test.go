//go:build draw && darwin

package workingview

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

func TestDispersedWorldMetalPreservesOrderedGeometry(t *testing.T) {
	view, _ := openView(t)
	cache := placement.NewDisplayCache()
	const farEast = 400000000
	const width, height = 321, 201
	sceneAt := func(east, camera int) *placement.DisplayScene {
		t.Helper()
		source := fmt.Sprintf(`$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Dispersed world precision
  setting: space
  seed: 3
  light: none
  film: {width: 321, height: 201, rays: 1, wide: 60}
  cameras:
    view:
      location: {at: [%d, 2, 0]}
      orientation: {look_at: [%d, 52, 0]}
  set:
    things:
      visitor: {kind: yurt, seed: 4, east: %d, north: 22}
  world:
    materials:
      first: {colour: {srgb_linear: [0.8, 0.6, 0.4]}, source: eye}
      second: {colour: {srgb_linear: [0.2, 0.7, 0.4]}, source: eye}
    boxes:
      marker: {middle: [%d.25, 31, 0], size: [2.5, 2.5, 2.5], material: first, source: eye}
      tied: {middle: [%d.25, 31, 0], size: [2.5, 2.5, 2.5], material: second, source: eye}
`, camera, camera, east+7, east, east)
		if east != 0 {
			source += "      far: {middle: [0, 31, 0], size: [2.5, 2.5, 2.5], material: first, source: eye}\n"
		}
		read, err := set.Read("split-world.yaml", []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		scene, err := cache.Scene(read)
		if err != nil {
			t.Fatal(err)
		}
		return scene
	}
	draw := func(scene *placement.DisplayScene) []byte {
		t.Helper()
		frame, err := drawUntilReady(view, scene, width, height)
		if err != nil {
			t.Fatal(err)
		}
		return bytes.Clone(frame.Pix)
	}
	local, distant := sceneAt(0, 0), sceneAt(farEast, farEast)
	if local.SplitWorldVertices || !distant.SplitWorldVertices {
		t.Fatal("producer fixtures did not choose narrow and split vertices")
	}
	source := distant.Objects[len(distant.Objects)-1].Mesh
	for _, mode := range []string{"solid", "edges", "wire"} {
		if err := view.SetMode(mode); err != nil {
			t.Fatal(err)
		}
		normal, split := draw(local), draw(distant)
		if oneColour(normal) || !bytes.Equal(normal, split) {
			t.Fatalf("%s changed ordered world or foreground pixels", mode)
		}
		if resident := view.meshes[source]; resident == nil || !resident.split {
			t.Fatalf("%s did not exercise the split pipeline", mode)
		}
	}
	for _, scene := range []*placement.DisplayScene{local, distant} {
		if hit, ok := scene.PickAt(.5, .5); !ok || hit.ID != "scene.world" || hit.Part != "first" || scene.Pick(.5, .5) != "" {
			t.Fatalf("changed world blocking pick or exact material tie: %+v", hit)
		}
	}
	previous, stats := draw(distant), view.Stats()
	panned := sceneAt(farEast, farEast+1)
	after := draw(panned)
	if view.Stats().Uploads != stats.Uploads || bytes.Equal(previous, after) {
		t.Fatal("panning uploaded geometry or returned stale pixels")
	}
	if err := view.DiscardScene(); err != nil {
		t.Fatal(err)
	}
	if fresh := draw(panned); !bytes.Equal(after, fresh) {
		t.Fatal("panning with the split resident differed from a fresh upload")
	}
	if err := view.SetMode("solid"); err != nil {
		t.Fatal(err)
	}
	solid := draw(panned)
	if err := view.DiscardScene(); err != nil {
		t.Fatal(err)
	}
	view.split.release()
	fallback := draw(panned)
	if !view.Stats().SplitWorldFallback {
		t.Fatal("missing split pipelines did not report reduced precision")
	}
	painted := 0
	for y := 2; y < height; y += 4 {
		for x := 2; x < width; x += 4 {
			hit, ok := panned.PickAt((float64(x)+.5)/width, (float64(y)+.5)/height)
			if !ok || hit.ID != "scene.set.things/visitor" {
				continue
			}
			offset := (y*width + x) * 4
			pixel := solid[offset : offset+4]
			if bytes.Equal(pixel, solid[:4]) {
				continue
			}
			painted++
			if !bytes.Equal(pixel, fallback[offset:offset+4]) {
				t.Fatalf("split fallback changed the nearby visitor at pixel %d,%d", x, y)
			}
		}
	}
	if painted == 0 {
		t.Fatal("fixture found no painted foreground visitor pixels")
	}
	t.Logf("split fallback preserved %d painted foreground samples", painted)
	draw(local)
	if view.Stats().SplitWorldFallback {
		t.Fatal("returning to a local world left its precision fallback set")
	}
}
