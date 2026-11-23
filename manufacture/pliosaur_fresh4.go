package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"sort"
)

// PliosaurFresh4 builds a new indexed body and four flippers sewn to
// their exact shared boundary loops. Only the skull dimensions are sourced.
func PliosaurFresh4(phase float64) ([]thing.Thing, error) {
	const n, m = 96, 48
	type section struct{ x, y, z float64 }
	profile := []section{{-1.55, .015, .015}, {-1.25, .08, .08}, {-.95, .15, .14}, {-.72, .21, .23}, {-.35, .255, .32}, {.18, .26, .34}, {.45, .21, .26}, {.65, .14, .135}, {.78, .11, .12}, {.96, .085, .09}, {1.13, .065, .05}, {1.23, .05, .042}, {1.30, .048, .056}, {1.34, .018, .025}}
	radii := func(x float64) (float64, float64) {
		for i := 1; i < len(profile); i++ {
			if x <= profile[i].x {
				a, b := profile[i-1], profile[i]
				t := (x - a.x) / (b.x - a.x)
				return a.y*(1-t) + b.y*t, a.z*(1-t) + b.z*t
			}
		}
		a := profile[len(profile)-1]
		return a.y, a.z
	}
	v := make([]maths.Vec, 0, 7000)
	f := make([][3]int, 0, 14000)
	for i := 0; i <= n; i++ {
		x := -1.55 + 2.89*float64(i)/n
		ry, rz := radii(x)
		for j := 0; j < m; j++ {
			a := 2 * math.Pi * float64(j) / m
			v = append(v, maths.V(x, ry*math.Cos(a), rz*math.Sin(a)))
		}
	}
	hole := func(i, j int) bool {
		return ((i >= 54 && i < 60) || (i >= 24 && i < 30)) && ((j >= 9 && j < 15) || (j >= 33 && j < 39))
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
			f = append(f, [3]int{a, b, c}, [3]int{a, c, d})
		}
	}
	tail := len(v)
	v = append(v, maths.V(-1.55, 0, 0))
	nose := len(v)
	v = append(v, maths.V(1.34, 0, 0))
	for j := 0; j < m; j++ {
		k := (j + 1) % m
		f = append(f, [3]int{tail, k, j}, [3]int{nose, n*m + j, n*m + k})
	}
	type key struct{ a, b int }
	type edge struct{ a, b int }
	boundary := map[key]edge{}
	for _, face := range f {
		for j := 0; j < 3; j++ {
			a, b := face[j], face[(j+1)%3]
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
		if _, ok := next[e.a]; ok {
			return nil, fmt.Errorf("branched skin boundary")
		}
		next[e.a] = e.b
	}
	starts := []int{}
	for a := range next {
		starts = append(starts, a)
	}
	sort.Ints(starts)
	loops := 0
	for _, start := range starts {
		if _, ok := next[start]; !ok {
			continue
		}
		ring := []int{}
		p := start
		for {
			q, ok := next[p]
			if !ok {
				return nil, fmt.Errorf("open skin boundary")
			}
			delete(next, p)
			ring = append(ring, p)
			p = q
			if p == start {
				break
			}
			if len(ring) > 100 {
				return nil, fmt.Errorf("boundary too long")
			}
		}
		loops++
		center := maths.Vec{}
		for _, i := range ring {
			center = center.Add(v[i])
		}
		center = center.Scale(1 / float64(len(ring)))
		side := 1.
		if center.Z < 0 {
			side = -1
		}
		lag := 0.
		if center.X < 0 {
			lag = .55
		}
		previous := ring
		const steps = 12
		for step := 1; step < steps; step++ {
			t := float64(step) / steps
			scale := math.Pow(1-t, .65) * (1 + 1.1*math.Sin(math.Pi*t))
			w := t * t * (3 - 2*t)
			ang := .42 * math.Sin(phase+lag) * w
			c, s := math.Cos(ang), math.Sin(ang)
			fresh := make([]int, len(ring))
			for j, id := range ring {
				delta := v[id].Sub(center).Scale(scale)
				out := .62*t + side*delta.Z
				pos := maths.V(center.X-.17*t*t+delta.X, center.Y+delta.Y*c-out*s, center.Z+side*(delta.Y*s+out*c))
				fresh[j] = len(v)
				v = append(v, pos)
			}
			for j := range fresh {
				k := (j + 1) % len(fresh)
				a, b, c, d := previous[j], previous[k], fresh[j], fresh[k]
				f = append(f, [3]int{b, a, c}, [3]int{b, c, d})
			}
			previous = fresh
		}
		angle := .42 * math.Sin(phase+lag)
		tip := len(v)
		v = append(v, maths.V(center.X-.17, center.Y-.62*math.Sin(angle), center.Z+side*.62*math.Cos(angle)))
		for j := range previous {
			k := (j + 1) % len(previous)
			f = append(f, [3]int{previous[k], previous[j], tip})
		}
	}
	if loops != 4 {
		return nil, fmt.Errorf("expected four fin roots, found %d", loops)
	}
	// Remove the unused grid vertices inside each excised rectangle.
	used := make([]bool, len(v))
	for _, face := range f {
		for _, id := range face {
			used[id] = true
		}
	}
	mapping := make([]int, len(v))
	compact := make([]maths.Vec, 0, len(v))
	for i, p := range v {
		if used[i] {
			mapping[i] = len(compact)
			compact = append(compact, p)
		}
	}
	for i := range f {
		for j := 0; j < 3; j++ {
			f[i][j] = mapping[f[i][j]]
		}
	}
	v = compact
	normals := make([]maths.Vec, len(v))
	for _, face := range f {
		a, b, c := v[face[0]], v[face[1]], v[face[2]]
		nn := b.Sub(a).Cross(c.Sub(a))
		for _, id := range face {
			normals[id] = normals[id].Add(nn)
		}
	}
	skin, e := shape.NewMesh(v, f, normals, true)
	if e != nil {
		return nil, e
	}
	if !skin.Closed() {
		return nil, fmt.Errorf("open pliosaur skin")
	}
	place := maths.NewSubspace(maths.Vec{}, maths.Identity, 3)
	coat := &thing.Material{Name: "pliosaur clay", Texture: pliosaurTexture4{}, Gloss: .14, Satin: .5}
	dark := &thing.Material{Name: "eye", Colour: maths.RGB(.025, .027, .023), Gloss: .28}
	ivory := &thing.Material{Name: "inferred teeth", Colour: maths.RGB(.7, .67, .56), Gloss: .1}
	parts := []thing.Thing{{Shape: skin, Material: coat, Motion: thing.Still(place)}}
	for _, side := range []float64{-1, 1} {
		x, z := .83, side*.08
		ry, rz := radii(x)
		y := ry*math.Sqrt(math.Max(0, 1-(z/rz)*(z/rz))) + .002
		at := maths.V(x, y, z)
		parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .012}, Material: dark, Motion: thing.Still(maths.NewSubspace(at, maths.Identity, 1).Within(place))})
		for j := 0; j < 8; j++ {
			x := .95 + .045*float64(j)
			ry, rz := radii(x)
			y := -.025
			z := side * (rz*math.Sqrt(math.Max(0, 1-(y/ry)*(y/ry))) - .002)
			tooth, e := pliosaurTooth4(maths.V(x, y, z), .006, .033)
			if e != nil {
				return nil, e
			}
			parts = append(parts, thing.Thing{Shape: tooth, Material: ivory, Motion: thing.Still(place)})
		}
	}
	return parts, nil
}

