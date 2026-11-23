package session

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// sequenceWrite keeps fixture setup errors from masquerading as refusals.
func sequenceWrite(t *testing.T, path string, raw []byte) {
	t.Helper()
	if err := os.WriteFile(path, raw, 0755); err != nil {
		t.Fatal(err)
	}
}

func TestSequenceResumeChecksEveryIndependentStamp(t *testing.T) {
	original := sequenceState{Plan: "p", Scene: "s", Build: "b", Built: "date",
		Executable: "exe", Sources: "sources", Terrain: "dem",
		Width: 24, Height: 16, FPS: 24, Frames: 2,
		Complete: map[int]sequenceEntry{}}
	for _, field := range []string{"Plan", "Scene", "Build", "Built",
		"Executable", "Sources", "Terrain", "Width", "Height", "FPS", "Frames", "Complete"} {
		t.Run(field, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "frames")
			if _, err := sequenceFolder(dir, original, false); err != nil {
				t.Fatal(err)
			}
			changed := original
			value := reflect.ValueOf(&changed).Elem().FieldByName(field)
			switch value.Kind() {
			case reflect.String:
				value.SetString(value.String() + "-changed")
			case reflect.Int:
				value.SetInt(value.Int() + 1)
			case reflect.Map:
				changed.Complete = nil
				if err := writeSequenceState(dir, &changed); err != nil {
					t.Fatal(err)
				}
				changed = original
			}
			if _, err := sequenceFolder(dir, changed, true); err == nil ||
				!strings.Contains(err.Error(), "does not match") {
				t.Fatalf("stamp %s ignored: %v", field, err)
			}
		})
	}
}

func TestSequenceRoundsFractionalFrameCounts(t *testing.T) {
	for _, test := range []struct {
		seconds     string
		fps, frames int
	}{
		{"0.07", 100, 7}, {"0.15", 10, 2}, {"0.001", 1, 1},
	} {
		path, _ := sequenceFixture(t)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		rate := fmt.Sprintf("fps: %d", test.fps)
		raw = bytes.Replace(raw, []byte("fps: 24"), []byte(rate), 1)
		raw = bytes.Replace(raw, []byte("duration: {value: 60"), []byte("duration: {value: "+test.seconds), 1)
		sequenceWrite(t, path, raw)
		plan, err := readSequence(path)
		if err != nil {
			t.Fatal(err)
		}
		if plan.frames != test.frames {
			t.Fatalf("%ss at %dfps: %d", test.seconds, test.fps, plan.frames)
		}
	}
}

func TestSequenceVerbRefusesMissingPlanArgumentsAndGround(t *testing.T) {
	for _, args := range [][]string{
		{"sequence"}, {"sequence", "--plan", "unused.yaml", "extra"},
	} {
		var out, wrong bytes.Buffer
		if sequenceRun(args, &out, &wrong) == 0 ||
			!strings.Contains(wrong.String(), "sequence wants --plan") {
			t.Fatalf("arguments %v: %s", args, wrong.String())
		}
	}
	for _, test := range []struct{ raw, reason string }{
		{"scene: {name: space}", "requires ground scenery"},
		{"scene: [", "line"},
	} {
		path, _ := sequenceFixture(t)
		scene := filepath.Join(filepath.Dir(path), "scene.yaml")
		sequenceWrite(t, scene, []byte(test.raw))
		var out, wrong bytes.Buffer
		if sequenceRun([]string{"sequence", "--plan", path}, &out, &wrong) == 0 ||
			!strings.Contains(wrong.String(), test.reason) {
			t.Fatalf("%s: %s", test.reason, wrong.String())
		}
	}
	path, _ := sequenceFixture(t)
	var out, wrong bytes.Buffer
	if sequenceRun([]string{"sequence", "--plan", path}, &out, &wrong) == 0 ||
		!strings.Contains(wrong.String(), "scene.yaml") {
		t.Fatalf("missing scene: %s", wrong.String())
	}
}

func TestSequenceMetadataAndFolderErrorsAreNotResumes(t *testing.T) {
	for _, test := range []string{"bad manifest", "manifest directory", "output file", "part directory", "final directory"} {
		t.Run(test, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "frames")
			if err := os.Mkdir(dir, 0755); err != nil {
				t.Fatal(err)
			}
			state := sequenceState{Complete: map[int]sequenceEntry{}}
			switch test {
			case "bad manifest":
				sequenceWrite(t, filepath.Join(dir, "sequence.json"), []byte("["))
			case "manifest directory":
				if err := os.Mkdir(filepath.Join(dir, "sequence.json"), 0755); err != nil {
					t.Fatal(err)
				}
			case "output file":
				if err := os.Remove(dir); err != nil {
					t.Fatal(err)
				}
				sequenceWrite(t, dir, []byte("keep"))
			case "part directory":
				if err := os.Mkdir(filepath.Join(dir, "sequence.json.part"), 0755); err != nil {
					t.Fatal(err)
				}
			case "final directory":
				if err := os.Mkdir(filepath.Join(dir, "sequence.json"), 0755); err != nil {
					t.Fatal(err)
				}
			}
			var err error
			if test == "part directory" || test == "final directory" {
				err = writeSequenceState(dir, &state)
			} else {
				_, err = sequenceFolder(dir, state, true)
			}
			if err == nil {
				t.Fatal("unreadable output accepted")
			}
		})
	}
	missing := filepath.Join(t.TempDir(), "missing")
	if _, err := sequenceHash(missing); err == nil {
		t.Fatal("missing dependency accepted")
	}
	if _, err := sequenceHash(t.TempDir()); err == nil {
		t.Fatal("directory dependency accepted")
	}
}
