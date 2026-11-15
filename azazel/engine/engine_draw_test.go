//go:build draw

package engine_test

import (
	"bytes"
	"testing"
)

// the adapter draws a set file through hillside's library: the picture it
// writes is the one hillside writes for the same set and settings, and
// its number is hillside's.
func TestRenderDrawsAsHillsideDoes(t *testing.T) {
	got, _, _, code := engineDraws(t, widow, settings...)
	want, _, wanted := hillsideDraws(t, widow, settings...)
	if code != nothing || wanted != nothing || len(got) == nothing {
		t.Fatalf("the adapter gave %d and hillside %d", code, wanted)
	}
	if !bytes.Equal(got, want) {
		t.Error("the adapter and hillside wrote different pictures")
	}
	small, _, _, _ := engineDraws(t, widow, "-w", "24", "-h", "14", "-n", "1")
	if bytes.Equal(got, small) {
		t.Error("the settings do not reach the picture")
	}
}
