package manufacture

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// bioformFieldMesh is generic scalar-field triangulation; it contains no animal shape.
func bioformFieldMesh(field func(maths.Vec) float64, warp func(maths.Vec) maths.Vec) (*shape.Mesh, error) {
	const nx, ny, nz = 70, 38, 44
	sx, sy, sz := 3.8/float64(nx), 1.5/float64(ny), 2.0/float64(nz)
	point := func(i, j, k int) maths.Vec {
		return maths.V(-1.9+float64(i)*sx+.000137, -.75+float64(j)*sy+.000193, -1+float64(k)*sz+.000233)
	}
	index := func(i, j, k int) int { return (i*(ny+1)+j)*(nz+1) + k }
	pts := make([]maths.Vec, (nx+1)*(ny+1)*(nz+1))
	val := make([]float64, len(pts))
	for i := 0; i <= nx; i++ {
		for j := 0; j <= ny; j++ {
			for k := 0; k <= nz; k++ {
				q := index(i, j, k)
				pts[q] = point(i, j, k)
				val[q] = field(pts[q])
			}
		}
	}
	type edge struct{ a, b int }
	edges := map[edge]int{}
	verts := []maths.Vec{}
	faces := [][3]int{}
	vertex := func(a, b int) int {
		if a > b {
			a, b = b, a
		}
		key := edge{a, b}
		if v, ok := edges[key]; ok {
			return v
		}
		t := val[a] / (val[a] - val[b])
		p := pts[a].Scale(1 - t).Add(pts[b].Scale(t))
		v := len(verts)
		verts = append(verts, p)
		edges[key] = v
		return v
	}
	emit := func(a, b, c int) {
		if a == b || b == c || a == c {
			return
		}
		p, q, r := verts[a], verts[b], verts[c]
		mid := p.Add(q).Add(r).Scale(1. / 3)
		h := .0005
		grad := maths.V(field(mid.Add(maths.V(h, 0, 0)))-field(mid.Add(maths.V(-h, 0, 0))), field(mid.Add(maths.V(0, h, 0)))-field(mid.Add(maths.V(0, -h, 0))), field(mid.Add(maths.V(0, 0, h)))-field(mid.Add(maths.V(0, 0, -h))))
		if q.Sub(p).Cross(r.Sub(p)).Dot(grad) < 0 {
			b, c = c, b
		}
		faces = append(faces, [3]int{a, b, c})
	}
	tetra := func(v [4]int) {
		in, out := []int{}, []int{}
		for _, p := range v {
			if val[p] < 0 {
				in = append(in, p)
			} else {
				out = append(out, p)
			}
		}
		switch len(in) {
		case 1:
			emit(vertex(in[0], out[0]), vertex(in[0], out[1]), vertex(in[0], out[2]))
		case 3:
			emit(vertex(out[0], in[0]), vertex(out[0], in[1]), vertex(out[0], in[2]))
		case 2:
			a, b := in[0], in[1]
			c, d := out[0], out[1]
			ac, ad, bc, bd := vertex(a, c), vertex(a, d), vertex(b, c), vertex(b, d)
			emit(ac, ad, bc)
			emit(ad, bd, bc)
		}
	}
	for i := 0; i < nx; i++ {
		for j := 0; j < ny; j++ {
			for k := 0; k < nz; k++ {
				p := [8]int{index(i, j, k), index(i+1, j, k), index(i, j+1, k), index(i+1, j+1, k), index(i, j, k+1), index(i+1, j, k+1), index(i, j+1, k+1), index(i+1, j+1, k+1)}
				for _, q := range [][4]int{{0, 1, 3, 7}, {0, 3, 2, 7}, {0, 2, 6, 7}, {0, 6, 4, 7}, {0, 4, 5, 7}, {0, 5, 1, 7}} {
					tetra([4]int{p[q[0]], p[q[1]], p[q[2]], p[q[3]]})
				}
			}
		}
	}
	if err := orientBioformFaces(verts, faces); err != nil {
		return nil, err
	}
	for i := range verts {
		verts[i] = warp(verts[i])
	}
	normals := make([]maths.Vec, len(verts))
	for _, f := range faces {
		a, b, c := verts[f[0]], verts[f[1]], verts[f[2]]
		n := b.Sub(a).Cross(c.Sub(a))
		for _, p := range f {
			normals[p] = normals[p].Add(n)
		}
	}
	skin, e := shape.NewMesh(verts, faces, normals, true)
	if e != nil {
		return nil, fmt.Errorf("implicit bioform skin: %w", e)
	}
	if !skin.Closed() {
		return nil, fmt.Errorf("implicit bioform skin is open")
	}
	return skin, nil
}

// Orient triangles by edge adjacency, then orient each closed component by
// signed volume. Gradient directions near a field crease can disagree.
func orientBioformFaces(vertices []maths.Vec, faces [][3]int) error {
	type key struct{ a, b int }
	type ref struct{ face, dir int }
	type link struct {
		other int
		flip  bool
	}
	edges := map[key][]ref{}
	for i, f := range faces {
		for j := 0; j < 3; j++ {
			a, b := f[j], f[(j+1)%3]
			dir := 1
			if a > b {
				a, b = b, a
				dir = -1
			}
			k := key{a, b}
			edges[k] = append(edges[k], ref{i, dir})
		}
	}
	links := make([][]link, len(faces))
	for k, refs := range edges {
		if len(refs) != 2 {
			return fmt.Errorf("edge %d-%d has %d adjacent faces", k.a, k.b, len(refs))
		}
		a, b := refs[0], refs[1]
		same := a.dir == b.dir
		links[a.face] = append(links[a.face], link{b.face, same})
		links[b.face] = append(links[b.face], link{a.face, same})
	}
	states := make([]int8, len(faces))
	for seed := range faces {
		if states[seed] != 0 {
			continue
		}
		states[seed] = 1
		queue := []int{seed}
		component := []int{}
		for len(queue) > 0 {
			a := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			component = append(component, a)
			for _, edge := range links[a] {
				want := states[a]
				if edge.flip {
					want = -want
				}
				if states[edge.other] == 0 {
					states[edge.other] = want
					queue = append(queue, edge.other)
				} else if states[edge.other] != want {
					return fmt.Errorf("nonorientable surface")
				}
			}
		}
		for _, i := range component {
			if states[i] < 0 {
				faces[i][1], faces[i][2] = faces[i][2], faces[i][1]
			}
		}
		volume := 0.
		for _, i := range component {
			f := faces[i]
			volume += vertices[f[0]].Dot(vertices[f[1]].Cross(vertices[f[2]])) / 6
		}
		if volume < 0 {
			for _, i := range component {
				faces[i][1], faces[i][2] = faces[i][2], faces[i][1]
			}
		}
	}
	return nil
}
