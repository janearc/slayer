package thing

import (
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// The table the attrs and motion tests read, beside numbers.yaml.
const attrsMotionPoints = "testdata/attrs-motion-points.csv"

// attrsMotionPoint is the point called name in the points table.
func attrsMotionPoint(t *testing.T, name string) maths.Vec {
	t.Helper()
	row := readSheet(t, attrsMotionPoints).row(t, attrsMotionPoints, name)
	return maths.V(cell(t, row, "x"), cell(t, row, "y"), cell(t, row, "z"))
}

// each switch of Attrs is its own and answers for itself: with one on
// the others are off, and an empty Attrs is visible, solid and casts
// its shadow.
func TestEachAttrIsItsOwn(t *testing.T) {
	for _, row := range []struct {
		name                  string
		attrs                 Attrs
		hidden, ghost, clears bool
	}{
		{"empty", Attrs{}, false, false, false},
		{"hidden", Attrs{Hidden: true}, true, false, false},
		{"ghost", Attrs{Ghost: true}, false, true, false},
		{"clear", Attrs{Clear: true}, false, false, true},
		{"all", Attrs{Hidden: true, Ghost: true, Clear: true},
			true, true, true},
	} {
		held := &Thing{Attrs: row.attrs}
		if held.Hidden() != row.hidden || held.Attrs.Ghost != row.ghost ||
			held.Clear() != row.clears {
			t.Errorf("%s thing: hidden %v ghost %v clear %v, want %v %v %v",
				row.name, held.Hidden(), held.Attrs.Ghost, held.Clear(),
				row.hidden, row.ghost, row.clears)
		}
		// a volume has the one switch that means anything to it
		volume := &Volume{Attrs: row.attrs}
		if volume.Hidden() != row.hidden {
			t.Errorf("%s volume: hidden %v, want %v", row.name,
				volume.Hidden(), row.hidden)
		}
	}
}

// still is where it is put, at any time.
func TestStillNeverMoves(t *testing.T) {
	origin := attrsMotionPoint(t, "place")
	turn := maths.Rotation(maths.Up, unity)
	scale := picked(t, "attrs-motion-still-scale")
	placed := maths.NewSubspace(origin, turn, scale)
	motion := Still(placed)
	hours := time.Duration(pickedCount(t, "attrs-motion-long-hours"))
	for _, at := range []time.Duration{nought, time.Second, -time.Hour,
		hours * time.Hour} {
		if got := motion.At(at); got != placed {
			t.Errorf("a still subspace at %v is %v, want %v", at, got, placed)
		}
	}
}

// a thing, a volume and a field with no motion are at the world's zero,
// at any time; with one they are where it says, at the time asked: here
// an orbit of radius 2, once in 4 s from +x, so at 1 s it is at -z, 2
// out.
func TestMotionsPlaceWhatCarriesThem(t *testing.T) {
	radius := picked(t, "attrs-motion-orbit-radius")
	seconds := time.Duration(pickedCount(t, "attrs-motion-orbit-seconds"))
	orbit := Orbit{Radius: radius, Period: seconds * time.Second}
	attrsMotionAtZero(t)
	want := maths.V(nought, nought, -radius)
	got := (&Thing{Motion: orbit}).At(time.Second).Origin
	if !attrsMotionNear(t, got, want) {
		t.Errorf("a thing in orbit at 1 s is at %v, want %v", got, want)
	}
	got = (&Volume{Motion: orbit}).At(time.Second).Origin
	if !attrsMotionNear(t, got, want) {
		t.Errorf("a volume in orbit at 1 s is at %v, want %v", got, want)
	}
	got = (&Field{Motion: orbit}).At(time.Second).Origin
	if !attrsMotionNear(t, got, want) {
		t.Errorf("a field in orbit at 1 s is at %v, want %v", got, want)
	}
	attrsMotionStill(t)
}

// attrsMotionNear is whether got is within the picked tolerance of want.
func attrsMotionNear(t *testing.T, got, want maths.Vec) bool {
	t.Helper()
	return got.Near(want, picked(t, "attrs-motion-tolerance"))
}

// attrsMotionAtZero checks that a thing, a volume and a field with no motion
// are at the world's zero.
func attrsMotionAtZero(t *testing.T) {
	t.Helper()
	for _, at := range []time.Duration{nought, time.Second} {
		if (&Thing{}).At(at) != maths.Local() {
			t.Errorf("a thing with no motion is not at the zero at %v", at)
		}
		if (&Volume{}).At(at) != maths.Local() {
			t.Errorf("a volume with no motion is not at the zero at %v", at)
		}
		if (&Field{}).At(at) != maths.Local() {
			t.Errorf("a field with no motion is not at the zero at %v", at)
		}
	}
}

// attrsMotionStill checks that a thing, a volume and a field with a still
// motion are where it puts them.
func attrsMotionStill(t *testing.T) {
	t.Helper()
	origin := attrsMotionPoint(t, "still-origin")
	unturned := maths.Local().Orientation
	still := Still(maths.NewSubspace(origin, unturned, nought))
	if got := (&Thing{Motion: still}).At(time.Hour).Origin; got != origin {
		t.Errorf("a still thing is at %v", got)
	}
	if got := (&Volume{Motion: still}).At(time.Hour).Origin; got != origin {
		t.Errorf("a still volume is at %v", got)
	}
	if got := (&Field{Motion: still}).At(time.Hour).Origin; got != origin {
		t.Errorf("a still field is at %v", got)
	}
}
