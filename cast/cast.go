// Package cast constructs the dual mesh of a hexagonal tiling. Each triple of
// mutually adjacent tiles forms a triangle joining their centres, so geometry
// and painted tiles share the same lattice.
//
// Neighbour relationships define the mesh for spheres, ring bands, and planes.
// Callers do not need to stitch tile boundaries by hand.
package cast

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/mesh"
)

// Lift is how far a tile's centre rises off the lattice, along the way out
// at that point: a height, for hills. Nil is flat.
type Lift func(tile hexes.Tile, centre maths.Vec) float64

// Mesh is the tiling's triangles, each turned to face out, a point's way away
// from the surface: up for a plane, away from the middle for a sphere, toward
// the axis for the floor of a ring.
//
// Smooth blends the normals across each triangle, so hills roll; without it
// every triangle shows as a facet. The count is how many triangles were cast.
func Mesh(t hexes.Tiling, out func(p maths.Vec) maths.Vec, lift Lift, smooth bool) (*mesh.Mesh, int, error) {
	tiles := t.Tiles()
	index := make(map[hexes.Tile]int, len(tiles))
	verts := make([]maths.Vec, len(tiles))
	for i, tile := range tiles {
		c := t.Centre(tile)
		if lift != nil {
			c = c.Add(out(c).Unit().Scale(lift(tile, c)))
		}
		index[tile], verts[i] = i, c
	}
	// a pair of a tile's neighbours that touch each other closes a
	// triangle with it; each is cast once, from its lowest tile
	var faces [][3]int
	for _, a := range tiles {
		ns := t.Neighbours(a)
		touching := map[hexes.Tile]map[hexes.Tile]bool{}
		for _, b := range ns {
			touching[b] = map[hexes.Tile]bool{}
			for _, c := range t.Neighbours(b) {
				touching[b][c] = true
			}
		}
		for i, b := range ns {
			for _, c := range ns[i+1:] {
				if !touching[b][c] || t.ID(a) > t.ID(b) || t.ID(a) > t.ID(c) {
					continue
				}
				f := [3]int{index[a], index[b], index[c]}
				p0, p1, p2 := verts[f[0]], verts[f[1]], verts[f[2]]
				centre := p0.Add(p1).Add(p2).Scale(1.0 / 3)
				if p1.Sub(p0).Cross(p2.Sub(p0)).Dot(out(centre)) < 0 {
					f[1], f[2] = f[2], f[1]
				}
				faces = append(faces, f)
			}
		}
	}
	if len(faces) == 0 {
		return nil, 0, fmt.Errorf("cast: the tiling's %d tiles close no triangles", len(tiles))
	}
	var normals []maths.Vec
	if smooth {
		// each vertex's normal is the sum of the faces round it, by
		// area, which is the surface's own lean once the hills are in
		normals = make([]maths.Vec, len(verts))
		for _, f := range faces {
			n := verts[f[1]].Sub(verts[f[0]]).Cross(verts[f[2]].Sub(verts[f[0]]))
			for _, v := range f {
				normals[v] = normals[v].Add(n)
			}
		}
		for i := range normals {
			normals[i] = normals[i].Unit()
		}
	}
	m, err := mesh.New(verts, faces, normals, smooth)
	return m, len(faces), err
}
