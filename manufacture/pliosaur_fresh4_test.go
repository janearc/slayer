package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"math"
	"sort"
	"testing"
)

func TestPliosaurFresh4(t *testing.T) {
	a, e := PliosaurFresh4(0)
	if e != nil {
		t.Fatal(e)
	}
	b, e := PliosaurFresh4(math.Pi / 2)
	if e != nil {
		t.Fatal(e)
	}
	c, e := PliosaurFresh4(2 * math.Pi)
	if e != nil {
		t.Fatal(e)
	}
	m := a[0].Shape.(*shape.Mesh)
	n := b[0].Shape.(*shape.Mesh)
	z := c[0].Shape.(*shape.Mesh)
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
	seen[0] = true
	stack := []int{0}
	count := 0
	for len(stack) > 0 {
		u := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		count++
		for _, v := range adj[u] {
			if !seen[v] {
				seen[v] = true
				stack = append(stack, v)
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
	chi := len(m.Vertices) - len(m.Faces)/2
	maxmove, loop := 0., 0.
	changes := make([]float64, 0, 3*len(m.Faces))
	for i := range m.Vertices {
		maxmove = math.Max(maxmove, n.Vertices[i].Sub(m.Vertices[i]).Len())
		loop = math.Max(loop, z.Vertices[i].Sub(m.Vertices[i]).Len())
	}
	for _, f := range m.Faces {
		for i := 0; i < 3; i++ {
			u, v := f[i], f[(i+1)%3]
			l := m.Vertices[u].Sub(m.Vertices[v]).Len()
			q := n.Vertices[u].Sub(n.Vertices[v]).Len()
			changes = append(changes, math.Abs(q/l-1))
		}
	}
	sort.Float64s(changes)
	volume := func(m *shape.Mesh) float64 {
		v := 0.
		for _, f := range m.Faces {
			a, b, c := m.Vertices[f[0]], m.Vertices[f[1]], m.Vertices[f[2]]
			v += a.Dot(b.Cross(c)) / 6
		}
		return math.Abs(v) * 27
	}
	xmin, xmax, zmin, zmax := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, p := range m.Vertices {
		if p.X >= .66 {
			xmin = math.Min(xmin, p.X)
			xmax = math.Max(xmax, p.X)
			zmin = math.Min(zmin, p.Z)
			zmax = math.Max(zmax, p.Z)
		}
	}
	if maxmove < .02 || loop > 1e-8 {
		t.Fatalf("displacement %.4f loop %.3g", maxmove, loop)
	}
	t.Logf("connected closed skin V=%d F=%d chi=%d; head bbox %.3f x %.3f m (threshold, not fossil landmarks)", len(m.Vertices), len(m.Faces), chi, (xmax-xmin)*3, (zmax-zmin)*3)
	t.Logf("volume %.3f -> %.3f m3; displacement %.3f m, loop %.3g m; edge change p95 %.2f%% p99 %.2f%% max %.2f%%", volume(m), volume(n), maxmove*3, loop*3, 100*changes[len(changes)*95/100], 100*changes[len(changes)*99/100], 100*changes[len(changes)-1])
}
func TestPliosaurFresh4WorstEdge(t *testing.T) {
	a, e := PliosaurFresh4(0)
	if e != nil {
		t.Fatal(e)
	}
	b, e := PliosaurFresh4(math.Pi / 2)
	if e != nil {
		t.Fatal(e)
	}
	m, n := a[0].Shape.(*shape.Mesh), b[0].Shape.(*shape.Mesh)
	worst, l0, l1 := 0., 0., 0.
	p := maths.Vec{}
	for _, f := range m.Faces {
		for i := 0; i < 3; i++ {
			u, v := f[i], f[(i+1)%3]
			q := m.Vertices[u].Sub(m.Vertices[v]).Len()
			r := n.Vertices[u].Sub(n.Vertices[v]).Len()
			change := math.Abs(r/q - 1)
			if change > worst {
				worst, l0, l1, p = change, q, r, m.Vertices[u]
			}
		}
	}
	t.Logf("worst edge %.2f%% rest %.8f m posed %.8f m at normalized %+v", 100*worst, l0*3, l1*3, p)
}

func TestPliosaurFresh4FormerControlSeam(t *testing.T) {
	a, e := PliosaurFresh4(0)
	if e != nil {
		t.Fatal(e)
	}
	b, e := PliosaurFresh4(math.Pi / 2)
	if e != nil {
		t.Fatal(e)
	}
	m, n := a[0].Shape.(*shape.Mesh), b[0].Shape.(*shape.Mesh)
	count := 0
	worst := 0.
	for i, p := range m.Vertices {
		if p.X > -.29 && p.X < -.25 && math.Abs(p.Z) < .4 {
			count++
			worst = math.Max(worst, n.Vertices[i].Sub(p).Len()*3)
		}
	}
	if count == 0 {
		t.Fatal("no former seam sample")
	}
	t.Logf("former x=-.27 control boundary: %d body vertices, max displacement %.8f m", count, worst)
}
