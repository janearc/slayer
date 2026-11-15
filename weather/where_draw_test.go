//go:build draw

package weather

import (
	"errors"
	"testing"
)

// HasGPU says whether the kernels open, as running on the gpu finds:
// where it says no, a run on the gpu is ErrNoGPU.
func TestHasGPUIsWhatRunningFinds(t *testing.T) {
	_, err := small(t, 5).RunOnGPU(1)
	if HasGPU() != (err == nil) {
		t.Errorf("HasGPU %t, and running on the gpu gave %v", HasGPU(),
			err)
	}
	if err != nil && !errors.Is(err, ErrNoGPU) {
		t.Errorf("no gpu is not ErrNoGPU: %v", err)
	}
}
