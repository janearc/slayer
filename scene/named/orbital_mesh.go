package named

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// rz is a point of a band's cross-section: its distance out from the axis
// and its height along it.
type rz struct{ out, along float64 }

// hoopMeshes is one band as three meshes: its floor, the inner face
// between the walls; the wall face that looks across the gap at its twin;
// and the rest, the walls and the hull, each strip with its own normals,
// smooth round and hard at the corners.
func hoopMeshes(y, radius float64, gapUp bool) (*mesh.Mesh, *mesh.Mesh,
	*mesh.Mesh, error) {
	lo, hi := y-bandWidth/2, y+bandWidth/2
	in, out := radius-bandWall, radius+bandThick
	floor := []rz{{radius, lo}, {radius, hi}}
	// each wall's outer face: the one toward the gap looks at the twin
	upper := []rz{{in, hi + bandThick}, {out, hi + bandThick}}
	lower := []rz{{out, lo - bandThick}, {in, lo - bandThick}}
	facing, away := lower, upper
	if gapUp {
		facing, away = upper, lower
	}
	rest := [][]rz{
		{{radius, hi}, {in, hi}},         // the upper wall's inner face
		{{in, hi}, {in, hi + bandThick}}, // its top
		away,
		{{out, hi + bandThick}, {out, lo - bandThick}}, // the hull
		{{in, lo - bandThick}, {in, lo}},
		{{in, lo}, {radius, lo}},
	}
	floorMesh, err := revolve([][]rz{floor})
	if err != nil {
		return nil, nil, nil, err
	}
	facingMesh, err := revolve([][]rz{facing})
	if err != nil {
		return nil, nil, nil, err
	}
	restMesh, err := revolve(rest)
	return floorMesh, facingMesh, restMesh, err
}

// revolve is strips of a cross-section swept round the axis, a step of
// the band's around at a time, as one mesh.
func revolve(strips [][]rz) (*mesh.Mesh, error) {
	var verts, normals []maths.Vec
	var faces [][3]int
	for _, strip := range strips {
		base := len(verts)
		verts, normals = sweep(strip, verts, normals)
		at := func(i, j int) int {
			return base + (i%bandAround)*2 + j
		}
		for i := range bandAround {
			quad := [4]int{at(i, 0), at(i+1, 0), at(i+1, 1),
				at(i, 1)}
			triangles := [][3]int{{quad[0], quad[1], quad[2]},
				{quad[0], quad[2], quad[3]}}
			for _, corners := range triangles {
				first, second := verts[corners[0]], verts[corners[1]]
				third := verts[corners[2]]
				across := second.Sub(first).Cross(third.Sub(first))
				if across.Dot(normals[corners[0]]) < 0 {
					corners[1], corners[2] = corners[2], corners[1]
				}
				faces = append(faces, corners)
			}
		}
	}
	return mesh.New(verts, faces, normals, true)
}

// sweep is one strip's two points at every step round, with the strip's
// face as their normal, added to verts and normals.
//
// the strip's face points to its left as the profile runs, which, as the
// profile goes round the section, is away from the solid: the floor
// toward the axis, the hull out
func sweep(strip []rz, verts, normals []maths.Vec) ([]maths.Vec,
	[]maths.Vec) {
	run := rz{strip[1].out - strip[0].out,
		strip[1].along - strip[0].along}
	face := rz{-run.along, run.out}
	length := math.Hypot(face.out, face.along)
	face = rz{face.out / length, face.along / length}
	for i := range bandAround {
		angle := 2 * math.Pi * float64(i) / bandAround
		cosine, sine := math.Cos(angle), math.Sin(angle)
		for _, point := range strip {
			verts = append(verts, maths.V(point.out*sine,
				point.along, point.out*cosine))
			normals = append(normals, maths.V(face.out*sine,
				face.along, face.out*cosine))
		}
	}
	return verts, normals
}
