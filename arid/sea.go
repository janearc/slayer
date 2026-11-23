// This file is a sea with no edge: the wave bands of one instant, summed at
// every point of the plane, and a ray's meeting with them.

package arid

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"sync/atomic"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
)

// seaTouch is how near the surface, in metres, a ray is taken to have met
// it: a tenth of a millimetre, far under the shortest wave any band holds
// and under anything a pixel can show. It is a tolerance of the search, not
// a property of any sea.
const seaTouch = 1e-4

// seaMostSteps is the most steps one ray may take before the search gives
// up and the ray ends on the mean plane. Counted over a draw of 960 by 540
// at two rays a pixel, seen from 2 m over the sea and level, the mean ray
// took 22 to 26 steps and about one in 60 took over a hundred. The most any
// took was 6,832 over one sea and was this limit over another, where 6 of
// 23 million rays reached it.
const seaMostSteps = 100000

// Sea is the surface of a sea with no edge. It holds the wave bands of
// oceantex.WaveCascades, each repeating over its own side and each read
// where its water is (oceantex.Field.Crested), so its crests are peaked
// and its troughs flat as the water's own sideways motion makes them.
//
// Its height at a point is the bands' heights summed, the longest band
// read at the point and each shorter one read where the longer ones'
// water came from: short waves ride the long ones and are carried by
// them. Nothing is sampled onto a square, so there is no square to end.
//
// A ray meets it by stepping the least distance it cannot cross the surface
// in: the gap above or below the surface, over the most the gap can change
// along the ray, which the bands' steepest rise per metre bounds. A ray
// therefore does not pass through a crest unseen, however low it skims.
//
// Every band is kept at every distance. A band the size of a pixel or less
// is not filtered away with distance, so far water shows the finest band's
// waves as the bands carry them. The finest band's side repeats, and so do
// the others'.
//
// A sea over ground (Ashore) has water only where the open sea reaches,
// and over each place no more of its waves than the depth there carries.
type Sea struct {
	// bands are the wave bands, longest side first, each crested.
	bands []*oceantex.Field
	// slopes is each band's own slope variance, along the wind and across
	// it together: what the band carries of the sea's slope.
	slopes   []float64
	reach    float64
	steepest float64
	// open is the steepest rise of the open sea's own surface, which is
	// steepest for a sea over no ground.
	open     float64
	period   float64
	finest   float64
	resolved float64
	// unseen is the slope variance of the waves shorter than any band
	// holds: Cox and Munk's total for the wind less the bands' own, and
	// nought for a sea made of bands alone or under a wind that leaves
	// the sea like a mirror.
	unseen float64
	// coast is the ground the sea meets, or nil for a sea over no ground,
	// which is water and the open sea's waves everywhere.
	coast *coast
	// surf is what the coast's bottom does to the waves (Surf), or nil
	// for waves cut only by the depth.
	surf *surf
	// onMean counts the rays that used up their steps and ended on the
	// mean plane.
	onMean atomic.Int64
	// leaning counts the rays that met the sea where the bands' slope
	// leaned the surface away from them (Seen).
	leaning atomic.Int64
}

// NewSea is the sea of the oceantex wave bands at one moment: the same
// bands, seed and time NewHeightField samples, kept whole instead. It
// refuses a time that is negative or not finite, and a sea that makes no
// band or a band that is not a square of finite heights and slopes.
func NewSea(sea oceantex.Sea, seconds float64, seed uint64) (*Sea, error) {
	cascades, err := oceantex.WaveCascades(sea, seconds, seed)
	if err != nil {
		return nil, fmt.Errorf("arid: synthesize the sea's bands: %w", err)
	}
	made, err := seaOf(cascades)
	if err != nil {
		return nil, err
	}
	budget, err := made.GlintBudget(sea)
	if err != nil {
		return nil, fmt.Errorf("arid: the sea's slope: %w", err)
	}
	if sea.Wind >= oceantex.MirrorCalm {
		made.unseen = budget.Unresolved
	}
	return made, nil
}

