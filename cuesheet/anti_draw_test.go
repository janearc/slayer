//go:build draw

package cuesheet_test

import (
	"math"
	"testing"
	"time"
)

// anti-test of TestThePieceClockAgreesWithThePiece
// The same comparison fails across two pieces: the clock of the plain
// piece is not the playhead of the piece whose tempo steps, past the step.
func TestThePieceClockDisagreesWithAPieceOfAnotherTempo(t *testing.T) {
	_, made := opened(t)
	stepped := steppedPiece(t)
	at := 15250 * time.Millisecond
	_, _, beats, _, ok := stepped.At(at)
	if !ok || math.Abs(made.Beats(at)-beats) < 1 {
		t.Errorf("the plain clock says %v beats, the stepped piece %v; "+
			"they should differ", made.Beats(at), beats)
	}
}
