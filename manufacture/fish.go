package manufacture

import (
	"github.com/janearc/slayer/manufacture/animals/vertebrates/fishes"
	"github.com/janearc/slayer/scene/shape"
)

var fishRaw = fishes.Embedded()

type fishRow struct{ recipe *fishes.Recipe }

func (row fishRow) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	if row.recipe == nil {
		return good{}, errKind
	}
	result, err := row.recipe.Build(from)
	if err != nil {
		return good{}, err
	}
	if len(result.Groups) > 0 {
		made := good{dressed: looks{}, groups: map[slot]*shape.MeshGroup{}}
		for i, group := range result.Groups {
			made.groups[slot(i)] = group.Mesh
			made.dressed[slot(i)] = group.Material
		}
		return made, nil
	}
	made := good{native: result.Things}
	if result.HasBounds {
		made.extent = &result.Bounds
	}
	return made, nil
}
