package main

import (
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// the stock: a hillside of a variety makes that many of each kind and stands
// each at every place that picks it. a made thing's mesh does not depend on
// where it stands, so one meshed once and moved is the thing made there, bit
// for bit, and its faces are kept in memory once.

// stocked is which made thing a place stands: a kind, and which of its
// variety, or the place itself where every thing is its own.
type stocked struct {
	name   string
	pick   int
	detail string
}

// kept is a made thing, meshed once where it stands at the origin, the
// faces it keeps, and its lamps wherever it stands.
type kept struct {
	things []thing.Thing
	faces  int
	stored int
	lamps  func(at maths.Subspace) ([]emit.Lit, error)
	bounds maths.Box
}

// stock is the things a hillside has made, by which they are.
type stock map[stocked]kept

// pick is which of a kind's variety stands at a place: one by the
// place's own seed, or the place itself with no variety.
func pick(given asked, place int) int {
	if given.variety == 0 {
		return place
	}
	return int(stir(given.seed, place) % uint64(given.variety))
}

// made is the thing a place stands, and whether it was made now: the
// stock's, or made and kept if the stock has none yet.
func (have stock) made(given asked, name string,
	place int) (kept, bool, error) {
	key := stocked{name: name, pick: pick(given, place)}
	if one, known := have[key]; known {
		return one, false, nil
	}
	one, err := manufactured(given, name, key.pick)
	if err != nil {
		return kept{}, false, err
	}
	have[key] = one
	return one, true, nil
}

// manufactured is one of a kind from the seed of a pick, meshed where
// it stands at the origin. the manufactory's light is a type nothing
// outside it can name, so its lamps are kept as the way to read them.
func manufactured(given asked, name string, pick int) (kept, error) {
	return manufacturedRecipe(given, name, pick, "quick", nil)
}

// manufacturedRecipe shares geometry across placements of one recipe.
func manufacturedRecipe(given asked, name string, pick int, detail string,
	spec *manufacture.Optical) (kept, error) {
	row, err := manufacture.KindNamed(name)
	if spec != nil {
		row, err = manufacture.Observatory(*spec)
	}
	if err != nil {
		return kept{}, err
	}
	return manufacturedKind(given, row, pick, detail)
}

// manufacturedKind meshes a checked recipe and retains only its local lights.
func manufacturedKind(given asked, row manufacture.Kind, pick int,
	detail string,
) (kept, error) {
	one, err := manufacture.MakeWithComplexity(row,
		manufacture.Sow(stir(given.seed, pick)), detail)
	if err != nil {
		return kept{}, err
	}
	things, err := one.Things(maths.Local())
	if err != nil {
		return kept{}, err
	}
	lights := one.Descriptions()
	lamps := localLamps{lights: lights,
		exposure: given.exposure, moment: given.at}
	return kept{things: things, faces: facesOf(things),
		stored: storedFaces(things), lamps: lamps.at,
		bounds: one.Bounds()}, nil
}

// lampOf is a carried light as a lamp of the picture, at its moment and with
// nothing less than zero in it.
func lampOf(light emit.Described, at, toward maths.Vec, beyond float64,
	given asked) (emit.Lit, error) {
	point, err := emit.Of(light, at, toward, given.exposure)
	if err != nil {
		return emit.Lit{}, err
	}
	point.Beyond = beyond
	shone := point.When(given.at)
	shone.Strength = seen(shone.Strength)
	return shone, nil
}

// stood is a made thing's things standing at a place: the same meshes
// and looks, moved there.
func stood(things []thing.Thing, at maths.Subspace) []thing.Thing {
	moved := make([]thing.Thing, len(things))
	for i, each := range things {
		each.Motion = thing.Still(at)
		moved[i] = each
	}
	return moved
}

// facesOf is how many faces some things' meshes have.
func facesOf(things []thing.Thing) int {
	faces := 0
	for _, each := range things {
		faces += facesIn(each.Shape)
	}
	return faces
}

// facesIn is how many faces a shape has: a mesh.Mesh's or a shape.Mesh's, and
// zero for any other shape.
func facesIn(solid shape.Shape) int {
	switch made := solid.(type) {
	case *mesh.Mesh:
		return made.FaceCount()
	case *shape.Mesh:
		return len(made.Faces)
	case *shape.MeshGroup:
		return made.FaceCount()
	}
	return 0
}

// storedFaces accounts for prototypes rather than every repeated needle.
func storedFaces(things []thing.Thing) int {
	count := 0
	for _, one := range things {
		if group, ok := one.Shape.(*shape.MeshGroup); ok {
			count += group.StoredFaces()
		} else {
			count += facesIn(one.Shape)
		}
	}
	return count
}
