//go:build draw && darwin

package session

import "testing"

// The lens curve must reach the Metal kernel, not only the parsed camera.
func TestSequenceZoomChangesGPUFraming(test *testing.T) {
	checkSequenceZoomFraming(test, true)
}
