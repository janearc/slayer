package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Reach is the box that holds every vertex of the design drawn at a level,
// in the design's own frame. It draws the elements and their meshes but
// makes nothing of them: no materials, no lamps, no trees and no things a
// design only mounts. A level finer than the one given may stand a little
// proud of this box, so a caller that must never be too small adds what
// the finer levels add. A design with no part has an empty box at nought.
func (design Design) Reach(level Detail) (maths.Box, error) {
	drawn, err := design.Rendered(level)
	if err != nil {
		return maths.Box{}, err
	}
	held := maths.Box{Min: maths.V(math.Inf(1), math.Inf(1), math.Inf(1)),
		Max: maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))}
	for _, one := range drawn {
		for _, triangle := range one.Mesh.Triangles {
			for _, corner := range triangle {
				held = holding(held, one.Place.Out(corner))
			}
		}
	}
	if held.Min.X > held.Max.X {
		return maths.Box{}, nil
	}
	return held, nil
}

// holding is the box with a point inside it as well.
func holding(box maths.Box, point maths.Vec) maths.Box {
	return maths.Box{Min: maths.V(math.Min(box.Min.X, point.X),
		math.Min(box.Min.Y, point.Y), math.Min(box.Min.Z, point.Z)),
		Max: maths.V(math.Max(box.Max.X, point.X),
			math.Max(box.Max.Y, point.Y), math.Max(box.Max.Z, point.Z))}
}
