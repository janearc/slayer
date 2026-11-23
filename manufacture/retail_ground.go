package manufacture

import (
	"fmt"
	"slices"
	"sort"
	"sync"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// A building with shops at its foot. The kind retail-block is a building
// of a kind the data names, drawn by its own row, with a podium of retail
// units stood against its front: the front face being the most forward
// thing the building has, and the units' own back standing on it.
//
// The building is not changed, and a set that does not name retail-block
// is not changed either. The units are drawn once for each type and each of
// a pool of variants, and stood along the front as instances of those, so
// that a street of many blocks holds as many distinct units as the pool
// has, and no more in the memory it keeps.

// retailBuildingSalt is mixed into the seed of the building a block
// stands on, so that the building is not the one its seed draws alone. A
// choice.
const retailBuildingSalt = 0x424c4f434b424c44

// retailPoolSalt is mixed into the seed of each unit of the pool, so that
// the pool does not repeat the seeds of the frontages. A choice.
const retailPoolSalt = 0x504f4f4c554e4954

// retailTypeShift is how far the place of a unit type is shifted up in
// the seed of a unit of the pool, clear of the variant's own number.
const retailTypeShift = 32

// retailBlockTries is how many units of the pool a block tries, at a
// place along the front, to find one that fits what is left of the
// front. A choice.
const retailBlockTries = 24

// retailDrawsPerTry is how many numbers a try draws: a type, and a
// variant of it.
const retailDrawsPerTry = 2

// retailBlockRow is the kind retail-block.
type retailBlockRow struct{}

// retailKey says one unit of the pool: its type, which variant of the type
// it is, and the level it is drawn at.
type retailKey struct {
	name           string
	variant, level int
}

// retailPrototype is a unit of the pool, drawn once: how wide it is, a mesh
// for each material it has, its lamps and its sign slots, in its own frame.
type retailPrototype struct {
	width  float64
	meshes map[string]*mesh.Mesh
	lamps  []retailLamp
	slots  []FrontageSlot
}

// retailPrototypes are the units drawn so far, by key. A unit is drawn the
// same every time, so that it is kept and not drawn again.
var retailPrototypes sync.Map

// retailPlaced is a unit of the pool stood at a place along the front.
type retailPlaced struct {
	unit *retailPrototype
	at   float64
}

// made is the block at the quick detail level.
func (row retailBlockRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt is the block at the named detail level: the building at the
// level, and the shops at it. It returns errGood when the seed is not
// made, and refuses a level there is none of. A building with no room
// for a unit has none.
func (row retailBlockRow) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	level := retailLevel(detail)
	if level < 0 {
		return good{}, fmt.Errorf("manufacture: retail-block has no %s "+
			"representation; choose %v", detail, retailLevels)
	}
	if retailBookError != nil {
		return good{}, retailBookError
	}
	building, err := row.building(from, detail)
	if err != nil {
		return good{}, err
	}
	box := building.Bounds()
	grounds := retailBookRead.Grounds
	placed, used, err := row.placed(from, level,
		box.Max.X-box.Min.X-2*grounds.Margin)
	if err != nil || len(placed) == 0 {
		return building, err
	}
	podium, err := row.podium(placed, box, used)
	if err != nil {
		return good{}, err
	}
	return gather(building, podium)
}

// building is the building the seed draws from the data's kinds, by their
// weights, made at a level.
func (row retailBlockRow) building(from seed, detail string) (good, error) {
	grounds := retailBookRead.Grounds
	names := make([]string, 0, len(grounds.Buildings))
	for name := range grounds.Buildings {
		names = append(names, name)
	}
	sort.Strings(names)
	name := retailWeighted(names, grounds.Buildings,
		from.draw(retailBlockDraw, 0, 0))
	made, err := KindNamed(name)
	if err != nil {
		return good{}, err
	}
	return MakeWithComplexity(made, sow(from.number^retailBuildingSalt),
		detail)
}

// retailWeighted is the name a share of the way through the weights of
// the names, which are not none.
func retailWeighted(names []string, weights map[string]float64,
	share float64) string {
	total := 0.0
	for _, name := range names {
		total += weights[name]
	}
	reach, last := share*total, len(names)-1
	for _, name := range names[:last] {
		reach -= weights[name]
		if reach < 0 {
			return name
		}
	}
	return names[last]
}

// placed is the units along a front that has room for them, left to
// right, with no two of a type side by side, each as wide as its draw
// and none past the room; and how much of the room they use.
func (row retailBlockRow) placed(from seed, level int,
	room float64) ([]retailPlaced, float64, error) {
	var placed []retailPlaced
	used, beside := 0.0, ""
	for index := 1; ; index++ {
		unit, name, err := row.next(from, level, index, beside, room-used)
		if err != nil || unit == nil {
			return placed, used, err
		}
		placed = append(placed, retailPlaced{unit: unit, at: used})
		used += unit.width
		beside = name
	}
}

// next is the unit that stands next along the front: the first of the
// tries the seed makes that fits what is left, or none.
func (row retailBlockRow) next(from seed, level, index int, beside string,
	left float64) (*retailPrototype, string, error) {
	book := retailBookRead
	names := book.unitNames()
	for attempt := range retailBlockTries {
		turn := attempt * retailDrawsPerTry
		name := book.pick(names, beside, from.draw(retailBlockDraw, index,
			turn))
		variant := int(from.draw(retailBlockDraw, index, turn+1) *
			float64(book.Grounds.Pool))
		unit, err := retailPrototypeOf(name, variant, level)
		if err != nil {
			return nil, "", err
		}
		if unit.width <= left+retailFit {
			return unit, name, nil
		}
	}
	return nil, "", nil
}

// retailPrototypeOf is the unit of the pool a type and a variant name at a
// level, drawn once and kept. Its room is as deep as the podium.
func retailPrototypeOf(name string, variant, level int) (*retailPrototype,
	error) {
	key := retailKey{name: name, variant: variant, level: level}
	if known, found := retailPrototypes.Load(key); found {
		return known.(*retailPrototype), nil
	}
	book := retailBookRead
	builder := &retailBuilder{book: book, level: level,
		skins: map[string]surface{}, podium: book.Grounds.Podium}
	number := retailPoolSalt ^ uint64(slices.Index(book.unitNames(),
		name))<<retailTypeShift ^ uint64(variant)
	builder.unit(0, name, sow(number))
	unit := &retailPrototype{width: builder.width,
		meshes: map[string]*mesh.Mesh{}, lamps: builder.lamps,
		slots: builder.slots}
	for material, skin := range builder.skins {
		polygon, err := skin.meshed()
		if err != nil {
			return nil, fmt.Errorf("manufacture: retail %s, %s: %w", name,
				material, err)
		}
		unit.meshes[material] = polygon
	}
	known, _ := retailPrototypes.LoadOrStore(key, unit)
	return known.(*retailPrototype), nil
}

// podium is the shops along a building's front as a good: each unit an
// instance of its prototype's meshes, and the plain wall at each end. It
// stands out from the building by the data's podium depth, from where the
// building's most forward thing is, and the units are centred along it.
func (row retailBlockRow) podium(placed []retailPlaced, box maths.Box,
	used float64) (good, error) {
	book := retailBookRead
	grounds := book.Grounds
	slots := book.slots()
	front := box.Max.Z + grounds.Podium
	start := box.Min.X + (box.Max.X-box.Min.X-used)*spaceHalf
	members := map[string][]shape.MeshMember{}
	made := good{dressed: looks{}, groups: map[slot]*shape.MeshGroup{},
		groupParts: map[slot]int{}}
	for _, one := range placed {
		space := maths.NewSubspace(maths.V(start+one.at, 0, front),
			maths.Identity, spaceOne)
		for material, polygon := range one.unit.meshes {
			members[material] = append(members[material],
				shape.MeshMember{Mesh: polygon, Space: space})
		}
		for _, lamp := range one.unit.lamps {
			shining, err := newRelativeLight(measure.Kelvin(lamp.heat),
				lamp.luminance)
			if err != nil {
				return good{}, err
			}
			made = made.lit(shining, lamp.at.Add(space.Origin), maths.Up,
				measure.Metres(lampBeyond))
		}
	}
	ends, err := row.ends(box, start, start+used, front)
	if err != nil {
		return good{}, err
	}
	members[grounds.Filler] = append(members[grounds.Filler], ends...)
	for material, list := range members {
		group, err := shape.NewMeshGroup(list)
		if err != nil {
			return good{}, err
		}
		made.groups[slots[material]] = group
		made.groupParts[slots[material]] = len(list)
		made.dressed[slots[material]] = book.look(material)
	}
	return made, nil
}

// ends are the plain walls at each end of the shops, from the building's
// side out to the first and last unit, in the podium's depth and the
// ground storey's height.
func (row retailBlockRow) ends(box maths.Box, from, to,
	front float64) ([]shape.MeshMember, error) {
	tall := cityValue("storey-max")
	back := front - retailBookRead.Grounds.Podium
	var members []shape.MeshMember
	for _, span := range [][2]float64{{box.Min.X, from}, {to, box.Max.X}} {
		if !(span[1]-span[0] > retailFit) {
			continue
		}
		polygon, err := cityBox(span[0], span[1], 0, tall, back,
			front).meshed()
		if err != nil {
			return nil, err
		}
		members = append(members, shape.MeshMember{Mesh: polygon,
			Space: maths.Local()})
	}
	return members, nil
}
