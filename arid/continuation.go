package arid

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// maxFlatMeanCollarMetresPerCoreExtent bounds the continuation collar to one
// core width on each side. It is a work/bounds ceiling, not a physical scale.
const maxFlatMeanCollarMetresPerCoreExtent = 1

// FlatMeanContinuation extends one finite sampled surface through a smooth
// collar to a flat, authored mean level. The core remains the exact same
// bilinear height function. The collar is deliberately an approximation: it
// damps the core's edge waves and the returned shape still has a finite outer
// boundary.
//
// This shape is CPU-only. The Metal encoder refuses its type rather than
// drawing a different boundary.
type FlatMeanContinuation struct {
	heights                    []float64
	samples                    int
	coreExtent, half, width    float64
	outerHalf, mean, top, base float64
	slant, gradient, curvature float64
	flatAtMean                 bool
	boundedEdge                bool
	// core is the sampled heights as a height field, for exact hits. It
	// shares heights.
	core *shape.HeightField
}

// InfiniteMeanContinuation extends the same authored core and smooth collar
// to an unbounded flat mean-level surface. The mean plane is the chosen
// far-field interface; it is not a measured infinite ocean.
type InfiniteMeanContinuation struct {
	finite *FlatMeanContinuation
	core   *shape.HeightField
	smooth *shape.HeightField
}

// WithApproximateSmoothNormals interpolates the core's sampled slopes
// for optical directions on the CPU. Heights, bounds and hits remain
// those of the bilinear interface. The outer collar uses its own slope.
func (surface *InfiniteMeanContinuation) WithApproximateSmoothNormals() (*InfiniteMeanContinuation, error) {
	if surface == nil || surface.core == nil {
		return nil, fmt.Errorf("scene/ocean: no continuation core to smooth")
	}
	smooth, err := surface.core.WithApproximateSmoothNormals()
	if err != nil {
		return nil, err
	}
	made := *surface
	made.smooth = smooth
	return &made, nil
}

// ExactPlaneLevel reports the one exact degenerate case: every core sample
// equals the authored far-field mean. Its bilinear core has zero slope and
// the collar blends equal constant values, so the whole shape is one plane.
// No tolerance is used: any nonzero displacement remains a continuation.
func (surface *InfiniteMeanContinuation) ExactPlaneLevel() (float64, bool) {
	if surface == nil || surface.finite == nil || surface.core == nil {
		return 0, false
	}
	return surface.finite.mean, surface.finite.flatAtMean
}

// infiniteContinuationSteps bounds work in the nonlinear smooth collar when
// the exact core or far-field plane does not supply the first hit.
const infiniteContinuationSteps = 1 << 15

// NewInfiniteMeanContinuation creates an explicit CPU continuation whose
// collar reaches an infinite flat mean-level surface. The sampled core and
// collar are identical to NewFlatMeanContinuation; only the far field and
// medium footprint differ.
func NewInfiniteMeanContinuation(core *shape.HeightField,
	meanLevelMetres, collarWidthMetres float64,
) (*InfiniteMeanContinuation, error) {
	return newInfiniteMeanContinuation(core, meanLevelMetres, collarWidthMetres, false)
}

// NewBoundedInfiniteMeanContinuation limits boundary slope extrapolation to
// one grid spacing while smoothly joining the core to the flat far field.
func NewBoundedInfiniteMeanContinuation(core *shape.HeightField,
	meanLevelMetres, collarWidthMetres float64,
) (*InfiniteMeanContinuation, error) {
	return newInfiniteMeanContinuation(core, meanLevelMetres, collarWidthMetres, true)
}

// newInfiniteMeanContinuation is the finite continuation of a core, with a
// copy of the core, made unbounded sideways. It refuses what the finite one
// refuses.
func newInfiniteMeanContinuation(core *shape.HeightField,
	meanLevelMetres, collarWidthMetres float64, boundedEdge bool,
) (*InfiniteMeanContinuation, error) {
	finite, err := newFlatMeanContinuation(core, meanLevelMetres,
		collarWidthMetres, boundedEdge)
	if err != nil {
		return nil, err
	}
	heights := append([]float64(nil), finite.heights...)
	core, err = shape.NewHeightField(heights, finite.samples, finite.coreExtent)
	if err != nil {
		return nil, fmt.Errorf("scene/ocean: copy infinite continuation core: %w", err)
	}
	return &InfiniteMeanContinuation{finite: finite, core: core}, nil
}

