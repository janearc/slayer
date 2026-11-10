package gpu

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// fire-night-design.md documents a 32-volume world budget. This is an
// authored resource limit exposed to callers, not a physical constant.
func TestVolumeLimitKeepsItsDocumentedThirtyTwoVolumeBudget(t *testing.T) {
	if got := VolumeLimit(); got != 32 {
		t.Fatalf("public volume limit is %d, documented budget is 32", got)
	}
	for _, count := range []int{0, 32, 33} {
		world := thing.NewWorld()
		for range count {
			world.AddVolume(thing.Volume{
				Medium: medium.Puff{Radius: 1, Thick: .1},
			})
		}
		encoded, err := encode(world, maths.Subspace{}, nil, 0, 4, 3)
		if count == 33 {
			if encoded != nil || err == nil ||
				!strings.Contains(err.Error(), "more than 32 volumes") {
				t.Fatalf("33-volume scene lost its budget refusal: %v", err)
			}
			continue
		}
		if err != nil || encoded == nil || encoded.counts.volumes != count {
			t.Fatalf("%d-volume scene refused or lost records: %v", count, err)
		}
	}
}
