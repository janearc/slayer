package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSmokeSequenceMovesResumesAndRefusesAChangedPath(test *testing.T) {
	raw, err := os.ReadFile("../../doc/camera-smoke-sequence.yaml")
	if err != nil {
		test.Fatal(err)
	}
	raw = bytes.Replace(raw, []byte("width: 640, height: 360"), []byte("width: 24, height: 18"), sequenceOne)
	raw = bytes.Replace(raw, []byte("fps: 24"), []byte("fps: 2"), sequenceOne)
	raw = bytes.Replace(raw, []byte("duration: {value: 4"), []byte("duration: {value: 1"), sequenceOne)
	path := filepath.Join(test.TempDir(), "move.yaml")
	sequenceWrite(test, path, raw)
	output := filepath.Join(test.TempDir(), "frames")
	args := []string{"sequence", "--plan", path, "--output", output}
	var out, wrong bytes.Buffer
	if code := sequenceRun(args, &out, &wrong); code != sequenceZero {
		test.Fatal(code, out.String(), wrong.String())
	}
	first, err := os.ReadFile(filepath.Join(output, "frame-000000.png"))
	if err != nil {
		test.Fatal(err)
	}
	second, err := os.ReadFile(filepath.Join(output, "frame-000001.png"))
	if err != nil || bytes.Equal(first, second) {
		test.Fatal("camera failed to move between actual smoke frames", err)
	}
	manifest, err := os.ReadFile(filepath.Join(output, "sequence.json"))
	var state sequenceState
	if err != nil || json.Unmarshal(manifest, &state) != nil || len(state.Complete) != sequencePair {
		test.Fatal("sequence did not commit exactly two actual frames", err)
	}
	out.Reset()
	if code := sequenceRun(append(args, "--resume"), &out, &wrong); code != sequenceZero || strings.Contains(out.String(), "frame ") {
		test.Fatal("matching resume redrew frames", code, out.String(), wrong.String())
	}
	changed := bytes.Replace(raw, []byte("value: 2, unit: m/s"),
		[]byte("value: 3, unit: m/s"), sequenceOne)
	sequenceWrite(test, path, changed)
	if code := sequenceRun(append(args, "--resume"), &out, &wrong); code == sequenceZero {
		test.Fatal("changed camera resumed into an existing film")
	}
}

func TestSequenceAssetIdentityIncludesActualBytesAndMissingFiles(test *testing.T) {
	path := filepath.Join(test.TempDir(), "data")
	sequenceWrite(test, path, []byte("original"))
	first, err := sequenceAssetHash([]string{path})
	if err != nil {
		test.Fatal(err)
	}
	sequenceWrite(test, path, []byte("changed"))
	second, err := sequenceAssetHash([]string{path})
	if err != nil || first == second {
		test.Fatal("changing input bytes did not change identity", err)
	}
	if err := os.Remove(path); err != nil {
		test.Fatal(err)
	}
	if _, err := sequenceAssetHash([]string{path}); err == nil {
		test.Fatal("missing input retained its identity")
	}
}

func TestSequenceSchemaRequiresExactlyOneCompleteSceneRoute(test *testing.T) {
	raw, err := os.ReadFile("../../doc/camera-smoke-sequence.yaml")
	if err != nil {
		test.Fatal(err)
	}
	for _, change := range []struct{ from, to string }{
		{"smoke: 1", "smoke: 1\nscene: scene.yaml"},
		{"film: {width: 640, height: 360, rays: 1}", ""},
		{"smoke: 1", "scene: scene.yaml"},
		{"smoke: 1", "smoke: 99999"},
	} {
		text := strings.Replace(string(raw), change.from, change.to, sequenceOne)
		path := filepath.Join(test.TempDir(), "invalid.yaml")
		sequenceWrite(test, path, []byte(text))
		var out, wrong bytes.Buffer
		args := []string{"sequence", "--plan", path, "--dry-run"}
		if code := sequenceRun(args, &out, &wrong); code == sequenceZero {
			test.Fatal("accepted incomplete or conflicting scene route", change.to)
		}
	}
}
