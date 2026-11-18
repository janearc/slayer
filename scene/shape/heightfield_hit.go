package shape

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// IntersectExact finds the first isolated intersection between r and the finite
// piecewise-bilinear surface. It traverses only the grid cells crossed by the
// ray; within each cell the ray/height equation is quadratic. It is an explicit
// query and does not opt ordinary HeightField scene tracing into Hitter
// dispatch; InfiniteMeanContinuation uses it for its sampled core.
//
// Like the analytic primitive hit tests, both ends of the requested interval
// are exclusive. A ray lying along a patch has no first isolated hit.
func (field *HeightField) IntersectExact(r maths.Ray, near, far float64) (float64, bool) {
	if field == nil || field.n < 2 || len(field.heights) != field.n*field.n ||
		!(near < far) || math.IsNaN(near) || math.IsNaN(far) {
		return 0, false
	}
	if !finiteHeightfieldVec(r.At) || !finiteHeightfieldVec(r.Dir) ||
		r.Dir == (maths.Vec{}) {
		return 0, false
	}
	half := field.extent / 2
	enter, leave := near, far
	if !clipHeightfieldAxis(r.At.X, r.Dir.X, -half, half, &enter, &leave) ||
		!clipHeightfieldAxis(r.At.Z, r.Dir.Z, -half, half, &enter, &leave) ||
		!(enter <= leave) {
		return 0, false
	}
	// X/Z traversal bounds the ray for every nonvertical path. A vertical
	// path can otherwise retain an infinite near or far; clip that case by
	// the field's finite height range. Avoid clipping ordinary rays to the
	// y slab, where rounded equal entry/exit values can exclude a true root.
	if math.IsInf(enter, 0) || math.IsInf(leave, 0) {
		if !clipHeightfieldAxis(r.At.Y, r.Dir.Y, field.bottom, field.top,
			&enter, &leave) || !(enter <= leave) {
			return 0, false
		}
	}
	if math.IsInf(enter, 0) {
		return 0, false
	}
	step := field.extent / float64(field.n-1)
	along := enter
	at := r.Point(along)
	column := heightfieldCell(at.X, half, step, field.n, r.Dir.X)
	row := heightfieldCell(at.Z, half, step, field.n, r.Dir.Z)
	// A ray can cross at most n-1 vertical and n-1 horizontal cell edges.
	// The bound also makes floating-point boundary degeneracies terminate.
	for visited := 0; visited < 2*field.n+2 && along <= leave; visited++ {
		nextX := heightfieldNextCell(along, r.At.X, r.Dir.X,
			column, half, step)
		nextZ := heightfieldNextCell(along, r.At.Z, r.Dir.Z,
			row, half, step)
		end := math.Min(leave, math.Min(nextX, nextZ))
		if hit, ok := field.hitCell(r, near, far, along, end,
			column, row, step, half); ok {
			return hit, true
		}
		if end >= leave || math.IsInf(end, 1) {
			break
		}
		// Advance the cell indices from the crossing itself. Reconstructing
		// them from ray.Point(Nextafter(end)) can select the old cell when the
		// coordinate rounds back to the grid edge.
		advanced := false
		if nextX <= end {
			column += sign(r.Dir.X)
			advanced = true
		}
		if nextZ <= end {
			row += sign(r.Dir.Z)
			advanced = true
		}
		if !advanced || column < 0 || column >= field.n-1 ||
			row < 0 || row >= field.n-1 {
			break
		}
		along = end
	}
	return 0, false
}

// finiteHeightfieldVec reports whether every component of v is a real number.
func finiteHeightfieldVec(v maths.Vec) bool {
	return !math.IsNaN(v.X) && !math.IsInf(v.X, 0) &&
		!math.IsNaN(v.Y) && !math.IsInf(v.Y, 0) &&
		!math.IsNaN(v.Z) && !math.IsInf(v.Z, 0)
}

// sign returns 1, -1 or 0 for the sign of value.
func sign(value float64) int {
	if value > 0 {
		return 1
	}
	if value < 0 {
		return -1
	}
	return 0
}

