package set_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// lensSet is a set of one camera standing at a place and looking at one,
// with what more the camera says.
const lensSet = "scene:\n  name: a lens\n  cameras:\n    down:\n" +
	"      location: {at: [0, 0, 1]}\n" +
	"      orientation: {look_at: [0, 2, 0]}\n%s"

// lensOf is the lens of the lens set's camera, with what more it says, or
// the refusal of the set.
func lensOf(t *testing.T, more string) (float64, error) {
	t.Helper()
	source := fmt.Sprintf(lensSet, more)
	read, err := set.Read("lens.yaml", []byte(source))
	if err != nil {
		return 0, err
	}
	shot, err := read.Shot("down", nil)
	if err != nil {
		t.Fatal(err)
	}
	return shot.Lens().Wide, nil
}

// A camera that says how wide its lens is has that lens, in radians.
func TestACameraSaysItsLens(t *testing.T) {
	wide, err := lensOf(t, "      wide: 90\n")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(wide-90*maths.Degree) > 1e-12 { // picked: as said, a rounding
		t.Errorf("the lens is %v radians", wide)
	}
}

// Anti: a camera that says nothing has no lens of its own.
func TestACameraWithoutALensHasNone(t *testing.T) {
	wide, err := lensOf(t, "")
	if err != nil || wide != 0 {
		t.Errorf("the lens is %v radians, %v", wide, err)
	}
}

// A lens of no width, or a half turn or more, is refused.
func TestACameraRefusesALensOutOfRange(t *testing.T) {
	for _, more := range []string{"      wide: 0\n", "      wide: 180\n"} {
		if _, err := lensOf(t, more); err == nil {
			t.Errorf("%q was read", more)
		}
	}
}