// Bounds keeps the finite vertical range while extending horizontally without
// an edge. This lets CPU tracing enter the ocean boundary from any footprint.
func (surface *InfiniteMeanContinuation) Bounds() maths.Box {
	if surface == nil || surface.finite == nil {
		return maths.Box{}
	}
	infinity := math.Inf(1)
	return maths.Box{Min: maths.V(math.Inf(-1), surface.finite.base, math.Inf(-1)),
		Max: maths.V(infinity, surface.finite.top, infinity)}
}

// Distance is a conservative signed vertical gap to the continuous surface.
// Unlike the finite collar, the mean plane continues beyond every x/z edge.
func (surface *InfiniteMeanContinuation) Distance(point maths.Vec) float64 {
	if surface == nil || surface.finite == nil {
		return math.Inf(1)
	}
	height, _, _ := surface.finite.sample(point.X, point.Z)
	return (point.Y - height) * surface.finite.slant
}

// Normal is the gradient of the inherited core/collar and is up in the
// infinite flat mean plane.
func (surface *InfiniteMeanContinuation) Normal(point maths.Vec) maths.Vec {
	if surface == nil || surface.finite == nil {
		return maths.Up
	}
	exact := surface.finite.Normal(point)
	if surface.smooth == nil {
		return exact
	}
	edge := math.Max(math.Abs(point.X), math.Abs(point.Z))
	half := surface.finite.half
	blend := 1 - maths.Smooth(half*.875, half, edge)
	if blend == 0 {
		return exact
	}
	return exact.Scale(1 - blend).Add(surface.smooth.Normal(point).Scale(blend)).Unit()
}

// OffsetToSide keeps an interface departure on the side requested by offset.
// A normal offset can cross a sharp bilinear crease and land above the adjacent
// face despite pointing into water. Reduce only that numerical offset; the
// surface, normal and outgoing ray direction remain unchanged.
func (surface *InfiniteMeanContinuation) OffsetToSide(point, offset maths.Vec) maths.Vec {
	if surface == nil || surface.finite == nil || offset.Y == 0 {
		return point.Add(offset)
	}
	for range 32 {
		candidate := point.Add(offset)
		height, _, _ := surface.finite.sample(candidate.X, candidate.Z)
		gap := candidate.Y - height
		if (offset.Y < 0 && gap < 0) || (offset.Y > 0 && gap > 0) {
			return candidate
		}
		offset = offset.Scale(.5)
	}
	// At coordinate precision a normal displacement can round back onto the
	// boundary. One representable vertical step establishes the requested side.
	height, _, _ := surface.finite.sample(point.X, point.Z)
	side := math.Copysign(math.Inf(1), offset.Y)
	return maths.V(point.X, math.Nextafter(height, side), point.Z)
}

