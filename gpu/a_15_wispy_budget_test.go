package gpu

import (
	"encoding/binary"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// volumes.go documents the authored 160-cell default. Hold its work
// budget with reservations alone; no voxel arrays or GPU jobs are made.
func TestDefaultWispyBakingKeepsItsDocumentedReservationBudget(t *testing.T) {
	if bakeSide != 160 {
		t.Fatalf("default wispy bake side is %d, documented budget is 160", bakeSide)
	}
	world := thing.NewWorld()
	for i := range 2 {
		world.AddVolume(thing.Volume{Medium: medium.Wispy{
			Of:    medium.Puff{Radius: 14, Thick: .3},
			Scale: 1, Amount: .5, Seed: uint64(i + 3),
		}})
	}
	encoded, err := encode(world, maths.Subspace{}, nil, 0, 4, 3)
	if err != nil {
		t.Fatal(err)
	}
	const perGrid = 160 * 160 * 160
	if len(encoded.bakes) != 2 || encoded.cells != 2*perGrid ||
		len(encoded.baked.bytes) != 0 || len(encoded.volumes.bytes) != 2*13*16 {
		t.Fatal("wispy reservation changed its job, cell or output-buffer budget")
	}
	for i := range 2 {
		// Volume lane six is the baked-grid first cell and side length.
		at := i*13*16 + 6*16
		first := int32(binary.LittleEndian.Uint32(encoded.volumes.bytes[at:]))
		side := int32(binary.LittleEndian.Uint32(encoded.volumes.bytes[at+4:]))
		if first != int32(i*perGrid) || side != 160 {
			t.Fatalf("wispy volume %d reserved %d/%d, want %d/160",
				i, first, side, i*perGrid)
		}
	}
}
