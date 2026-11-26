package seasonal

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/manufacture/branching"
	"github.com/janearc/slayer/maths"
)

// Roots is the root system a plant shows above the ground, an
// optional block of a profile. Heights and radii are physical metres, as
// leaf sizes are, and do not grow with the tree; the spread is a share
// of the tree's scale. Tip is a root's radius at its far end as a share of
// its radius at its foot, nought for a point.
type Roots struct {
	policy *Policy
	Habit  string
	Count  struct{ Minimum, Maximum int }
	Height struct{ Minimum, Maximum float64 }
	Radius float64
	Tip    float64
	Spread float64
}

// the habits a profile's roots may have: upright pegs and cones that
// breathe, standing in a field about the trunk, and arching props and
// buttresses that hold the trunk up, evenly spaced about it.
const (
	rootPencil = "pencil"
	rootProp   = "prop"
)

// rootsRow is the seed row the roots grow from: beyond the row of any
// stem, so no stem's draws are a root's.
const rootsRow = 100

// the places a root's own draws are taken from in its seed.
const (
	rootTurn = iota
	rootOut
	rootHeight
	rootThick
	rootLeanTurn
	rootLean
)

// exposed grows a profile's visible roots from the seed, or none. The
// count, places, heights and thicknesses are the seed's; every detail
// level shows the same roots.
func (profile Profile) exposed(from branching.Seed, scale float64) []plantAxis {
	roots := profile.Roots
	if roots == nil {
		return nil
	}
	grown := from.Sprout(rootsRow, spaceZero)
	counts := span{least: float64(roots.Count.Minimum),
		most: float64(roots.Count.Maximum)}
	count := counts.count(grown.Draw(picking, spaceZero, spaceZero))
	var axes []plantAxis
	for index := range count {
		if roots.Habit == rootProp {
			trunk := scale * profile.Slenderness
			axes = append(axes,
				roots.prop(grown, index, count, scale, trunk)...)
			continue
		}
		axes = append(axes, roots.pencil(grown, index, scale)...)
	}
	return axes
}

// thickness is a root's radius at its foot, varied by the seed.
func (roots Roots) thickness(grown branching.Seed, index int) float64 {
	return roots.Radius * (spaceOne + roots.policy.Value("root-variation")*
		grown.Swing(growing, index, rootThick))
}

// risen is a root's height above the ground, picked from its range.
func (roots Roots) risen(grown branching.Seed, index int) float64 {
	heights := span{least: roots.Height.Minimum, most: roots.Height.Maximum}
	return heights.pick(grown.Draw(growing, index, rootHeight))
}

// pencil is one upright root, a peg or a cone, in the field about the
// trunk. It stands anywhere between the trunk's clearance and the field's
// edge with an even chance for each patch of ground, and leans a little.
func (roots Roots) pencil(grown branching.Seed, index int,
	scale float64) []plantAxis {
	inner := roots.policy.Value("root-clearance")
	edge := roots.Spread * scale
	turn := fullTurn * grown.Draw(growing, index, rootTurn)
	out := inner + (edge-inner)*math.Sqrt(grown.Draw(growing, index, rootOut))
	sine, cosine := math.Sincos(turn)
	height := roots.risen(grown, index)
	leanTurn := fullTurn * grown.Draw(growing, index, rootLeanTurn)
	lean := height * roots.policy.Value("pencil-lean") *
		grown.Swing(growing, index, rootLean)
	sideways, forwards := math.Sincos(leanTurn)
	sink := roots.policy.Value("root-sink")
	foot := maths.V(out*cosine, -sink, out*sine)
	end := foot.Add(maths.V(lean*forwards, height+sink, lean*sideways))
	radius := roots.thickness(grown, index)
	return []plantAxis{{root: foot, end: end, radius: radius,
		tip: radius * roots.Tip}}
}

// prop is one arching root: it leaves the trunk at a height, goes out and
// a little down to an elbow, and goes on down into the ground. The props
// of a trunk are evenly spaced about it, each strayed a little by the seed.
func (roots Roots) prop(grown branching.Seed, index, count int,
	scale, trunk float64) []plantAxis {
	strayed := float64(index) + roots.policy.Value("prop-turn-variation")*
		grown.Swing(growing, index, rootTurn)
	sine, cosine := math.Sincos(fullTurn * strayed / float64(count))
	height := roots.risen(grown, index)
	reach := roots.Spread * scale * (spaceOne +
		roots.policy.Value("prop-landing-variation")*grown.Swing(growing, index, rootOut))
	start := trunk * roots.policy.Value("prop-start-share")
	top := maths.V(start*cosine, height, start*sine)
	across := reach * roots.policy.Value("prop-elbow-reach")
	elbow := maths.V(across*cosine, height*roots.policy.Value("prop-elbow-height"),
		across*sine)
	land := maths.V(reach*cosine, -roots.policy.Value("root-sink"), reach*sine)
	radius := roots.thickness(grown, index)
	narrow := radius * roots.policy.Value("prop-lower-share")
	return []plantAxis{
		{root: top, end: elbow, radius: radius, tip: narrow},
		{root: elbow, end: land, radius: narrow, tip: narrow * roots.Tip}}
}

// validRoots refuses a malformed roots block before any root is grown.
// A profile without one is valid.
func (profile Profile) validRoots() error {
	roots := profile.Roots
	if roots == nil {
		return nil
	}
	for _, value := range []float64{roots.Height.Minimum, roots.Height.Maximum,
		roots.Radius, roots.Tip, roots.Spread} {
		if math.IsNaN(value) || math.IsInf(value, spaceZero) || math.Abs(value) > MaximumFigure {
			return fmt.Errorf("manufacture: non-finite roots for %s", profile.Kind)
		}
	}
	reach := roots.Spread * profile.Scale.Minimum
	switch {
	case roots.Habit != rootPencil && roots.Habit != rootProp:
		return fmt.Errorf("manufacture: unknown root habit %q for %s",
			roots.Habit, profile.Kind)
	case roots.Count.Minimum < spaceOne ||
		roots.Count.Maximum < roots.Count.Minimum ||
		roots.Count.Maximum > int(profile.value("maximum-roots")):
		return fmt.Errorf("manufacture: invalid root count for %s", profile.Kind)
	case !positive(roots.Height.Minimum) ||
		roots.Height.Maximum < roots.Height.Minimum ||
		!positive(roots.Radius) || roots.Tip < spaceZero || roots.Tip >= spaceOne:
		return fmt.Errorf("manufacture: invalid root dimensions for %s", profile.Kind)
	case !positive(roots.Spread) ||
		(roots.Habit == rootPencil && reach <= profile.value("root-clearance")):
		return fmt.Errorf("manufacture: roots of %s reach nowhere", profile.Kind)
	}
	return nil
}