// Hit returns the first intersection with the core, smooth collar, or infinite
// mean plane. The flat far field is solved analytically; the nonlinear collar
// uses seam-aware adaptive root isolation with a local curvature bound.
func (surface *InfiniteMeanContinuation) Hit(ray maths.Ray, near,
	far float64,
) (float64, bool) {
	directionLength := ray.Dir.Len()
	if surface == nil || surface.finite == nil || surface.core == nil ||
		!(near < far) || math.IsNaN(near) || math.IsNaN(far) ||
		!continuationFinite(ray.At.X, ray.At.Y, ray.At.Z, ray.Dir.X, ray.Dir.Y,
			ray.Dir.Z, directionLength) || directionLength == 0 {
		return 0, false
	}
	bounded := surface.finite
	if bounded.flatAtMean {
		if ray.Dir.Y == 0 {
			return 0, false
		}
		along := (bounded.mean - ray.At.Y) / ray.Dir.Y
		return along, along > near && along < far
	}
	plane, hasPlane := surface.meanPlaneHit(ray, near, far)
	if hasPlane && surface.staysBeyondCollar(ray, near, plane) {
		return plane, true
	}
	limit := far
	if hasPlane && plane < limit {
		limit = plane
	}
	outer := bounded.outerHalf
	box := maths.Box{Min: maths.V(-outer, bounded.base, -outer),
		Max: maths.V(outer, bounded.top, outer)}
	start, end, meets := box.Hit(ray, near, limit)
	if meets {
		// The sampled core is piecewise bilinear and has an exact per-cell
		// intersection solver. Use it directly; conservative advancement can
		// require thousands of tiny steps for a ray that nearly follows a
		// wave before crossing it.
		coreBox := maths.Box{Min: maths.V(-bounded.half, math.Inf(-1), -bounded.half),
			Max: maths.V(bounded.half, math.Inf(1), bounded.half)}
		coreStart, coreEnd, crossesCore := coreBox.Hit(ray, start, end)
		coreHit, hasCoreHit := surface.core.IntersectExact(ray, near, limit)
		if crossesCore {
			justBefore := math.Nextafter(coreStart, math.Inf(-1))
			beforeCore := math.Min(end, justBefore)
			if start <= beforeCore {
				if hit, ok := surface.hitCoreOrCollar(ray, near, start,
					beforeCore); ok && (!hasCoreHit || hit < coreHit) {
					return hit, true
				}
			}
		}
		if hasCoreHit {
			return coreHit, true
		}
		if crossesCore {
			justAfter := math.Nextafter(coreEnd, math.Inf(1))
			afterCore := math.Max(start, justAfter)
			if afterCore <= end {
				if hit, ok := surface.hitCoreOrCollar(ray, near,
					afterCore, end); ok {
					return hit, true
				}
			}
		} else if hit, ok := surface.hitCoreOrCollar(ray, near, start, end); ok {
			return hit, true
		}
	}
	if hasPlane {
		return plane, true
	}
	return 0, false
}

// meanPlaneHit returns the ray distance to the flat mean plane when that hit
// lies outside the core.
func (surface *InfiniteMeanContinuation) meanPlaneHit(ray maths.Ray,
	near, far float64,
) (float64, bool) {
	if ray.Dir.Y == 0 {
		return 0, false
	}
	along := (surface.finite.mean - ray.At.Y) / ray.Dir.Y
	if !(along > near && along < far) {
		return 0, false
	}
	point := ray.Point(along)
	if !continuationFinite(point.X, point.Y, point.Z) {
		return 0, false
	}
	dx := math.Max(math.Abs(point.X)-surface.finite.half, 0)
	dz := math.Max(math.Abs(point.Z)-surface.finite.half, 0)
	if math.Hypot(dx, dz) < surface.finite.width {
		return 0, false
	}
	return along, true
}

// staysBeyondCollar cheaply proves a segment stays in the analytic plane.
func (surface *InfiniteMeanContinuation) staysBeyondCollar(ray maths.Ray,
	from, to float64,
) bool {
	outer := surface.finite.outerHalf
	start, end := ray.Point(from), ray.Point(to)
	return math.Min(start.X, end.X) >= outer ||
		math.Max(start.X, end.X) <= -outer ||
		math.Min(start.Z, end.Z) >= outer ||
		math.Max(start.Z, end.Z) <= -outer
}

// hitCoreOrCollar finds the first core or collar hit between start and end.
func (surface *InfiniteMeanContinuation) hitCoreOrCollar(ray maths.Ray,
	near, start, end float64,
) (float64, bool) {
	if !(start <= end) {
		return 0, false
	}
	length := ray.Dir.Len()
	hit, ok, complete := surface.hitCollarBySubdivision(ray, near, start, end,
		length)
	if ok {
		return hit, true
	}
	if complete {
		return 0, false
	}
	// Find a candidate upper bound at a fraction of the authored core cell
	// spacing. This pass is not an intersection answer: it only brackets a
	// sign change. The interval before that bracket must still be completely
	// isolated left-to-right before returning any hit.
	bound, bounded := surface.collarCandidateBound(ray, start, end, length)
	if !bounded {
		bound, bounded = surface.conservativeCollarBound(ray, near, start, end,
			length)
	}
	if !bounded || !(bound > start) {
		return 0, false
	}
	if hit, found, done := surface.hitCollarBySubdivision(ray, near, start,
		bound, length); found {
		return hit, true
	} else if !done {
		return 0, false
	}
	return 0, false
}

