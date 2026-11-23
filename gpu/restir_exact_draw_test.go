//go:build draw && darwin

package gpu

import (
	"errors"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
)

// exactDigests are the sha256 of the exact (software traversal) picture of
// crowd(40) and crowd(5) at 64 by 36, one ray a pixel, drawn by the
// tree at origin/rc-v0.1.3 (73623e49) BEFORE any restir code existed.
// They are recorded, not computed here: a change that makes the exact
// path draw anything else fails this test.
var exactDigests = map[int]string{
	5:  "b783762aec3c9062317d72cf1c0277dcfe3a6a3fb093ebbdb41360fb1f8862cc",
	40: "6598b288b192c3218e7b9b825e1fb5feba126a15b4f1e375c585f194dc8c480b",
}

func TestExactPathIsByteIdenticalWithRestirPresent(t *testing.T) {
	for lamps, want := range exactDigests {
		world, eye := crowd(t, lamps)
		lens := film.Pinhole{Wide: 50 * maths.Degree}
		picture, err := Sampled(world, eye, lens, 0, 64, 36, 1)
		if errors.Is(err, metal.ErrNoDevice) {
			t.Skip("no gpu here")
		}
		if err != nil {
			t.Fatal(err)
		}
		got := digest(picture, 64, 36)
		t.Logf("lamps %d digest %s", lamps, got)
		if got != want {
			t.Errorf("exact picture of %d lamps changed: %s, want %s",
				lamps, got, want)
		}
	}
}
