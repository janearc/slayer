package manufacture

import (
	"github.com/janearc/slayer/scene/shape"
	"math"
	"testing"
)

func TestPurussaurusFresh1(t *testing.T) {
	a, e := PurussaurusFresh1(0, 0, 0)
	if e != nil {
		t.Fatal(e)
	}
	b, e := PurussaurusFresh1(math.Pi/2, .3, 1)
	if e != nil {
		t.Fatal(e)
	}
	c, e := PurussaurusFresh1(2*math.Pi, 0, 0)
	if e != nil {
		t.Fatal(e)
	}
	m := a[0].Shape.(*shape.Mesh)
	n := b[0].Shape.(*shape.Mesh)
	loop := c[0].Shape.(*shape.Mesh)
	if !m.Closed() || !n.Closed() {
		t.Fatal("open skin")
	}
	adj := make([][]int, len(m.Vertices))
	for _, f := range m.Faces {
		for i := 0; i < 3; i++ {
			u, v := f[i], f[(i+1)%3]
			adj[u] = append(adj[u], v)
			adj[v] = append(adj[v], u)
		}
	}
	seen := make([]bool, len(adj))
	todo := []int{0}
	seen[0] = true
	count := 0
	for len(todo) > 0 {
		u := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		count++
		for _, v := range adj[u] {
			if !seen[v] {
				seen[v] = true
				todo = append(todo, v)
			}
		}
	}
	active := 0
	for _, es := range adj {
		if len(es) > 0 {
			active++
		}
	}
	if count != active || active != len(adj) {
		t.Fatalf("connected %d active %d all %d", count, active, len(adj))
	}
	delta, loopErr := 0., 0.
	for i := range m.Vertices {
		delta = math.Max(delta, n.Vertices[i].Sub(m.Vertices[i]).Len())
		loopErr = math.Max(loopErr, loop.Vertices[i].Sub(m.Vertices[i]).Len())
	}
	if delta < .01 || loopErr > 1e-8 {
		t.Fatalf("pose delta %.5g loop %.5g", delta, loopErr)
	}
	volume := func(m *shape.Mesh) float64 {
		v := 0.
		for _, f := range m.Faces {
			a, b, c := m.Vertices[f[0]], m.Vertices[f[1]], m.Vertices[f[2]]
			v += a.Dot(b.Cross(c)) / 6
		}
		return math.Abs(v) * math.Pow(2.8, 3)
	}
	t.Logf("skin vertices=%d faces=%d; volume rest %.4f m3 posed %.4f m3; pose max %.4f m loop %.3g m", len(m.Vertices), len(m.Faces), volume(m), volume(n), 2.8*delta, 2.8*loopErr)
}

func TestPurussaurusFresh1ShapeAndStrain(t *testing.T) {
	a, e := PurussaurusFresh1(0, 0, 0)
	if e != nil {
		t.Fatal(e)
	}
	b, e := PurussaurusFresh1(math.Pi/2, .3, 1)
	if e != nil {
		t.Fatal(e)
	}
	m := a[0].Shape.(*shape.Mesh)
	n := b[0].Shape.(*shape.Mesh)
	xmin, xmax, zmin, zmax := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, p := range m.Vertices {
		if p.X >= .9 {
			xmin = math.Min(xmin, p.X)
			xmax = math.Max(xmax, p.X)
			zmin = math.Min(zmin, p.Z)
			zmax = math.Max(zmax, p.Z)
		}
	}
	minStretch, maxStretch := math.Inf(1), 0.
	for _, f := range m.Faces {
		for i := 0; i < 3; i++ {
			u, v := f[i], f[(i+1)%3]
			l := m.Vertices[u].Sub(m.Vertices[v]).Len()
			r := n.Vertices[u].Sub(n.Vertices[v]).Len() / l
			minStretch = math.Min(minStretch, r)
			maxStretch = math.Max(maxStretch, r)
		}
	}
	t.Logf("authored cranial surface x>=.9 bounding length %.3f m and width %.3f m; reference max skull 1.400 x .820 m; not matching the same anatomical landmarks", (xmax-xmin)*2.8, (zmax-zmin)*2.8)
	t.Logf("posed/rest edge length range %.5f..%.5f; extrema alone do not prove absence of folds or collisions", minStretch, maxStretch)
}
