//go:build cgo

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/session/placement"
)

func TestPresetsCommandUsesProducerTable(t *testing.T) {
	var output, errors bytes.Buffer
	if status := command().RunWith([]string{"presets"}, strings.NewReader(""), &output, &errors); status != 0 {
		t.Fatalf("presets exited %d: %s", status, errors.String())
	}
	var got []stagewire.FilmPreset
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := job.Presets()
	if len(got) != len(want) {
		t.Fatalf("got %d presets, producer has %d", len(got), len(want))
	}
	for i, preset := range want {
		width, height := preset.Size()
		if got[i] != (stagewire.FilmPreset{Name: preset.Name(), Width: width, Height: height}) {
			t.Errorf("preset %d: got %+v, producer has %s %dx%d", i, got[i], preset.Name(), width, height)
		}
	}
}

func TestBuiltHelperReturnsProducerPresets(t *testing.T) {
	helper := os.Getenv("FERRUCCIO_STAGE_HELPER")
	if helper == "" {
		t.Skip("set FERRUCCIO_STAGE_HELPER to check the Game-built helper")
	}
	got, err := stagewire.ReadFilmPresets(context.Background(), helper)
	if err != nil {
		t.Fatal(err)
	}
	want := job.Presets()
	if len(got) != len(want) {
		t.Fatalf("built helper returned %d presets; producer has %d", len(got), len(want))
	}
	for i, preset := range want {
		width, height := preset.Size()
		if got[i] != (stagewire.FilmPreset{Name: preset.Name(), Width: width, Height: height}) {
			t.Fatalf("built helper preset %d differs from producer", i)
		}
	}
}

const source = `scene:
  film: {width: 32, height: 24, rays: 1, wide: 45, exposure: 1}
  cameras:
    view:
      location: {at: [0, -5, 2]}
      orientation: {look_at: [1, 0, 0]}
  world:
    materials:
      matte: {colour: {srgb_linear: [0.3, 0.4, 0.5]}, source: eye}
    boxes:
      first: {material: matte, middle: [10, 20, 3], size: [6, 4, 6], source: eye}
`

func TestStageCommandReadsBufferAndWritesSnapshot(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	path := filepath.Join(t.TempDir(), "scene.yaml")
	var output, errors bytes.Buffer
	status := command().RunWith([]string{"stage", "--path", path, "--kind", "legacy-set"}, strings.NewReader(source), &output, &errors)
	if status != 0 {
		t.Fatalf("stage exited %d: %s", status, errors.String())
	}
	snapshot, err := stagewire.Read(&output)
	if err != nil {
		t.Fatal(err)
	}
	batch := snapshot.Scene.Batches[0]
	low, high, ok := batch.Mesh.Bounds()
	low, high = batch.Places[0].Out(low), batch.Places[0].Out(high)
	if !ok || low.X != 7 || high.X != 13 {
		t.Fatalf("buffer geometry = %v %v", low, high)
	}
}

func TestRealSceneStagedByBuiltHelper(t *testing.T) {
	path, helper, cache := os.Getenv("FERRUCCIO_STAGE_SCENE"), os.Getenv("FERRUCCIO_STAGE_HELPER"), os.Getenv("FERRUCCIO_STAGE_CACHE")
	if path == "" || helper == "" || cache == "" {
		t.Skip("set FERRUCCIO_STAGE_SCENE, FERRUCCIO_STAGE_HELPER and FERRUCCIO_STAGE_CACHE for a real-scene probe")
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	started := time.Now()
	snapshot, err := stagewire.Run(ctx, helper, path, string(source), "set", cache, os.Getenv("FERRUCCIO_STAGE_SHELF"), "", 2)
	if err != nil {
		t.Fatalf("stage after %s: %v", time.Since(started), err)
	}
	meshes := make(map[*placement.DisplayMesh]bool)
	places := 0
	for _, batch := range snapshot.Scene.Batches {
		meshes[batch.Mesh] = true
		places += len(batch.Places)
	}
	t.Logf("staged in %s: %d meshes, %d places; %s", time.Since(started), len(meshes), places, snapshot.Notice)
}

func TestBuiltHelperRunsAndReturnsSnapshot(t *testing.T) {
	helper := os.Getenv("FERRUCCIO_STAGE_HELPER")
	if helper == "" {
		t.Skip("set FERRUCCIO_STAGE_HELPER to check a Game-built helper process")
	}
	path := filepath.Join(t.TempDir(), "unsaved.yaml")
	snapshot, err := stagewire.Run(context.Background(), helper, path, source, "legacy-set", t.TempDir(), "", "", 2)
	if err != nil {
		t.Fatal(err)
	}
	batch := snapshot.Scene.Batches[0]
	low, high, ok := batch.Mesh.Bounds()
	low, high = batch.Places[0].Out(low), batch.Places[0].Out(high)
	if !ok || low.X != 7 || high.X != 13 {
		t.Fatalf("child geometry = %v %v", low, high)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("child wrote the unsaved buffer")
	}
}

func TestStageCommandRefusesInvalidSourceAndPath(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	for _, test := range []struct{ path, source string }{{"relative.yaml", source}, {filepath.Join(t.TempDir(), "bad.yaml"), "scene: ["}} {
		var output, errors bytes.Buffer
		status := command().RunWith([]string{"stage", "--path", test.path, "--kind", "legacy-set"}, strings.NewReader(test.source), &output, &errors)
		if status == 0 || output.Len() != 0 {
			t.Fatalf("accepted invalid request: %s", errors.String())
		}
	}
}
