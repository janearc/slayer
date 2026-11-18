//go:build draw

package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/janearc/slayer/improv/searching"
	"github.com/janearc/slayer/interfaces/movie"
)

// TestReceiptRecordsTheProbedMovie writes a real two-frame movie, then
// checks the receipt's numbers against what was asked for.
func TestReceiptRecordsTheProbedMovie(t *testing.T) {
	path := shoot(t, wavBytes(8000, 1))
	edit, err := searching.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	into := t.TempDir()
	score, err := os.ReadFile(edit.Plan.Score)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(into, "score.yaml"), score, 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(into, "shoot.yaml"), plan, 0o644); err != nil {
		t.Fatal(err)
	}
	sound, err := movie.ReadSound(edit.Plan.Music)
	if err != nil {
		t.Fatal(err)
	}
	write := func(frames int, sound movie.Sound) {
		settings, err := movie.NewSettings(movie.Request{Path: filepath.Join(into, "animation.mp4"),
			Width: 160, Height: 90, Rate: 6, Frames: frames, Sound: sound, Pad: true, Replace: true})
		if err != nil {
			t.Fatal(err)
		}
		w, err := movie.New(context.Background(), settings)
		if err != nil {
			t.Fatal(err)
		}
		for i := range frames {
			if err := w.Frame(i, image.NewRGBA(image.Rect(0, 0, 160, 90))); err != nil {
				t.Fatal(err)
			}
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
	}
	write(6, sound)
	if err := receipt(edit, into, 3, 6, 1500*time.Millisecond, pipelineTiming{}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(into, "render.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	probed := got["probed_movie"].(map[string]any)
	if got["name"] != "test shoot" || got["source_start_frame"] != 3.0 || got["frames"] != 6.0 ||
		got["wall_seconds"] != 1.5 || got["status"] != "completed preview" ||
		probed["frames"] != 6.0 || probed["width"] != 160.0 || probed["audio_channels"] != 2.0 {
		t.Errorf("receipt = %s", raw)
	}
	if got["score_sha256"] != fmt.Sprintf("%x", sha256.Sum256(score)) {
		t.Errorf("score hash = %v", got["score_sha256"])
	}
	if got["plan_sha256"] != fmt.Sprintf("%x", sha256.Sum256(plan)) {
		t.Errorf("plan hash = %v", got["plan_sha256"])
	}

	// A movie with a different frame count than requested is refused.
	if err := receipt(edit, into, 0, 5, time.Second, pipelineTiming{}); err == nil {
		t.Error("receipt accepted a movie of another length")
	}
	// A missing movie is refused by the probe.
	if err := receipt(edit, t.TempDir(), 0, 6, time.Second, pipelineTiming{}); err == nil {
		t.Error("receipt accepted a missing movie")
	}
	// The checked plan copy is required to attest the actual production input.
	if err := os.Remove(filepath.Join(into, "shoot.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := receipt(edit, into, 3, 6, time.Second, pipelineTiming{}); err == nil {
		t.Error("receipt accepted a missing plan copy")
	}
	if err := os.WriteFile(filepath.Join(into, "shoot.yaml"), plan, 0o644); err != nil {
		t.Fatal(err)
	}
	// A missing score is refused after the movie checks out.
	if err := os.Remove(filepath.Join(into, "score.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := receipt(edit, into, 3, 6, time.Second, pipelineTiming{}); err == nil {
		t.Error("receipt accepted a missing score copy")
	}
}
