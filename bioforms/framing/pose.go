package framing

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// ThingsSample gives the root placement and subject parts to sample. Part
// motions are relative to Root and sampled at the same scene time as Root.
type ThingsSample struct {
	Root  maths.Subspace
	Parts []thing.Thing
	Time  time.Duration
}

// ThingsPose gets the posed bounds and transforms of visible subject parts.
func ThingsPose(sample ThingsSample) (Pose, error) {
	pose := Pose{Root: sample.Root, Parts: make([]Part, 0, len(sample.Parts))}
	for index, part := range sample.Parts {
		if part.Hidden() {
			continue
		}
		if part.Shape == nil {
			return Pose{}, fmt.Errorf("framing: part %d has no shape", index+1)
		}
		pose.Parts = append(pose.Parts, Part{
			Bounds: part.Shape.Bounds(), Placement: part.At(sample.Time),
		})
	}
	return pose, nil
}

// orientationTolerance allows a composed basis to round in float64.
const orientationTolerance = 1e-8

type posedBounds struct {
	corners []maths.Vec
	target  maths.Vec
	basis   maths.Orientation
}

func (pose Pose) bounds() (posedBounds, error) {
	if len(pose.Parts) == 0 {
		return posedBounds{}, fmt.Errorf("need at least one part")
	}
	if !validPlacement(pose.Root) {
		return posedBounds{}, fmt.Errorf("invalid root placement")
	}
	bounds := posedBounds{basis: pose.Root.Orientation}
	lo := maths.V(math.Inf(1), math.Inf(1), math.Inf(1))
	hi := lo.Neg()
	for index, part := range pose.Parts {
		if !validPart(part) {
			return posedBounds{}, fmt.Errorf("part %d has invalid bounds or placement", index+1)
		}
		box := part.Bounds
		for _, x := range [...]float64{box.Min.X, box.Max.X} {
			for _, y := range [...]float64{box.Min.Y, box.Max.Y} {
				for _, z := range [...]float64{box.Min.Z, box.Max.Z} {
					corner := pose.Root.Out(part.Placement.Out(maths.V(x, y, z)))
					if !finite(corner.Len()) {
						return posedBounds{}, fmt.Errorf("part %d exceeds the scene's numeric range", index+1)
					}
					bounds.corners = append(bounds.corners, corner)
					lo = maths.V(min(lo.X, corner.X), min(lo.Y, corner.Y), min(lo.Z, corner.Z))
					hi = maths.V(max(hi.X, corner.X), max(hi.Y, corner.Y), max(hi.Z, corner.Z))
				}
			}
		}
	}
	bounds.target = lo.Add(hi).Scale(.5)
	return bounds, nil
}

func validPart(part Part) bool {
	from, to := part.Bounds.Min, part.Bounds.Max
	return finite(from.Len()) && finite(to.Len()) &&
		from.X <= to.X && from.Y <= to.Y && from.Z <= to.Z &&
		validPlacement(part.Placement)
}

func validPlacement(placement maths.Subspace) bool {
	if !finite(placement.Origin.Len()) || !finite(placement.Scale) {
		return false
	}
	basis := placement.Orientation
	if basis == (maths.Orientation{}) {
		return true
	}
	axes := [...]maths.Vec{basis.X, basis.Y, basis.Z}
	for index, axis := range axes {
		if !(math.Abs(axis.Dot(axis)-1) <= orientationTolerance) {
			return false
		}
		for _, other := range axes[index+1:] {
			if math.Abs(axis.Dot(other)) > orientationTolerance {
				return false
			}
		}
	}
	return basis.X.Cross(basis.Y).Dot(basis.Z) > 0
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
