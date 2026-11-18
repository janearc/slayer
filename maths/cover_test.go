package maths

import "testing"

// TestLocalIsTheZeroSubspace checks Local is the frame with nothing set, whose scale reads as one.
func TestLocalIsTheZeroSubspace(t *testing.T) {
	if got := Local(); got != (Subspace{}) {
		t.Fatalf("Local()=%+v, want the zero Subspace", got)
	}
}

// TestNewSubspaceKeepsWhatItIsGiven checks the origin, orientation and scale, including a scale of zero, are stored as given.
func TestNewSubspaceKeepsWhatItIsGiven(t *testing.T) {
	origin := V(1, 2, 3)
	got := NewSubspace(origin, Orientation{}, 2.5)
	if got.Origin != origin || got.Orientation != (Orientation{}) || got.Scale != 2.5 {
		t.Errorf("got %+v", got)
	}
	if zero := NewSubspace(origin, Orientation{}, 0); zero.Scale != 0 {
		t.Errorf("a scale of zero was changed to %v", zero.Scale)
	}
}
