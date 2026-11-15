//go:build draw

package named_test

import (
	"errors"
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

// the magnetar's times the tests draw at: two a quarter of its turn
// apart, before the flare; and the turn itself, SGR 1806-20's period as
// the research data holds it, 7.55 s. Picked by the author of this
// scene, 2026-11-12.
const (
	quietAt  = 1 * time.Second
	turnTime = 7550 * time.Millisecond
	quarter  = turnTime / 4
)

// how far two pictures of one moment drawn a whole turn apart may be,
// as the gpu tests measure it: the arithmetic of a turn of 2 pi more,
// picked by the author of this scene, 2026-11-12
const turnSlack = 1e-6

// quietOnly is the magnetar with only its turning lines: no hoops, no
// flare, from close.
var quietOnly = map[string]any{"hoops": false, "flare": false}

// drawnAt is a built magnetar drawn small by the tracer at a time, told the
// time first, as a frame is.
func drawnAt(t *testing.T, built *named.Built,
	at time.Duration) *frame.Frame {
	t.Helper()
	built.Set(at)
	lens := film.Pinhole{Wide: mustGet(t, "magnetar").Defaults().Wide()}
	made, err := trace.Sampled(built.World(), built.Eye(), lens, at,
		smallWide, smallHigh, once)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// lightIn is the sum of a frame's light.
func lightIn(made *frame.Frame) float64 {
	var sum float64
	for _, pixel := range made.Pix {
		sum += float64(pixel.R + pixel.G + pixel.B)
	}
	return sum
}

// The magnetar builds, keeps time, and draws at a tiny size, in every
// view.
func TestTheMagnetarBuildsAndKeepsTime(t *testing.T) {
	for _, chosen := range []string{"near", "shells", "far", "beside",
		"world"} {
		built := mustBuild(t, "magnetar", map[string]any{"view": chosen,
			"zappy": chosen == "beside", "scale": chosen == "far",
			"world": chosen == "world"})
		if !built.Timed() {
			t.Errorf("%s: the magnetar does not keep time", chosen)
		}
		drawsTiny(t, "magnetar", built)
	}
}

// The star turns at its period: a quarter turn apart, two pictures;
// a whole turn apart, the same picture.
func TestTheMagnetarTurnsAtItsPeriod(t *testing.T) {
	built := mustBuild(t, "magnetar", quietOnly)
	first := drawnAt(t, built, quietAt)
	if same(first, drawnAt(t, built, quietAt+quarter)) {
		t.Error("a quarter turn apart the field drew one picture")
	}
	mean, largest := apartBy(first, drawnAt(t, built, quietAt+turnTime))
	if mean > turnSlack || largest > turnSlack {
		t.Errorf("a whole turn apart the field drew two pictures: "+
			"%g on average, %g at most", mean, largest)
	}
}

// anti-test: a turn the scene is told is not its own: a period asked of
// twice its own is a different picture a whole true turn later.
func TestATurnIsTheStarsOwnPeriod(t *testing.T) {
	with := map[string]any{"hoops": false, "flare": false,
		"period": 2 * turnTime}
	built := mustBuild(t, "magnetar", with)
	if same(drawnAt(t, built, quietAt), drawnAt(t, built, quietAt+turnTime)) {
		t.Error("a star of twice the period is where it was after " +
			"one of the real star's turns")
	}
}

// The flare's spike overwhelms the picture: at its peak the frame holds
// far more light than the quiet star's.
func TestTheFlareOverwhelmsThePicture(t *testing.T) {
	with := map[string]any{"hoops": false, "flare-at": "2s"}
	built := mustBuild(t, "magnetar", with)
	quiet := lightIn(drawnAt(t, built, quietAt))
	peak := lightIn(drawnAt(t, built, 2*time.Second+time.Millisecond))
	if !(peak > overwhelmed*quiet) {
		t.Errorf("at the spike the frame holds %g, quiet %g", peak, quiet)
	}
}

// anti-test: with the flare off, the same moment is the quiet star.
func TestNoFlareNoSpike(t *testing.T) {
	with := map[string]any{"hoops": false, "flare": false,
		"flare-at": "2s"}
	built := mustBuild(t, "magnetar", with)
	quiet := lightIn(drawnAt(t, built, quietAt))
	at := lightIn(drawnAt(t, built, 2*time.Second+time.Millisecond))
	if at > overwhelmed*quiet {
		t.Errorf("with no flare the frame holds %g, quiet %g", at, quiet)
	}
}

// overwhelmed is how many times the quiet frame's light the spike's
// frame holds at least: picked by the author of this scene, 2026-11-12,
// far below the spike's and far above a turn's change
const overwhelmed = 100

// The magnetar is drawn by the gpu within the gpu package's exact
// tolerance of the tracer, at two times, its lines where the tracer has
// them, and at the spike.
func TestTheGPUDrawsTheMagnetar(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("the kernel is Metal, mac only")
	}
	scene := mustGet(t, "magnetar")
	if !scene.Draws(named.GPU) {
		t.Fatal("the magnetar does not declare the gpu")
	}
	built := mustBuild(t, "magnetar", map[string]any{"flare-at": "2s"})
	lens := film.Pinhole{Wide: scene.Defaults().Wide()}
	for _, at := range []time.Duration{quietAt, quietAt + quarter,
		2*time.Second + 50*time.Millisecond} {
		truth := drawnAt(t, built, at)
		made, err := gpu.Sampled(built.World(), built.Eye(), lens, at,
			smallWide, smallHigh, once)
		if errors.Is(err, metal.ErrNoDevice) {
			t.Skip("no gpu here")
		}
		if err != nil {
			t.Fatalf("the gpu refuses the magnetar: %v", err)
		}
		mean, largest := apartBy(truth, made)
		t.Logf("at %v: mean %g, largest %g", at, mean, largest)
		if !(mean <= exactMean && largest <= exactLargest) {
			t.Errorf("at %v the gpu is %g from the tracer on average "+
				"and %g at most, want within %g and %g", at, mean,
				largest, exactMean, exactLargest)
		}
	}
}
