package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// WalrusFresh3 builds an original, connected skin from a smooth implicit
// union. Grid edge sharing makes adjacent tetrahedra share exact vertices.
func WalrusFresh3(yaw, phase float64) ([]thing.Thing, error) {
	ellipsoid := func(p, c, r maths.Vec) float64 {
		a := (p.X - c.X) / r.X
		b := (p.Y - c.Y) / r.Y
		d := (p.Z - c.Z) / r.Z
		return (math.Sqrt(a*a+b*b+d*d) - 1) * math.Min(r.X, math.Min(r.Y, r.Z))
	}
	smooth := func(a, b, k float64) float64 {
		h := math.Max(k-math.Abs(a-b), 0) / k
		return math.Min(a, b) - h*h*k*.25
	}
	field := func(p maths.Vec) float64 {
		d := ellipsoid(p, maths.V(-.12, 0, 0), maths.V(1.28, .48, .51))
		d = smooth(d, ellipsoid(p, maths.V(.88, .015, 0), maths.V(.50, .33, .35)), .16)
		for _, s := range []float64{-1, 1} {
			d = smooth(d, ellipsoid(p, maths.V(1.25, -.08, s*.13), maths.V(.28, .22, .18)), .12)
			d = smooth(d, ellipsoid(p, maths.V(.42, -.25, s*.51), maths.V(.34, .11, .38)), .14)
			d = smooth(d, ellipsoid(p, maths.V(-1.23, -.13, s*.24), maths.V(.38, .12, .32)), .16)
		}
		return d
	}
	const nx, ny, nz = 70, 38, 44
	sx, sy, sz := 3.5/float64(nx), 1.5/float64(ny), 2.0/float64(nz)
	point := func(i, j, k int) maths.Vec {
		return maths.V(-1.75+float64(i)*sx+.000137, -.75+float64(j)*sy+.000193, -1+float64(k)*sz+.000233)
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
	warp := func(p maths.Vec) maths.Vec {
		s := 1.55 - p.X
		if p.X < -.85 {
			w := math.Min(1, (-.85-p.X)/.55)
			p.Y += .12 * math.Sin(phase) * w
		}
		if math.Abs(yaw) < 1e-9 {
			return p
		}
		k := yaw / 3.1
		a := k * s
		return maths.V(1.55-math.Sin(a)/k+p.Z*math.Sin(a), p.Y, (1-math.Cos(a))/k+p.Z*math.Cos(a))
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
		return nil, fmt.Errorf("implicit walrus skin: %w", e)
	}
	if !skin.Closed() {
		return nil, fmt.Errorf("implicit walrus skin is open")
	}
	coat := &thing.Material{Name: "walrus skin", Colour: maths.RGB(.30, .185, .14), Gloss: .13, Satin: .59}
	ivory := &thing.Material{Name: "ivory", Colour: maths.RGB(.77, .72, .58), Gloss: .15, Satin: .42}
	eye := &thing.Material{Name: "eye", Colour: maths.RGB(.018, .013, .011), Gloss: .4}
	parts := []thing.Thing{{Shape: skin, Material: coat, Motion: thing.Still(maths.NewSubspace(maths.Vec{}, maths.Identity, 1))}}
	for _, side := range []float64{-1, 1} {
		at := warp(maths.V(1.03, .16, side*.285))
		parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .027}, Material: eye, Motion: thing.Still(maths.NewSubspace(at, maths.Identity, 1))})
		tusk, e := walrusFresh3Tooth(side, warp)
		if e != nil {
			return nil, e
		}
		parts = append(parts, thing.Thing{Shape: tusk, Material: ivory, Motion: thing.Still(maths.NewSubspace(maths.Vec{}, maths.Identity, 1))})
		for row := 0; row < 7; row++ {
			for col := 0; col < 10; col++ {
				y := .06 - float64(row)*.035
				z := side * (.015 + float64(col)*.022)
				lo, hi := 1.05, 1.75
				if field(maths.V(lo, y, z)) >= 0 {
					continue
				}
				for k := 0; k < 20; k++ {
					mid := (lo + hi) / 2
					if field(maths.V(mid, y, z)) < 0 {
						lo = mid
					} else {
						hi = mid
					}
				}
				a := warp(maths.V((lo+hi)/2, y, z))
				b := warp(maths.V((lo+hi)/2+.075, y-.04-.005*float64(row), z+side*.035))
				axis := b.Sub(a)
				parts = append(parts, thing.Thing{Shape: shape.Tube{Radius: .001, Thick: .002, Height: axis.Len()}, Material: ivory, Motion: thing.Still(maths.NewSubspace(a.Add(b).Scale(.5), maths.Upright(axis), 1))})
			}
		}
	}
	return parts, nil
}

// walrusFresh3Tooth is the mesh of one tusk, curved from its root to its
// tip and passed through warp; side is the sign of z.
func walrusFresh3Tooth(side float64, warp func(maths.Vec) maths.Vec) (*shape.Mesh, error) {
	const n, m = 18, 20
	v := []maths.Vec{}
	f := [][3]int{}
	// Generate each ring as a radial section of an authored curved root-to-tip path.
	center := func(t float64) maths.Vec { return maths.V(1.36+.08*t*t, -.17-.43*t, side*.14) }
	for i := 0; i <= n; i++ {
		t := float64(i) / n
		r := .04 * math.Pow(1-.98*t, .7)
		c := center(t)
		for j := 0; j < m; j++ {
			a := 2 * math.Pi * float64(j) / m
			v = append(v, warp(c.Add(maths.V(r*math.Cos(a), 0, r*math.Sin(a)))))
		}
	}
	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			a := i*m + j
			b := i*m + (j+1)%m
			c := (i+1)*m + (j+1)%m
			d := (i+1)*m + j
			f = append(f, [3]int{a, b, c}, [3]int{a, c, d})
		}
	}
	a := len(v)
	v = append(v, warp(center(0)))
	b := len(v)
	v = append(v, warp(center(1)))
	for j := 0; j < m; j++ {
		k := (j + 1) % m
		f = append(f, [3]int{a, k, j}, [3]int{b, n*m + j, n*m + k})
	}
	return shape.NewMesh(v, f, nil, false)
}