// seaOf is the sea of bands already made, the longest side first.
func seaOf(cascades []oceantex.WaveCascade) (*Sea, error) {
	if len(cascades) == 0 {
		return nil, errors.New("arid: a sea needs at least one band of waves")
	}
	cascades = append([]oceantex.WaveCascade(nil), cascades...)
	made := &Sea{finest: math.Inf(1)}
	for index, cascade := range cascades {
		if err := checkBand(cascade.Field); err != nil {
			return nil, fmt.Errorf("arid: band %d: %w", index, err)
		}
	}
	sort.SliceStable(cascades, func(one, other int) bool {
		return cascades[one].Field.Extent > cascades[other].Field.Extent
	})
	for _, cascade := range cascades {
		made.take(cascade)
	}
	made.steepest = nestedSteepest(made.bands)
	made.open = made.steepest
	if !finite(made.reach) || !finite(made.steepest) || !finite(made.resolved) {
		return nil, errors.New("arid: the sea's bands are not finite")
	}
	return made, nil
}

// take adds one band to a sea being made: the band crested, its slope
// variance, and what it adds to the sea's reach, period and finest
// sample.
func (sea *Sea) take(cascade oceantex.WaveCascade) {
	field := crestedBand(cascade.Field)
	sea.bands = append(sea.bands, field)
	sea.slopes = append(sea.slopes,
		cascade.ResolvedAlong+cascade.ResolvedAcross)
	var reach float64
	for _, height := range field.Height {
		reach = math.Max(reach, math.Abs(height))
	}
	sea.reach += reach
	sea.period = math.Max(sea.period, field.Extent)
	sea.finest = math.Min(sea.finest, field.Extent/float64(field.N))
	sea.resolved += cascade.ResolvedAlong + cascade.ResolvedAcross
}

// crestedBand is a band read where its water is. A band that says nothing
// of how its water is carried, no push and no stretch at every sample, is
// taken as carrying none: its own heights and slopes, with a push of
// nought.
func crestedBand(field *oceantex.Field) *oceantex.Field {
	count := field.N * field.N
	carried := true
	for _, grid := range [][]float64{field.PushX, field.PushY,
		field.StretchXX, field.StretchXY, field.StretchYY} {
		carried = carried && len(grid) == count
	}
	if carried {
		return field.Crested()
	}
	still := *field
	still.PushX, still.PushY = make([]float64, count), make([]float64, count)
	still.StretchXX = make([]float64, count)
	still.StretchXY = make([]float64, count)
	still.StretchYY = make([]float64, count)
	return &still
}

// checkBand is what is wrong with a band, or nothing: a square of at least
// two samples a side, a finite side, and a height and two slopes at every
// sample.
func checkBand(field *oceantex.Field) error {
	if field == nil || field.N < 2 || !finite(field.Extent) ||
		!(field.Extent > 0) {
		return errors.New("not a square of at least two samples and a side above nought")
	}
	count := field.N * field.N
	if len(field.Height) != count || len(field.SlopeX) != count ||
		len(field.SlopeY) != count {
		return errors.New("a height and two slopes are needed at every sample")
	}
	return nil
}

// steepestRise is the most the band's bilinear surface can rise in one
// metre, in any direction (steepestOf its heights).
func steepestRise(field *oceantex.Field) float64 {
	return steepestOf(field.Height, field.N, field.Extent)
}

// steepestOf is the most a square grid of values, count a side over
// extent metres and read on straight lines, can change in one metre, in
// any direction. Along each axis a bilinear cell's change is a blend of
// its edges' changes, so it is within the greatest change between
// neighbouring samples, the grid taken as wrapping; the change in any
// direction is within the two axes' greatest taken together.
func steepestOf(values []float64, count int, extent float64) float64 {
	var east, north float64
	for row := range count {
		for column := range count {
			here := values[row*count+column]
			along := values[row*count+(column+1)%count]
			across := values[((row+1)%count)*count+column]
			east = math.Max(east, math.Abs(along-here))
			north = math.Max(north, math.Abs(across-here))
		}
	}
	return math.Hypot(east, north) / (extent / float64(count))
}

