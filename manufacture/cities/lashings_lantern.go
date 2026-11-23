package cities

import (
	"fmt"

	"github.com/janearc/slayer/maths"
)

// ringsPerRib is how many bands of latitude a lantern's rib is cut into:
// one for the ridge and one for the groove beside it.
const ringsPerRib = 2

// ribbedLantern is a paper lantern of the kind hung in a street of
// shops: a globe about its own origin, as wide as twice its radius and as
// tall as its stretch times that, so that a stretch under one is squat,
// girdled with Ribs ridges each with a groove under it, the groove Depth
// of the radius nearer the axis than the ridge.
type ribbedLantern struct {
	Radius, Stretch float64
	Ribs            int
	Depth           float64
}

// Mesh makes the lantern as bands of latitude, wound outward. It refuses
// a lantern with no radius or no stretch, no rib, and a depth that is not
// from nought to under one. The corners round are as many as the detail
// level cuts a circle into.
func (lantern ribbedLantern) Mesh(level Detail) (*Mesh, error) {
	if !(lantern.Radius > 0 && lantern.Stretch > 0 && lantern.Ribs > 0 &&
		lantern.Depth >= 0 && lantern.Depth < 1) {
		return nil, fmt.Errorf("cities: a ribbed lantern needs a positive "+
			"radius and stretch, a rib, and a depth from nought to under "+
			"one, not %v", lantern)
	}
	sides, err := sidesAt(level)
	if err != nil {
		return nil, err
	}
	return bandMesh(lantern.Ribs*ringsPerRib, sides, lantern.latitude), nil
}

// latitude is the corners of the circle a number of bands down from the
// top, of the rings there are in all: those of the round lantern, drawn
// in toward the axis on every ring of a groove. The poles are ridges, as
// the rings in all are even.
func (lantern ribbedLantern) latitude(band, rings, sides int) []maths.Vec {
	corners := lashingLantern{lantern.Radius, lantern.Stretch}.latitude(band, rings,
		sides)
	if band%ringsPerRib == 0 {
		return corners
	}
	for index, corner := range corners {
		corners[index] = maths.V(corner.X*(1-lantern.Depth), corner.Y,
			corner.Z*(1-lantern.Depth))
	}
	return corners
}
