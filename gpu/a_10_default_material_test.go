package gpu

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The established CPU fallback in trace/trace.go is grey .8 with no glow.
// Compare nil with an explicit material, not the GPU fallback variable.
func TestMissingMaterialKeepsTheEstablishedGreyFallback(t *testing.T) {
	missing, explicit := newScene(), newScene()
	missing.material(nil)
	explicit.material(&thing.Material{Colour: maths.Grey(.8)})
	// Material is colour, glow, mirror and info: four 16-byte lanes.
	if len(missing.materials.bytes) != 4*16 ||
		len(explicit.materials.bytes) != 4*16 {
		t.Fatal("default or explicit material did not produce one complete record")
	}
	if !bytes.Equal(missing.materials.bytes, explicit.materials.bytes) {
		t.Fatal("missing material differs from the CPU's established grey fallback")
	}
	for channel := range 3 {
		got := math.Float32frombits(binary.LittleEndian.Uint32(
			missing.materials.bytes[channel*4:]))
		if got != float32(.8) {
			t.Fatalf("default colour channel %d is %g, want .8", channel, got)
		}
	}
}
