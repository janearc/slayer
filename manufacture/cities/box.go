package cities

import "fmt"

// Box is a closed box spanning its size from its own origin; where it
// sits is the place it is given.
type Box struct{ Width, Height, Depth float64 }

// Mesh makes the box's six faces, two triangles each, wound outward. A
// box is the same at every level: its detail is in what is placed on it.
// It is a prism of a rectangle, and meshed as one.
func (box Box) Mesh(level Detail) (*Mesh, error) {
	if !(box.Width > 0 && box.Height > 0 && box.Depth > 0) {
		return nil, fmt.Errorf("cities: a box needs a positive width, "+
			"height and depth, not %v", box)
	}
	rectangle := polygon{Corners: []outlineCorner{{0, 0}, {box.Width, 0},
		{box.Width, box.Depth}, {0, box.Depth}}}
	return prism{Bottom: rectangle, Top: rectangle,
		Height: box.Height}.Mesh(level)
}
