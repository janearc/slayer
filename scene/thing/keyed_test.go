package thing

import (
	"math"
	"slices"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// The tables the keyed tests read, beside numbers.yaml.
const (
	keyedEasesFile    = "testdata/keyed-eases.csv"
	keyedNotEasesFile = "testdata/keyed-not-eases.csv"
	keyedOutsideFile  = "testdata/keyed-outside.csv"
	keyedHalvingsFile = "testdata/keyed-halvings.csv"
	keyedRunsFile     = "testdata/keyed-runs.csv"
	keyedFramesFile   = "testdata/keyed-frames.csv"
	keyedProbesFile   = "testdata/keyed-probes.csv"
)

// keyedVec is the vector a row names in its x, y and z columns, or, for a
// prefix, in the columns the prefix names: base-x, base-y and base-z.
func keyedVec(t *testing.T, row map[string]string, prefix string) maths.Vec {
	t.Helper()
	if prefix != "" {
		prefix += "-"
	}
	return maths.V(cell(t, row, prefix+"x"), cell(t, row, prefix+"y"),
		cell(t, row, prefix+"z"))
}

// keyedFrames is the keyframes the frames table holds for a run, in the
// order written.
func keyedFrames(t *testing.T, run string) []Keyframe {
	t.Helper()
	var frames []Keyframe
	for _, row := range readSheet(t, keyedFramesFile) {
		if row["run"] == run {
			frames = append(frames, Keyframe{Share: cell(t, row, "share"),
				At: keyedVec(t, row, "")})
		}
	}
	return frames
}

// keyedRun is the run of a name in the runs table, held or not.
func keyedRun(t *testing.T, name string, hold bool) Keyed {
	t.Helper()
	row := readSheet(t, keyedRunsFile).row(t, keyedRunsFile, name)
	return Keyed{Frames: keyedFrames(t, name), Ease: row["ease"], Hold: hold,
		Length: time.Duration(cellCount(t, row, "seconds")) * time.Second,
		Base:   keyedVec(t, row, "base")}
}

// keyedProbes is the rows of the probes table that ask a run where it is.
func keyedProbes(t *testing.T, run string) []map[string]string {
	t.Helper()
	var probes []map[string]string
	for _, row := range readSheet(t, keyedProbesFile) {
		if row["run"] == run {
			probes = append(probes, row)
		}
	}
	return probes
}

// keyedTime is the time a probe asks about.
func keyedTime(t *testing.T, probe map[string]string) time.Duration {
	t.Helper()
	return time.Duration(cellCount(t, probe, "microseconds")) * time.Microsecond
}

// the four named eases are css's cubic beziers through (0, 0) and
// (1, 1). the curve at t is
//
//	x(t) = 3(1-t)^2 t x1 + 3(1-t) t^2 x2 + t^3
//	y(t) = 3(1-t)^2 t y1 + 3(1-t) t^2 y2 + t^3
//
// and the eased share at x is the y where the x is. t of a half has
// weights 0.375, 0.375 and 0.125; t of a quarter has 0.421875, 0.140625
// and 0.015625. each row of keyed-eases.csv is worked from css's
// published points.
func TestTheNamedEasesAreCssBeziers(t *testing.T) {
	tolerance := picked(t, "keyed-bezier-tolerance")
	for _, row := range readSheet(t, keyedEasesFile) {
		share, want := cell(t, row, "x"), cell(t, row, "want")
		if got := eased(row["ease"], share); !(math.Abs(got-want) <= tolerance) {
			t.Errorf("%q at %v is %v, want %v (%s)", row["ease"], share, got,
				want, row["working"])
		}
	}
	for _, row := range readSheet(t, keyedNotEasesFile) {
		if slices.Contains(Eases(), row["word"]) {
			t.Errorf("%q is an ease", row["word"])
		}
	}
}

// every name, linear and one that is no ease too, gives a share at or
// below nought nought and a share at or above one one, whatever its
// curve does between. The anti-test: a share inside the run is not taken
// to either end.
func TestEasesBelowNoughtAndPastOne(t *testing.T) {
	names := append(Eases(), "swoop")
	inside := picked(t, "keyed-inside-share")
	for _, name := range names {
		for _, row := range readSheet(t, keyedOutsideFile) {
			share, want := cell(t, row, "x"), cell(t, row, "want")
			if got := eased(name, share); got != want {
				t.Errorf("%s at %v (%s) is %v, want %v", name, share,
					row["name"], got, want)
			}
		}
		if got := eased(name, inside); !(got > nought && got < unity) {
			t.Errorf("%s at %v is %v, an end", name, inside, got)
		}
	}
}

// the curve is solved for the time axis by forty halvings, and the
// share it gives is the curve at the middle of what is left.
//
// where the curve meets x exactly, at t of a half, the half that holds
// is the lower one, so what is left ends a half of 2 to the -40 below
// a half: t is a half less 2 to the -41.
//
// the share is then the curve's y there, which a first order step from
// the y at a half gives: y of a half less y' of a half times 2 to the
// -41, the next term being of the order of 2 to the -82. y' is worked
// from y(t) = 3(1-t)^2 t y1 + 3(1-t) t^2 y2 + t^3:
//
//	ease (y1 .1, y2 1):  y' = .3[(1-t)^2 - 2t(1-t)] + 3[2t(1-t) - t^2] + 3t^2,
//	                     at a half -.075 + .75 + .75 = 1.425, and y is .5375
//	ease-in-out (0, 1):  y' = 3[2t(1-t) - t^2] + 3t^2,
//	                     at a half .75 + .75 = 1.5, and y is .5
func TestTheHalvingsSettleBelowAnExactMeet(t *testing.T) {
	step := math.Ldexp(unity, pickedCount(t, "keyed-halving-step-exponent"))
	for _, row := range readSheet(t, keyedHalvingsFile) {
		share := cell(t, row, "x")
		got := eased(row["ease"], share)
		want := cell(t, row, "level") - cell(t, row, "slope")*step
		if !(math.Abs(got-want) <= picked(t, "keyed-halving-tolerance")) {
			t.Errorf("%s at %v is %.17g, want %.17g (a half of 2 to the "+
				"-40 below the meet)", row["ease"], share, got, want)
		}
	}
}

// a share of the run is blended between the two keyframes it falls
// between: a quarter of the run is half way from the first to the
// second, and three quarters half way from the second to the third. The
// run is along x and up y, at nought, a half and one of its length.
func TestKeyframesBlendBetweenTheTwoAroundTheShare(t *testing.T) {
	held, loose := keyedRun(t, "along", true), keyedRun(t, "along", false)
	for _, row := range keyedProbes(t, "along") {
		at, want := keyedTime(t, row), keyedVec(t, row, "")
		near(t, "held, "+row["name"], held.At(at).Origin, want)
		near(t, "not held, "+row["name"], loose.At(at).Origin, want)
	}
	// past the end, a held run keeps the last frame and one not held
	// goes back to its base
	frames := keyedFrames(t, "along")
	last := frames[len(frames)-unity].At
	near(t, "an hour on, held", held.At(time.Hour).Origin, last)
	near(t, "an hour on, not held", loose.At(time.Hour).Origin, loose.Base)
	past := loose.Length + time.Millisecond
	near(t, "a moment past the end, not held", loose.At(past).Origin,
		loose.Base)
}

// the eased share is blended once over the whole run: an ease-in-out
// push is half way where the curve is a half, at the middle of the run,
// and a quarter of its way, 0.15625, where the share is 0.274375.
func TestAnEasedRunIsBlendedOnce(t *testing.T) {
	push := keyedRun(t, "push", true)
	for _, row := range keyedProbes(t, "push") {
		at := keyedTime(t, row)
		near(t, row["name"], push.At(at).Origin, keyedVec(t, row, ""))
	}
}

// a run of no length, or of none, or with no frames, stands at its base;
// one of a single nanosecond is run, and ends on its last frame at that
// nanosecond.
func TestRunsOfNoLengthAndOfOneNanosecond(t *testing.T) {
	short := keyedRun(t, "short", true)
	frames, base := short.Frames, short.Base
	for _, length := range []time.Duration{nought, -time.Second} {
		run := Keyed{Length: length, Frames: frames, Base: base, Hold: true}
		for _, at := range []time.Duration{nought, time.Second} {
			if got := run.At(at).Origin; got != base {
				t.Errorf("a run of %v at %v is at %v, want its base",
					length, at, got)
			}
		}
	}
	if got := (Keyed{Length: time.Second, Base: base}).At(nought).Origin; got != base {
		t.Errorf("a run of no frames is at %v", got)
	}
	run := Keyed{Length: time.Nanosecond, Frames: frames, Base: base,
		Ease: "linear"}
	if got, want := run.At(time.Nanosecond).Origin, frames[len(frames)-unity].At; got != want {
		t.Errorf("a run of one nanosecond at its end is at %v", got)
	}
	if got := run.At(nought).Origin; got != frames[nought].At {
		t.Errorf("a run of one nanosecond at its start is at %v", got)
	}
	if got := run.At(pair * time.Nanosecond).Origin; got != base {
		t.Errorf("a run of one nanosecond, two on, is at %v", got)
	}
}

// a first keyframe that is not at nought is held until its share; two
// frames at one share are each the place at that share, the later one
// winning at it and the earlier never showing, as css lets the later
// rule win.
func TestLateFirstFramesAndSharedShares(t *testing.T) {
	for _, run := range []string{"late", "same", "mid"} {
		keyed := keyedRun(t, run, true)
		for _, row := range keyedProbes(t, run) {
			at := keyedTime(t, row)
			near(t, run+", "+row["name"], keyed.At(at).Origin,
				keyedVec(t, row, ""))
		}
	}
}
