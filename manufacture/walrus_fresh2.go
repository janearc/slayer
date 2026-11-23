package manufacture

import (
	"fmt"
	"math"
	"sort"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// WalrusFresh2 constructs a new manifold body by growing fins from the
// actual boundary edges left by rectangular openings in an axial loft.
// Its design is original; the proportions and motion are study choices.
func WalrusFresh2(yaw, phase float64) ([]thing.Thing, error) {
	const n, m = 48, 48
	type key struct{ a, b int }
	type edge struct{ a, b int }
	verts := make([]maths.Vec, 0, 3000)
	faces := make([][3]int, 0, 6000)
	profile := [][2]float64{{0, .025}, {.08, .23}, {.19, .37}, {.38, .48}, {.58, .46}, {.70, .39}, {.80, .32}, {.88, .29}, {.94, .26}, {.98, .20}, {1, .06}}
	radius := func(t float64) float64 {
		for i := 1; i < len(profile); i++ {
			if t <= profile[i][0] {
				a, b := profile[i-1], profile[i]
				v := (t - a[0]) / (b[0] - a[0])
				return a[1]*(1-v) + b[1]*v
			}
		}
		return .06
	}
	for i := 0; i <= n; i++ {
		t := float64(i) / n
		x := -1.5 + 3*t
		r := radius(t)
		for j := 0; j < m; j++ {
			a := 2 * math.Pi * float64(j) / m
			y := r * .94 * math.Cos(a)
			z := r * math.Sin(a)
			// A few broad neck folds are small displacements of the same skin.
			if t > .68 && t < .84 {
				ripple := .012 * math.Sin((t-.68)*math.Pi*32) * math.Sin(math.Pi*(t-.68)/.16)
				y += ripple * math.Cos(a)
				z += ripple * math.Sin(a)
			}
			verts = append(verts, maths.V(x, y, z))
		}
	}
	hole := func(i, j int) bool {
		return ((i >= 27 && i < 32) || (i >= 5 && i < 10)) && ((j >= 9 && j < 15) || (j >= 33 && j < 39))
	}
	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			if hole(i, j) {
				continue
			}
			a := i*m + j
			b := i*m + (j+1)%m
			c := (i+1)*m + (j+1)%m
			d := (i+1)*m + j
			faces = append(faces, [3]int{a, b, c}, [3]int{a, c, d})
		}
	}
	tail := len(verts)
	verts = append(verts, maths.V(-1.5, 0, 0))
	nose := len(verts)
	verts = append(verts, maths.V(1.5, 0, 0))
	for j := 0; j < m; j++ {
		k := (j + 1) % m
		faces = append(faces, [3]int{tail, k, j}, [3]int{nose, n*m + j, n*m + k})
	}
	boundary := map[key]edge{}
	for _, f := range faces {
		for j := 0; j < 3; j++ {
			a, b := f[j], f[(j+1)%3]
			k := key{a, b}
			if a > b {
				k = key{b, a}
			}
			if _, ok := boundary[k]; ok {
				delete(boundary, k)
			} else {
				boundary[k] = edge{a, b}
			}
		}
	}
	next := map[int]int{}
	for _, e := range boundary {
		if _, exists := next[e.a]; exists {
			return nil, fmt.Errorf("branched boundary at %d", e.a)
		}
		next[e.a] = e.b
	}
	starts := make([]int, 0, len(next))
	for a := range next {
		starts = append(starts, a)
	}
	sort.Ints(starts)
	loops := 0
	for _, start := range starts {
		if _, ok := next[start]; !ok {
			continue
		}
		loop := []int{}
		p := start
		for {
			q, ok := next[p]
			if !ok {
				return nil, fmt.Errorf("open boundary %d", p)
			}
			delete(next, p)
			loop = append(loop, p)
			p = q
			if p == start {
				break
			}
			if len(loop) > 100 {
				return nil, fmt.Errorf("boundary loop too long")
			}
		}
		loops++
		center := maths.Vec{}
		for _, v := range loop {
			center = center.Add(verts[v])
		}
		center = center.Scale(1 / float64(len(loop)))
		side := 1.
		if center.Z < 0 {
			side = -1
		}
		hind := center.X < 0
		previous := loop
		const steps = 7
		for step := 1; step < steps; step++ {
			t := float64(step) / steps
			ring := make([]int, len(loop))
			for k, v := range loop {
				p := verts[v].Sub(center).Scale(1 - .8*t).Add(center)
				p.Z += side * .48 * t
				p.X -= .18 * t
				p.Y -= .12 * t
				if hind {
					p.Y += .16 * math.Sin(phase) * t
					p.Z += side * .08 * math.Sin(phase) * t
				}
				ring[k] = len(verts)
				verts = append(verts, p)
			}
			for k := range ring {
				q := (k + 1) % len(ring)
				a, b := previous[k], previous[q]
				c, d := ring[k], ring[q]
				faces = append(faces, [3]int{b, a, c}, [3]int{b, c, d})
			}
			previous = ring
		}
		tip := center.Add(maths.V(-.18, -.12, side*.48))
		if hind {
			tip.Y += .16 * math.Sin(phase)
			tip.Z += side * .08 * math.Sin(phase)
		}
		tipIndex := len(verts)
		verts = append(verts, tip)
		for k := range previous {
			q := (k + 1) % len(previous)
			faces = append(faces, [3]int{previous[q], previous[k], tipIndex})
		}
	}
	if loops != 4 {
		return nil, fmt.Errorf("expected 4 fin boundaries; found %d", loops)
	}
	// Arc-length-preserving centerline. Cross sections rotate with the tangent.
	warp := func(p maths.Vec) maths.Vec {
		s := 1.5 - p.X
		if math.Abs(yaw) < 1e-9 {
			return p
		}
		k := yaw / 3
		a := k * s
		return maths.V(1.5-math.Sin(a)/k+p.Z*math.Sin(a), p.Y, (1-math.Cos(a))/k+p.Z*math.Cos(a))
	}
	for i := range verts {
		verts[i] = warp(verts[i])
	}
	normals := make([]maths.Vec, len(verts))
	for _, f := range faces {
		a, b, c := verts[f[0]], verts[f[1]], verts[f[2]]
		normal := b.Sub(a).Cross(c.Sub(a))
		for _, v := range f {
			normals[v] = normals[v].Add(normal)
		}
	}
	skin, err := shape.NewMesh(verts, faces, normals, true)
	if err != nil {
		return nil, err
	}
	if !skin.Closed() {
		return nil, fmt.Errorf("connected skin is open")
	}
	coat := &thing.Material{Name: "walrus skin", Colour: maths.RGB(.30, .185, .14), Gloss: .13, Satin: .59}
	ivory := &thing.Material{Name: "ivory", Colour: maths.RGB(.77, .72, .58), Gloss: .15, Satin: .42}
	eye := &thing.Material{Name: "eye", Colour: maths.RGB(.018, .013, .011), Gloss: .4}
	parts := []thing.Thing{{Shape: skin, Material: coat, Motion: thing.Still(maths.NewSubspace(maths.Vec{}, maths.Identity, 1))}}
	for _, side := range []float64{-1, 1} {
		at := warp(maths.V(1.14, .14, side*.24))
		parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .025}, Material: eye, Motion: thing.Still(maths.NewSubspace(at, maths.Identity, 1))})
		tusk, e := walrusFresh2Tusk(side, warp)
		if e != nil {
			return nil, e
		}
		parts = append(parts, thing.Thing{Shape: tusk, Material: ivory, Motion: thing.Still(maths.NewSubspace(maths.Vec{}, maths.Identity, 1))})
		for row := 0; row < 5; row++ {
			for col := 0; col < 10; col++ {
				z := side * (.03 + .018*float64(col))
				root := warp(maths.V(1.46, -.01-.028*float64(row), z))
				end := warp(maths.V(1.55, -.08-.045*float64(row), z+side*.09))
				axis := end.Sub(root)
				parts = append(parts, thing.Thing{Shape: shape.Tube{Radius: .001, Thick: .002, Height: axis.Len()}, Material: ivory, Motion: thing.Still(maths.NewSubspace(root.Add(end).Scale(.5), maths.Upright(axis), 1))})
			}
		}
	}
	return parts, nil
}

// walrusFresh2Tusk is the mesh of one tusk, curved from its root to its
// tip and passed through warp; side is the sign of z.
func walrusFresh2Tusk(side float64, warp func(maths.Vec) maths.Vec) (*shape.Mesh, error) {
	const n, m = 12, 16
	v := []maths.Vec{}
	f := [][3]int{}
	for i := 0; i <= n; i++ {
		t := float64(i) / n
		r := .035 * (1 - .97*t)
		for j := 0; j < m; j++ {
			a := 2 * math.Pi * float64(j) / m
			v = append(v, warp(maths.V(1.35+.07*t*t+r*math.Cos(a), -.12-.35*t, side*.13+r*math.Sin(a))))
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
	v = append(v, warp(maths.V(1.35, -.12, side*.13)))
	b := len(v)
	v = append(v, warp(maths.V(1.42, -.47, side*.13)))
	for j := 0; j < m; j++ {
		k := (j + 1) % m
		f = append(f, [3]int{a, k, j}, [3]int{b, n*m + j, n*m + k})
	}
	return shape.NewMesh(v, f, nil, false)
}
