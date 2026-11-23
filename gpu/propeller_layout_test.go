package gpu

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// These literal offsets are the published append-only host/Metal contract,
// deliberately independent of propellerFields and its generated offsets.
func TestPropellerDispersalPublishedCardLayout(t *testing.T) {
	p := propellerFixture(t, 5).Parameters()
	p.WanderM, p.WanderScaleM = .8, 1.5
	p.SpreadStartM, p.SpreadPerM = 1, .12
	field, err := medium.NewPropellerCavitation(p)
	if err != nil {
		t.Fatal(err)
	}
	card, err := propellerCardOf(field)
	if err != nil {
		t.Fatal(err)
	}
	if len(card) != 31 {
		t.Fatalf("published propeller card has %d words, want exactly 31", len(card))
	}
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: field, Material: &thing.Material{Glow: maths.Grey(1)}})
	encoded, err := encode(world, maths.Subspace{}, film.Pinhole{Wide: 1}, 0, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded.boxes.bytes) != 124 {
		t.Fatalf("encoded propeller card has %d bytes, want exactly 124", len(encoded.boxes.bytes))
	}
	for _, item := range []struct {
		name   string
		offset int
		value  float32
	}{
		{"wander", 27, .8},
		{"wander_scale", 28, 1.5},
		{"spread_start", 29, 1},
		{"spread", 30, .12},
	} {
		want := math.Float32bits(item.value)
		if card[item.offset] != want {
			t.Errorf("card word %d (%s) = %#x, want %#x", item.offset, item.name, card[item.offset], want)
		}
		got := binary.LittleEndian.Uint32(encoded.boxes.bytes[item.offset*4:])
		if got != want {
			t.Errorf("encoded byte offset %d (%s) = %#x, want %#x", item.offset*4, item.name, got, want)
		}
		declaration := fmt.Sprintf("constexpr constant int propeller_%s = %d;\n", item.name, item.offset)
		for name, source := range map[string]string{
			"optical":    sourceWithOpticalState(0, probeVolumes),
			"standalone": sourceWithPropellerDispersal(0, probeVolumes),
		} {
			if count := strings.Count(source, declaration); count != 1 {
				t.Errorf("%s shader has %d exact declarations of %q, want one", name, count, declaration)
			}
		}
	}
}
