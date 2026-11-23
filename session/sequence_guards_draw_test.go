//go:build draw

package session

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/set"
)

func TestSequenceFingerprintsRealChangedFiles(t *testing.T) {
	for _, changed := range []string{"executable", "catalogue", "terrain", "scene", "plan"} {
		t.Run(changed, func(t *testing.T) {
			given := mustParse(t)
			given.set = onFixture(t, "", "")
			path, _ := sequenceFixture(t)
			planRaw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			planRaw = bytes.Replace(planRaw, []byte("scene: scene.yaml"),
				[]byte("scene: "+given.set), 1)
			sequenceWrite(t, path, planRaw)
			executable := filepath.Join(t.TempDir(), "renderer")
			sequenceWrite(t, executable, []byte("#!/bin/sh\nexit 0\n"))
			inputs := func() sequenceState {
				t.Helper()
				plan, err := readSequence(path)
				if err != nil {
					t.Fatal(err)
				}
				raw, err := os.ReadFile(given.set)
				if err != nil {
					t.Fatal(err)
				}
				read, err := set.Read(given.set, raw)
				if err != nil {
					t.Fatal(err)
				}
				filmedGiven, err := filmed(read, given)
				if err != nil {
					t.Fatal(err)
				}
				stamp, err := sequenceInputs(*plan, raw, filmedGiven, executable)
				if err != nil {
					t.Fatal(err)
				}
				return stamp
			}
			before := inputs()
			folder := filepath.Join(t.TempDir(), "frames")
			if _, err = sequenceFolder(folder, before, false); err != nil {
				t.Fatal(err)
			}
			root := filepath.Dir(filepath.Dir(given.set))
			files := map[string]string{
				"executable": executable,
				"catalogue":  filepath.Join(root, "sources.yaml"),
				"terrain":    filepath.Join(os.Getenv("SLAYER_SHELF"), "fixture-plain.tif"),
				"scene":      given.set, "plan": path,
			}
			target := files[changed]
			raw, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			raw = append(raw, []byte("\n# changed actual file\n")...)
			sequenceWrite(t, target, raw)
			after := inputs()
			got := map[string]string{
				"executable": after.Executable, "catalogue": after.Sources,
				"terrain": after.Terrain, "scene": after.Scene, "plan": after.Plan,
			}[changed]
			sum := sha256.Sum256(raw)
			if got != hex.EncodeToString(sum[:]) {
				t.Fatalf("%s hash is not the actual file: %s", changed, got)
			}
			if _, err = sequenceFolder(folder, after, true); err == nil ||
				!strings.Contains(err.Error(), "does not match") {
				t.Fatalf("changed %s accepted: %v", changed, err)
			}
		})
	}
}

func TestSequenceRedrawsOnlyMissingAlteredOrMistimedFrames(t *testing.T) {
	plan := sequencePlan{output: t.TempDir(), fps: 10, frames: 2, start: 130 * time.Millisecond}
	state := &sequenceState{Complete: map[int]sequenceEntry{}}
	var called []time.Duration
	draw := func(at time.Duration) (*image.RGBA, gpu.Profile, tally, error) {
		called = append(called, at)
		picture := image.NewRGBA(image.Rect(0, 0, 1, 1))
		picture.Pix[0], picture.Pix[3] = byte(at.Milliseconds()), 255
		return picture, gpu.Profile{}, tally{}, nil
	}
	check := func(want ...time.Duration) {
		t.Helper()
		called = nil
		if err := sequencePictures(plan, state, io.Discard, draw); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(called, want) {
			t.Fatalf("redrew %v, want %v", called, want)
		}
	}
	check(130*time.Millisecond, 230*time.Millisecond)
	check()
	second := filepath.Join(plan.output, "frame-000001.png")
	sequenceWrite(t, second, []byte("tampered PNG"))
	check(230 * time.Millisecond)
	first := filepath.Join(plan.output, "frame-000000.png")
	if err := os.Remove(first); err != nil {
		t.Fatal(err)
	}
	check(130 * time.Millisecond)
	entry := state.Complete[1]
	entry.AtNanoseconds++
	state.Complete[1] = entry
	check(230 * time.Millisecond)
	check()
	for i := 0; i < 2; i++ {
		raw, err := os.ReadFile(filepath.Join(plan.output, fmt.Sprintf("frame-%06d.png", i)))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		if state.Complete[i].SHA256 != hex.EncodeToString(sum[:]) {
			t.Fatal("repaired frame hash missing")
		}
	}
}

