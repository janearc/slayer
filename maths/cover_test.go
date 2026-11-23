package maths_test

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// TestLocalIsTheZeroSubspace checks Local is the frame with nothing set, whose
// scale reads as one.
func TestLocalIsTheZeroSubspace(test *testing.T) {
	if got := maths.Local(); got != (maths.Subspace{}) {
		test.Fatalf("Local()=%+v, want the zero Subspace", got)
	}
}

// TestNewSubspaceKeepsWhatItIsGiven checks the origin, orientation and scale,
// including a scale of zero, are stored as given.
func TestNewSubspaceKeepsWhatItIsGiven(test *testing.T) {
	origin, scale := vector(test, "a"), picked(test, "subspace-scale")
	got := maths.NewSubspace(origin, maths.Orientation{}, scale)
	if got.Origin != origin || got.Orientation != (maths.Orientation{}) ||
		got.Scale != scale {
		test.Errorf("got %+v", got)
	}
	if zero := maths.NewSubspace(origin, maths.Orientation{}, nought); zero.Scale != nought {
		test.Errorf("a scale of zero was changed to %v", zero.Scale)
	}
}