// nestedSteepest is the most the bands' summed surface can rise in one
// metre, the bands longest first. Each band but the last moves the place
// the shorter ones are read at by its push, so a metre along the surface
// is at most one and its push's steepest change along the shorter bands,
// and the whole is
//
//	rise[0] + (1 + push[0]) * (rise[1] + (1 + push[1]) * (rise[2] ...
//
// with each push's change the two ways' taken together.
func nestedSteepest(bands []*oceantex.Field) float64 {
	var steepest float64
	for index := len(bands) - 1; index >= 0; index-- {
		band := bands[index]
		push := math.Hypot(steepestOf(band.PushX, band.N, band.Extent),
			steepestOf(band.PushY, band.N, band.Extent))
		steepest = steepestRise(band) + (1+push)*steepest
	}
	return steepest
}

// Period is the side of the longest band, in metres: the distance after
// which the sea's longest waves repeat.
func (sea *Sea) Period() float64 { return sea.period }

// GlintBudget is the slope the sea's bands do not carry, for the glint's
// lobe to carry in their place: Cox and Munk's total for the wind, less
// the slope the bands resolve, which stops at the finest band's shortest
// wave, its cut's share of its Nyquist wave number. See
// oceantex.SlopeBudget. The sea given must be the one the bands were made
// from, for its wind.
func (sea *Sea) GlintBudget(made oceantex.Sea) (oceantex.SlopeBudget, error) {
	return made.BudgetOf(sea.resolved,
		oceantex.CascadeCut()*math.Pi/sea.finest)
}

// Unseen is the slope variance, both ways together, of the waves shorter
// than any band holds, for a mirrored look to be roughened by: Cox and
// Munk's total for the wind less the bands' own. It is nought under a
// wind that leaves the sea like a mirror (oceantex.MirrorCalm).
func (sea *Sea) Unseen() float64 { return sea.unseen }

// height is the surface's height at an east and a south, in metres: the
// open sea's waves, times the share of them the coast lets stand there.
func (sea *Sea) height(east, south float64) float64 {
	if sea.surf != nil {
		return sea.surf.height(sea, east, south)
	}
	share, _, _ := sea.coast.meet(east, south)
	return share * sea.waves(east, south)
}

// waves is the open sea's height at an east and a south, in metres: the
// bands summed, longest first, each after the first read where the longer
// bands' water came from. South is the shape's z; the bands run north, so
// each is read at the side's half less it, as NewHeightField reads them.
func (sea *Sea) waves(east, south float64) float64 {
	var total float64
	last := len(sea.bands) - 1
	for index, band := range sea.bands {
		half := band.Extent / 2
		if index == last {
			return total + periodicSample(band, band.Height, half+east,
				half-south)
		}
		height, pushEast, pushNorth := carriedSample(band, half+east,
			half-south)
		total += height
		east, south = east-pushEast, south+pushNorth
	}
	return total
}

// carriedSample is a band's height at an east and a north on its tile,
// and how far the water there has been carried east and north, each as
// periodicSample reads it, the place found once for the three.
func carriedSample(field *oceantex.Field, east, north float64) (height,
	pushEast, pushNorth float64) {
	count := field.N
	gx := east / field.Extent * float64(count)
	gy := north / field.Extent * float64(count)
	xFloor, yFloor := math.Floor(gx), math.Floor(gy)
	tx, ty := gx-xFloor, gy-yFloor
	x0 := ((int(xFloor) % count) + count) % count
	y0 := ((int(yFloor) % count) + count) % count
	x1, y1 := (x0+1)%count, (y0+1)%count
	near, far := y0*count, y1*count
	weights := [...]float64{(1 - tx) * (1 - ty), tx * (1 - ty),
		(1 - tx) * ty, tx * ty}
	for index, at := range [...]int{near + x0, near + x1, far + x0,
		far + x1} {
		height += weights[index] * field.Height[at]
		pushEast += weights[index] * field.PushX[at]
		pushNorth += weights[index] * field.PushY[at]
	}
	return height, pushEast, pushNorth
}

