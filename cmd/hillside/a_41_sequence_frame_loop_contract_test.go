package main

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/gpu"
)

// Keep this before the actual rendering tests. A backwards or stuck frame
// loop must fail on its next draw request, before it writes unbounded files.
// The callback supplies an authored one-pixel fixture; file saving, hashes
// and progress records still run through the actual sequence implementation.
func TestSequencePicturesRequestsExactlyTheCheckedForwardFrameTimes(t *testing.T) {
	path, output := sequenceFixture(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "fps: 24", "fps: 2", 1)
	text = strings.Replace(text, "duration: {value: 60", "duration: {value: 1", 1)
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	plan, err := readSequence(path)
	if err != nil || plan.frames != 2 || plan.fps != 2 || plan.start != 3*time.Second {
		t.Fatalf("checked two-frame plan changed: %+v, %v", plan, err)
	}
	if err := os.Mkdir(output, 0700); err != nil {
		t.Fatal(err)
	}
	state := sequenceState{Complete: map[int]sequenceEntry{}}
	calls := 0
	var progress bytes.Buffer
	err = sequencePictures(*plan, &state, &progress,
		func(at time.Duration) (*image.RGBA, gpu.Profile, tally, error) {
			calls++
			if calls > 2 {
				t.Fatal("sequence requested another frame after its checked two-frame plan")
			}
			want := []time.Duration{3 * time.Second, 3500 * time.Millisecond}[calls-1]
			if at != want {
				t.Fatalf("actual draw request %d moved to %s, want %s", calls, at, want)
			}
			// Authored report data, not timings claimed from a device.
			profile := gpu.Profile{TraceGPU: 3 * time.Millisecond, LightingGPU: 2 * time.Millisecond}
			return image.NewRGBA(image.Rect(0, 0, 1, 1)), profile, tally{sparks: 2, lamps: 3}, nil
		})
	if err != nil || calls != 2 || len(state.Complete) != 2 {
		t.Fatalf("actual sequence did not finish exactly two frames: calls%d, state%+v, error%v", calls, state, err)
	}
	for _, name := range []string{"frame-000000.png", "frame-000001.png", "sequence.json"} {
		if info, err := os.Stat(filepath.Join(output, name)); err != nil || info.Size() == 0 {
			t.Fatalf("actual sequence lost its saved %s: %v", name, err)
		}
	}
	entries, err := os.ReadDir(output)
	if err != nil || len(entries) != 3 ||
		!strings.Contains(progress.String(), "frame 1/2 at 3.000000000s:") ||
		!strings.Contains(progress.String(), "frame 2/2 at 3.500000000s:") ||
		strings.Count(progress.String(), "(GPU 0.005s), 2 sparks, 3 lamps\n") != 2 {
		t.Fatalf("sequence lost its actual bounded files or final progress: files%v, %v, progress%q", entries, err, progress.String())
	}
}

func TestSequencePicturesKeepsTheActualDrawRefusalBeforeSaving(t *testing.T) {
	path, output := sequenceFixture(t)
	plan, err := readSequence(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(output, 0700); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(t.TempDir(), "missing-draw-input")
	_, cause := os.ReadFile(missing)
	if cause == nil {
		t.Fatal("owned draw input should actually be absent")
	}
	state := sequenceState{Complete: map[int]sequenceEntry{}}
	var progress bytes.Buffer
	calls := 0
	defer func() {
		if failure := recover(); failure != nil {
			t.Fatalf("draw refusal reached saving instead: %v", failure)
		}
	}()
	err = sequencePictures(*plan, &state, &progress,
		func(time.Duration) (*image.RGBA, gpu.Profile, tally, error) {
			calls++
			if calls > 1 {
				t.Fatal("sequence continued requesting frames after its actual draw refusal")
			}
			return nil, gpu.Profile{}, tally{}, cause
		})
	if err != cause || calls != 1 || len(state.Complete) != 0 || progress.Len() != 0 {
		t.Fatalf("actual draw refusal was replaced or continued: %v, calls%d, state%+v, progress%q", err, calls, state, progress.String())
	}
	entries, err := os.ReadDir(output)
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed draw wrote actual sequence files: %v, %v", entries, err)
	}
}
