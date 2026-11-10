package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestSequenceFingerprintRefusesUnreadableRealInputs(t *testing.T) {
	for _, fault := range []string{"scene", "executable", "executable directory", "catalogue", "terrain"} {
		t.Run(fault, func(t *testing.T) {
			given := mustParse(t)
			given.set = onFixture(t, "", "")
			raw, err := os.ReadFile(given.set)
			if err != nil {
				t.Fatal(err)
			}
			read, err := set.Read(given.set, raw)
			if err != nil {
				t.Fatal(err)
			}
			executable := filepath.Join(t.TempDir(), "renderer")
			sequenceWrite(t, executable, []byte("actual executable bytes"))
			wanted := ""
			switch fault {
			case "scene":
				raw = []byte("scene: [")
				wanted = given.set
			case "executable":
				if err := os.Remove(executable); err != nil {
					t.Fatal(err)
				}
				wanted = executable
			case "executable directory":
				executable = t.TempDir()
				wanted = executable
			case "catalogue":
				path, err := nearest(filepath.Dir(given.set), "sources.yaml")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				wanted = "sources.yaml"
			case "terrain":
				ground, _ := read.Ground()
				path := filepath.Join(os.Getenv("SLAYER_SHELF"), ground.File())
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				wanted = filepath.Base(path)
			}
			_, err = sequenceInputs(sequencePlan{scene: given.set}, raw, given, executable)
			if err == nil || !strings.Contains(err.Error(), wanted) {
				t.Fatalf("%s: %v, want %q", fault, err, wanted)
			}
		})
	}
}

func TestSequenceFolderDoesNotClaimProgressItCannotWrite(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(dir, 0700); err != nil {
			t.Error(err)
		}
	})
	_, err := sequenceFolder(dir, sequenceState{Complete: map[int]sequenceEntry{}}, true)
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("read-only output: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "sequence.json")); !os.IsNotExist(err) {
		t.Fatalf("progress written after refusal: %v", err)
	}
}

func TestSequenceInputParseFailuresDoNotStartRendering(t *testing.T) {
	plan := sequencePlan{scene: filepath.Join(t.TempDir(), "missing.yaml")}
	if err := drawSequence(plan, nil, false, io.Discard); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing scene: %v", err)
	}
	sequenceWrite(t, plan.scene, []byte("scene: ["))
	if err := drawSequence(plan, nil, false, io.Discard); err == nil || !strings.Contains(err.Error(), plan.scene) {
		t.Fatalf("malformed scene: %v", err)
	}
}
