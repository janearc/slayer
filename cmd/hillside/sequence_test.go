package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sequenceFixture(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	plan := filepath.Join(dir, "sequence.yaml")
	output := filepath.Join(dir, "frames")
	text := fmt.Sprintf("$schema: https://github.com/janearc/slayer/schemas/sequence-v1.json\nscene: scene.yaml\noutput: %s\nfps: 24\nstart: {value: 3, unit: s}\nduration: {value: 60, unit: s}\nprint: {knee: 0.1, top: 100, bloom: 0.6}\n", output)
	if err := os.WriteFile(plan, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	return plan, output
}

func TestSequencePlanUsesExactFrameTimesAndPhysicalUnits(t *testing.T) {
	path, _ := sequenceFixture(t)
	plan, err := readSequence(path)
	if err != nil {
		t.Fatal(err)
	}
	if plan.frames != 1440 || plan.fps != 24 || plan.start != 3*time.Second ||
		plan.scene != filepath.Join(filepath.Dir(path), "scene.yaml") || plan.knee != .1 || plan.top != 100 {
		t.Fatal("sequence plan lost frame count, time or relative scene")
	}
}

func TestSequencePlanRejectsUnboundedWorkAndWrongUnits(t *testing.T) {
	path, _ := sequenceFixture(t)
	raw, _ := os.ReadFile(path)
	for _, change := range [][2]string{
		{"fps: 24", "fps: 0"}, {"unit: s", "unit: ms-ish"},
		{"duration: {value: 60", "duration: {value: -1"},
		{"top: 100", "top: 0"}, {"fps: 24", "fps: 24\ntypo: true"},
	} {
		changed := strings.Replace(string(raw), change[0], change[1], 1)
		if err := os.WriteFile(path, []byte(changed), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := readSequence(path); err == nil {
			t.Fatal("invalid sequence accepted")
		}
	}
}

func TestSequenceDryRunCreatesNoOutput(t *testing.T) {
	path, output := sequenceFixture(t)
	given := fireOnFixture(t)
	raw, _ := os.ReadFile(path)
	raw = bytes.Replace(raw, []byte("scene: scene.yaml"), []byte("scene: "+given.set), 1)
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}
	var out, wrong bytes.Buffer
	code := sequenceRun([]string{"sequence", "--plan", path, "--dry-run"}, &out, &wrong)
	if code != 0 || !strings.Contains(out.String(), "1440 frames") {
		t.Fatalf("dry run: %d %s %s", code, out.String(), wrong.String())
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("dry run wrote output")
	}
}

func TestSequenceUsesPzmRefusalsBeforeAnyWork(t *testing.T) {
	var out, wrong bytes.Buffer
	code := sequenceRun([]string{"sequence", "--typo"}, &out, &wrong)
	if code != 2 || !strings.Contains(wrong.String(), "unknown flag") {
		t.Fatal("new verb bypassed Puzzlemouth")
	}
}

func TestSequenceResumeRequiresMatchingInputsAndProtectsExistingFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "frames")
	expected := sequenceState{Plan: "p", Scene: "s", Build: "b", Width: 8, Height: 8, FPS: 24, Frames: 2, Complete: map[int]sequenceEntry{}}
	state, err := sequenceFolder(dir, expected, false)
	if err != nil {
		t.Fatal(err)
	}
	state.Complete[0] = sequenceEntry{SHA256: "hash", AtNanoseconds: 3}
	if err = writeSequenceState(dir, state); err != nil {
		t.Fatal(err)
	}
	read, err := sequenceFolder(dir, expected, true)
	if err != nil || len(read.Complete) != 1 {
		t.Fatal("matching progress was lost")
	}
	if _, err = sequenceFolder(dir, expected, false); err == nil {
		t.Fatal("existing output accepted without resume")
	}
	wrong := expected
	wrong.Scene = "different"
	if _, err = sequenceFolder(dir, wrong, true); err == nil {
		t.Fatal("mixed scene frames accepted")
	}
	unowned := filepath.Join(t.TempDir(), "unowned")
	os.Mkdir(unowned, 0755)
	marker := filepath.Join(unowned, "keep-me")
	os.WriteFile(marker, []byte("untouched"), 0644)
	if _, err = sequenceFolder(unowned, expected, true); err == nil {
		t.Fatal("unowned output accepted")
	}
	if data, _ := os.ReadFile(marker); string(data) != "untouched" {
		t.Fatal("unrelated data changed")
	}
}

func TestSequenceRendersDistinctTimedGPUFramesAndResumesThem(t *testing.T) {
	given := fireOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "width: 120", "width: 64", 1)
	// Set the film dimensions explicitly, independent of the fixture's defaults.
	start, end := strings.Index(text, "  film:"), strings.Index(text, "  cameras:")
	if start < 0 || end < start {
		t.Fatal("fixture film missing")
	}
	text = text[:start] + "  film:\n    width: 64\n    height: 40\n    rays: 1\n    exposure: 100\n" + text[end:]
	aCamera, bCamera := strings.Index(text, "  cameras:"), strings.Index(text, "  set:")
	text = text[:aCamera] + "  cameras:\n    near:\n      orientation: {point_to: the tree, alpha: -90}\n      location:\n        loc_relative: {object: the tree, standoff_x: 0.1, standoff_y: 0, standoff_z: 4}\n" + text[bCamera:]
	if err = os.WriteFile(given.set, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	planPath := filepath.Join(dir, "clip.yaml")
	output := filepath.Join(dir, "frames")
	plan := fmt.Sprintf("$schema: https://github.com/janearc/slayer/schemas/sequence-v1.json\nscene: %s\noutput: %s\nfps: 10\nstart: {value: 0.13, unit: s}\nduration: {value: 0.2, unit: s}\nprint: {knee: 0.1, top: 100, bloom: 0}\n", given.set, output)
	if err = os.WriteFile(planPath, []byte(plan), 0644); err != nil {
		t.Fatal(err)
	}
	var out, wrong bytes.Buffer
	if code := sequenceRun([]string{"sequence", "--plan", planPath}, &out, &wrong); code != 0 {
		t.Fatalf("render failed %d: %s", code, wrong.String())
	}
	a, err := os.ReadFile(filepath.Join(output, "frame-000000.png"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(output, "frame-000001.png"))
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(a) == sha256.Sum256(b) {
		t.Fatal("frame time did not change the fire")
	}
	progress, _ := os.ReadFile(filepath.Join(output, "sequence.json"))
	var state sequenceState
	if err = json.Unmarshal(progress, &state); err != nil {
		t.Fatal(err)
	}
	if len(state.Complete) != 2 || state.Complete[0].AtNanoseconds != 130000000 || state.Complete[1].AtNanoseconds != 230000000 {
		t.Fatal("frame timing or atomic progress lost")
	}
	out.Reset()
	wrong.Reset()
	if code := sequenceRun([]string{"sequence", "--plan", planPath, "--resume"}, &out, &wrong); code != 0 {
		t.Fatalf("resume %d: %s", code, wrong.String())
	}
	if strings.Contains(out.String(), "frame ") {
		t.Fatal("resume redrew a verified frame")
	}
}

// Example_sequence exercises the declared Pzm dry-run door.
func Example_sequence() {
	var out, wrong bytes.Buffer
	code := sequenceRun([]string{"sequence", "--plan", "missing-plan.yaml", "--dry-run"}, &out, &wrong)
	fmt.Println(code != 0, strings.Contains(wrong.String(), "missing-plan.yaml"))
	// Output: true true
}
