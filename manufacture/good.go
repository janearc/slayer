package manufacture

import (
	"errors"
	"maps"
	"sort"
	"time"

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
	parts []part
	// native keeps already indexed, locally placed surfaces intact, including
	// their rest coordinates and material. They are immutable after making.
	native     []thing.Thing
	dressed    looks
	lights     []lit
	extent     *maths.Box
	groups     map[slot]*shape.MeshGroup
	groupParts map[slot]int
}

// nativeWithin composes a native part's local time-varying placement with
// its enclosing placement without taking a premature phase-zero snapshot.
type nativeWithin struct {
	local thing.Motion
	place maths.Subspace
}

// At is the part's placement at a time: its own motion at that time, set
// inside the enclosing place.
func (motion nativeWithin) At(at time.Duration) maths.Subspace {
	local := maths.Local()
	if motion.local != nil {
		local = motion.local.At(at)
	}
	return local.Within(motion.place)
}

// lit is a light a thing carries: the light, where on the thing it is, which
// way it shines, and how far out from there its light starts, in the thing's
// own space.
//
// a light that shines every way still has a way; nothing reads it. the light
// starts a little way out so that the lamp's own housing, which it sits
// against, does not shade everything it lights: emit calls that distance
// Beyond.
//
// a light may have a name, which a scene addresses it by to program it:
// a saucer's cabin, say. most have none.
type lit struct {
	made       light
	at, toward maths.Vec
	beyond     measure.Length
	name       string
}

// lit is the thing carrying one more light.
func (made good) lit(by light, at, toward maths.Vec,
	beyond measure.Length,
) good {
	return made.litAs("", by, at, toward, beyond)
}

// litAs is the thing carrying one more light, by a name a scene can
// address it by.
func (made good) litAs(name string, by light, at, toward maths.Vec,
	beyond measure.Length,
) good {
	made.lights = append(append([]lit{}, made.lights...),
		lit{made: by, at: at, toward: toward.Unit(), beyond: beyond,
			name: name})
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
				all.groupParts = map[slot]int{}
			}
			all.groups[renumbered[of]] = group
			all.groupParts[renumbered[of]] = one.partCounts()[of]
		}
		all.native = append(all.native, one.native...)
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
	if len(made.native) > 0 {
		moved := append([]thing.Thing(nil), made.native...)
		for i := range moved {
			moved[i].Motion = nativeWithin{local: moved[i].Motion, place: space}
		}
		made.native = moved
	}
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

// mostFaces is the faces a made thing's own budgets are drawn up within: a
// tree's foliage, a city's chunks, a macro row's faces. it is tuning from
// a measurement taken before the mesh kept its corners as int32 and the
// gpu packed each tree's faces: flattening then cost the cpu about 1 KB a
// triangle (64 conifers, 1.5 million triangles, 1.6 GB).
//
// it has not been measured since. no thing is refused for having more:
// a thing past it costs memory and time, and is drawn.
const mostFaces = 100_000

var (
	errGood = errors.New("manufacture: a thing needs a form and a cut, " +
		"or a growth and a seed that was sown")
	errLook = errors.New("manufacture: a matter of this thing was " +
		"given no look")
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
	if grown.Stem.Pieces < 1 || !from.made() {
		return good{}, errGood
	}
	parts, err := grownParts(grown, place{Size: to}, from)
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
	for _, one := range made.native {
		switch solid := one.Shape.(type) {
		case *shape.Mesh:
			faces += len(solid.Faces)
		case *shape.MeshGroup:
			faces += solid.StoredFaces()
		case interface{ FaceCount() int }:
			faces += solid.FaceCount()
		}
	}
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
// nobody gave a look, and whatever slayer's mesh refuses. a matter with a
// rounded part in it is shaded smooth by shadings, its other faces flat.
func (made good) things(at maths.Subspace) ([]thing.Thing, error) {
	return made.thingsWithTouching(at, true)
}

// thingsStrict keeps the original one-mesh-per-material validation policy.
func (made good) thingsStrict(at maths.Subspace) ([]thing.Thing, error) {
	return made.thingsWithTouching(at, false)
}

func (made good) thingsWithTouching(at maths.Subspace, separate bool) ([]thing.Thing, error) {
	if !shape.ValidPlacement(at) {
		return nil, errors.New("manufacture: invalid placement")
	}
	if len(made.parts) == 0 && len(made.groups) == 0 && len(made.native) == 0 {
		return nil, errGood
	}
	skins, shadings := made.skins(), made.shadings()
	var things []thing.Thing
	for _, one := range made.native {
		one.Motion = thing.Still(one.At(0).Within(at))
		things = append(things, one)
	}
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
		solid, err := skin.shaded(shadings[of])
		if err != nil {
			if !separate {
				return nil, err
			}
			separate, ok := made.separateParts(of, shadings[of], look, at)
			if !ok {
				return nil, err
			}
			things = append(things, separate...)
			continue
		}
		things = append(things, thing.Thing{
			Shape: solid, Material: look,
			Motion: thing.Still(at),
		})
	}
	return things, nil
}

// separateParts preserves independently valid surfaces as separate things
// when their combined edges coincide. Their shared placement retains local
// material coordinates; no invalid constituent is accepted.
func (made good) separateParts(of slot, normals [][3]maths.Vec,
	look *thing.Material, at maths.Subspace) ([]thing.Thing, bool) {
	var things []thing.Thing
	offset := 0
	for _, one := range made.parts {
		if one.of != of {
			continue
		}
		skin := one.skin()
		if len(skin) == 0 {
			continue
		}
		var each [][3]maths.Vec
		if normals != nil {
			if offset > len(normals) || len(skin) > len(normals)-offset {
				return nil, false
			}
			each = normals[offset : offset+len(skin)]
		}
		offset += len(skin)
		mesh, err := skin.shaded(each)
		if err != nil {
			return nil, false
		}
		things = append(things, thing.Thing{Shape: mesh, Material: look, Motion: thing.Still(at)})
	}
	return things, len(things) >= 2 && (normals == nil || offset == len(normals))
}

// withMeshGroups validates each material before installing any groups.
// Slot order keeps body errors ahead of foliage errors.
func (made good) withMeshGroups(members map[slot][]shape.MeshMember) (good, error) {
	order := make([]int, 0, len(members))
	for of := range members {
		order = append(order, int(of))
	}
	sort.Ints(order)
	groups := make(map[slot]*shape.MeshGroup, len(members))
	for _, index := range order {
		of := slot(index)
		group, err := shape.NewMeshGroup(members[of])
		if err != nil {
			return good{}, err
		}
		groups[of] = group
	}
	if made.groupParts != nil {
		made.groupParts = maps.Clone(made.groupParts)
		for of := range groups {
			made.groupParts[of] = len(members[of])
		}
	}
	made.groups = groups
	return made, nil
}
