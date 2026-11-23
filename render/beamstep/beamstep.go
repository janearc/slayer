// Package beamstep refines volume integration where a ray may cross a narrow
// light beam. Bounds choose sampling effort; they do not discard beam tails.
package beamstep

import (
	_ "embed"
	"math"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies/scrooge"
	"github.com/janearc/slayer/scene/thing"
)

// zero and one are defined additive/multiplicative identities.
const zero, one = 0, 1

// MostIntervals is how many stretches of a ray a Refinement keeps apart:
// sixteen, one more than the fifteen beams of the widest fan a set has
// drawn, so such a fan keeps each beam's stretch to itself. Past it the
// last stretch holds the union of the rest. The Metal kernel keeps the
// same number.
const MostIntervals = 16

// parallel is how small the square of the sine of the angle between a ray
// and a beam's axis is before the ray is taken as along the axis: one in a
// million, an angle of a thousandth of a radian. The Metal kernel uses the
// same, below which its quadratic's divisor is too small to divide by.
const parallel = 1e-6

//go:embed choices.yaml
var choicesRaw []byte

type choices struct {
	radii, radial float64
	maximum       int
}

var chosen = readChoices()

// readChoices checks immutable embedded budgets through Puzzlemouth.
func readChoices() choices {
	read, err := numbers.Read("choices.yaml", choicesRaw)
	if err != nil {
		panic(err)
	}
	return choices{radii: choice(read, "refinement-radii"),
		radial:  choice(read, "samples-per-radius"),
		maximum: int(choice(read, "maximum-splits"))}
}

// choice names a checked embedded value; a missing name is a code fault.
func choice(read numbers.Numbers, name string) float64 {
	number, err := read.Get(name)
	if err != nil {
		panic(err)
	}
	return number.Value
}

// Radii is the shared CPU/Metal refinement-envelope padding.
func Radii() float64 { return chosen.radii }

// RadialSamples is the target samples across one launch radius.
func RadialSamples() float64 { return chosen.radial }

// MaximumSplits bounds work within one original volume-march cell.
func MaximumSplits() int { return chosen.maximum }

// source is one beam: its axis, its padding and its step. box is the axis's
// bounding box widened by the padding; the cylinder is the same axis with
// the same padding all round.
type source struct {
	box    maths.Box
	step   float64
	at     maths.Vec
	toward maths.Vec
	reach  float64
	margin float64
}

// Beams caches immutable launch envelopes once for a render's lamp snapshot.
// It keeps each beam's stretch of a ray apart when scrooge asks.
type Beams struct {
	sources []source
	apart   bool
}

// Of prepares sampling bounds for Gaussian beams; other lamps need no change.
func Of(lamps []thing.Lamp) *Beams {
	plan := &Beams{apart: scrooge.Asked()}
	for _, lamp := range lamps {
		beam, ok := lamp.(*emit.Beam)
		if !ok || beam == nil {
			continue
		}
		point := beam.Point()
		reach := beam.Reach().InMetres()
		radius := beam.Radius().InMetres()
		end := point.At.Add(point.Toward.Scale(reach))
		margin := (radius + reach*beam.Slope()) * chosen.radii
		plan.sources = append(plan.sources, source{
			box:    maths.SegmentBox(point.At, end, margin),
			step:   radius / chosen.radial,
			at:     point.At,
			toward: point.Toward,
			reach:  reach,
			margin: margin,
		})
	}
	return plan
}

// Refinement is a ray's conservative intervals and desired local step sizes.
// Merging intervals may refine gaps too; it never declares a gap empty.
// Unless scrooge is asked it is the one union of the beams' boxes, in the
// first place; asked, it is a place for each beam the ray meets, and the
// last place holds the union of any past MostIntervals. A count says how
// many places are in use.
type Refinement struct {
	count               int
	enter, leave, steps [MostIntervals]float64
}

// add keeps a stretch of the ray and its step: in the next free place, or,
// with none free, in the last, as the union with what it holds.
func (refine *Refinement) add(enter, leave, step float64) {
	place := refine.count
	if place >= MostIntervals {
		last := MostIntervals - one
		refine.enter[last] = math.Min(refine.enter[last], enter)
		refine.leave[last] = math.Max(refine.leave[last], leave)
		refine.steps[last] = math.Min(refine.steps[last], step)
		return
	}
	refine.enter[place], refine.leave[place] = enter, leave
	refine.steps[place] = step
	refine.count++
}

