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

func TestPlantSelectorsSurviveRegistryRemoval(t *testing.T) {
	for _, species := range []string{"norway", "fern", "saguaro", "valley-oak", "giant-sequoia-shoot"} {
		if _, err := parse([]string{"-out", "x.pfm", "-kind", species}, io.Discard); err != nil {
			t.Errorf("%s: %v", species, err)
		}
	}
	given, err := parse([]string{"-out", "x.pfm"}, io.Discard)
	if err != nil || given.kind != "norway" {
		t.Fatalf("default kind %q: %v", given.kind, err)
	}
	if _, err := parse([]string{"-out", "x.pfm", "-kind", "Norway"}, io.Discard); !errors.Is(err, named.ErrChoice) {
		t.Fatalf("unknown species: %v", err)
	}
}
