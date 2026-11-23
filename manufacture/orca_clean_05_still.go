package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// orcaClean05Still is the orca-clean-05 animal held in one shape that
// does not change with time: its skeleton alone at rest (bones), or its
// skin spy-hopping. It is the same animal and the same code as
// orcaClean05Kind; it is a type of its own because it does not deform,
// so a renderer that refuses deforming geometry can draw it.
type orcaClean05Still struct {
	bones bool
}

// build is the still kind's things: the skeleton, or the spy-hop.
func (kind orcaClean05Still) build() ([]thing.Thing, error) {
	if kind.bones {
		return OrcaClean05Bones(maths.Local())
	}
	return OrcaClean05SpyHop(maths.Local())
}

// made is the orca in the still pose the kind builds. It refuses a seed
// that was not made.
func (kind orcaClean05Still) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	parts, err := kind.build()
	if err != nil {
		return good{}, err
	}
	box := parts[0].Shape.Bounds()
	return good{native: parts, extent: &box}, nil
}
