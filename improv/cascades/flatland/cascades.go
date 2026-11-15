package flatland

import (
	"errors"
	"math"
	"sync/atomic"
	"time"
)

// branching is the number of child directions each direction has in the
// next cascade. The paper uses alpha = 2 in two dimensions, which is four.
const branching = 4

// firstDirections is the number of directions per probe on cascade 0.
// The paper's run used four.
const firstDirections = 4

// minimumSegment is the length in cells below which a bilinear-fix segment
// is skipped as empty. Picked: far shorter than a march step.
const minimumSegment = 1e-9

// Level is one cascade. Its probes sit in a square grid with spacing
// Spacing cells. Probe number n along an axis is at (n + 0.5) * Spacing
// cells from the edge.
//
// So cascade 0 sits on the cell centres, cascade 1 on
// the corners shared by four cells, and every probe of cascade i+1 lies in
// the middle of a square of four probes of cascade i. A cascade-i probe
// therefore always has four cascade-(i+1) probes around it, and the ones
// beyond the grid edge are handled by clamping (paper figure 4).
type Level struct {
	Index         int
	Spacing       int
	ProbesPerSide int
	Directions    int
	Start, End    float64
}

// Plan is every cascade, nearest first.
type Plan []Level

// Rays is how many rays the cascade traces in the basic method.
func (level Level) Rays() int {
	return level.ProbesPerSide * level.ProbesPerSide * level.Directions
}

// Position is where probe number n along an axis sits, in cells.
func (level Level) Position(probe int) float64 {
	return (float64(probe) + 0.5) * float64(level.Spacing)
}

// Direction is the unit vector of direction number k. Directions are
// evenly spaced and offset by half a step, so the four children of a
// direction are centred on it (paper figure 5).
func (level Level) Direction(direction int) (float64, float64) {
	angle := (float64(direction) + 0.5) * 2 * math.Pi / float64(level.Directions)
	sine, cosine := math.Sincos(angle)
	return cosine, sine
}

// MakePlan lays out the cascades for a grid whose side is a power of two.
// Cascade i covers distances from start to start + firstLength*4^i. New
// cascades are added until the next one would start beyond the grid
// diagonal, or the probes run down to one per side. The last cascade's
// rays run to the edge of the grid.
func MakePlan(side int, firstLength float64) (Plan, error) {
	if side < 2 || side&(side-1) != 0 {
		return nil, errors.New("side must be a power of two")
	}
	diagonal := float64(side) * math.Sqrt2
	var plan Plan
	start, length := 0.0, firstLength
	for index, spacing := 0, 1; ; index, spacing = index+1, spacing*2 {
		level := Level{Index: index, Spacing: spacing,
			ProbesPerSide: side / spacing,
			Directions:    firstDirections << (2 * index),
			Start:         start, End: start + length}
		if level.End > diagonal || level.ProbesPerSide == 1 {
			level.End = math.Inf(1)
			return append(plan, level), nil
		}
		plan = append(plan, level)
		start, length = level.End, length*branching
	}
}

// merged holds, for one cascade, the intensity of every ray after merging
// with all the cascades above it.
type merged struct {
	level  Level
	values []float64
}

// slot is the index in values of the first band for the ray at the probe in
// column and row going in direction; each ray takes Bands values.
func (cascade *merged) slot(column, row, direction int) int {
	probes := cascade.level.ProbesPerSide
	return ((row*probes+column)*cascade.level.Directions + direction) * Bands
}

// corner is one of the four probes around a point, with its bilinear weight.
type corner struct {
	column, row int
	weight      float64
}

// corners finds the four probes of this cascade around a point, with
// bilinear weights. Probes that would lie beyond the grid edge are
// replaced by the nearest probe inside it, as the paper does.
func (cascade *merged) corners(positionX, positionY float64) [4]corner {
	spacing := float64(cascade.level.Spacing)
	lastProbe := cascade.level.ProbesPerSide - 1
	scaledX := positionX/spacing - 0.5
	scaledY := positionY/spacing - 0.5
	firstColumn, firstRow := math.Floor(scaledX), math.Floor(scaledY)
	fractionX, fractionY := scaledX-firstColumn, scaledY-firstRow
	var result [4]corner
	for index := range result {
		stepX, stepY := index%2, index/2
		weight := weightOf(fractionX, stepX) * weightOf(fractionY, stepY)
		result[index] = corner{
			column: min(max(int(firstColumn)+stepX, 0), lastProbe),
			row:    min(max(int(firstRow)+stepY, 0), lastProbe),
			weight: weight,
		}
	}
	return result
}

// weightOf is the weight of one axis for a corner: fraction if step is one,
// else what is left of one.
func weightOf(fraction float64, step int) float64 {
	if step == 1 {
		return fraction
	}
	return 1 - fraction
}

// children averages the four child directions of one direction, each
// bilinearly interpolated from the probes around a point.
func (cascade *merged) children(positionX, positionY float64, direction int) [Bands]float64 {
	var sum [Bands]float64
	for _, probe := range cascade.corners(positionX, positionY) {
		for child := 0; child < branching; child++ {
			at := cascade.slot(probe.column, probe.row, direction*branching+child)
			for band := 0; band < Bands; band++ {
				sum[band] += probe.weight * cascade.values[at+band]
			}
		}
	}
	for band := range sum {
		sum[band] /= branching
	}
	return sum
}