// collarCandidateBound scans the collar at fixed steps for a sign change and
// returns the step where it appears.
func (surface *InfiniteMeanContinuation) collarCandidateBound(ray maths.Ray,
	start, end, length float64,
) (float64, bool) {
	if !(start < end) || !(length > 0) || !finite(length) {
		return 0, false
	}
	step := surface.finite.coreExtent / float64(surface.finite.samples-1)
	// At eight intervals per core cell, this samples the finite bilinear field
	// and its smooth collar densely enough to locate broad crossings. It is
	// deliberately only a candidate generator; adaptive interval isolation
	// establishes first-hit ordering before its candidate can be accepted.
	maxDelta := step / 8 / length
	if !(maxDelta > 0) || !finite(maxDelta) {
		return 0, false
	}
	previousAlong := start
	previous := surface.Distance(ray.Point(start))
	for along := math.Min(end, start+maxDelta); along > previousAlong; {
		gap := surface.Distance(ray.Point(along))
		if (gap < 0) != (previous < 0) {
			return along, true
		}
		previousAlong, previous = along, gap
		if along == end {
			break
		}
		along = math.Min(end, along+maxDelta)
	}
	return 0, false
}

// conservativeCollarBound walks the collar with a conservative step and
// returns the first sign change or near-zero residual.
func (surface *InfiniteMeanContinuation) conservativeCollarBound(ray maths.Ray,
	near, start, end, length float64,
) (float64, bool) {
	if !(length > 0) || !finite(length) {
		return 0, false
	}
	along := start
	previous := surface.Distance(ray.Point(along))
	for step := 0; step < infiniteContinuationSteps && along <= end; step++ {
		point := ray.Point(along)
		gap := surface.Distance(point)
		tolerance := continuationResidualTolerance(point, gap)
		if along > near && math.Abs(gap) <= tolerance {
			return along, true
		}
		if step > 0 && (gap < 0) != (previous < 0) {
			return continuationBisect(surface, ray, near,
				along-alongStep(previous, length), along)
		}
		previous = gap
		advance := math.Abs(gap) / length
		if !(advance > 0) || !finite(advance) {
			advance = continuationNextParameter(ray, along)
		}
		next := math.Min(end, along+advance)
		if !(next > along) {
			next = math.Min(end, continuationNextParameter(ray, along))
		}
		if !(next > along) {
			break
		}
		along = next
	}
	return 0, false
}

