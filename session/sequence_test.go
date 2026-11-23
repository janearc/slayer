package session

import (
	"bytes"
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

// Example_sequence exercises the declared Pzm dry-run door.
func Example_sequence() {
	var out, wrong bytes.Buffer
	code := sequenceRun([]string{"sequence", "--plan", "missing-plan.yaml", "--dry-run"}, &out, &wrong)
	named := strings.Contains(wrong.String(), "missing-plan.yaml")
	fmt.Println(code != 0, named)
	// Output: true true
}
