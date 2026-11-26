package manufacture

import (
	"github.com/janearc/slayer/manufacture/animals/cnidarians/coral"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// CoralSpec controls an authored branching colony in metres.
type CoralSpec = coral.Spec

// DefaultCoral gives a small branching colony for close review.
func DefaultCoral() CoralSpec { return coral.Default() }

// Coral constructs a seeded branching coral study.
func Coral(spec CoralSpec) (Kind, error) {
	if err := spec.Check(); err != nil {
		return kind{}, err
	}
	return kind{name: "branching-coral", how: coralMaker{spec}}, nil
}

type coralMaker struct{ spec CoralSpec }

func (maker coralMaker) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	branches, err := maker.spec.Grow(from, from.number)
	if err != nil {
		return good{}, err
	}
	colony := good{dressed: looks{body: &thing.Material{
		Name: "branching coral", Colour: maths.RGB(.49, .32, .22),
		Texture: coral.Texture(from.number),
	}}}
	for _, branch := range branches {
		colony.parts = append(colony.parts, part{shape: coralForm{branch.Branch}, space: branch.Space, fine: cut{around: 20}, of: body})
	}
	return colony, nil
}

type coralForm struct{ coral.Branch }

func (branch coralForm) skin(_ cut) surface { return coral.Skin[face](branch.Branch) }
func (branch coralForm) normals(skin surface) [][3]maths.Vec {
	return coral.Normals(branch.Branch, skin)
}
