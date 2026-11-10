package camera_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/camera"
	"github.com/janearc/slayer/maths"
)

// place is a point in scene metres: east, up, and south.
func place(east, up, south float64) maths.Vec {
	return maths.V(east, up, south)
}

// looking is a move with its own look, or the test stops.
func looking(test *testing.T, move camera.Move,
	from, to maths.Vec) camera.Move {
	test.Helper()
	made, err := move.Looking(from, to)
	if err != nil {
		test.Fatal(err)
	}
	return made
}

// facing is whether a sampled camera at a moment looks toward a target.
func facing(test *testing.T, path *camera.Path, at time.Duration,
	target maths.Vec) bool {
	test.Helper()
	eye, err := path.At(at)
	if err != nil {
		test.Fatal(err)
	}
	return eye.Forward().Near(target.Sub(eye.Origin).Unit(), 1e-9)
}

// the test: a hold that looks from one target to another pans: it faces
// the first as it starts, the second as it ends, and the point between
// them halfway, while it stays where it is.
func TestAHoldThatLooksPans(test *testing.T) {
	stand := place(0, 10, 0)
	north, west := place(0, 10, -100), place(-100, 10, 0)
	hold := looking(test, timed(test, stand, stand, 2*time.Second, ""),
		north, west)
	path, err := camera.New([]camera.Move{hold}, place(100, 10, 0),
		maths.Up)
	if err != nil {
		test.Fatal(err)
	}
	if !facing(test, path, 0, north) ||
		!facing(test, path, time.Second, place(-50, 10, -50)) ||
		!facing(test, path, 2*time.Second, west) {
		test.Error("the hold did not pan from north to west")
	}
	eye, err := path.At(time.Second)
	if err != nil || !eye.Origin.Near(stand, 0) {
		test.Errorf("the pan moved the camera: %v %v", eye, err)
	}
}

// the test: a move that says no look looks where the path does, and a
// move after it may take up its own look from there.
func TestAMoveWithoutALookUsesThePathsAndHandsItOn(test *testing.T) {
	ground, sky := place(0, 0, 0), place(0, 1000, 0)
	target, down := place(0, 2000, -10), place(0, 0, -10)
	rise := timed(test, ground, sky, 10*time.Second, "smooth")
	turn := looking(test, timed(test, sky, sky, 5*time.Second, ""),
		target, down)
	// up is north: the turn ends looking straight down, where the
	// world's up would be no way to hold the picture level.
	north := place(0, 0, -1)
	path, err := camera.New([]camera.Move{rise, turn}, target, north)
	if err != nil {
		test.Fatal(err)
	}
	if !facing(test, path, 5*time.Second, target) ||
		!facing(test, path, 15*time.Second, down) {
		test.Error("the rise did not look up, or the turn not down")
	}
}

// the anti-test: a move whose look starts where the one before it did
// not end is refused, as a move that starts somewhere else is.
func TestALookThatJumpsAtAJoinIsRefused(test *testing.T) {
	stand := place(0, 10, 0)
	first := looking(test, timed(test, stand, stand, time.Second, ""),
		place(0, 10, -100), place(-100, 10, 0))
	second := looking(test, timed(test, stand, stand, time.Second, ""),
		place(100, 10, 0), place(0, 10, 100))
	_, err := camera.New([]camera.Move{first, second}, place(1, 1, 1),
		maths.Up)
	if err == nil || !strings.Contains(err.Error(), "does not start "+
		"looking where the previous move ended") {
		test.Errorf("a jumping look was taken: %v", err)
	}
}

// the anti-test: a look whose target sweeps through the camera is
// refused, since the camera would have no way to face.
func TestALookThatSweepsThroughTheCameraIsRefused(test *testing.T) {
	stand := place(0, 10, 0)
	sweep := looking(test, timed(test, stand, stand, time.Second, ""),
		place(-50, 10, 0), place(50, 10, 0))
	_, err := camera.New([]camera.Move{sweep}, place(1, 1, 1), maths.Up)
	if err == nil || !strings.Contains(err.Error(),
		"reaches its look target") {
		test.Errorf("a look through the camera was taken: %v", err)
	}
}

// the anti-test: a camera and a target that pass each other from far
// apart are refused, though each travels a finite way: their difference
// travels twice as far, past where the crossing test's products are
// finite. A counterexample from review 015, round 2.
func TestALookTooFarApartIsRefusedNotMissed(test *testing.T) {
	far := 4e153
	move := looking(test, timed(test, place(-far, 0, 0),
		place(far, 0, 0), time.Second, ""),
		place(far, 0, 0), place(-far, 0, 0))
	_, err := camera.New([]camera.Move{move}, place(1, 1, 1), maths.Up)
	if err == nil || !strings.Contains(err.Error(), "numeric range") {
		test.Errorf("a look past the numeric range was taken: %v", err)
	}
}

// the anti-test: a look at no finite place is refused before any frame.
func TestALookAtNoPlaceIsRefused(test *testing.T) {
	stand := place(0, 10, 0)
	hold := timed(test, stand, stand, time.Second, "")
	for _, target := range []maths.Vec{place(math.NaN(), 0, 0),
		place(math.Inf(1), 0, 0)} {
		if _, err := hold.Looking(target, stand.Add(place(1, 0,
			0))); err == nil {
			test.Errorf("a look at %v was taken", target)
		}
	}
}
