// Package emptyspace prepares conservative occupancy for interpolated grids.
// It does not alter a renderer's sample positions, transport or field layout.
package emptyspace

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// Field is one nonnegative scalar/vector grid, with interleaved channels.
// Several fields can cover density and independent emission without copying.
type Field struct {
	Values   []float32
	Channels int
}

// Options bounds retained occupancy and the number of rays in one packet.
// BrickSide is measured in grid samples, not physical scene metres.
type Options struct {
	BrickSide   int
	Bytes       int64
	Rays        int
	QueryBricks int
}

// Mask owns compact support metadata; it does not retain the input arrays.
// All fields contributing transport must participate in its construction.
type Mask struct {
	bounds  maths.Box
	cells   [3]int
	bricks  [3]int
	options Options
	words   []uint64
	stamp   [32]byte
	ready   bool
}

// Build marks a brick occupied if any supporting interpolation sample is
// nonzero. Exactly-zero support can be skipped; no faintness or extinction
// cutoff is used.
func Build(bounds maths.Box, cells [3]int, fields []Field, options Options) (*Mask, error) {
	bricks, words, err := layout(bounds, cells, fields, options)
	if err != nil {
		return nil, err
	}
	stamp, any, err := fingerprint(bounds, cells, fields, options.BrickSide)
	if err != nil {
		return nil, err
	}
	mask := &Mask{bounds: bounds, cells: cells, bricks: bricks,
		options: options, stamp: stamp, ready: true}
	if !any {
		return mask, nil
	}
	mask.words = make([]uint64, words)
	mask.populate(fields)
	return mask, nil
}

// Fingerprint identifies bounds, sampling layout, fields and brick policy.
// The caller still owns time, transform and renderer-version invalidation.
func (mask *Mask) Fingerprint() [32]byte { return mask.stamp }

// Bytes is retained bit-array payload, excluding the fixed descriptor.
func (mask *Mask) Bytes() int64 { return int64(len(mask.words)) * 8 }

// Dimensions returns the compact brick grid, not the original sample grid.
func (mask *Mask) Dimensions() [3]int { return mask.bricks }

// finiteVector checks components without overflowing a squared vector length.
func finiteVector(point maths.Vec) bool {
	return finite(point.X) && finite(point.Y) && finite(point.Z)
}

// finite rejects invalid bounds before coordinate/index conversion.
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

// validBox permits point/flat query boxes; field support must have positive
// extent.
func validBox(bounds maths.Box) bool {
	return finiteVector(bounds.Min) && finiteVector(bounds.Max) &&
		bounds.Min.X <= bounds.Max.X && bounds.Min.Y <= bounds.Max.Y && bounds.Min.Z <= bounds.Max.Z
}

// layout guards sample products and bit-array capacity before any allocation.
func layout(bounds maths.Box, cells [3]int, fields []Field,
	options Options,
) ([3]int, int, error) {
	var bricks [3]int
	size := bounds.Max.Sub(bounds.Min)
	if !validBox(bounds) || !finiteVector(size) || size.X <= 0 || size.Y <= 0 || size.Z <= 0 ||
		options.BrickSide < 1 || options.Bytes < 8 || options.Rays < 1 ||
		options.QueryBricks < 1 || len(fields) == 0 {
		return bricks, 0, fmt.Errorf("emptyspace: invalid grid, support or policy")
	}
	for axis, count := range cells {
		if count < 2 {
			return bricks, 0, fmt.Errorf("emptyspace: interpolation needs at least two samples per axis")
		}
		bricks[axis] = (count-1)/options.BrickSide + 1
	}
	if err := fieldLengths(cells, fields); err != nil {
		return bricks, 0, err
	}
	limit := min(options.Bytes/8, int64(math.MaxInt)/64) * 64
	if int64(bricks[0]) > limit/int64(bricks[1])/int64(bricks[2]) {
		return bricks, 0, fmt.Errorf("emptyspace: occupancy exceeds its %d-byte payload budget", options.Bytes)
	}
	count := bricks[0] * bricks[1] * bricks[2]
	return bricks, (count-1)/64 + 1, nil
}

// fieldLengths refuses mismatched grids before multiplying their dimensions.
func fieldLengths(cells [3]int, fields []Field) error {
	for _, field := range fields {
		if field.Channels < 1 || cells[0] > len(field.Values)/cells[1]/cells[2]/field.Channels ||
			cells[0]*cells[1]*cells[2]*field.Channels != len(field.Values) {
			return fmt.Errorf("emptyspace: field payload does not match its grid and channels")
		}
	}
	return nil
}
