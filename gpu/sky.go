package gpu

import (
	"encoding/binary"
	"math"

	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/skytable"
)

// skyOf is the world's sky as the kernel reads it to light surfaces:
// skytable's table of it, a float4 a cell, row by row. the kernel's
// sky_at reads it as skytable does. with no sky, a table of none.
func (encoded *scene) skyOf(world *thing.World) {
	cells := skytable.Of(world.Sky).Cells()
	table := make([]byte, 0, len(cells)*16)
	for _, light := range cells {
		for _, part := range []float64{light.R, light.G, light.B, 0} {
			table = binary.LittleEndian.AppendUint32(table,
				math.Float32bits(float32(part)))
		}
	}
	encoded.sky = packer{bytes: table}
}