// extend is paper equation 11: the near interval, then whatever lies
// beyond it, dimmed by the near interval's optical depth.
func extend(near Interval, beyond [Bands]float64) [Bands]float64 {
	var result [Bands]float64
	for band := 0; band < Bands; band++ {
		result[band] = near.Intensity[band] + math.Exp(-near.Tau[band])*beyond[band]
	}
	return result
}

// tracer finds the merged intensity of one ray: the probe position, then
// the direction number. It also says how many rays it marched and how many
// cells it read.
type tracer func(positionX, positionY float64, direction int) ([Bands]float64, int64, int64)

// basicTracer is the tracer that marches one ray over the level's range and,
// if there is a cascade above, extends it with that cascade's children
// interpolated around the same point. It counts one ray and its steps.
func basicTracer(grid *Grid, level Level, above *merged) tracer {
	return func(positionX, positionY float64, direction int) ([Bands]float64, int64, int64) {
		directionX, directionY := level.Direction(direction)
		near, steps := grid.March(positionX, positionY, directionX, directionY,
			level.Start, level.End)
		if above == nil {
			return near.Intensity, 1, steps
		}
		beyond := above.children(positionX, positionY, direction)
		return extend(near, beyond), 1, steps
	}
}

// fixedTracer is the bilinear fix (paper section 2.5). Instead of one
// ray merged with interpolated children, each child ray of each of the
// four probes above is reached by its own ray from this ray's start to
// that child's start, merged with exactly that child, and the results are
// blended with the bilinear weights. Sixteen rays per ray.
func fixedTracer(grid *Grid, level Level, above *merged) tracer {
	return func(positionX, positionY float64, direction int) ([Bands]float64, int64, int64) {
		directionX, directionY := level.Direction(direction)
		startX := positionX + level.Start*directionX
		startY := positionY + level.Start*directionY
		var total [Bands]float64
		var rays, steps int64
		for _, probe := range above.corners(positionX, positionY) {
			for child := 0; child < branching; child++ {
				used := fixedChild(grid, above, probe, direction*branching+child,
					startX, startY, &total)
				rays++
				steps += used
			}
		}
		return total, rays, steps
	}
}

// fixedChild adds one weighted contribution of the bilinear fix to total.
func fixedChild(grid *Grid, above *merged, probe corner, child int,
	startX, startY float64, total *[Bands]float64) int64 {
	childX, childY := above.level.Direction(child)
	endX := above.level.Position(probe.column) + above.level.Start*childX
	endY := above.level.Position(probe.row) + above.level.Start*childY
	length := math.Hypot(endX-startX, endY-startY)
	var near Interval
	var steps int64
	if length > minimumSegment {
		near, steps = grid.March(startX, startY, (endX-startX)/length,
			(endY-startY)/length, 0, length)
	}
	at := above.slot(probe.column, probe.row, child)
	var beyond [Bands]float64
	copy(beyond[:], above.values[at:])
	result := extend(near, beyond)
	for band := 0; band < Bands; band++ {
		total[band] += probe.weight * result[band] / branching
	}
	return steps
}

// runLevel traces and merges every ray of one cascade.
func runLevel(level Level, workers int, trace tracer, rays, steps *atomic.Int64) *merged {
	count := level.ProbesPerSide
	result := &merged{level: level, values: make([]float64, count*count*level.Directions*Bands)}
	forEachRow(count, workers, func(row int) {
		var rowRays, rowSteps int64
		for column := 0; column < count; column++ {
			for direction := 0; direction < level.Directions; direction++ {
				value, usedRays, usedSteps := trace(level.Position(column),
					level.Position(row), direction)
				copy(result.values[result.slot(column, row, direction):], value[:])
				rowRays += usedRays
				rowSteps += usedSteps
			}
		}
		rays.Add(rowRays)
		steps.Add(rowSteps)
	})
	return result
}

// Cascades computes J by radiance cascades, top cascade first, each merged
// down into the one below it. With fix set, every cascade but the top uses
// the bilinear fix.
func Cascades(grid *Grid, plan Plan, workers int, fix bool) (Field, Stats) {
	begin := time.Now()
	var rays, steps atomic.Int64
	var above *merged
	for index := len(plan) - 1; index >= 0; index-- {
		trace := basicTracer(grid, plan[index], above)
		if fix && above != nil {
			trace = fixedTracer(grid, plan[index], above)
		}
		above = runLevel(plan[index], workers, trace, &rays, &steps)
	}
	field := meanOverDirections(above, grid.Side)
	return field, Stats{Rays: rays.Load(), Steps: steps.Load(), Elapsed: time.Since(begin)}
}

// meanOverDirections turns cascade 0's merged directions into J.
func meanOverDirections(cascade *merged, side int) Field {
	field := Field{Side: side, Values: make([]float64, side*side*Bands)}
	directions := cascade.level.Directions
	for cell := 0; cell < side*side; cell++ {
		for direction := 0; direction < directions; direction++ {
			for band := 0; band < Bands; band++ {
				field.Values[cell*Bands+band] +=
					cascade.values[(cell*directions+direction)*Bands+band] / float64(directions)
			}
		}
	}
	return field
}
