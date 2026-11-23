package weathering

import (
	"errors"
	"slices"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// settled is a time long after any motion would have moved a thing: a
// thing that is where it was then, as at the start, is not moving.
const settled = 24 * time.Hour

// Apply lays a layer over a world: every surface that holds still and is
// not clear gets a texture that asks the layer wherever it is drawn, the
// panes of empty dwellings are boarded or broken, and the lamps behind
// them go dark. The layer's shelter is made of the world as it stands
// now, before the panes are boarded, and laid at its first look.
//
// Surfaces that move are left as they are, since a texture is given the
// world's place of a point and a moving thing has no one place; clear
// surfaces are left as they are, since glass takes no stain here.
func Apply(world *thing.World, layer *Layer) error {
	if !(layer.Block > 0) {
		return errors.New("weathering: the block a look overhead is " +
			"kept by must be wider than nothing")
	}
	layer.Over = NewShelter(layer.Block)
	for _, each := range world.Things {
		if holdsStill(each) {
			place := each.At(0)
			if nameOf(each) == SpoutMaterial {
				layer.Spouts = append(layer.Spouts, place.Out(maths.Vec{}))
			}
			layer.Over.holdPlaced(each.Shape, place, nameOf(each))
		}
	}
	for _, agent := range layer.Agents {
		if agent.Kind == Vacancy {
			layer.vacate(world, agent)
		}
	}
	for _, each := range world.Things {
		layer.dress(each)
	}
	return nil
}

// nameOf is the name of a thing's material, or none.
func nameOf(each *thing.Thing) string {
	if each.Material == nil {
		return ""
	}
	return each.Material.Name
}

// holdsStill is whether a thing is where it always is.
func holdsStill(each *thing.Thing) bool {
	return each.Motion == nil || each.At(0) == each.At(settled)
}

// dress puts the layer over one thing's material, as a copy, so that no
// other thing wearing the material is changed. A lamp's glow goes dark
// wherever a dwelling stands empty.
func (layer *Layer) dress(each *thing.Thing) {
	material := each.Material
	if material == nil || material.Through > 0 || !holdsStill(each) {
		return
	}
	own := *material
	own.Texture = Weathered{Base: material.Texture, Colour: material.Colour,
		Name: material.Name, Layer: layer}
	if material.Glow != (maths.Light{}) && material.Shine == nil &&
		layer.anyVacancy() {
		own.Shine = Dimmed{Light: material.Glow, Place: each.At(0),
			Layer: layer}
	}
	each.Material = &own
}

// anyVacancy is whether the layer has an agent that empties dwellings.
func (layer *Layer) anyVacancy() bool {
	return slices.ContainsFunc(layer.Agents, func(agent Agent) bool {
		return agent.Kind == Vacancy
	})
}

// triangles calls visit with each triangle of a shape, its corners taken
// through out. Shapes that are not meshes have none to visit.
func triangles(of shape.Shape, out func(maths.Vec) maths.Vec,
	visit func(first, second, third maths.Vec)) {
	switch each := of.(type) {
	case *mesh.Mesh:
		for face := range each.FaceCount() {
			first, second, third := each.Corners(face)
			visit(out(first), out(second), out(third))
		}
	case *shape.Mesh:
		for _, face := range each.Faces {
			visit(out(each.Vertices[face[0]]), out(each.Vertices[face[1]]),
				out(each.Vertices[face[2]]))
		}
	case *shape.MeshGroup:
		for _, member := range each.Members() {
			triangles(member.Mesh, func(point maths.Vec) maths.Vec {
				return out(member.Space.Out(point))
			}, visit)
		}
	}
}