// hitCollarBySubdivision searches the collar by splitting intervals until each
// is proven empty or holds a hit.
func (surface *InfiniteMeanContinuation) hitCollarBySubdivision(ray maths.Ray,
	near, start, end, length float64,
) (float64, bool, bool) {
	if !(start < end) || !(length > 0) || !continuationFinite(start, end,
		length, ray.Dir.X, ray.Dir.Y, ray.Dir.Z) {
		return 0, false, true
	}
	type interval struct {
		low, high       float64
		lowGap, highGap float64
		lowRaw, highRaw float64
	}
	residual := func(along float64) float64 {
		point := ray.Point(along)
		height, _, _ := surface.finite.sample(point.X, point.Z)
		return point.Y - height
	}
	stack := []interval{{low: start, high: end,
		lowGap:  surface.Distance(ray.Point(start)),
		highGap: surface.Distance(ray.Point(end)),
		lowRaw:  residual(start), highRaw: residual(end)}}
	for steps := 0; len(stack) > 0; steps++ {
		if steps >= infiniteContinuationSteps {
			return 0, false, false
		}
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current.low > near && math.Abs(current.lowRaw) <=
			continuationResidualTolerance(ray.Point(current.low), current.lowRaw) {
			return current.low, true, true
		}
		span := length * (current.high - current.low)
		if !finite(span) {
			span = math.Inf(1)
		}
		if seam := surface.nextCoreGridSeam(ray, current.low, current.high); seam > current.low && seam < current.high {
			// The bilinear base is continuous but its gradient can jump at
			// sample-grid lines. Curvature bounds apply only within one cell.
			seamGap := surface.Distance(ray.Point(seam))
			seamRaw := residual(seam)
			stack = append(stack,
				interval{low: seam, high: current.high,
					lowGap: seamGap, highGap: current.highGap,
					lowRaw: seamRaw, highRaw: current.highRaw},
				interval{low: current.low, high: seam,
					lowGap: current.lowGap, highGap: seamGap,
					lowRaw: current.lowRaw, highRaw: seamRaw})
			continue
		}
		if (current.lowGap < 0) == (current.highGap < 0) {
			middle := current.low + (current.high-current.low)/2
			middlePoint := ray.Point(middle)
			_, slopeX, slopeZ := surface.finite.sample(middlePoint.X,
				middlePoint.Z)
			dirX, dirY, dirZ := ray.Dir.X/length, ray.Dir.Y/length,
				ray.Dir.Z/length
			derivative := dirY - slopeX*dirX - slopeZ*dirZ
			curvature := surface.finite.curvature * (dirX*dirX + dirZ*dirZ)
			variation := curvature * span / 2
			// The directional derivative can be bounded locally by the
			// authored surface Hessian. If it keeps one sign, equal-sign
			// endpoints prove the whole interval root-free, even for a shallow
			// ray that hugs the interface and defeats the global slope bound.
			if math.Abs(derivative) > variation {
				continue
			}
			// Otherwise use a local Lipschitz cone. Distance is scaled by
			// slant, and its directional derivative varies by at most the
			// Hessian bound over half this interval.
			localLipschitz := surface.finite.slant *
				(math.Abs(derivative) + variation)
			above := math.Nextafter(localLipschitz, math.Inf(1))
			localLipschitz = math.Min(1, above)
			if math.Abs(current.lowGap)+math.Abs(current.highGap) >
				localLipschitz*span {
				continue
			}
		}
		middle := current.low + (current.high-current.low)/2
		if !(middle > current.low && middle < current.high) {
			if (current.lowGap < 0) != (current.highGap < 0) {
				hit, found := continuationBisect(surface, ray, near,
					current.low, current.high)
				return hit, found, true
			}
			if current.high > near && math.Abs(current.highRaw) <=
				continuationResidualTolerance(ray.Point(current.high), current.highRaw) {
				return current.high, true, true
			}
			continue
		}
		middleGap := surface.Distance(ray.Point(middle))
		middleRaw := residual(middle)
		// The stack is LIFO; push right first so the entire earlier interval
		// is proven empty or hit before visiting the later half.
		stack = append(stack,
			interval{low: middle, high: current.high,
				lowGap: middleGap, highGap: current.highGap,
				lowRaw: middleRaw, highRaw: current.highRaw},
			interval{low: current.low, high: middle,
				lowGap: current.lowGap, highGap: middleGap,
				lowRaw: current.lowRaw, highRaw: middleRaw})
	}
	return 0, false, true
}

// nextCoreGridSeam returns the ray distance to the next core grid line after
// start, or infinity when none lies before end.
func (surface *InfiniteMeanContinuation) nextCoreGridSeam(ray maths.Ray,
	start, end float64,
) float64 {
	step := surface.finite.coreExtent / float64(surface.finite.samples-1)
	next := math.Inf(1)
	for _, axis := range [][2]float64{{ray.At.X, ray.Dir.X},
		{ray.At.Z, ray.Dir.Z}} {
		origin, direction := axis[0], axis[1]
		if direction == 0 {
			continue
		}
		coordinate := origin + direction*start
		grid := (coordinate + surface.finite.half) / step
		var index float64
		if direction > 0 {
			grid = math.Nextafter(grid, math.Inf(1))
			index = math.Floor(grid) + 1
		} else {
			grid = math.Nextafter(grid, math.Inf(-1))
			index = math.Ceil(grid) - 1
		}
		if index <= 0 || index >= float64(surface.finite.samples-1) {
			continue
		}
		line := -surface.finite.half + index*step
		along := start + (line-coordinate)/direction
		if along > start && along < end {
			next = math.Min(next, along)
		}
	}
	return next
}

// continuationResidualTolerance is the rounding allowance for a residual at
// point, scaled to the point's magnitude.
func continuationResidualTolerance(point maths.Vec, gap float64) float64 {
	widest := math.Max(math.Abs(point.Z), math.Abs(gap))
	widest = math.Max(math.Abs(point.Y), widest)
	widest = math.Max(math.Abs(point.X), widest)
	scale := math.Max(1, widest)
	return 8 * (math.Nextafter(1, 2) - 1) * scale
}

// continuationFinite reports whether every value is a real number.
func continuationFinite(values ...float64) bool {
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
}

