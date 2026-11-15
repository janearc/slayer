package main

import (
	"errors"
	"io"
	"testing"

	"github.com/janearc/slayer/scene/named"
)

// where the raw goes must be said, and a lens of zero, a sun below the horizon
// or a frame of no size is refused; the sun by the scene, by name.
func TestWhatCannotBeDrawnIsRefused(t *testing.T) {
	for _, wrong := range [][]string{{}, {"-out", "x.pfm", "-wide", "0"},
		{"-out", "x.pfm", "-sun", "-1"}, {"-out", "x.pfm", "-w", "0"}} {
		if _, err := parse(wrong, io.Discard); !errors.Is(err, errAsked) {
			t.Errorf("%v is taken: %v", wrong, err)
		}
	}
	_, err := parse([]string{"-out", "x.pfm", "-sun", "-1"}, io.Discard)
	if !errors.Is(err, named.ErrBounds) {
		t.Errorf("a sun below the horizon is refused as %v, not by "+
			"the scene's bounds", err)
	}
}
