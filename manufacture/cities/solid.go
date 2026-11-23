package cities

import (
	"errors"
	"fmt"
	"slices"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// errNoMaterial is the refusal of a solid that has no material.
var errNoMaterial = errors.New("cities: a solid needs a material")

// solid is a form in a material it is given. A mass takes its material
// from a role; a solid is for the parts whose material a culture draws,
// a wall painted this colour, a window lit at that heat. It stands where
// its place says, in the frame of whatever holds it.
type solid struct {
	Name     string
	Form     form
	Material *thing.Material
	// Lamp, if given, makes the solid a light as well.
	Lamp *Lamp
}

// Rendered is the solid's form meshed at the level, in its material, at
// the origin of its own frame.
func (solid solid) Rendered(level Detail) (Rendered, error) {
	if solid.Material == nil {
		return Rendered{}, fmt.Errorf("cities: %s: %w", solid.Name,
			errNoMaterial)
	}
	mesh, err := solid.Form.Mesh(level)
	if err != nil {
		return Rendered{}, fmt.Errorf("cities: %s: %w", solid.Name, err)
	}
	mesh.Name = solid.Name
	return Rendered{Mesh: mesh, Place: standingAt(maths.Vec{}),
		Material: solid.Material, Lamp: solid.Lamp}, nil
}

// put is a solid placed with its own origin at a point of its holder's
// frame.
func put(name string, form form, material *thing.Material,
	at maths.Vec) placedElement {
	return placedElement{Element: solid{Name: name, Form: form,
		Material: material}, Place: standingAt(at)}
}

// within is the parts as they stand in a parent frame: each place taken
// out through the parent's. The parts given are not changed.
func within(parts []placedElement, parent maths.Subspace) []placedElement {
	moved := make([]placedElement, len(parts))
	for index, part := range parts {
		moved[index] = liftedInto(part, parent)
	}
	return moved
}

// ridge is the way a roof's ridge, or an extrusion's length, runs on
// the ground: along the x axis or along the z axis.
type ridge int

const (
	// ridgeAlongX runs the length along the x axis, and the section
	// across the z axis.
	ridgeAlongX ridge = iota
	// ridgeAlongZ runs the length along the z axis, and the section
	// across the x axis.
	ridgeAlongZ
)

// extrusion is a convex section carried a length along its ridge: a
// triangle laid on its side is a gable. A corner of the section says how
// far across the ridge it stands, in X, and how high, in Z; the section
// is flat in that plane, and its height stands on the ground.
type extrusion struct {
	Section []outlineCorner
	Length  float64
	Ridge   ridge
}

// Mesh makes the extrusion's two ends and the walls between them, wound
// outward. It refuses a length that is not positive and a section that
// is not convex with at least three corners.
func (extrusion extrusion) Mesh(Detail) (*Mesh, error) {
	if !(extrusion.Length > 0) || len(extrusion.Section) < leastSides {
		return nil, fmt.Errorf("cities: an extrusion needs a positive "+
			"length and at least %d corners in its section, not %v and %d",
			leastSides, extrusion.Length, len(extrusion.Section))
	}
	if !convex(extrusion.Section) {
		return nil, fmt.Errorf("cities: an extrusion's end is a fan, which " +
			"a section that turns both ways is not filled by: it is concave")
	}
	near, far := extrusion.ends()
	inside := middle(slices.Concat(near, far))
	mesh := &Mesh{Triangles: sides(near, far, inside)}
	mesh.Triangles = append(mesh.Triangles,
		fill(near, middle(near).Sub(inside))...)
	mesh.Triangles = append(mesh.Triangles,
		fill(far, middle(far).Sub(inside))...)
	return mesh, nil
}

// ends is the section laid at the start of the length and at its end.
func (extrusion extrusion) ends() (near, far []maths.Vec) {
	for _, corner := range extrusion.Section {
		near = append(near, extrusion.lay(corner, 0))
		far = append(far, extrusion.lay(corner, extrusion.Length))
	}
	return near, far
}

// lay is a corner of the section, at a distance along the ridge, as a
// point on the ground and above it.
func (extrusion extrusion) lay(corner outlineCorner, along float64) maths.Vec {
	if extrusion.Ridge == ridgeAlongX {
		return maths.V(along, corner.Z, corner.X)
	}
	return maths.V(corner.X, corner.Z, along)
}

// sides joins each edge of one end to the same edge of the other as two
// triangles, facing away from the point inside the solid.
func sides(near, far []maths.Vec, inside maths.Vec) []Triangle {
	var skin []Triangle
	for index := range near {
		next := (index + 1) % len(near)
		quad := []maths.Vec{near[index], near[next], far[next], far[index]}
		away := middle(quad).Sub(inside)
		skin = append(skin,
			outward(Triangle{quad[0], quad[1], quad[2]}, away),
			outward(Triangle{quad[0], quad[2], quad[3]}, away))
	}
	return skin
}