// Bounds is the slab the surface stays in, with no edge east or south: the
// bands' highest crest above the mean and deepest trough below it.
func (sea *Sea) Bounds() maths.Box {
	infinity := math.Inf(1)
	return maths.Box{Min: maths.V(-infinity, -sea.reach, -infinity),
		Max: maths.V(infinity, sea.reach, infinity)}
}

// Distance is a lower bound on how far point is from the surface: the gap
// above it, less where the surface leans, and negative below it. A surface
// that rises at most steepest a metre is at least gap over the square root
// of one and steepest squared away.
func (sea *Sea) Distance(point maths.Vec) float64 {
	gap := point.Y - sea.height(point.X, point.Z)
	return gap / math.Sqrt(1+sea.steepest*sea.steepest)
}

// Normal is the way the surface faces at point, from the bands' own
// slopes, as one unit vector. It is not the normal of the surface Hit
// finds: the hit is on the sum of the bands' bilinear heights, and the
// slopes are the bands' own, read smooth, where the heights have creases.
//
// The two agree to the bands' sampling. A tracer that knows how wide its
// pixel is shades by Seen, which is this normal with the bands the pixel
// cannot show taken out.
//
// Over a coast the surface is the share times the open waves, so its
// slope is the share times theirs and the open height times the share's.
func (sea *Sea) Normal(point maths.Vec) maths.Vec {
	if sea.surf != nil {
		return sea.surf.normal(sea, point.X, point.Z)
	}
	eastward, northward := sea.slope(point.X, point.Z)
	share, shareEast, shareSouth := sea.coast.meet(point.X, point.Z)
	open := sea.waves(point.X, point.Z)
	eastward = share*eastward + open*shareEast
	northward = share*northward - open*shareSouth
	return maths.V(-eastward, 1, northward).Unit()
}

// Hit is the distance along ray, in lengths of ray.Dir and between near and
// far, at which it first meets the surface from either side, if it does.
// A ray that begins on the surface and leaves it has met nothing.
func (sea *Sea) Hit(ray maths.Ray, near, far float64) (float64, bool) {
	if !sea.usable(ray, near, far) {
		return 0, false
	}
	enter, leave, inside := sea.within(ray, near, far)
	if !inside {
		return 0, false
	}
	return sea.search(ray, near, enter, leave)
}

// usable is whether a ray and the span asked of it are ones Hit can
// answer: a sea, a span of numbers with something in it, and a ray of
// finite numbers that goes somewhere.
func (sea *Sea) usable(ray maths.Ray, near, far float64) bool {
	return !(sea == nil || !(near < far) || math.IsNaN(near) ||
		math.IsNaN(far) || !continuationFinite(ray.At.X, ray.At.Y,
		ray.At.Z, ray.Dir.X, ray.Dir.Y, ray.Dir.Z) ||
		ray.Dir == (maths.Vec{}))
}

// within is the part of the ray, between near and far, that is inside the
// slab the surface stays in, and whether any of it is.
func (sea *Sea) within(ray maths.Ray, near, far float64) (enter, leave float64,
	inside bool) {
	if ray.Dir.Y == 0 {
		return near, far, math.Abs(ray.At.Y) <= sea.reach
	}
	one, two := sea.faces(ray)
	enter, leave = math.Max(near, one), math.Min(far, two)
	return enter, leave, enter <= leave
}

// faces is where a ray that does not run level meets the slab's two
// faces, the first it meets and then the second, in lengths of ray.Dir.
func (sea *Sea) faces(ray maths.Ray) (first, second float64) {
	one := (-sea.reach - ray.At.Y) / ray.Dir.Y
	two := (sea.reach - ray.At.Y) / ray.Dir.Y
	if one > two {
		one, two = two, one
	}
	return one, two
}

