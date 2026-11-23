package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// wardrobe reads the materials of roles, and keeps the first it could
// not read. After that it gives nothing, so a run of reads is checked
// once at the end.
type wardrobe struct{ trouble error }

// of is the material a role is made of.
func (clothes *wardrobe) of(role partRole) *thing.Material {
	if clothes.trouble != nil {
		return nil
	}
	look, err := material(role)
	clothes.trouble = err
	return look
}

// colour is the colour of the material a role is made of, or nothing if
// it could not be read.
func (clothes *wardrobe) colour(role partRole) maths.Light {
	look := clothes.of(role)
	if look == nil {
		return maths.Light{}
	}
	return look.Colour
}

// dressed is a copy of the material a role is made of, under a name of its
// own and with a texture over its colour. The role's own material is not
// changed, so the levels that draw it plain are not.
func (clothes *wardrobe) dressed(role partRole, name string,
	painted texture.Texture) *thing.Material {
	base := clothes.of(role)
	if base == nil {
		return nil
	}
	look := *base
	look.Name += " " + name
	look.Texture = painted
	return &look
}

// part is a solid of a name, a form and a material, standing at a place
// in the frame of whatever holds it.
func part(name string, form form, material *thing.Material,
	place maths.Subspace) placedElement {
	return placedElement{Element: solid{Name: name, Form: form,
		Material: material}, Place: place}
}

// seated is the place of a box of a size standing on the origin of a
// frame, centred across and through, with the frame's own place put on
// it.
func seated(size Box, frame maths.Subspace) maths.Subspace {
	return standingAt(maths.V(-size.Width/half, 0, -size.Depth/half)).
		Within(frame)
}

// mounted is the place of a box of a size centred on the origin of a
// frame, turned there about its middle, with the frame's own place put
// on it.
func mounted(size Box, turn maths.Orientation,
	frame maths.Subspace) maths.Subspace {
	centre := standingAt(maths.V(-size.Width/half, -size.Height/half,
		-size.Depth/half))
	return centre.Within(maths.NewSubspace(maths.Vec{}, turn, 0)).Within(frame)
}

// radial is a place at a heading about the vertical, a distance out from
// the axis and a height up, facing out along its own positive z. At
// heading nought, out is along the vehicle's or the yurt's positive z.
func radial(heading, out, up float64) maths.Subspace {
	turn := maths.Rotation(maths.YAxis, heading)
	return maths.NewSubspace(
		turn.Out(maths.V(0, 0, out)).Add(maths.V(0, up, 0)), turn, 0)
}

// spun is the parts turned about the vertical axis through the origin by
// a heading. The parts given are not changed.
func spun(parts []placedElement, heading float64) []placedElement {
	return within(parts, maths.NewSubspace(maths.Vec{},
		maths.Rotation(maths.YAxis, heading), 0))
}

// leaning is the turn that carries the y axis onto a direction, by the
// shortest way. A direction along the y axis, either way, is no turn.
func leaning(direction maths.Vec) maths.Orientation {
	axis := maths.YAxis.Cross(direction)
	return maths.Rotation(axis, math.Atan2(axis.Len(),
		maths.YAxis.Dot(direction)))
}

// rod is a bar of a square section, from one point to another, so it
// has no side to roll onto.
func rod(name string, from, to maths.Vec, thickness float64,
	material *thing.Material) placedElement {
	size := Box{thickness, to.Sub(from).Len(), thickness}
	midpoint := standingAt(from.Add(to).Scale(1 / float64(half)))
	return part(name, size, material, mounted(size, leaning(to.Sub(from)),
		midpoint))
}