// continuationNextParameter returns the next ray distance after along at which
// a coordinate reaches its next representable boundary.
func continuationNextParameter(ray maths.Ray, along float64) float64 {
	point := ray.Point(along)
	next := math.Inf(1)
	for _, axis := range [][2]float64{{point.X, ray.Dir.X},
		{point.Y, ray.Dir.Y}, {point.Z, ray.Dir.Z}} {
		if axis[1] == 0 {
			continue
		}
		direction := math.Inf(1)
		if axis[1] < 0 {
			direction = math.Inf(-1)
		}
		coordinate := math.Nextafter(axis[0], direction)
		candidate := along + (coordinate-axis[0])/axis[1]
		if candidate > along {
			next = math.Min(next, candidate)
		}
	}
	if math.IsInf(next, 1) {
		return math.Nextafter(along, math.Inf(1))
	}
	return next
}

// continuationBisect narrows a sign change between low and high, and returns
// the midpoint of the final bracket and whether it lies beyond near.
func continuationBisect(surface *InfiniteMeanContinuation, ray maths.Ray,
	near, low, high float64,
) (float64, bool) {
	lowGap := surface.Distance(ray.Point(low))
	for range 64 {
		middle := low + (high-low)/2
		if !(middle > low && middle < high) {
			break
		}
		gap := surface.Distance(ray.Point(middle))
		if (gap < 0) == (lowGap < 0) {
			low, lowGap = middle, gap
		} else {
			high = middle
		}
	}
	hit := low + (high-low)/2
	return hit, hit > near
}

// alongStep converts a distance into a ray parameter step for a ray of the
// given length.
func alongStep(previous, length float64) float64 {
	return math.Abs(previous) / length
}

// NewFlatMeanContinuation wraps core in a flat-mean collar. meanLevelMetres
// is the explicitly chosen far-field level in the core's local y coordinate;
// collarWidthMetres is both the blend width and the added extent on each side.
// The collar width is limited to one core width. Existing core samples and
// their sampling quality are not changed or reduced. The returned shape owns
// a copy of the core samples so later caller mutation cannot invalidate bounds.
func NewFlatMeanContinuation(core *shape.HeightField, meanLevelMetres,
	collarWidthMetres float64,
) (*FlatMeanContinuation, error) {
	return newFlatMeanContinuation(core, meanLevelMetres, collarWidthMetres, false)
}

