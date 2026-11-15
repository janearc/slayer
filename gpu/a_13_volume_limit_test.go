//go:build draw

package gpu

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// The kernel sizes its crossing arrays to the scene, so the limit is a
// ceiling against pathological packs, not a per-scene budget. It is an
// authored resource limit exposed to callers, not a physical constant;
// doc/fire-night-design.md says what it costs to approach it.
func TestVolumeLimitIsACeilingNotABudget(t *testing.T) {
	if got := VolumeLimit(); got != mostVolumes {
		t.Fatalf("public volume limit is %d, the ceiling is %d", got,
			mostVolumes)
	}
	for _, count := range []int{0, 33, mostVolumes, mostVolumes + 1} {
		world := thing.NewWorld()
		for range count {
			world.AddVolume(thing.Volume{
				Medium: medium.Puff{Radius: 1, Thick: .1},
			})
		}
		encoded, err := encode(world, maths.Subspace{}, nil, 0, 4, 3)
		if count > mostVolumes {
			if encoded != nil || err == nil ||
				!strings.Contains(err.Error(), "more than") {
				t.Fatalf("%d-volume scene lost its ceiling refusal: %v",
					count, err)
			}
			continue
		}
		if err != nil || encoded == nil || encoded.counts.volumes != count {
			t.Fatalf("%d-volume scene refused or lost records: %v", count, err)
		}
	}
}
