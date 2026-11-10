package manufacture

import (
	"errors"
	"fmt"
	"sort"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// good is a made thing: its parts, in its own space, standing on the
// ground at the space's start, and what each of its matters looks
// like. a stone is a good of one part; a spruce is a good of some
// hundreds.
//
// made only by newGood and grow, dressed by in, lit by lit, and
// gathered from others by gather.
type good struct {
	parts   []part
	dressed looks
	lights  []lit
	extent  *maths.Box
	groups  map[slot]*shape.MeshGroup
}

// lit is a light a thing carries: the light, where on the thing it is, which
// way it shines, and how far out from there its light starts, in the thing's
// own space.
//
// a light that shines every way still has a way; nothing reads it. the light
// starts a little way out so that the lamp's own housing, which it sits
// against, does not shade everything it lights: emit calls that distance
// Beyond.
type lit struct {
	made       light
	at, toward maths.Vec
	beyond     measure.Length
}

// lit is the thing carrying one more light.
func (made good) lit(by light, at, toward maths.Vec,
	beyond measure.Length,
) good {
	made.lights = append(append([]lit{}, made.lights...),
		lit{made: by, at: at, toward: toward.Unit(), beyond: beyond})
	return made
}

// slots is the thing's matters, in order.
func (made good) slots() []slot {
	seen := map[slot]bool{}
	var slots []slot
	for _, one := range made.parts {
		if !seen[one.of] {
			seen[one.of] = true
			slots = append(slots, one.of)
		}
	}
	for of := range made.groups {
		if !seen[of] {
			slots = append(slots, of)
		}
	}
	sort.Slice(slots, func(one, other int) bool {
		return slots[one] < slots[other]
	})
	return slots
}

// gather is several things as one: every part and every light of each, where it
// was.
//
// each thing's matters stay its own, in its own looks, so stones and sticks
// gathered into a fire are still stone and wood: the matters are numbered on
// from one thing to the next. a thing with a matter nobody gave a look is
// refused here, as it is when a scene asks for it alone.
func gather(things ...good) (good, error) {
	all := good{dressed: looks{}}
	next := slot(0)
	for _, one := range things {
		renumbered := map[slot]slot{}
		for _, of := range one.slots() {
			look, known := one.dressed[of]
			if !known || look == nil {
				return good{}, errLook
			}
			renumbered[of] = next
			all.dressed[next] = look
			next++
		}
		for _, piece := range one.parts {
			piece.of = renumbered[piece.of]
			all.parts = append(all.parts, piece)
		}
		for of, group := range one.groups {
			if all.groups == nil {
				all.groups = map[slot]*shape.MeshGroup{}
			}
			all.groups[renumbered[of]] = group
		}
		all.lights = append(all.lights, one.lights...)
	}
	// Only assemblies with a prescribed physical extent need metadata.
	// Ordinary goods retain the extent of their actual surfaces.
	for _, one := range things {
		if one.extent != nil {
			box := partBounds(all.parts)
			for _, member := range things {
				box = enclosing(box, member.Bounds())
			}
			all.extent = &box
			break
		}
	}
	return all, nil
}

// looks is what each of a thing's matters looks like, as
// thing.Materials. a kind's row says what its things look like.
type looks map[slot]*thing.Material

// in is the thing in these looks.
func (made good) in(dressed looks) good {
	made.dressed = dressed
	return made
}

// placed is the thing put in a space of something larger's: every part
// of it and every light it carries, moved together.
func (made good) placed(space maths.Subspace) (good, error) {
	if !shape.ValidPlacement(space) {
		return good{}, errors.New("manufacture: invalid placement")
	}
	moved := make([]part, len(made.parts))
	for index, one := range made.parts {
		one.space = one.space.Within(space)
		moved[index] = one
	}
	shone := make([]lit, len(made.lights))
	for index, one := range made.lights {
		one.at = space.Out(one.at)
		one.toward = space.Orientation.Out(one.toward)
		one.beyond = one.beyond.Times(scaleOf(space))
		shone[index] = one
	}
	made.parts, made.lights = moved, shone
	if len(made.groups) > 0 {
		groups := map[slot]*shape.MeshGroup{}
		for of, group := range made.groups {
			members := group.Members()
			for i := range members {
				members[i].Space = members[i].Space.Within(space)
			}
			moved, err := shape.NewMeshGroup(members)
			if err != nil {
				return good{}, err
			}
			groups[of] = moved
		}
		made.groups = groups
	}
	if made.extent != nil {
		box := transformed(*made.extent, space)
		made.extent = &box
	}
	return made, nil
}

// mostFaces is the most faces a thing may have and still be flattened
// into meshes. it is tuning from a measurement: ninja-gaiden found
// flattening costs the cpu about 1 KB a triangle (64 conifers, 1.5
// million triangles, 1.6 GB), so this many is about 100 MB a thing.
const mostFaces = 100_000

var (
	errGood = errors.New("manufacture: a thing needs a form and a cut, " +
		"or a growth and a seed that was sown")
	errLook = errors.New("manufacture: a matter of this thing was " +
		"given no look")
	errTooMany = fmt.Errorf("manufacture: a thing of more than %d "+
		"faces is not flattened into a mesh", mostFaces)
)

// newGood is a thing of one form, cut so finely, or why there can be
// none: a stone, a can.
func newGood(shape form, fine cut) (good, error) {
	if shape == nil || fine.around < leastAround {
		return good{}, errGood
	}
	return good{parts: []part{{
		shape: shape, fine: fine, of: body,
		space: maths.Subspace{Scale: 1},
	}}}, nil
}

// grow is a thing grown from a seed to a size, or why it cannot be: a
// spruce, a fern.
func grow(grown growth, to size, from seed) (good, error) {
	if grown.stem.pieces < 1 || !from.made() {
		return good{}, errGood
	}
	parts, err := grown.parts(place{size: to}, from)
	if err != nil {
		return good{}, err
	}
	return good{parts: parts}, nil
}

// skins is the explicit parts' surfaces, matter by matter. Repeated mesh
// groups remain instances and are not expanded into this temporary surface.
func (made good) skins() map[slot]surface {
	skins := map[slot]surface{}
	for _, one := range made.parts {
		skins[one.of] = append(skins[one.of], one.skin()...)
	}
	return skins
}

// faces is stored face cost: each prototype once within its material group.
func (made good) faces() int {
	faces := 0
	for of, skin := range made.skins() {
		if made.groups[of] == nil {
			faces += len(skin)
		}
	}
	for _, group := range made.groups {
		faces += group.StoredFaces()
	}
	return faces
}

// things is the thing as a scene holds it, standing at a place: one thing.Thing
// a matter, each a mesh or shared mesh group, in the order of the
// matters. it refuses a thing of no parts, of too many faces, or with a matter
// nobody gave a look, and whatever slayer's mesh refuses.
func (made good) things(at maths.Subspace) ([]thing.Thing, error) {
	if !shape.ValidPlacement(at) {
		return nil, errors.New("manufacture: invalid placement")
	}
	if len(made.parts) == 0 && len(made.groups) == 0 {
		return nil, errGood
	}
	skins := made.skins()
	// The flattening limit bounds temporary expanded surfaces. Groups
	// retain their prototypes; the caller budgets their stored geometry.
	flattened := 0
	for of, skin := range skins {
		if made.groups[of] == nil {
			flattened += len(skin)
		}
	}
	if flattened > mostFaces {
		return nil, errTooMany
	}
	var things []thing.Thing
	for _, of := range made.slots() {
		look, known := made.dressed[of]
		if !known || look == nil {
			return nil, errLook
		}
		if group := made.groups[of]; group != nil {
			things = append(things, thing.Thing{Shape: group, Material: look, Motion: thing.Still(at)})
			continue
		}
		skin := skins[of]
		mesh, err := skin.meshed()
		if err != nil {
			return nil, err
		}
		things = append(things, thing.Thing{
			Shape: mesh, Material: look,
			Motion: thing.Still(at),
		})
	}
	return things, nil
}
