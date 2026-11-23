package manufacture

import (
	"fmt"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
)

// gargoyleCorniceRow is the kind of a cornice wall of a stone, with a row
// of gargoyles under its eave, each a different creature the seed draws.
// The wall's face is in the plane z is nought and faces positive z; a
// gargoyle's corbel stands on it and the rest of the creature goes into
// the wall behind.
type gargoyleCorniceRow struct{ stone gargoyleStone }

// made is the cornice at the quick detail level.
func (row gargoyleCorniceRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt is the cornice wall at the named detail level, in its stone, with
// the gargoyles of its mounts. It returns errGood when the seed is not
// made.
func (row gargoyleCorniceRow) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	design, err := cities.GargoyleCornice(cities.Seed(from.number),
		row.stone.name)
	if err != nil {
		return good{}, err
	}
	drawn, err := design.Rendered(cities.Detail(detail))
	if err != nil {
		return good{}, err
	}
	wall, err := goodOfRendered(drawn)
	if err != nil {
		return good{}, err
	}
	for of, look := range wall.dressed {
		wall.dressed[of] = row.stone.look(look.Name, from.number, 1)
	}
	return row.withGargoyles(wall, design.Mounts, from, detail, nil)
}

// withGargoyles is the wall with a gargoyle of each mount's kind stood at
// the mount. Each is made from a seed of its own, drawn from the wall's, so
// that the five are not five of one. A mount of a kind there is none of is
// refused, naming the kinds there are. With a sight, a mount is made only
// as fine as its span in pixels shows; with none, every mount is whole.
func (row gargoyleCorniceRow) withGargoyles(wall good,
	mounts []cities.Mount, from seed, detail string,
	sight *mountSight) (good, error) {
	things := []good{wall}
	for index, mount := range mounts {
		one, made, err := row.mountAt(mount, from.sprout(housing, index),
			detail, sight)
		if err != nil {
			return good{}, err
		}
		if made {
			things = append(things, one)
		}
	}
	return gather(things...)
}

// mountAt is the mount made as its sight says, and whether it is made at
// all: not under mountLeastPixels, a block of its box under
// mountWholePixels, and the gargoyle whole from there. With no sight it is
// the gargoyle whole.
func (row gargoyleCorniceRow) mountAt(mount cities.Mount, from seed,
	detail string, sight *mountSight) (good, bool, error) {
	if sight == nil {
		one, err := row.gargoyleAt(mount, from, detail)
		return one, true, err
	}
	kind, err := kindNamed(mount.Kind)
	if err != nil {
		return good{}, false, err
	}
	box, pixels, err := sight.span(kind, mount.Place, from)
	switch {
	case err != nil:
		return good{}, false, err
	case pixels < mountLeastPixels:
		return good{}, false, nil
	case pixels < mountWholePixels:
		one, err := row.blockOfBox(box, mount.Place, from)
		return one, true, err
	}
	one, err := row.gargoyleAt(mount, from, detail)
	return one, true, err
}

// gargoyleAt is the gargoyle of a mount, stood where the mount puts it.
func (row gargoyleCorniceRow) gargoyleAt(mount cities.Mount, from seed,
	detail string) (good, error) {
	kind, err := kindNamed(mount.Kind)
	if err != nil {
		return good{}, err
	}
	made, err := mountedAt(kind, from, detail)
	if err != nil {
		return good{}, err
	}
	return made.placed(mount.Place)
}

// spouts is where every gargoyle of the cornice lets water go, in the
// cornice's frame, in the order of the mounts.
func (row gargoyleCorniceRow) spouts(from seed, detail string) (
	[]maths.Vec, error) {
	design, err := cities.GargoyleCornice(cities.Seed(from.number),
		row.stone.name)
	if err != nil {
		return nil, err
	}
	var all []maths.Vec
	for index, mount := range design.Mounts {
		kind, err := kindNamed(mount.Kind)
		if err != nil {
			return nil, err
		}
		each, err := spoutsOf(kind, from.sprout(housing, index), detail)
		if err != nil {
			return nil, err
		}
		for _, point := range each {
			all = append(all, mount.Place.Out(point))
		}
	}
	return all, nil
}

// spouting is what a kind with water to let go says: where, in its own
// frame, at a detail level, for a seed.
type spouting interface {
	spouts(from seed, detail string) ([]maths.Vec, error)
}

// spoutsOf is the spouts of a kind, or why it has none.
func spoutsOf(row kind, from seed, detail string) ([]maths.Vec, error) {
	spouter, has := row.how.(spouting)
	if !has {
		return nil, fmt.Errorf("manufacture: %s has no spout", row.name)
	}
	return spouter.spouts(from, detail)
}

// Spouts is where the kind of a name lets water go, for a seed, in the
// kind's own frame, as points: the mouth of each gargoyle's creature, or
// its funnel. These are the places rain would run from, for an agent that
// streaks the wall below them. A kind that has no spout is refused.
func Spouts(name string, number uint64) ([]maths.Vec, error) {
	row, err := kindNamed(name)
	if err != nil {
		return nil, err
	}
	return spoutsOf(row, sow(number), "quick")
}
