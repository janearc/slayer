package geometry

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// Triangle permits a caller's own named triangle type.
type Triangle interface{ ~[3]maths.Vec }

// Quad adds nondegenerate triangles in the same order as the original surface.
func Quad[F Triangle](skin []F, a, b, c, d maths.Vec) []F {
	for _, triangle := range []F{{a, b, c}, {a, c, d}} {
		first, second, third := triangle[0], triangle[1], triangle[2]
		if first == second || second == third || third == first {
			continue
		}
		if second.Sub(first).Cross(third.Sub(first)).Len() > 1e-14 {
			skin = append(skin, triangle)
		}
	}
	return skin
}

// Middle returns a ring's mean point.
func Middle(ring []maths.Vec) maths.Vec {
	sum := maths.V(0, 0, 0)
	for _, corner := range ring {
		sum = sum.Add(corner)
	}
	return sum.Scale(1 / float64(len(ring)))
}

// Loft joins and caps rings, orienting the skin outwards.
func Loft[F Triangle](rings [][]maths.Vec) []F {
	var skin []F
	for step := 1; step < len(rings); step++ {
		low, high := rings[step-1], rings[step]
		for at := range low {
			next := (at + 1) % len(low)
			skin = Quad(skin, low[at], low[next], high[next], high[at])
		}
	}
	first, last := rings[0], rings[len(rings)-1]
	middle, end := Middle(first), Middle(last)
	for at := range first {
		next := (at + 1) % len(first)
		skin = append(skin, F{middle, first[next], first[at]}, F{end, last[at], last[next]})
	}
	if Volume(skin) < 0 {
		for i := range skin {
			skin[i][1], skin[i][2] = skin[i][2], skin[i][1]
		}
	}
	return skin
}

// IndexedLoftFaces joins equal rings and caps them with two final vertices.
// Side faces precede the first and last cap fans.
func IndexedLoftFaces(rows, around int) [][3]int {
	faces := AppendIndexedLoftSides(nil, 0, rows, around)
	for end, i := range []int{0, rows - 1} {
		n := rows*around + end
		for j := 0; j < around; j++ {
			k := (j + 1) % around
			if i == 0 {
				faces = append(faces, [3]int{n, k, j})
			} else {
				faces = append(faces, [3]int{n, i*around + j, i*around + k})
			}
		}
	}
	return faces
}

// AppendIndexedLoftSides joins equal rings, preserving face order and offset.
func AppendIndexedLoftSides(faces [][3]int, first, rows, around int) [][3]int {
	for i := 0; i < rows-1; i++ {
		for j := 0; j < around; j++ {
			a := first + i*around + j
			b := first + i*around + (j+1)%around
			c, d := b+around, a+around
			faces = append(faces, [3]int{a, b, c}, [3]int{a, c, d})
		}
	}
	return faces
}

// Volume is the signed volume held by an ordered triangle skin.
func Volume[F Triangle](skin []F) float64 {
	volume := float64(0)
	for _, one := range skin {
		volume += one[0].Dot(one[1].Cross(one[2]))
	}
	return volume / 6
}

// Creased smooths neighbouring faces while retaining edges over sixty degrees.
func Creased[F Triangle](skin []F) [][3]maths.Vec {
	meeting := map[maths.Vec][]int{}
	areas := make([]maths.Vec, len(skin))
	for index, one := range skin {
		areas[index] = one[1].Sub(one[0]).Cross(one[2].Sub(one[0]))
		for _, corner := range one {
			meeting[corner] = append(meeting[corner], index)
		}
	}
	normals := make([][3]maths.Vec, len(skin))
	for index, one := range skin {
		own := areas[index].Unit()
		for corner, point := range one {
			sum := maths.V(0, 0, 0)
			for _, other := range meeting[point] {
				if areas[other].Unit().Dot(own) >= .5 {
					sum = sum.Add(areas[other])
				}
			}
			normals[index][corner] = sum.Unit()
		}
	}
	return normals
}

// FaceMesh preserves separate corners, with optional corner normals.
func FaceMesh[F Triangle](skin []F, normals [][3]maths.Vec) (*mesh.Mesh, error) {
	corners := make([]maths.Vec, 0, len(skin)*3)
	faces := make([][3]int, 0, len(skin))
	for _, one := range skin {
		first := len(corners)
		corners = append(corners, one[0], one[1], one[2])
		faces = append(faces, [3]int{first, first + 1, first + 2})
	}
	if normals == nil {
		return mesh.New(corners, faces, nil, false)
	}
	each := make([]maths.Vec, 0, len(corners))
	for _, three := range normals {
		each = append(each, three[:]...)
	}
	return mesh.New(corners, faces, each, true)
}

// Ring makes a bent elliptical ring.
func Ring(centre, up, across maths.Vec, bend func(maths.Vec) maths.Vec) []maths.Vec {
	const corners = 24
	ring := make([]maths.Vec, corners)
	for index := range ring {
		sine, cosine := math.Sincos(2 * math.Pi * float64(index) / corners)
		ring[index] = bend(centre.Add(up.Scale(cosine)).Add(across.Scale(sine)))
	}
	return ring
}

// LoftPiece makes a smoothly shaded loft in its requested placement.
func LoftPiece(rings [][]maths.Vec, material *thing.Material, at maths.Subspace) (thing.Thing, error) {
	skin := Loft[[3]maths.Vec](rings)
	solid, err := FaceMesh(skin, Creased(skin))
	return thing.Thing{Shape: solid, Material: material, Motion: thing.Still(at)}, err
}

// JointLoft builds a loft through control joints and bends its rest rings.
func JointLoft(centres []maths.Vec, radii [][2]float64, material *thing.Material, at maths.Subspace, bend func(maths.Vec) maths.Vec) (thing.Thing, error) {
	rings := make([][]maths.Vec, 0, len(centres))
	for i, p := range centres {
		axis := centres[min(i+1, len(centres)-1)].Sub(centres[max(i-1, 0)]).Unit()
		a := axis.Cross(maths.V(0, 0, 1))
		if a.Len() < 1e-9 {
			a = axis.Cross(maths.Up)
		}
		a = a.Unit()
		b := axis.Cross(a).Unit()
		rings = append(rings, Ring(p, a.Scale(radii[i][0]), b.Scale(radii[i][1]), bend))
	}
	return LoftPiece(rings, material, at)
}

// HasArea rejects a repeated or nearly collinear triangle.
func HasArea[F Triangle](one F) bool {
	first, second := one[1].Sub(one[0]), one[2].Sub(one[0])
	across := first.Cross(second)
	return across.Dot(across) > 1e-12*1e-12*first.Dot(first)*second.Dot(second)
}
