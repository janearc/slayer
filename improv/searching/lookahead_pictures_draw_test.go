//go:build draw && darwin && lookahead

package searching

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
)

// The harness's moving case, as the BenchmarkHeardWorkers benchmark draws it:
// its cue, start, progress and step, its size, and its frames after the
// cold one.
const (
	picturesCue      = "17.1"
	picturesAt       = 60.0
	picturesProgress = .3
	picturesStep     = .01
	picturesAcross   = 480
	picturesDown     = 270
	picturesFrames   = 3
	picturesBeats    = 66.0 / 60
	picturesRate     = 6
)

// picturesSetting is the value an environment variable names, or the
// harness's own when it names none.
func picturesSetting(name, own string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return own
}

// picturesStart is the harness's start in seconds: HEARD_BENCH_AT's,
// as the benchmark reads it, or its own.
func picturesStart(t *testing.T) float64 {
	t.Helper()
	text := picturesSetting("HEARD_BENCH_AT", "")
	if text == "" {
		return picturesAt
	}
	start, err := strconv.ParseFloat(text, 64)
	if err != nil {
		t.Fatal(err)
	}
	return start
}

// TestLookAheadPictures draws the harness's cold frame and its moving
// frames and writes each picture's pixels, every bit, to the folder
// LOOKAHEAD_PICTURES names, or else HEARD_BENCH_OUT, for a byte
// comparison of two builds. HEARD_BENCH_CUE and HEARD_BENCH_AT choose
// another stage and start, as they do for the benchmark.
func TestLookAheadPictures(t *testing.T) {
	out := picturesSetting("LOOKAHEAD_PICTURES",
		os.Getenv("HEARD_BENCH_OUT"))
	if out == "" {
		t.Skip("LOOKAHEAD_PICTURES names where the pictures go")
	}
	edit, err := Read(os.Getenv("HEARD_BENCH_PLAN"))
	if err != nil {
		t.Fatal(err)
	}
	cue, start := picturesSetting("HEARD_BENCH_CUE", picturesCue),
		picturesStart(t)
	var plan *stagePlan
	for _, shot := range edit.Plan.Shots {
		if shot.From == cue && shot.Stage != nil {
			plan = shot.Stage
			break
		}
	}
	if plan == nil {
		t.Fatal("the harness's stage is not in the plan")
	}
	stage, err := newHeardStage(plan, "")
	if err != nil {
		t.Fatal(err)
	}
	drawer, err := gpu.NewFrames(gpu.FasterLighting)
	if err != nil {
		t.Fatal(err)
	}
	defer drawer.Close()
	lens := film.Pinhole{Wide: plan.Wide * maths.Degree}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	for index := range picturesFrames + 1 {
		at := time.Duration(start*float64(time.Second)) +
			time.Duration(index)*time.Second/picturesRate
		eye, err := stage.update(plan, State{At: at.Seconds(),
			Beat: at.Seconds() * picturesBeats},
			picturesProgress+float64(index)*picturesStep)
		if err != nil {
			t.Fatal(err)
		}
		drawn, _, err := drawer.Draw(stage.world, uint64(index+1), eye,
			lens, at, picturesAcross, picturesDown, 1)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.New()
		if err := binary.Write(sum, binary.LittleEndian,
			drawn.Pix); err != nil {
			t.Fatal(err)
		}
		name := filepath.Join(out, fmt.Sprintf("frame-%d.sha256", index))
		said := fmt.Sprintf("%x\n", sum.Sum(nil))
		if err := os.WriteFile(name, []byte(said), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