// newFlatMeanContinuation is a core height field extended through a smooth
// collar of a width in metres to a flat mean level. It refuses a nil core,
// a level or width that is not finite, and a width not above nought.
func newFlatMeanContinuation(core *shape.HeightField, meanLevelMetres,
	collarWidthMetres float64, boundedEdge bool,
) (*FlatMeanContinuation, error) {
	if core == nil || !finite(meanLevelMetres) || !finite(collarWidthMetres) ||
		collarWidthMetres <= 0 {
		return nil, errors.New("scene/ocean: invalid flat-mean continuation inputs")
	}
	if mode := core.ApproximateShadingNormalMode(); mode != "" {
		return nil, fmt.Errorf("scene/ocean: flat-mean continuation cannot preserve shading-normal %q",
			mode)
	}
	heights, samples, extent, coreBottom, coreTop, _ := core.Pack()
	if samples < 2 || samples > MaxHeightFieldSamples ||
		len(heights) != samples*samples || !finite(extent) || extent <= 0 ||
		!finite(coreBottom) || !finite(coreTop) || coreBottom > coreTop ||
		collarWidthMetres > maxFlatMeanCollarMetresPerCoreExtent*extent {
		return nil, errors.New("scene/ocean: core or collar exceeds its finite bounds")
	}
	step := extent / float64(samples-1)
	if !finite(step) || step <= 0 {
		return nil, errors.New("scene/ocean: invalid core sample spacing")
	}

	maxX, maxZ, maxCross := 0.0, 0.0, 0.0
	maxDeviation := 0.0
	for _, height := range heights {
		if !finite(height) {
			return nil, errors.New("scene/ocean: nonfinite core height")
		}
		maxDeviation = math.Max(maxDeviation, math.Abs(height-meanLevelMetres))
	}
	for row := 0; row < samples; row++ {
		for column := 0; column < samples; column++ {
			at := row*samples + column
			if column+1 < samples {
				maxX = math.Max(maxX,
					math.Abs(heights[at+1]-heights[at])/step)
			}
			if row+1 < samples {
				maxZ = math.Max(maxZ,
					math.Abs(heights[at+samples]-heights[at])/step)
			}
			if column+1 < samples && row+1 < samples {
				cross := heights[at+samples+1] - heights[at+1] -
					heights[at+samples] + heights[at]
				maxCross = math.Max(maxCross, math.Abs(cross)/(step*step))
			}
		}
	}
	if !finite(maxX) || !finite(maxZ) || !finite(maxCross) {
		return nil, errors.New("scene/ocean: nonfinite core slope bound")
	}

	width := collarWidthMetres
	extension := width
	if boundedEdge {
		extension = math.Min(width, step)
	}
	boundX := math.Nextafter(maxX+extension*maxCross, math.Inf(1))
	boundZ := math.Nextafter(maxZ+extension*maxCross, math.Inf(1))
	extendedSlope := math.Nextafter(math.Hypot(boundX, boundZ), math.Inf(1))
	deviation := math.Nextafter(maxDeviation+extendedSlope*width, math.Inf(1))
	if boundedEdge {
		deviation = math.Nextafter(maxDeviation+extendedSlope*math.Sqrt2*extension, math.Inf(1))
	}
	gradient := math.Nextafter(extendedSlope+1.5*deviation/width, math.Inf(1))
	// Bound the operator norm of the surface Hessian throughout the smooth
	// collar. The radial blend has |f''| <= 6/w² and |f' Hess(r)| <= 6/w²;
	// its two mixed product terms contribute at most 3*|grad(base)|/w.
	extraCurvature := 0.0
	if boundedEdge {
		extraCurvature = (boundX + boundZ) / step
	}
	curvature := math.Nextafter(maxCross+extraCurvature+3*extendedSlope/width+
		12*deviation/(width*width), math.Inf(1))
	slant := 1 / math.Hypot(1, gradient)
	outerHalf := extent/2 + width
	base, top := meanLevelMetres-deviation, meanLevelMetres+deviation
	if !finite(outerHalf) || !finite(base) || !finite(top) ||
		!finite(gradient) || !finite(curvature) || !finite(slant) || slant <= 0 {
		return nil, errors.New("scene/ocean: continuation exceeds finite numeric bounds")
	}

	ownedHeights := append([]float64(nil), heights...)
	owned, err := shape.NewHeightField(ownedHeights, samples, extent)
	if err != nil {
		return nil, fmt.Errorf("scene/ocean: core as a height field: %w", err)
	}
	return &FlatMeanContinuation{heights: ownedHeights, samples: samples,
		coreExtent: extent, half: extent / 2, width: width,
		outerHalf: outerHalf, mean: meanLevelMetres,
		top: top, base: base, slant: slant, gradient: gradient,
		curvature: curvature, flatAtMean: maxDeviation == 0,
		boundedEdge: boundedEdge, core: owned}, nil
}

// Hit is the first intersection with the core or the collar, found as the
// infinite continuation finds it and not by marching Distance, so a ray at a
// shallow angle arrives. The surface ends at its square: where the first
// intersection is on the mean plane beyond the collar, there is none.
func (surface *FlatMeanContinuation) Hit(ray maths.Ray, near,
	far float64) (float64, bool) {
	if surface == nil || surface.core == nil {
		return 0, false
	}
	whole := &InfiniteMeanContinuation{finite: surface, core: surface.core}
	along, hit := whole.Hit(ray, near, far)
	if !hit {
		return 0, false
	}
	at := ray.Point(along)
	if math.Max(math.Abs(at.X), math.Abs(at.Z)) > surface.outerHalf {
		return 0, false
	}
	return along, true
}

// Bounds includes the entire finite core and continuation collar. Its y
// limits conservatively include the linearly extended edge cell and fade.
func (surface *FlatMeanContinuation) Bounds() maths.Box {
	return maths.Box{Min: maths.V(-surface.outerHalf, surface.base, -surface.outerHalf),
		Max: maths.V(surface.outerHalf, surface.top, surface.outerHalf)}
}

