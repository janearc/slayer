package shape

import (
	"errors"
	"math"

	"github.com/janearc/slayer/maths"
)

// leastSamples is how few samples a side a height field has: a cell has
// two.
const leastSamples = pair // defined: a cell is bounded by two samples a side

// lastCellBack is how far back from the last sample the last cell begins:
// a row of n samples has n-1 cells, and the last starts at n-2.
const lastCellBack = pair // defined: the last cell starts two from the end

// HeightField is a surface given as heights on a square grid: samples by
// samples over a square extent across, centred on the shape's own
// zero in x and z, heights along y, the surface running on straight
// lines between the samples and ending at the square's edge. Water
// from a wave field, ground from a survey: anything that is a height
// wherever you stand.
//
// Its heights are the slice it was made from, and are not copied. They
// must not be written once the field is made: its range and the slant
// that bounds its Distance were measured from them then, and a later
// write would leave Distance longer than the true distance.
type HeightField struct {
	heights     []float64
	smoothX     []float64
	smoothZ     []float64
	shadingMode string
	samples     int
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

// NewHeightField checks and measures a field: samples at least two, extent
// above nought, samples by samples finite heights, row by row, and an extent
// that leaves every cell a width. It keeps heights and does not copy them;
// the caller must not write them afterward.
func NewHeightField(heights []float64, samples int, extent float64) (
	*HeightField, error) {
	if samples < leastSamples || len(heights) != samples*samples ||
		!(extent > nought) || math.IsInf(extent, unity) {
		return nil, errors.New("shape: a height field is samples by " +
			"samples finite heights, samples at least two, over a positive " +
			"extent")
	}
	step := extent / float64(samples-unity)
	if !(step > nought) {
		return nil, errors.New("shape: a height field's extent is too " +
			"small for its samples: a cell has no width")
	}
	field := &HeightField{heights: heights, samples: samples, extent: extent,
		top: math.Inf(-unity), bottom: math.Inf(unity)}
	if !field.measureRange() {
		return nil, errors.New("shape: a height field is finite")
	}
	steepestX, steepestZ := steepestSlopes(heights, samples, step)
	field.slant = unity / math.Sqrt(unity+steepestX*steepestX+steepestZ*steepestZ)
	return field, nil
}

// measureRange finds the lowest and highest height, and says whether every
// height is a finite number.
func (field *HeightField) measureRange() bool {
	for _, height := range field.heights {
		if !finiteNumber(height) {
			return false
		}
		field.top = math.Max(field.top, height)
		field.bottom = math.Min(field.bottom, height)
	}
	return true
}

// steepestSlopes are the largest slope along x and the largest along z
// between neighbouring samples, taken separately: the two need not be in
// the same cell.
func steepestSlopes(heights []float64, samples int,
	step float64) (steepestX, steepestZ float64) {
	for row := range samples {
		for column := range samples {
			at := heights[row*samples+column]
			if column+unity < samples {
				rise := math.Abs(heights[row*samples+column+unity]-at) / step
				steepestX = math.Max(steepestX, rise)
			}
			if row+unity < samples {
				rise := math.Abs(heights[(row+unity)*samples+column]-at) / step
				steepestZ = math.Max(steepestZ, rise)
			}
		}
	}
	return steepestX, steepestZ
}

// Pack is the field as flat numbers, for a renderer that keeps it on
// a device: the samples row by row, how many a side, the extent, the
// lowest and highest sample, and the slant that flattens a vertical
// gap into a true distance bound. The slice is the field's own, and must
// not be written.
func (field *HeightField) Pack() (heights []float64, samples int,
	extent, bottom, top, slant float64) {
	return field.heights, field.samples, field.extent, field.bottom,
		field.top, field.slant
}

// hasSlopes is whether the field carries shading slopes for every sample.
func (field *HeightField) hasSlopes() bool {
	return field != nil && len(field.heights) > nought &&
		len(field.smoothX) == len(field.heights) &&
		len(field.smoothZ) == len(field.heights)
}

// ApproximateShadingNormalMode names a shading-only normal approximation,
// or is empty when the exact bilinear cell normal is in use.
func (field *HeightField) ApproximateShadingNormalMode() string {
	if !field.hasSlopes() {
		return ""
	}
	return field.shadingMode
}

// ApproximateShadingSlopes returns the optional shading-only vertex slopes
// in local x and z axes. They are empty, and ok is false, when this field
// uses its exact bilinear-cell normal. The slices are the field's own, not
// copies, so a scene's fields do not each hold a second set; they must not
// be written.
func (field *HeightField) ApproximateShadingSlopes() (x, z []float64, ok bool) {
	if !field.hasSlopes() {
		return nil, nil, false
	}
	return field.smoothX, field.smoothZ, true
}

// Bounds is the box round the square at its highest and lowest.
func (field *HeightField) Bounds() maths.Box {
	reach := field.extent * half
	return maths.Box{Min: maths.V(-reach, field.bottom, -reach),
		Max: maths.V(reach, field.top, reach)}
}

// cellBelow is the index of the cell a grid position falls in, held to the
// last cell. A position that is not a number is the first cell, and the
// fraction measured from it is still not a number, so the answer stays one.
func cellBelow(grid float64, samples int) int {
	if math.IsNaN(grid) {
		return nought
	}
	return min(int(grid), samples-lastCellBack)
}

// place is the grid cell under x and z and how far across it the
// point is, held to the square. A coordinate that is not a number gives a
// fraction that is not a number. A field that was not made by
// NewHeightField is refused, as a Mesh that was not made by NewMesh is.
func (field *HeightField) place(x, z float64) (column, row int,
	across, down float64) {
	if len(field.heights) == nought {
		panic("shape: a HeightField is made by NewHeightField, which " +
			"measures it")
	}
	step := field.extent / float64(field.samples-unity)
	gx := (x + field.extent*half) / step
	gz := (z + field.extent*half) / step
	last := float64(field.samples - unity)
	gx = math.Min(math.Max(gx, nought), last)
	gz = math.Min(math.Max(gz, nought), last)
	column = cellBelow(gx, field.samples)
	row = cellBelow(gz, field.samples)
	return column, row, gx - float64(column), gz - float64(row)
}

// height is the surface's height over x and z, on the straight lines
// between the four nearest samples, held to the square.
func (field *HeightField) height(x, z float64) float64 {
	column, row, across, down := field.place(x, z)
	at := row*field.samples + column
	low := field.heights[at]*(unity-across) + field.heights[at+unity]*across
	upper := field.heights[at+field.samples]*(unity-across) +
		field.heights[at+field.samples+unity]*across
	return low*(unity-down) + upper*down
}

// Distance is a lower bound on how far point is from the surface, and
// negative below it: the vertical gap flattened by the steepest
// gradient, and never less than the sideways gap to the square, since
// the surface ends there.
//
// It is a bound and no more, and a ray is not met by marching it. The slant
// is one number for the whole field, made from its steepest slope in x and
// its steepest in z wherever they are, so over level water beside one steep
// cell every step is shorter than it need be.
//
// A ray that comes to the
// surface at a shallow angle closes the gap by only about the slant times
// the sine of that angle each step, so it can use all the steps a marcher
// allows before it arrives. Hit answers every ray exactly, and is what
// tracing asks.
func (field *HeightField) Distance(point maths.Vec) float64 {
	vertical := (point.Y - field.height(point.X, point.Z)) * field.slant
	reach := field.extent * half
	outX := math.Max(math.Abs(point.X)-reach, nought)
	outZ := math.Max(math.Abs(point.Z)-reach, nought)
	sideways := math.Hypot(outX, outZ)
	if sideways > nought {
		return math.Max(vertical, sideways)
	}
	return vertical
}

// Normal is which way the surface faces over point: square to the cell's
// own slopes, exactly.
func (field *HeightField) Normal(point maths.Vec) maths.Vec {
	if len(field.smoothX) != nought {
		return field.smoothNormal(point)
	}
	return field.GeometricNormal(point)
}

// GeometricNormal is the exact normal of the bilinear cell under point. Optional
// interpolated slopes change Normal for shading only; refraction and TIR keep
// using this geometry normal.
func (field *HeightField) GeometricNormal(point maths.Vec) maths.Vec {
	column, row, across, down := field.place(point.X, point.Z)
	at := row*field.samples + column
	step := field.extent / float64(field.samples-unity)
	// the bilinear patch's slopes at the point, blended as its
	// heights are
	lowX := (field.heights[at+unity] - field.heights[at]) / step
	highX := (field.heights[at+field.samples+unity] -
		field.heights[at+field.samples]) / step
	lowZ := (field.heights[at+field.samples] - field.heights[at]) / step
	highZ := (field.heights[at+field.samples+unity] -
		field.heights[at+unity]) / step
	slopeX := lowX*(unity-down) + highX*down
	slopeZ := lowZ*(unity-across) + highZ*across
	return maths.V(-slopeX, unity, -slopeZ).Unit()
}

// WithApproximateSmoothNormals copies a field and changes only its shading
// normals. It estimates slopes at each grid vertex with one-sided differences
// at the outer edges and centered differences inside, then bilinearly blends
// those slopes. This normal is intentionally not the exact normal of the
// piecewise bilinear surface, and is not a measured or exact surface normal.
//
// Bounds, packed heights, and Distance remain those of the original field;
// the heights are shared with it, not copied.
func (field *HeightField) WithApproximateSmoothNormals() (*HeightField, error) {
	if field == nil || len(field.heights) == nought {
		return nil, errors.New("shape: approximate smooth normals need a valid height field")
	}
	made := *field
	made.smoothX = make([]float64, len(field.heights))
	made.smoothZ = make([]float64, len(field.heights))
	made.shadingMode = "bilinear-vertex-slopes"
	if !made.fillVertexSlopes() {
		return nil, errors.New("shape: approximate smooth normals need finite vertex slopes")
	}
	return &made, nil
}

// fillVertexSlopes estimates the slopes at every grid vertex from the
// heights, and says whether every one is a finite number.
func (field *HeightField) fillVertexSlopes() bool {
	step := field.extent / float64(field.samples-unity)
	for row := range field.samples {
		for column := range field.samples {
			at := row*field.samples + column
			slopeX := finiteDifference(field.heights, field.samples,
				at, column, unity, step)
			slopeZ := finiteDifference(field.heights, field.samples,
				at, row, field.samples, step)
			if !finiteNumber(slopeX) || !finiteNumber(slopeZ) {
				return false
			}
			field.smoothX[at], field.smoothZ[at] = slopeX, slopeZ
		}
	}
	return true
}

// WithApproximateSpectralNormals copies a height field and uses the supplied
// spectral slopes at each grid vertex for shading. xSlopes and zSlopes are
// derivatives in the field's local x and z axes, row by row. The resulting
// bilinearly interpolated normal is intentionally not the exact normal of
// the piecewise-bilinear geometry, and is not a measured surface normal.
//
// Heights, bounds, packed samples, and Distance remain unchanged; the
// heights are shared with the field, not copied.
func (field *HeightField) WithApproximateSpectralNormals(
	xSlopes, zSlopes []float64,
) (*HeightField, error) {
	if field == nil || len(field.heights) == nought {
		return nil, errors.New("shape: spectral shading normals need a valid height field")
	}
	if len(xSlopes) != len(field.heights) || len(zSlopes) != len(field.heights) {
		return nil, errors.New("shape: spectral shading slopes must match the height field")
	}
	for index := range xSlopes {
		if !finiteNumber(xSlopes[index]) || !finiteNumber(zSlopes[index]) {
			return nil, errors.New("shape: spectral shading slopes must be finite")
		}
	}
	made := *field
	made.smoothX = append([]float64(nil), xSlopes...)
	made.smoothZ = append([]float64(nil), zSlopes...)
	made.shadingMode = "spectral-slopes"
	return &made, nil
}

// finiteDifference is one vertex slope on the selected grid axis. Edges use
// adjacent one-sided samples; interior vertices use their two-sided span.
func finiteDifference(heights []float64, samples, at, coordinate, stride int,
	step float64) float64 {
	if coordinate == nought {
		return (heights[at+stride] - heights[at]) / step
	}
	if coordinate == samples-unity {
		return (heights[at] - heights[at-stride]) / step
	}
	return (heights[at+stride] - heights[at-stride]) / (pair * step)
}

// smoothNormal interpolates the precomputed vertex slopes at point, then
// normalizes the shading direction.
func (field *HeightField) smoothNormal(point maths.Vec) maths.Vec {
	column, row, across, down := field.place(point.X, point.Z)
	at := row*field.samples + column
	lowX := field.smoothX[at]*(unity-across) + field.smoothX[at+unity]*across
	highX := field.smoothX[at+field.samples]*(unity-across) +
		field.smoothX[at+field.samples+unity]*across
	lowZ := field.smoothZ[at]*(unity-across) + field.smoothZ[at+unity]*across
	highZ := field.smoothZ[at+field.samples]*(unity-across) +
		field.smoothZ[at+field.samples+unity]*across
	slopeX := lowX*(unity-down) + highX*down
	slopeZ := lowZ*(unity-down) + highZ*down
	return maths.V(-slopeX, unity, -slopeZ).Unit()
}
