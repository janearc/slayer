package manufacture

import (
	"fmt"
	birds "github.com/janearc/slayer/manufacture/animals/vertebrates/dinosaurs/theropods/birds"
	"github.com/janearc/slayer/maths"
)

type birdKind struct {
	species string
	profile *birds.Specimen
}

func (kind birdKind) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	if kind.profile == nil {
		return good{}, fmt.Errorf("manufacture: missing bird profile %q", kind.species)
	}
	parts, box, err := kind.profile.Build(from, maths.Local())
	if err != nil {
		return good{}, err
	}
	return good{native: parts, extent: &box}, nil
}
