//go:build cgo

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
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

func TestStageCommandSelectsNamedCameraWithoutChangingSource(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	path := filepath.Join(t.TempDir(), "scene.yaml")
	scene := strings.Replace(source, "  world:", `    zoomed side:
      location: {at: [12, -5, 2]}
      orientation: {look_at: [1, 0, 0]}
  world:`, 1)
	args := []string{"stage", "--path", path, "--kind", "legacy-set"}
	var output, errors bytes.Buffer
	status := command().RunWith(append(args, "--camera", "zoomed side", "--camera-wide", "31.25"), strings.NewReader(scene), &output, &errors)
	if status != 0 {
		t.Fatalf("named stage exited %d: %s", status, errors.String())
	}
	snapshot, err := stagewire.Read(&output)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Scene.Eye.Origin.X != 12 {
		t.Fatalf("named camera eye = %v", snapshot.Scene.Eye.Origin)
	}
	if got := snapshot.Scene.Lens.Wide * 180 / math.Pi; math.Abs(got-31.25) > 1e-12 {
		t.Fatalf("named working lens = %g degrees", got)
	}
	output.Reset()
	errors.Reset()
	if status := command().RunWith(args, strings.NewReader(scene), &output, &errors); status != 0 {
		t.Fatalf("default stage exited %d: %s", status, errors.String())
	}
	snapshot, err = stagewire.Read(&output)
	if err != nil || snapshot.Scene.Eye.Origin.X != 0 {
		t.Fatalf("default camera eye = %v, %v", snapshot.Scene.Eye.Origin, err)
	}
	output.Reset()
	errors.Reset()
	if status := command().RunWith(append(args, "--camera", "missing"), strings.NewReader(scene), &output, &errors); status == 0 {
		t.Fatal("named stage fell back when the camera was absent")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("staging source buffer wrote to its path: %v", err)
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

func TestStageCommandReportsProducerFilmClock(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	root := t.TempDir()
	scenePath := filepath.Join(root, "scene.yaml")
	if err := os.WriteFile(scenePath, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	const schema = "$schema: https://github.com/janearc/slayer/schemas/job-v1.json\n"
	const set = "scene: {set: scene.yaml}\n"
	const film = "duration: {value: 1, unit: s}\n"
	for _, tc := range []struct {
		name, job, notice string
		want              stagewire.FilmFrame
	}{
		{"film", set + "fps: 4\nduration: {value: 2, unit: s}\nstart: {value: 20.125, unit: s}\nscene-duration: {value: 24, unit: s}\n", "Working view shows film frame 1 of 8 at scene time 20.125s", stagewire.FilmFrame{Number: 1, Count: 8, Time: 20125 * time.Millisecond}},
		{"single-frame film", set + "fps: 1\n" + film, "Working view shows film frame 1 of 1 at scene time 0s", stagewire.FilmFrame{Number: 1, Count: 1}},
		{"inherited named-scene start", "scene: {named: orbital, with: {far: true}}\nrenderer: cpu\nfilm: {width: 24, height: 14, rays: 1}\nfps: 2\n" + film, "Working view shows film frame 1 of 2 at scene time 500ms", stagewire.FilmFrame{Number: 1, Count: 2, Time: 500 * time.Millisecond}},
		{"timed still", set + "at: {value: 20.125, unit: s}\n", "", stagewire.FilmFrame{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output, errors bytes.Buffer
			args := []string{"stage", "--path", filepath.Join(root, "job.yaml"), "--kind", "job"}
			if status := command().RunWith(args, strings.NewReader(schema+tc.job), &output, &errors); status != 0 {
				t.Fatalf("stage exited %d: %s", status, errors.String())
			}
			snapshot, err := stagewire.Read(&output)
			if err != nil {
				t.Fatal(err)
			}
			if tc.notice == "" {
				if snapshot.Frame != nil || strings.Contains(snapshot.Notice, "film frame") {
					t.Fatalf("still inherited a film notice: %q", snapshot.Notice)
				}
				return
			}
			if snapshot.Frame == nil || *snapshot.Frame != tc.want {
				t.Fatalf("frame = %+v, want %+v", snapshot.Frame, tc.want)
			}
			if !strings.HasPrefix(snapshot.Notice, tc.notice+" · ") {
				t.Fatalf("notice = %q, want prefix %q", snapshot.Notice, tc.notice)
			}
		})
	}
}

func TestStageCommandMovesTheWorkingGeometryAtSelectedFrame(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	root := t.TempDir()
	moving := source + `    spheres:
      mover: {material: matte, middle: [0, 0, 0], radius: 1, motion: march, source: eye}
    motions:
      march:
        seconds: 1
        ease: linear
        hold: true
        frames:
          - {percent: 0, at: [0, 0, 0]}
          - {percent: 100, at: [2, 0, 0]}
`
	if err := os.WriteFile(filepath.Join(root, "scene.yaml"), []byte(moving), 0600); err != nil {
		t.Fatal(err)
	}
	const job = "$schema: https://github.com/janearc/slayer/schemas/job-v1.json\nscene: {set: scene.yaml}\n"
	const film = job + "fps: 4\nduration: {value: 1, unit: s}\n"
	path := filepath.Join(root, "job.yaml")
	for _, row := range []struct {
		name, text, frame, refusal string
		number                     int
		x                          float64
	}{
		{"first", film, "1", "", 1, 0},
		{"third", film, "3", "", 3, 1},
		{"zero", film, "0", "positive film job frame number", 0, 0},
		{"too late", film, "5", "outside the producer's 4 frames", 0, 0},
		{"still", job + "at: {value: 0.5, unit: s}\n", "1", "requires a film job", 0, 0},
	} {
		t.Run(row.name, func(t *testing.T) {
			var output, errors bytes.Buffer
			args := []string{"stage", "--path", path, "--kind", "job", "--frame", row.frame}
			status := command().RunWith(args, strings.NewReader(row.text), &output, &errors)
			if row.refusal != "" {
				if status == 0 || output.Len() != 0 || !strings.Contains(errors.String(), row.refusal) {
					t.Fatalf("invalid frame = status %d, %d bytes, %s", status, output.Len(), errors.String())
				}
				return
			}
			if status != 0 {
				t.Fatal(errors.String())
			}
			got, err := stagewire.Read(&output)
			if err != nil {
				t.Fatal(err)
			}
			wantTime := time.Duration(row.number-1) * 250 * time.Millisecond
			if got.Frame == nil || *got.Frame != (stagewire.FilmFrame{Number: row.number, Count: 4, Time: wantTime}) {
				t.Fatalf("frame = %+v, want frame %d at %v", got.Frame, row.number, wantTime)
			}
			for _, batch := range got.Scene.Batches {
				if batch.Authored && batch.Author.Name == "mover" && batch.Places[0].Origin.X == row.x {
					return
				}
			}
			t.Fatalf("moving sphere was not staged at x=%g", row.x)
		})
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("inspection wrote the film job: %v", err)
	}
}

func TestStageCommandKeepsAuthoredStillPath(t *testing.T) {
	t.Setenv("SLAYER_CACHE", t.TempDir())
	root := t.TempDir()
	scenePath, jobPath := filepath.Join(root, "scene.yaml"), filepath.Join(root, "job.yaml")
	if err := os.WriteFile(scenePath, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	authored := `$schema: https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: scene.yaml}
at: {value: 20.125, unit: s}
camera:
  unit: m
  up: {x: 0, y: 1, z: 0}
  look-at: {x: 0, y: 0, z: 0}
  moves:
    - from: {x: 3, y: 2, z: 8}
      to: {x: 1, y: 2, z: 8}
      speed: {value: 2, unit: m/s}
      look: {from: {x: 1, y: 0, z: 0}, to: {x: 2, y: 0, z: 0}}
  zoom: {from: {value: 31.25, unit: deg}, to: {value: 65, unit: deg}, duration: {value: 1, unit: s}}
`
	wanted, err := job.ReadSource(jobPath, []byte(authored))
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := wanted.StillCameraRecipe()
	if err != nil {
		t.Fatal(err)
	}
	setStill, err := job.StillFrom(scenePath, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	borrowed, err := setStill.With(job.StillCamera(recipe))
	if err != nil {
		t.Fatal(err)
	}
	at, said := borrowed.Clock().At()
	lens, _ := borrowed.Camera()
	if unit, _ := lens.Unit(); unit != "m" || !said || at != 20*time.Second+125*time.Millisecond {
		t.Fatalf("borrowed unit and time = %q, %v (%t)", unit, at, said)
	}
	stage := func(path, kind, input string, extra ...string) stagewire.Snapshot {
		t.Helper()
		var output, errors bytes.Buffer
		args := append([]string{"stage", "--path", path, "--kind", kind}, extra...)
		if status := command().RunWith(args, strings.NewReader(input), &output, &errors); status != 0 {
			t.Fatalf("stage %s exited %d: %s", kind, status, errors.String())
		}
		snapshot, err := stagewire.Read(&output)
		if err != nil {
			t.Fatal(err)
		}
		return snapshot
	}
	original := stage(jobPath, "job", authored)
	edited := stage(scenePath, "legacy-set", source, "--still-camera", recipe)
	if edited.Scene.Eye != original.Scene.Eye || edited.Scene.Lens != original.Scene.Lens {
		t.Fatalf("authored and borrowed camera differ: %+v vs %+v", original.Scene.Eye, edited.Scene.Eye)
	}
	if eye := edited.Scene.Eye.Origin; eye.X != 3 || eye.Y != 2 || eye.Z != 8 {
		t.Fatalf("still path did not begin at the first authored place: %v", eye)
	}
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