type pliosaurTexture4 struct{}

// At is the skin's colour at a point: a dark mark on each side of the
// head, and a grey-green elsewhere.
func (pliosaurTexture4) At(p maths.Vec) maths.Light {
	if p.X > .73 && p.X < 1.33 && p.Y < -.022 && p.Y > -.028 && math.Abs(p.Z) > .025 {
		return maths.RGB(.055, .065, .06)
	}
	return maths.RGB(.31, .35, .32)
}

// pliosaurTooth4 is the mesh of one pointed tooth of a radius and length,
// set at a root point.
func pliosaurTooth4(root maths.Vec, radius, length float64) (*shape.Mesh, error) {
	const n = 12
	v := []maths.Vec{}
	f := [][3]int{}
	for j := 0; j < n; j++ {
		a := 2 * math.Pi * float64(j) / n
		v = append(v, root.Add(maths.V(radius*math.Cos(a), 0, radius*math.Sin(a))))
	}
	base := len(v)
	v = append(v, root)
	tip := len(v)
	v = append(v, root.Add(maths.V(.005, -length, 0)))
	for j := 0; j < n; j++ {
		k := (j + 1) % n
		f = append(f, [3]int{j, k, tip}, [3]int{base, k, j})
	}
	return shape.NewMesh(v, f, nil, false)
}
