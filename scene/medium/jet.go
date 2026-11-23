package medium

import (
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// A stream's gaussian is cut at chosen.skirtWidths half widths from its
// middle: past it the density is exactly zero, so a span can find the edge and
// the box can be drawn round the gas. It is exp(-chosen.streamBell d^2) of the
// distance d from the middle in half widths, and a stream with no base is
// probed in cells of a chosen.streamEighth-th of its widest.

// Jet is a pair of jets, one up y and one down, from the origin out to Length:
// a cone each, Base across at the origin and spreading by Spread of the
// distance out, Thick at the heart of the base and thinning to zero at the end,
// gaussian across so the edges are soft.
//
// From is where each starts, out from the origin: a hole's jets are launched
// from the disc, not from the shadow.
//
// Knots is a dressing: the jet bunched along its length into bright knots by
// noise, as m87's is in the pictures, zero smooth and one all knots; Seed makes
// a jet's own.
type Jet struct {
	Length, Base, Spread float64
	Thick                float64
	From                 float64
	Knots                float64
	Seed                 uint64
	// Flow is how fast the knots move out along the jet, in base widths a
	// second, and Time when the jet is seen, in seconds: zero for knots
	// that stand still.
	Flow, Time float64
}

// the knots of a jet, chosen for the picture: a knot's noise is read at the
// side's own offset, chosen.jetLowerSide for the lower jet and nought for the
// upper, so each side has its own; knots are chosen.jetKnotSpacing base widths
// apart; the noise has chosen.jetKnotOctaves octaves; and Knots weighs it
// chosen.jetKnotWeight times.

// wide is how far from the y axis the jets' gas reaches: the cut of the
// gaussian at the widest of the cone.
func (jet Jet) wide() float64 {
	return chosen.skirtWidths * (jet.Base/double + jet.Length*jet.Spread)
}

// Bounds is the box both cones fit in.
func (jet Jet) Bounds() maths.Box {
	across := jet.wide()
	return corners(maths.V(-across, -jet.Length, -across),
		maths.V(across, jet.Length, across))
}

// Span is where a ray is in the jets: the slab and the cylinder they
// fit in, then probed in from both ends to where there is gas.
func (jet Jet) Span(ray maths.Ray, near, far float64) (float64, float64,
	bool) {
	enter, leave, hit := slab(jet.Length, ray, near, far)
	if !hit {
		return nought, nought, false
	}
	enter, leave, hit = cylinder(jet.wide(), ray, enter, leave)
	if !hit {
		return nought, nought, false
	}
	// probing no coarser than the narrowest the jet gets, so a ray
	// across the base does not step over it
	finest := jet.Base / double
	if finest <= nought {
		finest = jet.Length * jet.Spread / chosen.streamEighth
	}
	return probed(jet, ray, enter, leave, finest)
}

// Density is the cone's, thinning outward and soft across.
func (jet Jet) Density(point maths.Vec) float64 {
	if jet.Length <= nought {
		return nought
	}
	along := math.Abs(point.Y)
	if along >= jet.Length || along < jet.From {
		return nought
	}
	width := jet.Base/double + along*jet.Spread
	if width <= nought {
		return nought
	}
	across := math.Hypot(point.X, point.Z) / width
	if across > chosen.skirtWidths {
		return nought
	}
	density := jet.Thick * (unity - along/jet.Length) *
		math.Exp(-chosen.streamBell*across*across)
	if jet.Knots > nought {
		// knots a few base widths apart along the jet, the same up
		// and down a side, each side its own
		var side float64
		if point.Y < nought {
			side = chosen.jetLowerSide
		}
		position := (along - jet.Flow*jet.Time*jet.Base) /
			(chosen.jetKnotSpacing * jet.Base)
		grain := noise.Fractal(jet.Seed, side, position, nought,
			chosen.jetKnotOctaves)
		density *= math.Max(nought,
			unity+jet.Knots*chosen.jetKnotWeight*grain)
	}
	return density
}

// slab is where a ray is between y of minus and plus half, within near
// and far.
func slab(half float64, ray maths.Ray, near, far float64) (float64, float64,
	bool) {
	if ray.Dir.Y == nought {
		if math.Abs(ray.At.Y) > half {
			return nought, nought, false
		}
		return near, far, near < far
	}
	lower := (-half - ray.At.Y) / ray.Dir.Y
	upper := (half - ray.At.Y) / ray.Dir.Y
	if lower > upper {
		lower, upper = upper, lower
	}
	enter, leave := math.Max(near, lower), math.Min(far, upper)
	return enter, leave, enter < leave
}

// quadraticFour is the four of the quadratic's discriminant, b^2 - 4ac.
const quadraticFour = 4 // defined: the discriminant of a quadratic

// cylinder is where a ray is inside a cylinder of a radius round y,
// within near and far.
func cylinder(radius float64, ray maths.Ray, near, far float64) (float64,
	float64, bool) {
	sideways := ray.Dir.X*ray.Dir.X + ray.Dir.Z*ray.Dir.Z
	linear := double * (ray.At.X*ray.Dir.X + ray.At.Z*ray.Dir.Z)
	constant := ray.At.X*ray.At.X + ray.At.Z*ray.At.Z - radius*radius
	if sideways < chosen.alongTheAxis {
		if constant > nought {
			return nought, nought, false
		}
		return near, far, near < far
	}
	discriminant := linear*linear - quadraticFour*sideways*constant
	if discriminant < nought {
		return nought, nought, false
	}
	root := math.Sqrt(discriminant)
	first := (-linear - root) / (double * sideways)
	second := (-linear + root) / (double * sideways)
	enter, leave := math.Max(near, first), math.Min(far, second)
	return enter, leave, enter < leave
}

// cellHalf is the distance from a cell's middle, where the density is read, to
// its edge, as a share of the cell. A probe is at least chosen.probesLeast
// cells and at most chosen.probesMost.
const cellHalf = 0.5 // defined: half a cell

// probed is a span pulled in from both ends to where the medium has
// any density at all, in cells no coarser than finest and at least
// forty-eight of them, so a march starts on the gas and not on the box
// round it; a ray that finds none has no span.
//
// The density is read at each cell's middle. The gas of the first cell read
// as gas begins somewhere after the middle of the cell before it, and the
// gas of the last ends before the middle of the cell after it, so the span
// is the cells widened by half a cell at each end, kept within enter and
// leave. A crossing shorter than a cell, with no middle in it, is not found.
func probed(field Medium, ray maths.Ray, enter, leave, finest float64) (
	float64, float64, bool) {
	probes := chosen.probesLeast
	if finest > nought {
		steps := int(math.Ceil((leave - enter) / finest))
		probes = max(probes, min(chosen.probesMost, steps))
	}
	step := (leave - enter) / float64(probes)
	// absent is the cell index of gas not yet found, and nextCell is the one
	// after a cell.
	const (
		absent   = -1 // defined: no cell
		nextCell = 1  // defined: one cell on
	)
	first, last := absent, absent
	for cell := range probes {
		middle := enter + (float64(cell)+cellHalf)*step
		if field.Density(ray.Point(middle)) > nought {
			if first == absent {
				first = cell
			}
			last = cell
		}
	}
	if first == absent {
		return nought, nought, false
	}
	begin := enter + (float64(first)-cellHalf)*step
	end := enter + (float64(last+nextCell)+cellHalf)*step
	return math.Max(enter, begin), math.Min(leave, end), true
}

// Tick is the jet as it is at a time: the same jet, its knots moved
// along.
func (jet Jet) Tick(at time.Duration) Medium {
	jet.Time = at.Seconds()
	return jet
}
