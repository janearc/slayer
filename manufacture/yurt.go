package manufacture

import (
	"slices"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// yurtRow is the kind yurt: a Mongolian ger, drawn by package cities from
// its design and made a good here, which is where a material becomes a
// slot.
type yurtRow struct{}

// made is the yurt at the quick detail level.
func (row yurtRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt is the yurt at the named detail level. It returns errGood when
// the seed is not made.
func (yurtRow) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	design, err := cities.YurtAt(cities.Detail(detail))
	if err != nil {
		return good{}, err
	}
	drawn, err := design.Rendered(cities.Detail(detail))
	if err != nil {
		return good{}, err
	}
	return goodOfRendered(drawn)
}

// sharedMeshes are the meshes made so far from what package cities drew,
// each with the triangles it was made from.
type sharedMeshes []sharedMesh

// sharedMesh is one mesh made, and the triangles of what it was made of.
type sharedMesh struct {
	drawn []cities.Triangle
	made  *mesh.Mesh
}

// of is the mesh for what was drawn. Two draws with the same triangles
// get the same mesh, however they came by them, so a window placed a
// thousand times is one mesh placed a thousand times.
func (meshes *sharedMeshes) of(drawn *cities.Mesh) (*mesh.Mesh, error) {
	for _, known := range *meshes {
		if slices.Equal(known.drawn, drawn.Triangles) {
			return known.made, nil
		}
	}
	skin := make(surface, len(drawn.Triangles))
	for index, triangle := range drawn.Triangles {
		skin[index] = face(triangle)
	}
	made, err := skin.meshed()
	if err != nil {
		return nil, err
	}
	*meshes = append(*meshes, sharedMesh{drawn.Triangles, made})
	return made, nil
}

// goodOfRendered is a good of what package cities drew. Each material is
// a slot, in the order first met; each slot is one group of the meshes of
// its material, each placed where it was drawn. A drawing with no mesh or
// no material is refused.
func goodOfRendered(drawn []cities.Rendered) (good, error) {
	made := good{dressed: looks{}, groups: map[slot]*shape.MeshGroup{},
		groupParts: map[slot]int{}}
	slots := map[string]slot{}
	members := map[slot][]shape.MeshMember{}
	var meshes sharedMeshes
	var lamps []cities.Rendered
	for _, one := range drawn {
		if one.Mesh == nil || one.Material == nil {
			return good{}, errLook
		}
		of, known := slots[one.Material.Name]
		if !known {
			of = slot(len(slots))
			slots[one.Material.Name], made.dressed[of] = of, one.Material
		}
		shared, err := meshes.of(one.Mesh)
		if err != nil {
			return good{}, err
		}
		members[of] = append(members[of],
			shape.MeshMember{Mesh: shared, Space: one.Place})
		if one.Lamp != nil {
			lamps = append(lamps, one)
		}
	}
	shaped, err := grouped(made, members)
	if err != nil {
		return good{}, err
	}
	return lamped(shaped, lamps)
}

// grouped is the good with a group of the members of each slot.
func grouped(made good, members map[slot][]shape.MeshMember) (good, error) {
	for of, each := range members {
		group, err := shape.NewMeshGroup(each)
		if err != nil {
			return good{}, err
		}
		made.groups[of], made.groupParts[of] = group, len(each)
	}
	return made, nil
}
