package embers

import (
	"math"
	"strings"
	"testing"
)

// TestTerminalSpeedIsFoundForAReachableHeightAndRefusedPastTheLift checks
// that the speed found puts the shell at the height asked at the end of its
// fuse, and that a height past what the lift reaches in no air at all is
// refused.
func TestTerminalSpeedIsFoundForAReachableHeightAndRefusedPastTheLift(t *testing.T) {
	shell := Shell{Lift: displayLift, Gravity: displayGravity, Fuse: seconds(3)}
	// With no air at all the shell is at 111.76*3 - 9.80665*9/2 metres at
	// three seconds, about 291.1; the air only lowers that.
	reach := displayLift*3 - displayGravity*9/2
	terminal, err := terminalFor(shell, 250)
	if err != nil {
		t.Fatal(err)
	}
	shell.Terminal = terminal
	if height, _ := shell.rise(3); math.Abs(height-250) > 1e-6 {
		t.Errorf("the shell is %v m up at the end of its fuse, want 250", height)
	}
	if _, err := terminalFor(shell, reach+1); err == nil || !strings.Contains(err.Error(), "the lift cannot carry the shell") {
		t.Errorf("a height past the lift: %v", err)
	}
}
