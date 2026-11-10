package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSequenceReadKeepsTheActualMissingPlanFileCause(t *testing.T) {
	control, _ := sequenceFixture(t)
	if _, err := readSequence(control); err != nil {
		t.Fatal("real valid plan control failed", err)
	}
	path := filepath.Join(t.TempDir(), "missing-plan.yaml")
	_, err := readSequence(path)
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), path) {
		t.Fatalf("actual plan-open failure became a schema or decode failure: %v", err)
	}
}

func TestSequenceCommandKeepsActualMissingInputCausesBeforeOutput(t *testing.T) {
	for _, fault := range []string{"plan", "scene"} {
		t.Run(fault, func(t *testing.T) {
			path, output := sequenceFixture(t)
			scene := onFixture(t, "alight: true", "alight: false")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			text := strings.Replace(string(raw), "scene: scene.yaml", "scene: "+scene, 1)
			if err := os.WriteFile(path, []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			const keep = "owned output obstacle; never render the control plan"
			if err := os.WriteFile(output, []byte(keep), 0600); err != nil {
				t.Fatal(err)
			}
			var said, wrong bytes.Buffer
			if code := sequenceRun([]string{"sequence", "--plan", path, "--dry-run"}, &said, &wrong); code != 0 || wrong.Len() != 0 {
				t.Fatalf("actual checked dry-run control failed: code%d, %s", code, wrong.String())
			}
			missing := path
			if fault == "scene" {
				missing = scene
			}
			if err := os.Remove(missing); err != nil {
				t.Fatal(err)
			}
			_, cause := os.ReadFile(missing)
			if !errors.Is(cause, os.ErrNotExist) {
				t.Fatal("fixture did not produce the actual missing-input refusal", cause)
			}
			said.Reset()
			wrong.Reset()
			code := sequenceRun([]string{"sequence", "--plan", path, "--dry-run"}, &said, &wrong)
			if code != 1 || said.Len() != 0 || !strings.Contains(wrong.String(), cause.Error()) {
				t.Fatalf("%s-open cause was hidden by later work: code%d, progress%q, diagnostic%q, want%q",
					fault, code, said.String(), wrong.String(), cause.Error())
			}
			got, err := os.ReadFile(output)
			if err != nil || string(got) != keep {
				t.Fatalf("failed input changed its actual output obstacle: %q, %v", got, err)
			}
		})
	}
}

// A dry run must succeed even when its proposed output path is an existing
// regular file. If the dry-run branch disappears, that real obstacle stops
// the small CPU scene before a frame, rather than rendering a long plan.
func TestSequenceDryRunNeverTouchesAnActualOutputFileObstacle(t *testing.T) {
	path, output := sequenceFixture(t)
	scene := onFixture(t, "alight: true", "alight: false")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "scene: scene.yaml", "scene: "+scene, 1)
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	const keep = "owned regular file; a dry run must leave these bytes alone"
	if err := os.WriteFile(output, []byte(keep), 0600); err != nil {
		t.Fatal(err)
	}
	var said, wrong bytes.Buffer
	code := sequenceRun([]string{"sequence", "--plan", path, "--dry-run"}, &said, &wrong)
	if code != 0 || wrong.Len() != 0 || !strings.Contains(said.String(), "would render 1440 frames at 24 fps") {
		t.Fatalf("real dry-run route performed output work: code%d, %q, %q", code, said.String(), wrong.String())
	}
	got, err := os.ReadFile(output)
	if err != nil || string(got) != keep {
		t.Fatalf("dry run changed its actual output obstacle: %q, %v", got, err)
	}
}

func actualSequenceFolderFixture(t *testing.T) (string, sequenceState) {
	t.Helper()
	given := mustParse(t)
	given.set = onFixture(t, "", "")
	path, output := sequenceFixture(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "scene: scene.yaml", "scene: "+given.set, 1)
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	plan, err := readSequence(path)
	if err != nil {
		t.Fatal(err)
	}
	given, err = filmed(readBoundarySet(t, given.set), given)
	if err != nil {
		t.Fatal(err)
	}
	scene, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	expected, err := sequenceInputs(*plan, scene, given, executable)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sequenceFolder(output, expected, false); err != nil {
		t.Fatal("actual manifest creation control failed", err)
	}
	if _, err := sequenceFolder(output, expected, true); err != nil {
		t.Fatal("actual unchanged manifest resume control failed", err)
	}
	return output, expected
}

func TestSequenceFolderKeepsTheActualManifestDecodeRefusal(t *testing.T) {
	output, expected := actualSequenceFolderFixture(t)
	path := filepath.Join(output, "sequence.json")
	broken := []byte(`{"Complete":`)
	if err := os.WriteFile(path, broken, 0600); err != nil {
		t.Fatal(err)
	}
	var reference sequenceState
	cause := json.Unmarshal(broken, &reference)
	var want *json.SyntaxError
	if !errors.As(cause, &want) {
		t.Fatal("owned manifest did not actually fail JSON decoding", cause)
	}
	_, err := sequenceFolder(output, expected, true)
	var got *json.SyntaxError
	if !errors.As(err, &got) || got.Offset != want.Offset || err.Error() != cause.Error() {
		t.Fatalf("actual manifest decoding cause was hidden by later mismatch: %v, want %v", err, cause)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(after, broken) {
		t.Fatalf("refused resume changed the actual broken manifest: %q, %v", after, err)
	}
}

func TestSequenceFolderKeepsTheActualManifestReadRefusal(t *testing.T) {
	output, expected := actualSequenceFolderFixture(t)
	path := filepath.Join(output, "sequence.json")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	_, cause := os.ReadFile(path)
	var want *os.PathError
	if !errors.As(cause, &want) || os.IsNotExist(cause) {
		t.Fatal("owned directory did not produce a real non-missing manifest read error", cause)
	}
	_, err := sequenceFolder(output, expected, true)
	var got *os.PathError
	if !errors.As(err, &got) || got.Op != want.Op || got.Path != want.Path || err.Error() != cause.Error() {
		t.Fatalf("actual manifest read cause was hidden by a later ownership error: %v, want %v", err, cause)
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		t.Fatalf("refused resume changed its actual manifest obstacle: %v", err)
	}
}

// Compare actual access bits under the current process umask. The test
// neither changes that global setting nor assumes its value is 022.
func TestSequenceFolderKeepsNormalOutputAccessBits(t *testing.T) {
	output, _ := actualSequenceFolderFixture(t)
	control := filepath.Join(t.TempDir(), "normal-output")
	if err := os.Mkdir(control, 0755); err != nil {
		t.Fatal(err)
	}
	want, err := os.Stat(control)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsDir() || got.Mode().Perm() != want.Mode().Perm() {
		t.Fatalf("actual sequence directory has mode%#o, normal output has%#o under the same umask", got.Mode().Perm(), want.Mode().Perm())
	}
}

func TestSequenceManifestKeepsNormalDataFileAccessBits(t *testing.T) {
	output, _ := actualSequenceFolderFixture(t)
	control := filepath.Join(t.TempDir(), "normal-manifest")
	if err := os.WriteFile(control, []byte("authored access-bit control\n"), 0644); err != nil {
		t.Fatal(err)
	}
	want, err := os.Stat(control)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.Stat(filepath.Join(output, "sequence.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !got.Mode().IsRegular() || got.Mode().Perm() != want.Mode().Perm() {
		t.Fatalf("actual manifest has mode%#o, normal data file has%#o under the same umask", got.Mode().Perm(), want.Mode().Perm())
	}
}
