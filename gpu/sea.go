// This file packs a sea with no edge for the kernel: a field row whose
// bands the kernel sums at every point of the plane, as arid.Sea does.

package gpu

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies/scrooge"
	"github.com/janearc/slayer/oceantex"
)

// seaBandsMost is how many bands a sea's row has room for: their sides
// are one row member of four numbers.
const seaBandsMost = 4

// sea appends a sea with no edge as a field row the kernel tells from a
// height field by its bands: its placement, each band's heights and its
// slopes east and north after whatever is packed already, and the slab,
// step bound and tolerances arid.Sea.Hit searches by.
//
// The row's fourth number in `sea` is one when scrooge is asked for, and
// the kernel then takes the sea as its mean plane from where a pixel is as
// wide as the sea is tall; nought, it searches the whole way.
//
// It refuses a sea whose bands the row cannot hold or whose grids differ in
// size.
func (encoded *scene) sea(packed arid.SeaPacked, space maths.Subspace,
	material int, clear bool) error {
	if err := seaRefused(packed.Bands); err != nil {
		return err
	}
	first := len(encoded.heights.bytes) / wordBytes
	sides := make([]float64, 0, seaBandsMost)
	for _, band := range packed.Bands {
		encoded.heights.grid(band.Height)
		encoded.heights.grid(band.SlopeX)
		encoded.heights.grid(band.SlopeY)
		sides = append(sides, band.Extent)
	}
	given := placedAt(space)
	given["data"] = wholes(first, packed.Bands[0].N, material, flag(clear))
	given["size"] = []float64{0, -packed.Reach, packed.Reach,
		packed.Steepest}
	given["normal"] = wholes(0, 0, 0)
	given["sides"] = sides
	given["sea"] = []float64{packed.Touch, float64(packed.MostSteps),
		float64(len(packed.Bands)), float64(flag(scrooge.Asked()))}
	given["slopes"] = packed.Slopes
	given["shown"] = []float64{packed.ShownWhole, packed.ShownNone}
	encoded.fields.row(fieldStruct, given)
	encoded.counts.fields++
	return nil
}

// seaRefused is why the kernel cannot hold a sea's bands, or nothing: none,
// more than its row has room for, or grids of different sizes, since the
// row carries one grid size for all.
func seaRefused(bands []*oceantex.Field) error {
	if len(bands) == 0 || len(bands) > seaBandsMost {
		return unsupported{fmt.Sprintf("a sea of %d bands; the kernel "+
			"holds one to %d", len(bands), seaBandsMost)}
	}
	for _, band := range bands {
		if band.N != bands[0].N {
			return unsupported{"a sea whose bands' grids differ in size"}
		}
	}
	return nil
}

// grid appends a grid of numbers as floats.
func (buffer *packer) grid(numbers []float64) {
	for _, number := range numbers {
		buffer.word(math.Float32bits(float32(number)))
	}
}
