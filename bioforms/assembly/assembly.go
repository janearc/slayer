// Package assembly poses independently supplied skin and attachment meshes.
package assembly

import (
	"fmt"

	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// Part is a material on a rest-space surface. Carry optionally overrides the
// figure's pose function, for example to attach a rigid part to one bone.
type Part struct {
	Rest []maths.Vec
	// Paint optionally supplies per-vertex texture coordinates, for example a strand chart.
	Paint    []maths.Vec
	Faces    [][3]int
	Material *thing.Material
	Carry    func(maths.Vec) maths.Vec
}

// Build validates each part and returns its posed surface with original
// coordinates for material sampling. The caller chooses the parts to include.
func Build(parts []Part, carry func(maths.Vec) maths.Vec) ([]thing.Thing, error) {
	objects := make([]thing.Thing, 0, len(parts))
	for _, p := range parts {
		move := p.Carry
		if move == nil {
			move = carry
		}
		placed := make([]maths.Vec, len(p.Rest))
		for i, r := range p.Rest {
			placed[i] = r
			if move != nil {
				placed[i] = move(r)
			}
		}
		// Validate topology before computing normals from vertex indices.
		for _, face := range p.Faces {
			for _, index := range face {
				if index < 0 || index >= len(placed) {
					return nil, fmt.Errorf("assembly: vertex index %d outside %d rest points", index, len(placed))
				}
			}
		}
		paint := p.Paint
		if paint == nil {
			paint = p.Rest
		}
		surface, err := mesh.NewWithRestPoints(placed, paint, p.Faces, geometry.AreaNormals(placed, p.Faces), true)
		if err != nil {
			return nil, err
		}
		objects = append(objects, thing.Thing{Shape: surface, Material: p.Material})
	}
	return objects, nil
}
