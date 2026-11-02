package thing

import (
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// each switch of Attrs is its own and answers for itself: with one on
// the others are off, and an empty Attrs is visible, solid and casts
// its shadow.
func TestEachAttrIsItsOwn(t *testing.T) {
	for _, c := range []struct {
		name                  string
		attrs                 Attrs
		hidden, ghost, clears bool
	}{
		{"empty", Attrs{}, false, false, false},
		{"hidden", Attrs{Hidden: true}, true, false, false},
		{"ghost", Attrs{Ghost: true}, false, true, false},
		{"clear", Attrs{Clear: true}, false, false, true},
		{"all", Attrs{Hidden: true, Ghost: true, Clear: true}, true, true, true},
	} {
		th := &Thing{Attrs: c.attrs}
		if th.Hidden() != c.hidden || th.Ghost() != c.ghost || th.Clear() != c.clears {
			t.Errorf("%s thing: hidden %v ghost %v clear %v, want %v %v %v", c.name,
				th.Hidden(), th.Ghost(), th.Clear(), c.hidden, c.ghost, c.clears)
		}
		// a volume has the one switch that means anything to it
		if v := (&Volume{Attrs: c.attrs}); v.Hidden() != c.hidden {
			t.Errorf("%s volume: hidden %v, want %v", c.name, v.Hidden(), c.hidden)
		}
	}
}

// still is where it is put, at any time.
func TestStillNeverMoves(t *testing.T) {
	s := maths.Subspace{Origin: maths.V(1, 2, 3),
		Orientation: maths.Rotation(maths.Up, 1), Scale: 4}
	m := Still(s)
	for _, at := range []time.Duration{0, time.Second, -time.Hour, 1000 * time.Hour} {
		if got := m.At(at); got != s {
			t.Errorf("a still subspace at %v is %v, want %v", at, got, s)
		}
	}
}

// a thing, a volume and a field with no motion are at the world's zero,
// at any time; with one they are where it says, at the time asked: here
// an orbit of radius 2, once in 4 s from +x, so at 1 s it is at -z, 2
// out.
func TestMotionsPlaceWhatCarriesThem(t *testing.T) {
	orbit := Orbit{Radius: 2, Period: 4 * time.Second}
	still := Still(maths.Subspace{Origin: maths.V(7, 8, 9)})
	want := maths.V(0, 0, -2)
	for _, at := range []time.Duration{0, time.Second} {
		if (&Thing{}).At(at) != (maths.Subspace{}) {
			t.Errorf("a thing with no motion is not at the zero at %v", at)
		}
		if (&Volume{}).At(at) != (maths.Subspace{}) {
			t.Errorf("a volume with no motion is not at the zero at %v", at)
		}
		if (&Field{}).At(at) != (maths.Subspace{}) {
			t.Errorf("a field with no motion is not at the zero at %v", at)
		}
	}
	if got := (&Thing{Motion: orbit}).At(time.Second).Origin; !got.Near(want, 1e-9) {
		t.Errorf("a thing in orbit at 1 s is at %v, want %v", got, want)
	}
	if got := (&Volume{Motion: orbit}).At(time.Second).Origin; !got.Near(want, 1e-9) {
		t.Errorf("a volume in orbit at 1 s is at %v, want %v", got, want)
	}
	if got := (&Field{Motion: orbit}).At(time.Second).Origin; !got.Near(want, 1e-9) {
		t.Errorf("a field in orbit at 1 s is at %v, want %v", got, want)
	}
	if got := (&Thing{Motion: still}).At(time.Hour).Origin; got != maths.V(7, 8, 9) {
		t.Errorf("a still thing is at %v", got)
	}
	if got := (&Volume{Motion: still}).At(time.Hour).Origin; got != maths.V(7, 8, 9) {
		t.Errorf("a still volume is at %v", got)
	}
	if got := (&Field{Motion: still}).At(time.Hour).Origin; got != maths.V(7, 8, 9) {
		t.Errorf("a still field is at %v", got)
	}
}