// search steps along the ray from enter to leave by the gap over the most
// the gap can change along it (stepAt), which cannot carry the ray across
// the surface, until the ray is within seaTouch of the surface and is
// closing on it. It is a miss at leave. A ray still inside the slab after
// seaMostSteps skims the surface, and a skimming ray is over water, so it
// ends on the mean plane (endOnMean) and is counted, not dropped as a miss.
//
// Where the coast is dry the surface is there for the search to step by,
// but no water stands on it, so the ray goes through it and on.
//
// A ray straight up or down changes its gap by exactly its step, so its
// step lands on the surface itself, and by the last bit of the sum it may
// land a hair past it. The search keeps the gap of the step before, so
// that such a ray is known to have arrived (see arrived) and is not taken
// for one that began there and is leaving.
func (sea *Sea) search(ray maths.Ray, near, enter, leave float64) (float64,
	bool) {
	level := math.Hypot(ray.Dir.X, ray.Dir.Z)
	along, before := enter, 0.0
	for range seaMostSteps {
		if along > leave {
			return 0, false
		}
		rate, most := sea.stepAt(ray.Point(along), ray.Dir.Y, level)
		gap := sea.gap(ray, along)
		if sea.meets(ray, along, near, before, gap, rate) {
			return along, true
		}
		before = gap
		stride := math.Max(math.Abs(gap), seaTouch) / rate
		along += math.Min(stride, most)
	}
	sea.onMean.Add(1)
	along, ended := endOnMean(ray, near, leave)
	return along, ended && sea.wetAt(ray, along)
}

// stepAt is the most the gap can change for each length of a ray at
// point, for a ray that rises rise and goes level along the plane each
// length; and the most lengths one step may go for that to hold. Over the
// open sea it is the open surface's steepest rise and any step.
//
// Over a coast it is the steepest the surface is within a tile of the point
// (coast.steepestNear), which holds for a step of a tile's side and no
// further.
func (sea *Sea) stepAt(point maths.Vec, rise, level float64) (rate,
	most float64) {
	if sea.coast == nil {
		return math.Abs(rise) + sea.steepest*level, math.Inf(1)
	}
	steep := shoredSteepest(sea.open,
		sea.coast.steepestNear(point.X, point.Z))
	return math.Abs(rise) + steep*level, sea.coast.tileSide() / level
}

// meets is whether the ray meets water at along: within seaTouch of the
// surface, past near, closing on it or just arrived from its other side,
// and where the coast lets water stand. before is the gap at the step
// before this one, or nought at the first.
func (sea *Sea) meets(ray maths.Ray, along, near, before, gap,
	rate float64) bool {
	return math.Abs(gap) < seaTouch && along > near &&
		(arrived(before, gap) || sea.closing(ray, along, gap, rate)) &&
		sea.wetAt(ray, along)
}

// arrived is whether a ray has come to the surface from its other side in
// one step: at the step before it was seaTouch or more from the surface,
// and now it is on the other side of it. A ray that was already within
// seaTouch began at the surface, and one that began there and is leaving
// has met nothing.
func arrived(before, gap float64) bool {
	return math.Abs(before) >= seaTouch && before*gap < 0
}

// wetAt is whether water may stand under the ray's point at along.
func (sea *Sea) wetAt(ray maths.Ray, along float64) bool {
	point := ray.Point(along)
	return sea.coast.wet(point.X, point.Z)
}

// endOnMean is where the ray crosses the mean plane, the sea's level where
// its waves average out, if that is between near and far.
func endOnMean(ray maths.Ray, near, far float64) (float64, bool) {
	if ray.Dir.Y == 0 {
		return 0, false
	}
	along := -ray.At.Y / ray.Dir.Y
	return along, along > near && along < far
}

// gap is how far above the surface the ray is at along, in metres, and
// negative below it.
func (sea *Sea) gap(ray maths.Ray, along float64) float64 {
	point := ray.Point(along)
	return point.Y - sea.height(point.X, point.Z)
}

// closing is whether the ray is nearer the surface a short way on: the
// gap a step of seaTouch on is smaller, or on the other side of the surface.
func (sea *Sea) closing(ray maths.Ray, along, gap, rate float64) bool {
	next := sea.gap(ray, along+seaTouch/rate)
	return gap*next <= 0 || math.Abs(next) < math.Abs(gap)
}

// slope is the surface's rise a metre east and a metre north at an east
// and a south, as a pixel of no width shows it: every band whole.
func (sea *Sea) slope(east, south float64) (eastward, northward float64) {
	eastward, northward, _ = sea.slopeWithin(east, south, 0)
	return eastward, northward
}