// Distance is a lower bound to the finite composite surface. Outside its
// square, the sideways distance prevents an unbounded plane from being implied.
func (surface *FlatMeanContinuation) Distance(point maths.Vec) float64 {
	x := math.Max(-surface.outerHalf, math.Min(surface.outerHalf, point.X))
	z := math.Max(-surface.outerHalf, math.Min(surface.outerHalf, point.Z))
	height, _, _ := surface.sample(x, z)
	vertical := (point.Y - height) * surface.slant
	outX := math.Max(math.Abs(point.X)-surface.outerHalf, 0)
	outZ := math.Max(math.Abs(point.Z)-surface.outerHalf, 0)
	sideways := math.Hypot(outX, outZ)
	if sideways > 0 {
		return math.Max(vertical, sideways)
	}
	return vertical
}

// Normal gives the exact gradient of the piecewise bilinear and smoothstep
// height function. At a core edge it uses the same boundary cell derivative as
// shape.HeightField.Normal.
func (surface *FlatMeanContinuation) Normal(point maths.Vec) maths.Vec {
	_, dx, dz := surface.sample(point.X, point.Z)
	return maths.V(-dx, 1, -dz).Unit()
}

// sample returns height and its east/south slopes. Within the core it uses the
// same interpolation and operation order as shape.HeightField. Outside, the
// edge bilinear cell is continued to match both value and slope, then smoothly
// blended to the authored mean level.
func (surface *FlatMeanContinuation) sample(x, z float64) (height, dx, dz float64) {
	base, baseX, baseZ := surface.edgeCell(x, z)
	dxOut := math.Max(math.Abs(x)-surface.half, 0)
	dzOut := math.Max(math.Abs(z)-surface.half, 0)
	distance := math.Hypot(dxOut, dzOut)
	if distance == 0 {
		return base, baseX, baseZ
	}
	if distance >= surface.width {
		return surface.mean, 0, 0
	}
	t := distance / surface.width
	blend := t * t * (3 - 2*t)
	blendSlope := 6 * t * (1 - t) / surface.width
	gradDistanceX, gradDistanceZ := 0.0, 0.0
	if dxOut > 0 {
		gradDistanceX = math.Copysign(dxOut/distance, x)
	}
	if dzOut > 0 {
		gradDistanceZ = math.Copysign(dzOut/distance, z)
	}
	height = base + blend*(surface.mean-base)
	dx = (1-blend)*baseX + (surface.mean-base)*blendSlope*gradDistanceX
	dz = (1-blend)*baseZ + (surface.mean-base)*blendSlope*gradDistanceZ
	return height, dx, dz
}

// edgeCell continues the boundary cell. The optional saturating coordinate
// preserves the edge slope while limiting extrapolation to one sample spacing.
func (surface *FlatMeanContinuation) edgeCell(x, z float64) (height, dx, dz float64) {
	step := surface.coreExtent / float64(surface.samples-1)
	mapCoordinate := func(value float64) (float64, float64) {
		distance := math.Abs(value) - surface.half
		if distance <= 0 {
			return value, 1
		}
		derivative := math.Exp(-distance / step)
		return math.Copysign(surface.half+step*(1-derivative), value), derivative
	}
	scaleX, scaleZ := 1.0, 1.0
	if surface.boundedEdge {
		x, scaleX = mapCoordinate(x)
		z, scaleZ = mapCoordinate(z)
	}
	gx := (x + surface.half) / step
	gz := (z + surface.half) / step
	columnFloor := int(math.Floor(gx))
	rowFloor := int(math.Floor(gz))
	column := max(0, min(columnFloor, surface.samples-2))
	row := max(0, min(rowFloor, surface.samples-2))
	tx := gx - float64(column)
	tz := gz - float64(row)
	at := row*surface.samples + column
	h00 := surface.heights[at]
	h10 := surface.heights[at+1]
	h01 := surface.heights[at+surface.samples]
	h11 := surface.heights[at+surface.samples+1]
	low := h00*(1-tx) + h10*tx
	upper := h01*(1-tx) + h11*tx
	height = low*(1-tz) + upper*tz
	dxLow := (h10 - h00) / step
	dxHigh := (h11 - h01) / step
	dzLow := (h01 - h00) / step
	dzHigh := (h11 - h10) / step
	dx = (dxLow*(1-tz) + dxHigh*tz) * scaleX
	dz = (dzLow*(1-tx) + dzHigh*tx) * scaleZ
	return height, dx, dz
}
