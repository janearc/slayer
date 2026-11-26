package manufacture

import (
	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
)

// madeAtDetail is a kind made at a detail level by a kind that makes it
// at the level asked, and a kind that has only the one way made as that.
type madeAtDetail interface {
	madeAt(from seed, detail string) (good, error)
}

// withMounts is a good with a thing of each mount's kind stood at the
// mount, in the good's frame: the telescopes a condominium keeps on its
// roofs, made by the catalogue's kind of that name. A mount of a kind
// there is none of is refused, naming the kinds there are.
//
// A thing that is refused as a mesh where it stands is left out, and the
// rest are kept.
//
// A finding of the telescope study, 2026-11-19: the catalogue's
// observatory builds its drum and its dome through different turns, so the
// two rims that meet differ, and at some places, at any scale, its mesh is
// refused for an edge two faces run the same way.
//
// A nudge of up to a fifth of a metre does not change that. The kind is
// in the geometry reference, so it is not changed here; it is the
// catalogue's to fix, and until it is some telescopes are not drawn.
func withMounts(made good, mounts []cities.Mount, from seed,
	detail string) (good, error) {
	if len(mounts) == 0 {
		return made, nil
	}
	things := []good{made}
	for _, mount := range mounts {
		row, err := kindNamed(mount.Kind)
		if err != nil {
			return good{}, err
		}
		one, err := mountedAt(row, from, detail)
		if err != nil {
			return good{}, err
		}
		placed, err := one.placed(mount.Place)
		if err != nil {
			return good{}, err
		}
		if _, refused := placed.thingsStrict(maths.Local()); refused == nil {
			things = append(things, placed)
		}
	}
	return gather(things...)
}

// mountedAt is one of a kind at a detail level, if the kind has levels.
func mountedAt(row kind, from seed, detail string) (good, error) {
	if leveled, has := row.how.(madeAtDetail); has {
		return leveled.madeAt(from, detail)
	}
	return makeOf(row, from)
}