type sequenceBrokenWriter struct{ err error }

func TestSequencePictureFailuresDoNotRecordIncompleteFrames(t *testing.T) {
	for _, fault := range []string{"draw", "nil image", "part path", "target path", "manifest", "progress"} {
		t.Run(fault, func(t *testing.T) {
			plan := sequencePlan{output: t.TempDir(), fps: 24, frames: 1}
			state := &sequenceState{Complete: map[int]sequenceEntry{}}
			failure := errors.New("sequence output failed")
			var out io.Writer = io.Discard
			if fault == "progress" {
				out = sequenceBrokenWriter{failure}
			}
			draw := func(time.Duration) (*image.RGBA, gpu.Profile, tally, error) {
				if fault == "draw" {
					return nil, gpu.Profile{}, tally{}, failure
				}
				if fault == "nil image" {
					return nil, gpu.Profile{}, tally{}, nil
				}
				for name, part := range map[string]string{"part path": "frame-000000.png.part", "target path": "frame-000000.png", "manifest": "sequence.json.part"} {
					if fault == name {
						if err := os.Mkdir(filepath.Join(plan.output, part), 0755); err != nil {
							t.Fatal(err)
						}
					}
				}
				return image.NewRGBA(image.Rect(0, 0, 1, 1)), gpu.Profile{}, tally{}, nil
			}
			if err := sequencePictures(plan, state, out, draw); err == nil {
				t.Fatal("failed frame accepted")
			}
			if fault != "progress" {
				if _, err := os.Stat(filepath.Join(plan.output, "sequence.json")); !os.IsNotExist(err) {
					t.Fatal("incomplete progress committed")
				}
			}
		})
	}
	if err := writeSequenceState(t.TempDir(), &sequenceState{Complete: map[int]sequenceEntry{0: {Seconds: math.NaN()}}}); err == nil {
		t.Fatal("non-JSON progress accepted")
	}
}

func TestSequencePreparationRefusesRigLegacyFireAndChangedScene(t *testing.T) {
	for _, mode := range []string{"rig", "legacy", "changed", "missing", "malformed"} {
		t.Run(mode, func(t *testing.T) {
			given := fireOnFixture(t)
			reason := "scene changed while preparing"
			switch mode {
			case "rig":
				given.set = onFixtureRigged(t, "0.5", false)
				reason = "uses pinhole frames"
			case "legacy":
				given.set = onFixture(t, "", "")
				reason = "cannot retain"
			case "missing":
				given.set = filepath.Join(t.TempDir(), "missing.yaml")
				reason = "no such file"
			case "malformed":
				sequenceWrite(t, given.set, []byte("scene: ["))
				reason = "line"
			}
			raw, err := os.ReadFile(given.set)
			if mode != "missing" && err != nil {
				t.Fatal(err)
			}
			if mode == "changed" {
				raw = append(raw, []byte("\n# earlier scene bytes\n")...)
			}
			plan := sequencePlan{scene: given.set, output: filepath.Join(t.TempDir(), "frames"), fps: 24, frames: 1}
			if err = drawSequence(plan, raw, false, io.Discard); err == nil || !strings.Contains(err.Error(), reason) {
				t.Fatalf("%s accepted: %v", mode, err)
			}
			if _, err = os.Stat(plan.output); !os.IsNotExist(err) {
				t.Fatal("refusal created output")
			}
		})
	}
}

func (writer sequenceBrokenWriter) Write([]byte) (int, error) { return 0, writer.err }
