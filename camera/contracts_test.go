package camera_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/camera"
	"github.com/janearc/slayer/maths"
)

// refusal holds the public diagnostic and ensures no unchecked path escapes.
func refusal(test *testing.T, moves []camera.Move, look maths.Vec,
	message string) {
	test.Helper()
	path, err := camera.New(moves, look, maths.Up)
	if err == nil || path != nil {
		test.Fatal("accepted an invalid path")
	}
	if !strings.Contains(err.Error(), message) {
		test.Fatalf("%v does not identify %q", err, message)
	}
}

func TestPathRefusalsIdentifyTheAffectedMove(test *testing.T) {
	data := fixture(test)
	look, origin := data.vector(test, "target"), data.vector(test, "origin")
	refusal(test, []camera.Move{{}}, look, "camera move 1: need finite endpoints")
	from, to := data.vector(test, "first"), data.vector(test, "corner")
	valid := travel(test, from, to, whole)
	refusal(test, []camera.Move{valid, valid}, origin, "camera move 2: does not start")
	crossFrom, crossTo := data.vector(test, "crossing-from"), data.vector(test, "crossing-to")
	crossing := timed(test, crossFrom, crossTo, time.Second, "")
	refusal(test, []camera.Move{crossing}, origin, "camera move 1: reaches its look target")
	farFrom, farTarget := data.vector(test, "far-from"), data.vector(test, "far-target")
	partial := timed(test, farFrom, origin, time.Second, "")
	refusal(test, []camera.Move{partial}, farTarget,
		"camera move 1: target direction exceeds the numeric range")
}

func TestMoveRefusesUnmeasurablePositionsAndDeltas(test *testing.T) {
	data := fixture(test)
	from, to := data.vector(test, "overflow-from"), data.vector(test, "overflow-to")
	if _, err := camera.Travel(from, to, whole); err == nil {
		test.Fatal("accepted an unmeasurable start with a finite displacement")
	}
	if _, err := camera.Travel(to, from, whole); err == nil {
		test.Fatal("accepted an unmeasurable end with a finite displacement")
	}
	from, to = data.vector(test, "far-from"), data.vector(test, "far-target")
	if _, err := camera.Timed(from, to, time.Second, ""); err == nil {
		test.Fatal("accepted finite endpoints with an unmeasurable displacement")
	}
}

func TestPathIncludesTheDocumentedDurationBoundary(test *testing.T) {
	data := fixture(test)
	from, to := data.vector(test, "first"), data.vector(test, "corner")
	look := data.vector(test, "origin")
	// The API reserves a second below the clock type's upper bound.
	longest := time.Duration(math.MaxInt64) - time.Second
	move := timed(test, from, to, longest, "linear")
	if _, err := camera.New([]camera.Move{move}, look, maths.Up); err != nil {
		test.Fatal("excluded the supported duration boundary", err)
	}
}

func TestPathPreservesExactEndpointsAcrossCancellationAndSignedZero(test *testing.T) {
	data := fixture(test)
	from, to := data.vector(test, "distant-start"), data.vector(test, "close-end")
	look := data.vector(test, "origin")
	move := timed(test, from, to, time.Second, "linear")
	path, err := camera.New([]camera.Move{move}, look, maths.Up)
	if err != nil {
		test.Fatal(err)
	}
	eye, err := path.At(time.Second)
	if err != nil {
		test.Fatal(err)
	}
	if eye.Origin != to {
		test.Fatal("endpoint lost to cancellation", eye.Origin)
	}
	from = data.vector(test, "first")
	from.X = math.Copysign(nothing, -whole)
	move = timed(test, from, to, time.Second, "linear")
	path, err = camera.New([]camera.Move{move}, look, maths.Up)
	if err != nil {
		test.Fatal(err)
	}
	eye, err = path.At(nothing)
	if err != nil {
		test.Fatal(err)
	}
	if !math.Signbit(eye.Origin.X) {
		test.Fatal("lost the signed-zero starting coordinate")
	}
}

func TestPathJoinUsesTheNextMoveStartingCoordinate(test *testing.T) {
	data := fixture(test)
	first, corner := data.vector(test, "first"), data.vector(test, "corner")
	end, look := data.vector(test, "close-end"), data.vector(test, "origin")
	oldEnd := first
	oldEnd.X = math.Copysign(nothing, -whole)
	moves := []camera.Move{
		timed(test, end, oldEnd, time.Second, "linear"),
		timed(test, first, corner, time.Second, "linear"),
	}
	path, err := camera.New(moves, look, maths.Up)
	if err != nil {
		test.Fatal(err)
	}
	eye, err := path.At(time.Second)
	if err != nil {
		test.Fatal(err)
	}
	if eye.Origin != first || math.Signbit(eye.Origin.X) {
		test.Fatal("join selected the old endpoint instead of the next start")
	}
}
