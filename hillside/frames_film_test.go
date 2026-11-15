//go:build darwin

package hillside

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/gpu"
)

// TestFramesModeFollowsDrawMeasuredsOrder picks the retained drawer's
// mode as drawMeasured picks its per-call function: cached lighting
// before native meshes before native shadows, and software with none.
func TestFramesModeFollowsDrawMeasuredsOrder(t *testing.T) {
	for _, each := range []struct {
		given asked
		want  gpu.Mode
	}{
		{asked{}, gpu.Software},
		{asked{nativeShadows: true}, gpu.NativeShadows},
		{asked{nativeMeshes: true, nativeShadows: true}, gpu.NativeMeshes},
		{asked{cachedLighting: true, nativeMeshes: true}, gpu.CachedLighting},
	} {
		if got := framesMode(each.given); got != each.want {
			t.Errorf("%+v picks mode %d, want %d", each.given, got, each.want)
		}
	}
}

// TestFrameStagesAreSaidForADeviceFrameOnly says nothing for a frame
// drawn on no device, the stages for a per-call frame, and what was sent
// for a retained one.
func TestFrameStagesAreSaidForADeviceFrameOnly(t *testing.T) {
	var said bytes.Buffer
	if err := reportFrameStages(&said, gpu.Profile{}); err != nil ||
		said.Len() != 0 {
		t.Fatalf("a cpu frame said %q (%v)", said.String(), err)
	}
	perCall := gpu.Profile{Device: "a device", Encode: time.Second}
	if err := reportFrameStages(&said, perCall); err != nil ||
		!strings.Contains(said.String(), "encode 1.000s") ||
		strings.Contains(said.String(), "retained") {
		t.Fatalf("a per-call frame said %q (%v)", said.String(), err)
	}
	said.Reset()
	retained := perCall
	retained.Retained = gpu.FrameInputs{Held: 18, Sent: 2, SentBytes: 512}
	if err := reportFrameStages(&said, retained); err != nil ||
		!strings.Contains(said.String(),
			"retained: sent 2 of 18 inputs held, 512 bytes") {
		t.Fatalf("a retained frame said %q (%v)", said.String(), err)
	}
}
