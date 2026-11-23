package cities

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// A building drawn for flat ground stands with its floor at height nought.
// On a slope it must not float, and it must not sink: its floor stands as
// high as the highest ground under it, so that no corner of it is buried,
// and a plinth carries it down to the lowest ground under it, so that no
// part of it hangs in the air. The plinth's height is the drop of the
// ground under the footprint, and nothing is picked for it.

// rolePlinth is the concrete a building stands on where the ground falls
// away from it.
const rolePlinth partRole = "plinth"

// Terrain is the height of the ground at a place in a building's own
// frame, x and z in metres from the frame's origin, in metres over the
// height the frame's origin stands at. It is how a slope is told to a
// building.
type Terrain func(x, z float64) float64

// groundFooting is where a building stands on the ground under its footprint:
// Base is the height of its floor, the highest ground under it, and Low the
// lowest, to which its plinth reaches. Heights are as a Terrain gives them.
type groundFooting struct{ Base, Low float64 }

// drop is how far the ground falls under the footprint: the height of the
// plinth, and nought on level ground.
func (footing groundFooting) drop() float64 { return footing.Base - footing.Low }

// Founding is which ground a building's floor stands at, where the ground
// under it is not level.
type Founding int

const (
	// FoundHighest stands the floor at the highest ground under the
	// footprint, so that no corner is buried. A building always was.
	FoundHighest Founding = iota
	// FoundAtDoor stands the floor at the highest ground along the front
	// wall, which is the wall the door is in, so that the door is at the
	// ground outside it. Ground that rises above the floor behind the
	// front wall is the hill the building is cut into, and a plinth
	// carries the front down to the lowest ground under the footprint
	// where the ground falls away from it.
	FoundAtDoor
)

// footingOn reads the ground at places spread evenly over a footprint, no
// further apart than the footing step and with one at each corner, and
// gives the highest and the lowest. It refuses a footprint of no area and
// a ground that gives a height that is not a number.
func footingOn(footprint rect, ground Terrain) (groundFooting, error) {
	return footingFounded(footprint, ground, FoundHighest)
}

// footingFounded is FootingOn with the floor founded as said: at the
// highest ground under the footprint, or at the highest along its front
// wall, the edge at the footprint's far z, where the door is. The lowest
// ground is the lowest under the whole footprint either way.
func footingFounded(footprint rect, ground Terrain,
	founding Founding) (groundFooting, error) {
	step, err := choice("footing-step")
	if err != nil {
		return groundFooting{}, err
	}
	if !(footprint.width() > 0 && footprint.depth() > 0) {
		return groundFooting{}, fmt.Errorf("cities: a footprint needs an area, "+
			"not %v by %v m", footprint.width(), footprint.depth())
	}
	across := int(math.Ceil(footprint.width() / step))
	along := int(math.Ceil(footprint.depth() / step))
	footing := groundFooting{Base: math.Inf(-1), Low: math.Inf(1)}
	door := math.Inf(-1)
	for column := range across + 1 {
		for row := range along + 1 {
			height := ground(
				footprint.X0+footprint.width()*float64(column)/float64(across),
				footprint.Z0+footprint.depth()*float64(row)/float64(along))
			if math.IsNaN(height) || math.IsInf(height, 0) {
				return groundFooting{}, fmt.Errorf("cities: the ground under a "+
					"building is %v high, which is no height", height)
			}
			footing.Base = math.Max(footing.Base, height)
			footing.Low = math.Min(footing.Low, height)
			if row == along {
				door = math.Max(door, height)
			}
		}
	}
	if founding == FoundAtDoor {
		footing.Base = door
	}
	return footing, nil
}

// planted is a building stood on its footing: every part raised to the
// footing's base, and under the footprint a plinth from the lowest ground
// up to the floor, where the ground falls at all.
func planted(parts []placedElement, footprint rect, footing groundFooting) ([]placedElement,
	error) {
	raised := within(parts, standingAt(maths.V(0, footing.Base, 0)))
	if !(footing.drop() > 0) {
		return raised, nil
	}
	concrete, err := material(rolePlinth)
	if err != nil {
		return nil, err
	}
	return append(raised, put("plinth", Box{footprint.width(),
		footing.drop(), footprint.depth()}, concrete,
		maths.V(footprint.X0, footing.Low, footprint.Z0))), nil
}

// plant stands a building on a ground, as it was drawn at its place, or
// leaves it as it is where there is no ground to stand it on.
func plant(parts []placedElement, footprint rect, ground Terrain) ([]placedElement,
	error) {
	return plantFounded(parts, footprint, ground, FoundHighest)
}

// plantFounded is plant with the floor founded as said.
func plantFounded(parts []placedElement, footprint rect, ground Terrain,
	founding Founding) ([]placedElement, error) {
	if ground == nil {
		return parts, nil
	}
	footing, err := footingFounded(footprint, ground, founding)
	if err != nil {
		return nil, err
	}
	return planted(parts, footprint, footing)
}

// footprintOf is the ground a box takes when its own frame stands in
// another: the least rectangle, along the axes, that holds its four
// corners.
func footprintOf(space maths.Subspace, width, depth float64) rect {
	box := rect{X0: math.Inf(1), Z0: math.Inf(1), X1: math.Inf(-1),
		Z1: math.Inf(-1)}
	for _, corner := range []maths.Vec{maths.V(0, 0, 0),
		maths.V(width, 0, 0), maths.V(0, 0, depth),
		maths.V(width, 0, depth)} {
		at := space.Out(corner)
		box = rect{math.Min(box.X0, at.X), math.Min(box.Z0, at.Z),
			math.Max(box.X1, at.X), math.Max(box.Z1, at.Z)}
	}
	return box
}
