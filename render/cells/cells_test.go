package cells_test

import (
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/cells"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/smoke"
	"github.com/janearc/slayer/scene/thing"
)

// failer is what the helpers need of a test. Both testing.T and
// rapid.T are one.
type failer interface {
	Helper()
	Fatal(args ...any)
	Fatalf(format string, args ...any)
}

// paceOf is the pace of a start in milliseconds and a rate, or the test
// stops.
func paceOf(t failer, milliseconds, rate int) cells.Pace {
	t.Helper()
	pace, err := cells.NewPace(time.Duration(milliseconds)*time.Millisecond,
		rate)
	if err != nil {
		t.Fatal(err)
	}
	return pace
}

// Each row of the table is a pace and a frame, and when the frame
// happens, worked by hand. A frame a second is exact, and a rate that
// does not divide a second cuts the nanosecond off.
func TestAPaceIsWhenEachFrameHappens(t *testing.T) {
	for _, row := range casesOf(t, pacesFile) {
		pace := paceOf(t, wholeIn(t, row, "start_ms"),
			wholeIn(t, row, "rate"))
		want := time.Duration(wholeIn(t, row, "want_ns"))
		number := wholeIn(t, row, "number")
		if got := pace.At(number); got != want {
			t.Errorf("row %v: frame %d is at %v; want %v", row, number,
				got, want)
		}
	}
}

// Each row of the table is a pace that cannot run: a rate under one or a start
// before zero. All are refused.
func TestAPaceIsRefusedWhatItCannotRun(t *testing.T) {
	for _, row := range casesOf(t, pacesBad) {
		start := time.Duration(wholeIn(t, row, "start_ms")) * time.Millisecond
		pace, err := cells.NewPace(start, wholeIn(t, row, "rate"))
		if err == nil {
			t.Errorf("row %v gave %v; want a refusal", row, pace)
		}
	}
}

// smokeScene is the smoke scene the tests draw, fresh.
func smokeScene(t failer) (*thing.World, maths.Subspace) {
	t.Helper()
	chosen, found := smoke.Get(picked("smoke-number"))
	if !found {
		t.Fatalf("there is no smoke number %d", picked("smoke-number"))
	}
	return chosen.Scene()
}

// placed is a scene of the smoke scene through a square lens, a plain
// tone and one ray a side, with the clock given. The frame happens at
// the picked pace.
func placed(t failer, clock cells.Clock) *cells.Scene {
	t.Helper()
	var lens film.Pinhole
	var tone colour.Tone
	start, rate := picked("start-ms"), picked("rate")
	scene, err := cells.New(lens, tone, once, paceOf(t, start, rate), clock)
	if err != nil {
		t.Fatal(err)
	}
	world, eye := smokeScene(t)
	scene.Place(world, eye)
	return scene
}

// A size under a pixel each way is refused by the tracer. The scene
// says so and draws nothing.
func TestAFrameOfNoSizeIsRefused(t *testing.T) {
	scene := placed(t, nil)
	for _, row := range casesOf(t, sizesRefused) {
		drawn, err := scene.Frame(picked("frame-number"),
			wholeIn(t, row, "wide"), wholeIn(t, row, "high"))
		if err == nil || drawn != nil {
			t.Errorf("row %v gave %v, %v; want a refusal", row, drawn, err)
		}
	}
}

// A scene with no world placed is refused when it is asked for a frame.
func TestAFrameWithNoWorldIsRefused(t *testing.T) {
	var lens film.Pinhole
	var tone colour.Tone
	scene, err := cells.New(lens, tone, once, paceOf(t, nothing, once), nil)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := scene.Frame(nothing, picked("frame-wide"),
		picked("frame-high"))
	if err == nil || drawn != nil {
		t.Errorf("gave %v, %v; want a refusal", drawn, err)
	}
}

// A scene needs a lens and at least one ray a side. Neither no lens nor
// fewer rays is accepted.
func TestASceneIsRefusedWhatItCannotDraw(t *testing.T) {
	var lens film.Pinhole
	var tone colour.Tone
	var noLens film.Sensor
	pace := paceOf(t, nothing, once)
	if scene, err := cells.New(noLens, tone, once, pace, nil); err == nil {
		t.Errorf("a scene with no lens was made: %v", scene)
	}
	for rays := -once; rays < once; rays++ {
		if scene, err := cells.New(lens, tone, rays, pace, nil); err == nil {
			t.Errorf("a scene of %d rays was made: %v", rays, scene)
		}
	}
}
