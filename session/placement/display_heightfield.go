package placement

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// maxDisplayHeightCells bounds one field before allocating its display mesh.
const maxDisplayHeightCells = 262144

// heightField connects the original grid samples as a labeled approximation.
func (builder *displayBuilder) heightField(field *shape.HeightField, at maths.Subspace, colour maths.Light) error {
	if field == nil {
		return fmt.Errorf("working height field is unavailable")
	}
	heights, samples, extent, _, _, _ := field.Pack()
	cells := samples - 1
	if cells < 1 || cells > maxDisplayHeightCells/cells || len(heights) != samples*samples {
		return fmt.Errorf("working height field exceeds the %d-cell limit or is invalid", maxDisplayHeightCells)
	}
	step := extent / float64(cells)
	point := func(x, z int) maths.Vec {
		return maths.V(-extent/2+float64(x)*step, heights[z*samples+x], -extent/2+float64(z)*step)
	}
	for z := 0; z < cells; z++ {
		for x := 0; x < cells; x++ {
			a, b, c, d := point(x, z), point(x+1, z), point(x, z+1), point(x+1, z+1)
			builder.triangle([3]maths.Vec{a, c, b}, [3]maths.Vec{}, false, at, colour)
			builder.triangle([3]maths.Vec{b, c, d}, [3]maths.Vec{}, false, at, colour)
		}
	}
	builder.skin.Notice = "Height field shown as a triangulated approximation between samples"
	return nil
}
