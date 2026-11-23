//go:build darwin

package gpu

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// tradeSea is a 7 m/s sea, fully developed and deep, running north: the
// fixture of arid's own sea tests.
var tradeSea = oceantex.Sea{Wind: 7, Fetch: 500000, Depth: 0, Toward: 1.5708}

// tradeSeed is the seed the trade sea's bands are made from. Chosen.
const tradeSeed = 7

// seaRowMember is where the `sea` member starts in a field row: after the
// eight members before it, sixteen bytes each.
const seaRowMember = 8 * 16

// sidesRowMember is where the `sides` member starts in a field row.
const sidesRowMember = 7 * 16

// float32Step is the gap between one and the next float32, as a share:
// how far a number may move when it is packed as a float. Defined by the
// format, 2 to the minus 23.
const float32Step = 1.1920929e-7

// newTradeSea is the trade sea at time nought.
func newTradeSea(t *testing.T) *arid.Sea {
	t.Helper()
	sea, err := arid.NewSea(tradeSea, 0, tradeSeed)
	if err != nil {
		t.Fatal(err)
	}
	return sea
}

// rowFloat is the float at a byte offset of a packed buffer.
func rowFloat(bytes []byte, at int) float64 {
	bits := binary.LittleEndian.Uint32(bytes[at:])
	return float64(math.Float32frombits(bits))
}

// A sea with no edge packs as one field row, its bands' three grids each
// in the heights, its bands' sides and its search's numbers in the row.
func TestASeaPacksItsBandsForMetal(t *testing.T) {
	sea := newTradeSea(t)
	packed := sea.Pack()
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: sea, Material: &thing.Material{Name: "sea"}})
	encoded, err := encode(world, maths.Local(), film.Pinhole{}, 0, 1, 1)
	if err != nil {
		t.Fatalf("encoded a sea: %v", err)
	}
	if got := len(encoded.fields.bytes); got != fieldStruct.size() {
		t.Fatalf("field rows are %d bytes, want one row of %d", got,
			fieldStruct.size())
	}
	words := 0
	for index, band := range packed.Bands {
		words += 3 * band.N * band.N
		side := rowFloat(encoded.fields.bytes, sidesRowMember+4*index)
		if side != band.Extent {
			t.Errorf("band %d packed a side of %g, its side is %g", index,
				side, band.Extent)
		}
	}
	if got := len(encoded.heights.bytes) / wordBytes; got != words {
		t.Errorf("packed %d words of bands, want %d", got, words)
	}
	bands := rowFloat(encoded.fields.bytes, seaRowMember+8)
	if bands != float64(len(packed.Bands)) {
		t.Errorf("the row says %g bands, the sea has %d", bands,
			len(packed.Bands))
	}
}

// Anti-test: a height field's row says no bands, so the kernel never takes
// a height field for a sea.
func TestAHeightFieldPacksNoBands(t *testing.T) {
	field, err := shape.NewHeightField(
		[]float64{0, 0, 0, 0, 1, 0, 0, 0, 0}, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: field, Material: &thing.Material{Name: "sea"}})
	encoded, err := encode(world, maths.Local(), film.Pinhole{}, 0, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if bands := rowFloat(encoded.fields.bytes, seaRowMember+8); bands != 0 {
		t.Errorf("a height field's row says %g bands, want none", bands)
	}
}

// Anti-test: bands the row cannot hold are refused by name, not packed
// wrong: none, more than the row has room for, or grids of two sizes.
func TestASeaWhoseBandsTheRowCannotHoldIsRefused(t *testing.T) {
	small := &oceantex.Field{N: 2, Extent: 1}
	large := &oceantex.Field{N: 4, Extent: 1}
	for name, bands := range map[string][]*oceantex.Field{
		"no bands":   nil,
		"five bands": {small, small, small, small, small},
		"two sizes":  {small, large},
	} {
		var refused unsupported
		if err := seaRefused(bands); !errors.As(err, &refused) {
			t.Errorf("%s: refused with %v, want the kernel's refusal",
				name, err)
		}
	}
}

// A material's slope variance reaches its row, for Beckmann's lobe.
func TestASlopeVariancePacksForBeckmannsLobe(t *testing.T) {
	const variance = 0.0213 // a mean-square slope; any above nought
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: newTradeSea(t), Material: &thing.Material{
		Name: "sea", Gloss: 1, Index: 1.333, SlopeVariance: variance}})
	encoded, err := encode(world, maths.Local(), film.Pinhole{}, 0, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	glint := materialStruct.size() - 16
	got := rowFloat(encoded.materials.bytes, glint)
	if math.Abs(got-variance) > variance*float32Step {
		t.Errorf("packed a slope variance of %g, want %g", got, variance)
	}
}