// clipHeightfieldAxis narrows the ray's entry and exit parameters to one axis
// slab, and reports whether any range is left.
func clipHeightfieldAxis(origin, direction, low, high float64,
	enter, leave *float64,
) bool {
	if math.IsNaN(origin) || math.IsNaN(direction) {
		return false
	}
	if direction == 0 {
		return origin >= low && origin <= high
	}
	one, two := (low-origin)/direction, (high-origin)/direction
	if one > two {
		one, two = two, one
	}
	*enter = math.Max(*enter, one)
	*leave = math.Min(*leave, two)
	return *enter <= *leave
}

// heightfieldCell returns the sample cell index holding at, clamped to the
// grid.
func heightfieldCell(at, half, step float64, samples int, direction float64) int {
	grid := (at + half) / step
	if direction < 0 && grid == math.Trunc(grid) {
		grid = math.Nextafter(grid, math.Inf(-1))
	}
	return min(max(int(math.Floor(grid)), 0), samples-2)
}

// heightfieldNextCell returns the ray distance to the next cell boundary on
// one axis, after along.
func heightfieldNextCell(along, origin, direction float64, cell int,
	half, step float64,
) float64 {
	if direction == 0 {
		return math.Inf(1)
	}
	boundary := -half + float64(cell)*step
	if direction > 0 {
		boundary += step
	}
	next := (boundary - origin) / direction
	if !(next > along) {
		return math.Nextafter(along, math.Inf(1))
	}
	return next
}

// hitCell solves the ray against one bilinear cell and returns the first hit
// inside the range.
func (field *HeightField) hitCell(r maths.Ray, near, far, start, end float64,
	column, row int, step, half float64,
) (float64, bool) {
	if !(start <= end) {
		return 0, false
	}
	base := row*field.n + column
	h00 := field.heights[base]
	dxHeight := field.heights[base+1] - h00
	dzHeight := field.heights[base+field.n] - h00
	dxzHeight := field.heights[base+field.n+1] - field.heights[base+1] -
		field.heights[base+field.n] + h00
	xCorner := -half + float64(column)*step
	zCorner := -half + float64(row)*step
	at := r.Point(start)
	qx, qz := (at.X-xCorner)/step, (at.Z-zCorner)/step
	dx, dz := r.Dir.X/step, r.Dir.Z/step
	// F(u) = rayY(start+u) - h(rayX(start+u), rayZ(start+u)).
	c := at.Y - (h00 + dxHeight*qx + dzHeight*qz + dxzHeight*qx*qz)
	b := r.Dir.Y - (dxHeight*dx + dzHeight*dz +
		dxzHeight*(qx*dz+qz*dx))
	a := -dxzHeight * dx * dz
	if math.IsNaN(a) || math.IsInf(a, 0) || math.IsNaN(b) ||
		math.IsInf(b, 0) || math.IsNaN(c) || math.IsInf(c, 0) {
		return 0, false
	}
	width := end - start
	one, two, roots := heightfieldRoots(a, b, c)
	for index, root := range [2]float64{one, two} {
		if index >= roots {
			break
		}
		along := start + root
		if root >= 0 && root <= width && along > near && along < far {
			return along, true
		}
	}
	return 0, false
}

// heightfieldRoots returns the real roots of a quadratic, or none when any
// coefficient is not finite.
func heightfieldRoots(a, b, c float64) (one, two float64, count int) {
	if math.IsNaN(a) || math.IsInf(a, 0) || math.IsNaN(b) ||
		math.IsInf(b, 0) || math.IsNaN(c) || math.IsInf(c, 0) {
		return 0, 0, 0
	}
	scale := max(math.Abs(a), math.Abs(b), math.Abs(c))
	if scale == 0 {
		return 0, 0, 0
	}
	a, b, c = a/scale, b/scale, c/scale
	if a == 0 {
		if b == 0 {
			return 0, 0, 0
		}
		return -c / b, 0, 1
	}
	disc := math.FMA(-4*a, c, b*b)
	if math.IsNaN(disc) || math.IsInf(disc, 0) {
		return 0, 0, 0
	}
	if disc < 0 {
		errorBound := 8 * (math.Nextafter(1, 2) - 1) *
			(math.Abs(b*b) + math.Abs(4*a*c))
		if disc < -errorBound {
			return 0, 0, 0
		}
		disc = 0
	}
	square := math.Sqrt(disc)
	q := -0.5 * (b + math.Copysign(square, b))
	if q == 0 {
		return -b / (2 * a), 0, 1
	}
	one, two = q/a, c/q
	if one > two {
		one, two = two, one
	}
	return one, two, 2
}
