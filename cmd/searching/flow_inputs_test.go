package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/improv/searching"
)

const copiedFlowGeometry = `paths:
- points: [[10, 0, 0], [9, 0, 0], [6, 0, 0]]
  width: 0.02
  delay-beats: 0.25
landmarks:
- at: [8, 2, -4]
  radius: 0.1
  rgb: [1, 0.5, 2]
clouds:
- at: [4, 5, 6]
  rgb: [0.1, 0.2, 0.3]
  radius: 2
  density: 0.04
  seed: 71
`

func shootWithFlowReference(t *testing.T) (string, []byte, string) {
	t.Helper()
	path := shoot(t, wavBytes(8000, 1))
	directory := filepath.Dir(path)
	geometry := []byte(copiedFlowGeometry)
	if err := os.WriteFile(filepath.Join(directory, "flow-geometry.yaml"), geometry, 0o644); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	plan := strings.ReplaceAll(string(raw), "music: song.wav", "music: "+filepath.Join(directory, "song.wav"))
	plan = strings.ReplaceAll(plan, "score: score.yaml", "score: "+filepath.Join(directory, "score.yaml"))
	plan += fmt.Sprintf(`flow-study:
  provenance: authored-previs
  start: "2.1"
  fold: "2.2"
  answer: "2.3"
  eye: [0, 0, 10]
  wide-degrees: 60
  return-path: 0
  geometry:
    file: flow-geometry.yaml
    sha256: %x
`, sha256.Sum256(geometry))
	if err := os.WriteFile(path, []byte(plan), 0o644); err != nil {
		t.Fatal(err)
	}
	return path, geometry, fmt.Sprintf("%x", sha256.Sum256(geometry))
}

func TestCopyFlowReferenceKeepsCopiedPlanReadableAndProvenancePinned(t *testing.T) {
	path, geometry, digest := shootWithFlowReference(t)
	edit, err := searching.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	into := filepath.Join(t.TempDir(), "take")
	var h harness
	h.run("copy", []string{path}, func(call pzm.Call) error {
		return copyInputs(call, edit, into)
	})
	if h.failed != nil {
		t.Fatal(h.failed)
	}
	for _, name := range []string{"shoot.yaml", "shoot.yaml.txt", "score.yaml", "score.yaml.txt",
		"flow-geometry.yaml", "flow-geometry.yaml.txt"} {
		if _, err := os.Stat(filepath.Join(into, name)); err != nil {
			t.Fatalf("copied dependency %s: %v", name, err)
		}
	}
	gotBytes, err := os.ReadFile(filepath.Join(into, "flow-geometry.yaml"))
	if err != nil || string(gotBytes) != string(geometry) {
		t.Fatalf("copied geometry differs: %v", err)
	}
	gotText, err := os.ReadFile(filepath.Join(into, "flow-geometry.yaml.txt"))
	if err != nil || string(gotText) != string(geometry) {
		t.Fatalf("geometry source view differs: %v", err)
	}
	opened, err := searching.Read(filepath.Join(into, "shoot.yaml"))
	if err != nil {
		t.Fatalf("copied plan cannot be reread: %v", err)
	}
	if opened.Plan.FlowStudy.Geometry.SHA != digest ||
		!reflect.DeepEqual(opened.Plan.FlowStudy.Paths, edit.Plan.FlowStudy.Paths) ||
		!reflect.DeepEqual(opened.Plan.FlowStudy.Landmarks, edit.Plan.FlowStudy.Landmarks) ||
		!reflect.DeepEqual(opened.Plan.FlowStudy.Clouds, edit.Plan.FlowStudy.Clouds) {
		t.Fatal("copied plan lost its exact referenced geometry")
	}
}

func TestCopyFlowReferenceRefusesReservedAndUnsafeTargetsBeforeWriting(t *testing.T) {
	path := shoot(t, wavBytes(8000, 1))
	for _, target := range []string{"shoot.yaml", "shoot.yaml.txt", "score.yaml", "score.yaml.txt",
		"catalogue.json", "catalogue.json.txt", "render.json", "animation.mp4", "frames",
		"frames/dependency.yaml", "behind-the-scenes.json", "behind-the-scenes.js",
		"behind-the-scenes.html", "behind-the-scenes.html/child.yaml", "../escape.yaml"} {
		t.Run(target, func(t *testing.T) {
			edit := &searching.Edit{Plan: searching.Plan{Score: filepath.Join(filepath.Dir(path), "score.yaml"),
				FlowStudy: &searching.FlowPlan{Geometry: &searching.FlowGeometryReference{File: target}}}}
			into := t.TempDir()
			var h harness
			h.run("copy", []string{path}, func(call pzm.Call) error {
				return copyInputs(call, edit, into)
			})
			if h.failed == nil {
				t.Fatal("reserved or unsafe dependency target was accepted")
			}
			entries, err := os.ReadDir(into)
			if err != nil || len(entries) != 0 {
				t.Fatalf("refusal left partial copied inputs: %v %v", entries, err)
			}
		})
	}
}

func TestCopyFlowReferenceRefusesChangedBytesBeforeWriting(t *testing.T) {
	path, _, _ := shootWithFlowReference(t)
	edit, err := searching.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	geometryPath := filepath.Join(filepath.Dir(path), edit.Plan.FlowStudy.Geometry.File)
	if err := os.WriteFile(geometryPath, []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	into := t.TempDir()
	var h harness
	h.run("copy", []string{path}, func(call pzm.Call) error {
		return copyInputs(call, edit, into)
	})
	if h.failed == nil || !strings.Contains(h.failed.Error(), "changed after validation") {
		t.Fatalf("changed geometry: %v", h.failed)
	}
	entries, err := os.ReadDir(into)
	if err != nil || len(entries) != 0 {
		t.Fatalf("tamper refusal left partial copied inputs: %v %v", entries, err)
	}
}

func TestCopyFlowReferenceRefusesOversizeGrowthBeforeWriting(t *testing.T) {
	path, _, _ := shootWithFlowReference(t)
	edit, err := searching.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	geometryPath := filepath.Join(filepath.Dir(path), edit.Plan.FlowStudy.Geometry.File)
	if err := os.WriteFile(geometryPath, make([]byte, (2<<20)+1), 0o600); err != nil {
		t.Fatal(err)
	}
	into := t.TempDir()
	var h harness
	h.run("copy", []string{path}, func(call pzm.Call) error {
		return copyInputs(call, edit, into)
	})
	if h.failed == nil || !strings.Contains(h.failed.Error(), "exceeds 2097152 bytes") {
		t.Fatalf("oversize geometry growth: %v", h.failed)
	}
	entries, err := os.ReadDir(into)
	if err != nil || len(entries) != 0 {
		t.Fatalf("oversize refusal left partial copied inputs: %v %v", entries, err)
	}
}
