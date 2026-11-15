package job

import (
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// lensOf is the camera of the job text says.
func lensOf(t *testing.T, text string) Camera {
	t.Helper()
	lens, has := read(t, text).Camera()
	if !has {
		t.Fatal("the job has no camera")
	}
	return lens
}

// secondsNamed is a number of the tests' as a duration of seconds.
func secondsNamed(t *testing.T, name string) time.Duration {
	t.Helper()
	return quantity{Value: number(t, name)}.duration()
}

// TestATimedSmoothMoveWithItsOwnUpReachesTheRenderer holds a smooth move
// to its hand-worked place a quarter of the way, and an up as said.
func TestATimedSmoothMoveWithItsOwnUpReachesTheRenderer(t *testing.T) {
	text := strings.Replace(pathCamera, "speed: {value: 2, unit: m/s}",
		"duration: {value: 2, unit: s}\n      ease: smooth", one)
	text = strings.Replace(text, "  unit: m\n",
		"  unit: m\n  up: {x: 1, y: 0, z: 0}\n", one)
	path, _ := lensOf(t, still+text).Path()
	eye, err := path.At(secondsNamed(t, "smooth-quarter-seconds"))
	if err != nil {
		t.Fatal(err)
	}
	want := maths.V(nothing, nothing, number(t, "smooth-quarter-z"))
	if eye.Origin != want {
		t.Errorf("a quarter along a smooth move is %v, want %v",
			eye.Origin, want)
	}
	if eye.Orientation.Y != maths.XAxis {
		t.Errorf("the camera's up is %v, want east", eye.Orientation.Y)
	}
}

// TestAMovesOwnLookReachesTheCamera holds the camera facing the look's
// first target as the move starts and its second as it ends.
func TestAMovesOwnLookReachesTheCamera(t *testing.T) {
	text := strings.Replace(pathCamera, "speed: {value: 2, unit: m/s}",
		"speed: {value: 2, unit: m/s}\n      look: {from: {x: 0, y: 1, "+
			"z: 0}, to: {x: 1, y: 1, z: 0}}", one)
	path, _ := lensOf(t, still+text).Path()
	ahead := number(t, "look-ahead")
	for _, each := range []struct {
		at     time.Duration
		target maths.Vec
	}{
		{nothing, maths.V(nothing, ahead, nothing)},
		{path.Duration(), maths.V(ahead, ahead, nothing)},
	} {
		eye, err := path.At(each.at)
		if err != nil {
			t.Fatal(err)
		}
		toward := each.target.Sub(eye.Origin).Unit()
		if !eye.Forward().Near(toward, number(t, "within")) {
			t.Errorf("at %s the camera does not face %v", each.at,
				each.target)
		}
	}
}

// TestALookHalfSaidIsRefused is the anti-test: a look with one target
// is refused before any frame.
// anti-test of TestAMovesOwnLookReachesTheCamera
func TestALookHalfSaidIsRefused(t *testing.T) {
	text := strings.Replace(pathCamera, "speed: {value: 2, unit: m/s}",
		"speed: {value: 2, unit: m/s}\n      look: {from: {x: 0, y: 1, "+
			"z: 0}}", one)
	refused(t, still+text, "job.yaml:", "/camera/moves/0/look")
}

// TestAMoveSaysSpeedOrDuration is the anti-test of a move's pace: both,
// or neither, is refused.
// anti-test of TestATimedSmoothMoveWithItsOwnUpReachesTheRenderer
func TestAMoveSaysSpeedOrDuration(t *testing.T) {
	both := strings.Replace(pathCamera, "speed: {value: 2, unit: m/s}",
		"speed: {value: 2, unit: m/s}\n      duration: {value: 2, "+
			"unit: s}", one)
	neither := strings.Replace(pathCamera,
		"      speed: {value: 2, unit: m/s}\n", "", one)
	for _, text := range []string{both, neither} {
		refused(t, still+text, "job.yaml:", "/camera/moves/0")
	}
}

// TestANamedCameraHasNoPath is the anti-test of the path: a camera by
// name leaves the path to the scene.
// anti-test of TestEveryKeyReads
func TestANamedCameraHasNoPath(t *testing.T) {
	lens := lensOf(t, still+"camera: {name: rim}\n")
	if _, has := lens.Path(); has {
		t.Error("a named camera has a path")
	}
	if _, has := lens.Zoom(); has {
		t.Error("a named camera with no zoom has one")
	}
}

// TestAPathSaysItsUnit is the unit as written, for the renderer to hold
// to the scene's: m for a set, a named scene's own words for it.
func TestAPathSaysItsUnit(t *testing.T) {
	for _, unit := range []string{"m", "a thousand kilometres"} {
		text := strings.Replace(pathCamera, "  unit: m\n",
			"  unit: "+unit+"\n", one)
		said, has := lensOf(t, still+text).Unit()
		if !has || said != unit {
			t.Errorf("a path in %q says %q, %v", unit, said, has)
		}
	}
}

// TestANamedCameraSaysNoUnit is the anti-test: a camera by name has no
// unit of its own.
// anti-test of TestAPathSaysItsUnit
func TestANamedCameraSaysNoUnit(t *testing.T) {
	if unit, has := lensOf(t, still+"camera: {name: rim}\n").Unit(); has {
		t.Errorf("a named camera says the unit %q", unit)
	}
}
