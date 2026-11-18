//go:build !darwin || !cgo

package footprint

import "testing"

func TestNativeUnavailableNeverReportsZeroAsFootprint(t *testing.T) {
	source := Native()
	if _, err := source.Group(1); err == nil {
		t.Fatal("group unexpectedly available")
	}
	if _, err := source.Identity(1); err == nil {
		t.Fatal("identity unexpectedly available")
	}
	if _, err := source.Footprint(1); err == nil {
		t.Fatal("footprint unexpectedly available")
	}
	if sample := Sample(1, source); sample.Complete || sample.Reason == "" {
		t.Fatal(sample)
	}
}
