//go:build darwin

package gpu

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/plume"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// Each smoke's cache belongs to its own packed volume record. A direct
// puff before the smokes keeps a mistaken record-zero write observable.
func TestLightingReservationUpdatesEverySmokeAndLeavesOtherRecords(t *testing.T) {
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{
		Medium:   medium.Puff{Radius: 2, Thick: .2},
		Material: &thing.Material{Colour: maths.Grey(.3)},
	})
	for i, top := range []float64{12, 20} {
		column, err := plume.New(plume.Fire{
			Heat: 4000, Convective: .71, Base: 2.4,
			Yield: .0024, Combustion: 21, Extinction: 8,
			Air: 288, Density: 1.2, Top: top, Seed: uint64(i + 5),
		})
		if err != nil {
			t.Fatal(err)
		}
		world.AddVolume(thing.Volume{
			Medium:   column,
			Material: &thing.Material{Colour: maths.Grey(.3)},
		})
	}
	encoded, err := encode(world, maths.Subspace{}, film.Pinhole{}, 0, 4, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded.smokes) != 2 || encoded.smokes[0].index != 1 ||
		encoded.smokes[1].index != 2 {
		t.Fatal("fixture needs a direct puff followed by two cacheable smokes")
	}
	// Volume has thirteen 16-byte lanes; lighting is lane nine. Decode
	// the wire directly rather than using the writer's offset helper.
	const stride, lighting, lanes = 13 * 16, 9 * 16, 4
	if len(encoded.volumes.bytes) != 3*stride {
		t.Fatalf("fixture packed %d volume bytes", len(encoded.volumes.bytes))
	}
	before := append([]byte(nil), encoded.volumes.bytes...)
	prefix := append([]byte(nil), encoded.boxes.bytes...)
	if len(prefix) == 0 || len(prefix)%4 != 0 {
		t.Fatal("fixture lost its initial smoke-density samples")
	}
	// A requested resolution of one needs two planes on every axis for
	// trilinear interpolation: eight cells and three source channels.
	encoded.reserveLightingAt(1)
	const perSmoke = 2 * 2 * 2 * 3
	if len(encoded.boxes.bytes) != len(prefix)+2*perSmoke*4 {
		t.Fatal("two lighting ranges have the wrong total storage")
	}
	if !bytes.Equal(encoded.boxes.bytes[:len(prefix)], prefix) {
		t.Fatal("lighting reservation changed earlier density samples")
	}
	if !bytes.Equal(encoded.volumes.bytes[:stride], before[:stride]) {
		t.Fatal("lighting reservation overwrote the direct puff's record")
	}
	for i, row := range []int{1, 2} {
		first := row * stride
		after := encoded.volumes.bytes[first : first+stride]
		original := before[first : first+stride]
		if !bytes.Equal(after[:lighting], original[:lighting]) ||
			!bytes.Equal(after[lighting+lanes*4:], original[lighting+lanes*4:]) {
			t.Fatalf("cache changed another field in volume %d", row)
		}
		want := [4]int32{2, 2, 2, int32(len(prefix)/4 + i*perSmoke)}
		for lane, value := range want {
			got := int32(binary.LittleEndian.Uint32(after[lighting+lane*4:]))
			if got != value {
				t.Fatalf("volume %d lighting lane %d: got %d, want %d",
					row, lane, got, value)
			}
		}
	}
}
