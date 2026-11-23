package cities

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// at is the design stood at a place in a bigger design: every mass and
// every placed element taken out through the place. It is how any kind is
// a part of a bigger kind, a tower on a bridge's deck or a bridge on
// another's. A design that is made in its own space, and reads no world
// coordinate, can be stood anywhere; the tests hold every kind to that.
func (design Design) at(place maths.Subspace) Design {
	moved := Design{Name: design.Name}
	for _, mass := range design.Massing {
		mass.Place = mass.Place.Within(place)
		moved.Massing = append(moved.Massing, mass)
	}
	for _, placed := range design.Elements {
		moved.Elements = append(moved.Elements, liftedInto(placed, place))
	}
	for _, mount := range design.Mounts {
		mount.Place = mount.Place.Within(place)
		moved.Mounts = append(moved.Mounts, mount)
	}
	return moved
}

// extent is the lowest and the highest corner of a design along each
// axis, drawn at the level, or why it cannot be drawn.
func (design Design) extent(level Detail) (low, high maths.Vec, err error) {
	drawn, err := design.Rendered(level)
	if err != nil {
		return maths.Vec{}, maths.Vec{}, err
	}
	low = maths.V(math.Inf(1), math.Inf(1), math.Inf(1))
	high = maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))
	for _, one := range drawn {
		for _, triangle := range one.Mesh.Triangles {
			for _, corner := range triangle {
				at := one.Place.Out(corner)
				low = maths.V(math.Min(low.X, at.X), math.Min(low.Y, at.Y),
					math.Min(low.Z, at.Z))
				high = maths.V(math.Max(high.X, at.X),
					math.Max(high.Y, at.Y), math.Max(high.Z, at.Z))
			}
		}
	}
	if math.IsInf(low.X, 1) {
		return maths.Vec{}, maths.Vec{}, fmt.Errorf("cities: design %s has "+
			"nothing in it", design.Name)
	}
	return low, high, nil
}
