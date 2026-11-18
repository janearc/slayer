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
	smoothX     []float64
	smoothZ     []float64
	shadingMode string
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

// HasApproximateSmoothNormals reports whether shading uses interpolated
// finite-difference vertex slopes instead of each bilinear cell's exact slope.
func (field *HeightField) HasApproximateSmoothNormals() bool {
	return field != nil && field.shadingMode == "bilinear-vertex-slopes" &&
		len(field.heights) > 0 &&
		len(field.smoothX) == len(field.heights) &&
		len(field.smoothZ) == len(field.heights)
}

// ApproximateShadingNormalMode names a shading-only normal approximation,
// or is empty when the exact bilinear cell normal is in use.
func (field *HeightField) ApproximateShadingNormalMode() string {
	if field == nil || len(field.heights) == 0 ||
		len(field.smoothX) != len(field.heights) ||
		len(field.smoothZ) != len(field.heights) {
		return ""
	}
	return field.shadingMode
}

// ApproximateShadingSlopes returns copies of the optional shading-only
// vertex slopes in local x and z axes. The returned slices are empty when
// this field uses its exact bilinear-cell normal. Copies keep a renderer
// from mutating the field's geometry or its shading record.
func (field *HeightField) ApproximateShadingSlopes() (x, z []float64, ok bool) {
	if field == nil || len(field.heights) == 0 ||
		len(field.smoothX) != len(field.heights) ||
		len(field.smoothZ) != len(field.heights) {
		return nil, nil, false
	}
	return append([]float64(nil), field.smoothX...),
		append([]float64(nil), field.smoothZ...), true
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
	if len(field.smoothX) != 0 {
		return field.smoothNormal(p)
	}
	return field.GeometricNormal(p)
}

// GeometricNormal is the exact normal of the bilinear cell under p. Optional
// interpolated slopes change Normal for shading only; refraction and TIR keep
// using this geometry normal.
func (field *HeightField) GeometricNormal(p maths.Vec) maths.Vec {
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

// WithApproximateSmoothNormals copies a field and changes only its shading
// normals. It estimates slopes at each grid vertex with one-sided differences
// at the outer edges and centered differences inside, then bilinearly blends
// those slopes. This normal is intentionally not the exact normal of the
// piecewise bilinear surface, and is not a measured or exact surface normal.
// Bounds, packed heights, and Distance remain those of the original field.
func (field *HeightField) WithApproximateSmoothNormals() (*HeightField, error) {
	if field == nil || field.n < 2 || len(field.heights) != field.n*field.n {
		return nil, errors.New("shape: approximate smooth normals need a valid height field")
	}
	step := field.extent / float64(field.n-1)
	if !(step > 0) || math.IsInf(step, 0) || math.IsNaN(step) {
		return nil, errors.New("shape: approximate smooth normals need finite sample spacing")
	}
	made := *field
	made.heights = append([]float64(nil), field.heights...)
	made.smoothX = make([]float64, len(field.heights))
	made.smoothZ = make([]float64, len(field.heights))
	made.shadingMode = "bilinear-vertex-slopes"
	for row := range field.n {
		for column := range field.n {
			at := row*field.n + column
			made.smoothX[at] = finiteDifference(field.heights, field.n,
				at, column, 1, step)
			made.smoothZ[at] = finiteDifference(field.heights, field.n,
				at, row, field.n, step)
			if math.IsNaN(made.smoothX[at]) || math.IsInf(made.smoothX[at], 0) ||
				math.IsNaN(made.smoothZ[at]) || math.IsInf(made.smoothZ[at], 0) {
				return nil, errors.New("shape: approximate smooth normals need finite vertex slopes")
			}
		}
	}
	return &made, nil
}

// WithApproximateSpectralNormals copies a height field and uses the supplied
// spectral slopes at each grid vertex for shading. xSlopes and zSlopes are
// derivatives in the field's local x and z axes, row by row. The resulting
// bilinearly interpolated normal is intentionally not the exact normal of
// the piecewise-bilinear geometry, and is not a measured surface normal.
// Heights, bounds, packed samples, and Distance remain unchanged.
func (field *HeightField) WithApproximateSpectralNormals(
	xSlopes, zSlopes []float64,
) (*HeightField, error) {
	if field == nil || field.n < 2 || len(field.heights) != field.n*field.n {
		return nil, errors.New("shape: spectral shading normals need a valid height field")
	}
	if len(xSlopes) != len(field.heights) || len(zSlopes) != len(field.heights) {
		return nil, errors.New("shape: spectral shading slopes must match the height field")
	}
	for i := range xSlopes {
		if math.IsNaN(xSlopes[i]) || math.IsInf(xSlopes[i], 0) ||
			math.IsNaN(zSlopes[i]) || math.IsInf(zSlopes[i], 0) {
			return nil, errors.New("shape: spectral shading slopes must be finite")
		}
	}
	made := *field
	made.heights = append([]float64(nil), field.heights...)
	made.smoothX = append([]float64(nil), xSlopes...)
	made.smoothZ = append([]float64(nil), zSlopes...)
	made.shadingMode = "spectral-slopes"
	return &made, nil
}

// finiteDifference is one vertex slope on the selected grid axis. Edges use
// adjacent one-sided samples; interior vertices use their two-sided span.
func finiteDifference(heights []float64, samples, at, coordinate, stride int,
	step float64) float64 {
	if coordinate == 0 {
		return (heights[at+stride] - heights[at]) / step
	}
	if coordinate == samples-1 {
		return (heights[at] - heights[at-stride]) / step
	}
	return (heights[at+stride] - heights[at-stride]) / (2 * step)
}

// smoothNormal interpolates the precomputed vertex slopes at p, then
// normalizes the shading direction.
func (field *HeightField) smoothNormal(p maths.Vec) maths.Vec {
	column, row, tx, tz := field.place(p.X, p.Z)
	at := row*field.n + column
	lowX := field.smoothX[at]*(1-tx) + field.smoothX[at+1]*tx
	highX := field.smoothX[at+field.n]*(1-tx) +
		field.smoothX[at+field.n+1]*tx
	lowZ := field.smoothZ[at]*(1-tx) + field.smoothZ[at+1]*tx
	highZ := field.smoothZ[at+field.n]*(1-tx) +
		field.smoothZ[at+field.n+1]*tx
	dx := lowX*(1-tz) + highX*tz
	dz := lowZ*(1-tz) + highZ*tz
	return maths.V(-dx, 1, -dz).Unit()
}
