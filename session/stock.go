package session

import (
	"math"
	"sync"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/session/placement"
	"time"
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
	deform func(time.Duration) ([]thing.Thing, error)
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
	var book func() (*manufacture.PlantBook, error)
	switch name {
	case "spruce", "norway":
		book = hillsideConifers
	case "fern":
		book = hillsideFerns
	default:
		return manufacturedRecipe(given, name, pick, "quick", nil)
	}
	plants, err := book()
	if err != nil {
		return kept{}, err
	}
	row, err := plants.New(manufacture.PlantOptions{Species: name})
	if err != nil {
		return kept{}, err
	}
	return manufacturedKind(given, row, pick, "quick")
}

// Hillside's species selectors use the same checked, embedded books as
// authored models. Each book is read once and shared across its specimens.
func hillsideBook(family string) (*manufacture.PlantBook, error) {
	name, raw, err := manufacture.EmbeddedPlantBook(family)
	if err != nil {
		return nil, err
	}
	return manufacture.ReadPlantBook(family, name, raw)
}

var hillsideConifers = sync.OnceValues(func() (*manufacture.PlantBook, error) {
	return hillsideBook("conifer")
})
var hillsideFerns = sync.OnceValues(func() (*manufacture.PlantBook, error) {
	return hillsideBook("fern")
})

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
	one, err := placement.Make(placement.Construction{Kind: row, SceneSeed: given.seed, ObjectSeed: pick, Complexity: detail})
	if err != nil {
		return kept{}, err
	}
	result, err := keptOf(given, one)
	if err != nil {
		return kept{}, err
	}
	if manufacture.Deforms(row) {
		seed := manufacture.Sow(placement.Seed(given.seed, pick))
		sample, err := manufacture.PrepareDeformation(row, seed, detail)
		if err != nil {
			return kept{}, err
		}
		result.deform = func(at time.Duration) ([]thing.Thing, error) {
			frame, err := sample(at)
			if err != nil {
				return nil, err
			}
			return frame.Things(maths.Local())
		}
		// Stock retains the recipe, not a shared frame mesh.
		result.things = nil
		result.stored = 0
	}
	return result, nil
}

// keptOf is a made thing meshed where it stands at the origin, with the
// faces it keeps and its lamps.
func keptOf(given asked, one placement.Specimen) (kept, error) {
	things, err := one.Things(maths.Local())
	if err != nil {
		return kept{}, err
	}
	lamps := localLamps{lights: one.Descriptions(),
		exposure: given.exposure, moment: given.at}
	return kept{things: things, faces: facesOf(things),
		stored: storedFaces(things), lamps: lamps.at,
		bounds: one.Bounds()}, nil
}

// manufacturedLevelled is a kind drawn part by part, standing at a place
// in the world a pass sees, from the pick's seed as manufacturedKind's.
func manufacturedLevelled(given asked, row manufacture.Kind, pick int,
	pass cities.Pass, at maths.Subspace) (kept, error) {
	one, err := manufacture.MakeLevelled(row,
		manufacture.Sow(placement.Seed(given.seed, pick)), pass, at)
	if err != nil {
		return kept{}, err
	}
	return keptOf(given, one)
}

// pointOf is a carried light as a point emitter. A light that gives its
// strength in the kit's relative light, as a set's lamp does, is made the
// way a set's lamp is, from its heat and luminance; any other is scaled by
// the picture's exposure from its watts a steradian. A light with a
// colour of its own, as a neon sign's is, is made from that colour and its
// luminance.
//
// Either is kept to a cone where the light's own arc is less
// than a turn, as a boat's side light is, and shines every way where it is
// not.
func pointOf(light emit.Described, at, toward maths.Vec,
	exposure float64) (emit.Point, error) {
	if tinted, ok := light.(emit.Tinted); ok {
		if colour, luminance, yes := tinted.Tint(); yes {
			spectrum, err := emit.RelativeTint(colour, luminance)
			return coned(emit.Point{At: at, Light: spectrum,
				Pulse: light.Share, Toward: toward.Unit()}, light), err
		}
	}
	if relative, ok := light.(emit.Relatively); ok {
		if kelvin, luminance, yes := relative.Relative(); yes {
			spectrum, err := emit.Relative(kelvin, luminance)
			return coned(emit.Point{At: at, Light: spectrum,
				Pulse: light.Share, Toward: toward.Unit()}, light), err
		}
	}
	return emit.Of(light, at, toward, exposure)
}

// coned is a point emitter kept to the cone its light describes: its half
// angle and the soft edge past it. A half turn is every way, and has no
// width, as emit.Of says.
func coned(point emit.Point, light emit.Described) emit.Point {
	half, soft := light.Angles()
	if half.InRadians() < math.Pi {
		point.Width, point.Soft = half.InRadians(), soft.InRadians()
	}
	return point
}

// lampOf is a carried light as a lamp of the picture, at its moment and with
// nothing less than zero in it.
func lampOf(light emit.Described, at, toward maths.Vec, beyond float64,
	given asked) (emit.Lit, error) {
	point, err := pointOf(light, at, toward, given.exposure)
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
		each.Motion = thing.Still(each.At(0).Within(at))
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
