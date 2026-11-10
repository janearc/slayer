package camera_test

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/camera"
	"github.com/janearc/slayer/maths"
)

// travel checks that a fixture describes valid constant-speed travel.
func travel(test *testing.T, from, to maths.Vec, speed float64) camera.Move {
	test.Helper()
	move, err := camera.Travel(from, to, speed)
	if err != nil {
		test.Fatal(err)
	}
	return move
}

// timed checks that a fixture describes valid timed travel or a hold.
func timed(test *testing.T, from, to maths.Vec,
	duration time.Duration, ease string) camera.Move {
	test.Helper()
	move, err := camera.Timed(from, to, duration, ease)
	if err != nil {
		test.Fatal(err)
	}
	return move
}

func TestSpeedMeansDistancePerSecondAndFramesCanBeReordered(test *testing.T) {
	data := fixture(test)
	origin, end := data.vector(test, "origin"), data.vector(test, "speed-end")
	target, speed := data.vector(test, "target"), data.number(test, "speed")
	move := travel(test, origin, end, speed)
	path, err := camera.New([]camera.Move{move}, target, maths.Up)
	if err != nil {
		test.Fatal(err)
	}
	if path.Duration() != data.duration(test, "speed-duration") {
		test.Fatalf("wrong duration: %s", path.Duration())
	}
	for _, sample := range data.Samples["speed"] {
		at := time.Duration(sample.Seconds * float64(time.Second))
		eye, err := path.At(at)
		if err != nil {
			test.Fatal(err)
		}
		want, tolerance := data.vector(test, sample.Point), data.number(test, "tolerance")
		if !eye.Origin.Near(want, tolerance) {
			test.Fatalf("at %s: %v, want %v", at, eye.Origin, want)
		}
		direction := target.Sub(eye.Origin).Unit()
		if !eye.Forward().Near(direction, tolerance) {
			test.Fatal("camera stopped facing its target")
		}
	}
}

func TestMovesAndHoldsKeepTheirTimingAndOwnTheirValues(test *testing.T) {
	data := fixture(test)
	first, corner := data.vector(test, "first"), data.vector(test, "corner")
	end, origin := data.vector(test, "end"), data.vector(test, "origin")
	travelTime, hold := data.duration(test, "travel-duration"), data.duration(test, "hold-duration")
	moves := []camera.Move{
		timed(test, first, corner, travelTime, ""),
		timed(test, corner, corner, hold, ""),
		timed(test, corner, end, travelTime, "smooth"),
	}
	path, err := camera.New(moves, origin, maths.Up)
	if err != nil {
		test.Fatal(err)
	}
	changed := data.vector(test, "mutation")
	moves[nothing] = timed(test, first, changed, hold, "")
	for _, sample := range data.Samples["holds"] {
		at := time.Duration(sample.Seconds * float64(time.Second))
		eye, err := path.At(at)
		if err != nil {
			test.Fatal(err)
		}
		want := data.vector(test, sample.Point)
		if eye.Origin != want {
			test.Fatalf("at %s: %v, want %v", at, eye.Origin, want)
		}
	}
	if path.Duration() != data.duration(test, "total-duration") {
		test.Fatal("total duration changed")
	}
}

func TestInvalidMovesAreRefusedBeforeRendering(test *testing.T) {
	data := fixture(test)
	origin, end := data.vector(test, "origin"), data.vector(test, "speed-end")
	for _, speed := range []float64{nothing, -whole, math.NaN(),
		math.Inf(whole), math.SmallestNonzeroFloat64} {
		if _, err := camera.Travel(origin, end, speed); err == nil {
			test.Fatalf("accepted speed %g", speed)
		}
	}
	for _, duration := range []time.Duration{nothing, -time.Second} {
		if _, err := camera.Timed(origin, end, duration, "linear"); err == nil {
			test.Fatal("accepted nonpositive duration")
		}
	}
	if _, err := camera.Timed(origin, end, time.Second, "wibble"); err == nil {
		test.Fatal("accepted unknown easing")
	}
	for _, point := range []maths.Vec{origin,
		maths.V(math.Inf(whole), nothing, nothing),
		maths.V(math.NaN(), nothing, nothing)} {
		if _, err := camera.Travel(origin, point, whole); err == nil {
			test.Fatal("accepted absent or non-finite travel")
		}
	}
}

func TestInvalidPathsAreRefusedBeforeRendering(test *testing.T) {
	data := fixture(test)
	zero, first := data.vector(test, "origin"), data.vector(test, "valid-first")
	end := data.vector(test, "valid-end")
	valid := travel(test, first, end, whole)
	from, to := data.vector(test, "crossing-from"), data.vector(test, "crossing-to")
	crossing := timed(test, from, to, time.Second, "")
	stillAtTarget := timed(test, zero, zero, time.Second, "")
	for _, moves := range [][]camera.Move{
		nil, {{}}, {valid, valid}, {crossing}, {stillAtTarget},
	} {
		if _, err := camera.New(moves, zero, maths.Up); err == nil {
			test.Fatal("accepted absent, unchecked, discontinuous or target-crossing moves")
		}
	}
	for _, up := range []maths.Vec{zero, maths.V(math.NaN(), nothing, nothing)} {
		if _, err := camera.New([]camera.Move{valid}, zero, up); err == nil {
			test.Fatal("accepted invalid up")
		}
	}
	var empty camera.Path
	if _, err := empty.At(nothing); err == nil || empty.Duration() != nothing {
		test.Fatal("zero path supplied a camera")
	}
}

func TestSpeedPathIncludesOneNanosecondButRefusesLess(test *testing.T) {
	data := fixture(test)
	from, to := data.vector(test, "origin"), data.vector(test, "nanosecond-end")
	look := data.vector(test, "target")
	move := travel(test, from, to, whole)
	path, err := camera.New([]camera.Move{move}, look, maths.Up)
	if err != nil {
		test.Fatal(err)
	}
	if path.Duration() != time.Nanosecond {
		test.Fatal(path.Duration())
	}
	eye, err := path.At(time.Nanosecond)
	if err != nil {
		test.Fatal(err)
	}
	if eye.Origin != to {
		test.Fatal("lost the exact endpoint", eye.Origin)
	}
	shorter := data.vector(test, "shorter-end")
	if _, err := camera.Travel(from, shorter, whole); err == nil {
		test.Fatal("accepted sub-nanosecond travel")
	}
}

func TestPathRefusesTimeAndDirectionBeyondItsNumericRange(test *testing.T) {
	data := fixture(test)
	from, to := data.vector(test, "first"), data.vector(test, "corner")
	origin := data.vector(test, "origin")
	longTime := data.duration(test, "long-duration")
	long := timed(test, from, to, longTime, "linear")
	for _, holdTime := range []time.Duration{longTime, time.Nanosecond} {
		hold := timed(test, to, to, holdTime, "linear")
		if _, err := camera.New([]camera.Move{long, hold}, origin, maths.Up); err == nil {
			test.Fatalf("accepted unrepresentable duration %s after %s", holdTime, longTime)
		}
	}
	farFrom, farTarget := data.vector(test, "far-from"), data.vector(test, "far-target")
	hold := timed(test, farFrom, farFrom, time.Second, "linear")
	if _, err := camera.New([]camera.Move{hold}, farTarget, maths.Up); err == nil {
		test.Fatal("accepted an unmeasurable look direction")
	}
}