// Ray computes bounds for this ray. No pilot ray stands in for neighbours.
func (beams *Beams) Ray(ray maths.Ray, far float64) Refinement {
	result := Refinement{}
	if beams == nil {
		return result
	}
	if !beams.apart {
		return beams.union(ray, far)
	}
	for _, source := range beams.sources {
		enter, leave := source.cylinder(ray, far)
		if leave <= enter {
			continue
		}
		result.add(enter, leave, source.step)
	}
	return result
}

// union is the stretch from the nearest entry to the furthest exit over
// every beam's box, at the smallest step, in the first place.
func (beams *Beams) union(ray maths.Ray, far float64) Refinement {
	var span struct{ enter, leave, step float64 }
	for _, source := range beams.sources {
		// A miss gives an empty interval. Point contacts also have no
		// length and must not extend another beam's interval.
		enter, leave, _ := source.box.Hit(ray, zero, far)
		if leave <= enter {
			continue
		}
		if span.step == zero {
			span.enter, span.leave, span.step = enter, leave, source.step
			continue
		}
		span.enter = math.Min(span.enter, enter)
		span.leave = math.Max(span.leave, leave)
		span.step = math.Min(span.step, source.step)
	}
	result := Refinement{}
	if span.step != zero {
		result.add(span.enter, span.leave, span.step)
	}
	return result
}

// slab is the stretch of a ray between two planes along one axis. A ray
// parallel to the planes is inside everywhere or nowhere.
func slab(at, way, low, high float64) (enter, leave float64) {
	if way == zero {
		if at >= low && at <= high {
			return math.Inf(-one), math.Inf(one)
		}
		return math.Inf(one), math.Inf(-one)
	}
	first, second := (low-at)/way, (high-at)/way
	return math.Min(first, second), math.Max(first, second)
}

// cylinder is the part of the ray inside the beam's bounding cylinder: the
// axis from the beam's start to its end, as wide as the margin, with caps a
// margin further out at both ends. It is the distances along the ray, from
// nought to far; a leave not past the enter is none.
func (source source) cylinder(ray maths.Ray, far float64) (
	enter, leave float64) {
	from := ray.At.Sub(source.at)
	start, rate := from.Dot(source.toward), ray.Dir.Dot(source.toward)
	capEnter, capLeave := slab(start, rate, -source.margin,
		source.reach+source.margin)
	across := from.Sub(source.toward.Scale(start))
	aside := ray.Dir.Sub(source.toward.Scale(rate))
	enter, leave, ok := source.sides(across, aside, far)
	if !ok {
		return far, zero
	}
	return math.Max(enter, capEnter), math.Min(leave, capLeave)
}

// sides is the stretch of the ray, from nought to far, inside the infinite
// cylinder about the axis, given where the ray starts and which way it
// goes, each across the axis. It is not ok when the ray misses it.
func (source source) sides(across, aside maths.Vec, far float64) (
	enter, leave float64, ok bool) {
	slant := aside.Dot(aside)
	if slant <= parallel {
		// Along the axis: the ray strays from it by no more than the
		// sine of its angle times its length, so a ray this close to the
		// axis at its start, widened by that, is inside the cylinder the
		// whole way.
		reach := source.margin + math.Sqrt(slant)*far
		return zero, far, across.Dot(across) <= reach*reach
	}
	half := across.Dot(aside)
	cut := half*half - slant*(across.Dot(across)-source.margin*source.margin)
	if cut < zero {
		return zero, far, false
	}
	root := math.Sqrt(cut)
	return math.Max(zero, (-half-root)/slant),
		math.Min(far, (-half+root)/slant), true
}

// Splits preserves original cell boundaries. Unaffected cells still use their
// original midpoint; intersected cells divide into a bounded regular grid,
// by the smallest step of the intervals the cell overlaps.
func (refine Refinement) Splits(enter, leave float64) int {
	step := float64(zero)
	for place := 0; place < refine.count; place++ {
		if leave <= refine.enter[place] || enter >= refine.leave[place] {
			continue
		}
		if step == zero {
			step = refine.steps[place]
		} else {
			step = math.Min(step, refine.steps[place])
		}
	}
	if step <= zero {
		return one
	}
	count := math.Ceil((leave - enter) / step)
	limit := float64(chosen.maximum)
	count = math.Min(limit, count)
	count = math.Max(one, count)
	return int(count)
}
