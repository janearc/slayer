package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// cityFoliagePart carries existing immutable meshes in one material role.
// Shared plant foliage cells remain shared, including across all four trees.
type cityFoliagePart struct {
	role    slot
	members []shape.MeshMember
}

// prepareCityFoliage lifts a tree's existing flat cell members into the city
// group. It does not flatten millions of repeated leaf triangles or introduce
// unsupported groups inside groups. Native acceleration receives the original
// shared mesh pointers plus composed planting/cell transforms.
func prepareCityFoliage(prototypes []cityTreePrototype, palette looks) (map[cityTreeKey][]cityFoliagePart, error) {
	prepared := map[cityTreeKey][]cityFoliagePart{}
	// Different plant families can use the same role name with different
	// colours. Do not silently assign every family the first one's palette.
	type plantLook struct {
		name                  string
		colour, glow          maths.Light
		gloss, index, through float64
	}
	roles := map[plantLook]slot{}
	for _, prototype := range prototypes {
		things, err := prototype.made.Things(maths.Local())
		if err != nil {
			return nil, err
		}
		for _, part := range things {
			look := plantLook{part.Material.Name, part.Material.Colour, part.Material.Glow,
				part.Material.Gloss, part.Material.Index, part.Material.Through}
			role, known := roles[look]
			if !known {
				role = slot(20 + len(roles))
				roles[look] = role
				palette[role] = part.Material
			}
			var members []shape.MeshMember
			if group, grouped := part.Shape.(*shape.MeshGroup); grouped {
				members = group.Members()
			} else {
				members = []shape.MeshMember{{Mesh: part.Shape, Space: maths.Local()}}
			}
			key := cityTreeKey{prototype.kind, prototype.seed}
			prepared[key] = append(prepared[key], cityFoliagePart{role, members})
		}
	}
	return prepared, nil
}
