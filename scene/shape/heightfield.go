package shape

import (
	"errors"
	"math"

	"github.com/janearc/slayer/maths"
)

// HeightField is a surface given as heights on a square grid: n by n
// samples over a square extent across, centred on the shape's own
// zero in x and z, heights along y, the surface running on straight
// lines between the samples and ending at the square's edge. Water
// from a wave field, ground from a survey: anything that is a height
// wherever you stand.
type HeightField struct {
	heights     []float64
	n           int
	extent      float64
	top, bottom float64
	// slant flattens the vertical gap into a true lower bound on the
	// distance: one over the root of one plus the gradient bound
	// squared, so the march never steps through the surface.
	//
	// Inside a cell the surface is bilinear, so its slope along x is a
	// blend of the cell's two edge slopes along x, and likewise along
	// z. The gradient is then at most the root of the largest x edge
	// slope squared plus the largest z edge slope squared.
	slant float64
}

// NewHeightField checks and measures a field: n at least two, extent
// above nought, n by n finite heights, row by row.
func NewHeightField(heights []float64, n int, extent float64) (
	*HeightField, error) {
	if n < 2 || len(heights) != n*n || !(extent > 0) ||
		math.IsInf(extent, 0) {
		return nil, errors.New("shape: a height field is n by n " +
			"finite heights, n at least two, over a positive extent")
	}
	field := &HeightField{heights: heights, n: n, extent: extent,
		top: math.Inf(-1), bottom: math.Inf(1)}
	for _, height := range heights {
		if math.IsNaN(height) || math.IsInf(height, 0) {
			return nil, errors.New("shape: a height field is finite")
		}
		field.top = math.Max(field.top, height)
		field.bottom = math.Min(field.bottom, height)
	}
	step := extent / float64(n-1)
	steepestX, steepestZ := 0.0, 0.0
	for row := 0; row < n; row++ {
		for column := 0; column < n; column++ {
			at := heights[row*n+column]
			if column+1 < n {
				rise := math.Abs(heights[row*n+column+1]-at) / step
				steepestX = math.Max(steepestX, rise)
			}
			if row+1 < n {
				rise := math.Abs(heights[(row+1)*n+column]-at) / step
				steepestZ = math.Max(steepestZ, rise)
			}
		}
	}
	field.slant = 1 / math.Sqrt(1+steepestX*steepestX+steepestZ*steepestZ)
	return field, nil
}

// Pack is the field as flat numbers, for a renderer that keeps it on
// a device: the samples row by row, how many a side, the extent, the
// lowest and highest sample, and the slant that flattens a vertical
// gap into a true distance bound.
func (field *HeightField) Pack() (heights []float64, n int,
	extent, bottom, top, slant float64) {
	return field.heights, field.n, field.extent, field.bottom,
		field.top, field.slant
}

// Bounds is the box round the square at its highest and lowest.
func (field *HeightField) Bounds() maths.Box {
	half := field.extent / 2
	return maths.Box{Min: maths.V(-half, field.bottom, -half),
		Max: maths.V(half, field.top, half)}
}

// place is the grid cell under x and z and how far across it the
// point is, held to the square.
func (field *HeightField) place(x, z float64) (column, row int,
	tx, tz float64) {
	step := field.extent / float64(field.n-1)
	gx := (x + field.extent/2) / step
	gz := (z + field.extent/2) / step
	last := float64(field.n - 1)
	gx = math.Min(math.Max(gx, 0), last)
	gz = math.Min(math.Max(gz, 0), last)
	column = min(int(gx), field.n-2)
	row = min(int(gz), field.n-2)
	return column, row, gx - float64(column), gz - float64(row)
}

// height is the surface's height over x and z, on the straight lines
// between the four nearest samples, held to the square.
func (field *HeightField) height(x, z float64) float64 {
	column, row, tx, tz := field.place(x, z)
	at := row*field.n + column
	low := field.heights[at]*(1-tx) + field.heights[at+1]*tx
	upper := field.heights[at+field.n]*(1-tx) +
		field.heights[at+field.n+1]*tx
	return low*(1-tz) + upper*tz
}

// Distance is a lower bound on how far p is from the surface, and
// negative below it: the vertical gap flattened by the steepest
// gradient, and never less than the sideways gap to the square, since
// the surface ends there.
func (field *HeightField) Distance(p maths.Vec) float64 {
	vertical := (p.Y - field.height(p.X, p.Z)) * field.slant
	half := field.extent / 2
	outX := math.Max(math.Abs(p.X)-half, 0)
	outZ := math.Max(math.Abs(p.Z)-half, 0)
	sideways := math.Hypot(outX, outZ)
	if sideways > 0 {
		return math.Max(vertical, sideways)
	}
	return vertical
}

// Normal is which way the surface faces over p: square to the cell's
// own slopes, exactly.
func (field *HeightField) Normal(p maths.Vec) maths.Vec {
	column, row, tx, tz := field.place(p.X, p.Z)
	at := row*field.n + column
	step := field.extent / float64(field.n-1)
	// the bilinear patch's slopes at the point, blended as its
	// heights are
	lowX := (field.heights[at+1] - field.heights[at]) / step
	highX := (field.heights[at+field.n+1] - field.heights[at+field.n]) / step
	lowZ := (field.heights[at+field.n] - field.heights[at]) / step
	highZ := (field.heights[at+field.n+1] - field.heights[at+1]) / step
	dx := lowX*(1-tz) + highX*tz
	dz := lowZ*(1-tx) + highZ*tx
	return maths.V(-dx, 1, -dz).Unit()
}
