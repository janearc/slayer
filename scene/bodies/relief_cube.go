package bodies

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/world"
)

// ReliefCube makes a closed, uniformly subdivided cube-sphere mesh from a
// fixed relief field. Its output coordinates are metres times scale. A sea
// choice clamps below-sea heights to sea level; it is not an ocean model.
// maxFaces is an explicit output bound, and level controls sampling; this
// function does not select detail by camera, estimate error, or stitch
// different levels. The result is a geometric shape, not a normal map.
func ReliefCube(field *world.Relief, level, maxFaces int, sea bool,
	scale float64) (*shape.Mesh, error) {
	if field == nil || level < 0 || level > 8 || maxFaces < 1 ||
		!(scale > 0) || math.IsInf(scale, 0) {
		return nil, fmt.Errorf("bodies: invalid relief cube field, level, budget or scale")
	}
	width := 1 << level
	facesNeeded := 12 * width * width
	if facesNeeded > maxFaces {
		return nil, fmt.Errorf("bodies: relief cube needs %d faces, budget is %d",
			facesNeeded, maxFaces)
	}
	vertices := make([]maths.Vec, 0, facesNeeded/2+2)
	faces := make([][3]int, 0, facesNeeded)
	indices := make(map[world.ReliefVertex]int, facesNeeded/2+2)
	vertex := func(key world.ReliefVertex) (int, error) {
		if found, ok := indices[key]; ok {
			return found, nil
		}
		point, err := field.Point(key, sea)
		if err != nil {
			return 0, err
		}
		index := len(vertices)
		vertices = append(vertices, point.Scale(scale))
		indices[key] = index
		return index, nil
	}
	face := func(a, b, c int) {
		n := vertices[b].Sub(vertices[a]).Cross(vertices[c].Sub(vertices[a]))
		if n.Dot(vertices[a]) < 0 {
			b, c = c, b
		}
		faces = append(faces, [3]int{a, b, c})
	}
	for side := world.ReliefEast; side <= world.ReliefBack; side++ {
		for y := 0; y < width; y++ {
			for x := 0; x < width; x++ {
				patch, err := world.NewReliefPatch(side, level, x, y)
				if err != nil {
					return nil, err
				}
				corners := patch.Corners()
				var index [4]int
				for i, key := range corners {
					index[i], err = vertex(key)
					if err != nil {
						return nil, err
					}
				}
				face(index[0], index[1], index[2])
				face(index[1], index[3], index[2])
			}
		}
	}
	// Smooth only the shading normals; intersections and silhouettes use
	// the actual triangles. Shared keys make the position continuous.
	normals := make([]maths.Vec, len(vertices))
	for _, tri := range faces {
		a, b, c := tri[0], tri[1], tri[2]
		n := vertices[b].Sub(vertices[a]).Cross(vertices[c].Sub(vertices[a]))
		normals[a] = normals[a].Add(n)
		normals[b] = normals[b].Add(n)
		normals[c] = normals[c].Add(n)
	}
	return shape.NewMesh(vertices, faces, normals, true)
}
