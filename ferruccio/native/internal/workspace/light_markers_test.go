//go:build cgo

package workspace

import (
	"context"
	"image"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestStageSnapshotNamesReadOnlyAuthoredLightMarkers(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	source := stagedTestSet + `    lamps:
      porch: {at: [1, 0, 0], temperature: 2700, strength: 0.35, radius: 0.0625, source: eye}
    spots:
      beam: {at: [1, 0, 0], aim: [1, 0, 0], cone: 20, soft: 5, temperature: 3000, strength: 40, source: eye}
    suns:
      morning: {toward: [1, 5, -2], colour: {srgb_linear: [0.35, 0.34, 0.32]}, source: eye}
    floods:
      fill: {colour: {srgb_linear: [0.01, 0.013, 0.025]}, strength: 2, source: eye}
`
	got, err := StageSnapshot(context.Background(), filepath.Join(t.TempDir(), "scene.yaml"), source, "legacy-set")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Scene.Lights) != 3 {
		t.Fatalf("unexpected lights: %+v", got.Scene.Lights)
	}
	for _, light := range got.Scene.Lights {
		collection := map[string]string{"porch": "lamps", "beam": "spots", "morning": "suns"}[light.Author.Name]
		if collection == "" || light.Author.Collection != "scene.world."+collection {
			t.Fatalf("missing authored light identity: %+v", light)
		}
		if light.Kind != "Sun" && light.At != maths.V(1, 0, 0) {
			t.Fatalf("light marker moved away from its staged location: %+v", light)
		}
	}
	if !strings.Contains(got.Notice, "1 light markers") || !strings.Contains(got.Notice, "lighting and shadows are omitted") {
		t.Fatalf("missing light limitations: %s", got.Notice)
	}
	view := newImageView()
	view.lightMarkers = visibleLightMarkers(&got.Scene)
	if len(view.lightMarkers) != 3 {
		t.Fatalf("expected three in-frame light markers: %+v", view.lightMarkers)
	}
	view.set(image.NewRGBA(image.Rect(0, 0, 32, 24)))
	if len(view.lightMarkers) != 0 {
		t.Fatal("photograph inherited working-view light markers")
	}
}
