//go:build draw

package named_test

import (
	"errors"
	"math"
	"runtime"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/named"
	"github.com/janearc/slayer/trace"
)

// the small picture zappy is drawn at to see its beams move, and to hold
// the gpu to the tracer: a ninth of its tall frame each way, one ray a
// pixel; and the two times it is drawn at, a quarter of a beam's spin
// apart. Picked by the author of this scene, 2026-11-12.
const (
	smallWide, smallHigh = 120, 213
	firstAt              = 500 * time.Millisecond
	laterAt              = 875 * time.Millisecond
)

// the gpu package's stated tolerance for a picture it draws exactly, as
// gpu/sampled_test.go's exact holds it: the mean difference as a share of
// the light, and the largest in one pixel as a share of the brightest
const (
	exactMean    = 1e-5
	exactLargest = 1e-3
)

// small is a built zappy drawn small by the tracer at a time, told the
// time first, as a frame is.
func small(t *testing.T, built *named.Built, at time.Duration) *frame.Frame {
	t.Helper()
	built.Set(at)
	lens := film.Pinhole{Wide: mustGet(t, "zappy").Defaults().Wide()}
	drawn, err := trace.Sampled(built.World(), built.Eye(), lens, at,
		smallWide, smallHigh, once)
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

// same is whether two frames hold the same light in every pixel.
func same(one, other *frame.Frame) bool {
	for i := range one.Pix {
		if one.Pix[i] != other.Pix[i] {
			return false
		}
	}
	return true
}

// Zappy builds alone and keeps time: its beams have a clock, and it
// draws at a tiny size.
func TestZappyBuildsAndKeepsTime(t *testing.T) {
	built := mustBuild(t, "zappy", nil)
	if !built.Timed() {
		t.Error("zappy does not keep time; its beams would not move")
	}
	drawsTiny(t, "zappy", built)
}

// The beams move: zappy told two times draws two pictures.
func TestZappysBeamsMove(t *testing.T) {
	built := mustBuild(t, "zappy", nil)
	if same(small(t, built, firstAt), small(t, built, laterAt)) {
		t.Error("zappy at two times is one picture; the beams are still")
	}
}

// anti-tests: told the same time twice, zappy draws the same picture;
// what moves is the time, not the drawing.
func TestZappyAtOneTimeIsOnePicture(t *testing.T) {
	built := mustBuild(t, "zappy", nil)
	if !same(small(t, built, firstAt), small(t, built, firstAt)) {
		t.Error("zappy at one time drew two pictures")
	}
}

// still holds the swarm, neither turning nor tumbling, and the beams go
// on: every panel is where it was, and the picture still changes.
func TestStillHoldsTheSwarmAndNotTheBeams(t *testing.T) {
	built := mustBuild(t, "zappy", map[string]any{"still": true})
	if moving := moved(built); moving != none {
		t.Errorf("held still, %d of zappy's things move", moving)
	}
	if same(small(t, built, firstAt), small(t, built, laterAt)) {
		t.Error("held still, zappy's beams stopped too")
	}
}

// anti-tests: not held still, the swarm's rings move.
func TestUnheldTheSwarmMoves(t *testing.T) {
	if moved(mustBuild(t, "zappy", nil)) == none {
		t.Error("not held still, nothing in zappy moves")
	}
}

// moved is how many of a built scene's things are somewhere else at the
// later time than at the first.
func moved(built *named.Built) int {
	count := none
	for _, each := range built.World().Things {
		if each.Motion != nil &&
			each.Motion.At(firstAt) != each.Motion.At(laterAt) {
			count++
		}
	}
	return count
}

// Zappy is drawn by the gpu within the gpu package's exact tolerance of
// the tracer, at two times, its beams where the tracer has them.
func TestTheGPUDrawsZappy(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("the kernel is Metal, mac only")
	}
	scene := mustGet(t, "zappy")
	if !scene.Draws(named.GPU) {
		t.Fatal("zappy does not declare the gpu")
	}
	built := mustBuild(t, "zappy", nil)
	lens := film.Pinhole{Wide: scene.Defaults().Wide()}
	for _, at := range []time.Duration{firstAt, laterAt} {
		truth := small(t, built, at)
		drawn, err := gpu.Sampled(built.World(), built.Eye(), lens, at,
			smallWide, smallHigh, once)
		if errors.Is(err, metal.ErrNoDevice) {
			t.Skip("no gpu here")
		}
		if err != nil {
			t.Fatalf("the gpu refuses zappy: %v", err)
		}
		mean, largest := apartBy(truth, drawn)
		t.Logf("at %v: mean %g, largest %g", at, mean, largest)
		if !(mean <= exactMean && largest <= exactLargest) {
			t.Errorf("at %v the gpu is %g from the tracer on average "+
				"and %g at most, want within %g and %g", at, mean,
				largest, exactMean, exactLargest)
		}
	}
}

// apartBy is how far drawn is from truth, as the gpu tests measure it:
// the sum of the differences in each channel as a share of truth's light,
// and the largest in one pixel as a share of truth's brightest. A
// difference that is not a number counts as endless.
func apartBy(truth, drawn *frame.Frame) (float64, float64) {
	var total, brightest, sum, largest float64
	for i, one := range truth.Pix {
		other := drawn.Pix[i]
		light := float64(one.R + one.G + one.B)
		difference := math.Abs(float64(one.R-other.R)) +
			math.Abs(float64(one.G-other.G)) +
			math.Abs(float64(one.B-other.B))
		if math.IsNaN(difference) {
			difference = math.Inf(once)
		}
		total += light
		brightest = math.Max(brightest, light)
		sum += difference
		largest = math.Max(largest, difference)
	}
	return sum / total, largest / brightest
}
