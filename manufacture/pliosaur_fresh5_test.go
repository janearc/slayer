package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"math"
	"sort"
	"testing"
)

func TestPliosaurFresh5(t *testing.T) {
	a, e := PliosaurFresh5(0)
	if e != nil {
		t.Fatal(e)
	}
	b, e := PliosaurFresh5(math.Pi / 2)
	if e != nil {
		t.Fatal(e)
	}
	c, e := PliosaurFresh5(2 * math.Pi)
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

func TestPliosaurFresh5WorstEdge(t *testing.T) {
	a, e := PliosaurFresh5(0)
	if e != nil {
		t.Fatal(e)
	}
	b, e := PliosaurFresh5(math.Pi / 2)
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

func TestPliosaurFresh5AttachmentsAndSeam(t *testing.T) {
	a, e := PliosaurFresh5(0)
	if e != nil {
		t.Fatal(e)
	}
	b, e := PliosaurFresh5(math.Pi / 2)
	if e != nil {
		t.Fatal(e)
	}
	m, n := a[0].Shape.(*shape.Mesh), b[0].Shape.(*shape.Mesh)
	maxGap, maxDrift := 0., 0.
	for _, i := range []int{2, 3, 4, 5, 6, 7, 8, 9, 11, 12, 13, 14, 15, 16, 17, 18} {
		u := a[i].Shape.(*shape.Mesh)
		v := b[i].Shape.(*shape.Mesh)
		root := u.Vertices[len(u.Vertices)-2]
		nearest := math.Inf(1)
		for _, p := range m.Vertices {
			nearest = math.Min(nearest, p.Sub(root).Len())
		}
		maxGap = math.Max(maxGap, nearest)
		for j := range u.Vertices {
			maxDrift = math.Max(maxDrift, v.Vertices[j].Sub(u.Vertices[j]).Len())
		}
	}
	worst := 0.
	count := 0
	for _, f := range m.Faces {
		for j := 0; j < 3; j++ {
			u, v := f[j], f[(j+1)%3]
			p, q := m.Vertices[u], m.Vertices[v]
			if p.X > -.29 && p.X < -.25 && q.X > -.29 && q.X < -.25 {
				l := p.Sub(q).Len()
				r := n.Vertices[u].Sub(n.Vertices[v]).Len()
				worst = math.Max(worst, math.Abs(r/l-1))
				count++
			}
		}
	}
	if count == 0 {
		t.Fatal("former seam has no sampled edges")
	}
	t.Logf("sixteen illustrative teeth: max nearest-vertex root distance %.4f m; rigid point drift %.3g m; former x=-.27 region %d face edges, worst length change %.3f%%", maxGap*3, maxDrift*3, count, worst*100)
	if maxDrift*3 > 1e-8 {
		t.Fatal("unexpected tooth deformation")
	}
}

func TestPliosaurFresh5BoundsAndFourPaddles(t *testing.T) {
	a, e := PliosaurFresh5(0)
	if e != nil {
		t.Fatal(e)
	}
	b, e := PliosaurFresh5(math.Pi / 2)
	if e != nil {
		t.Fatal(e)
	}
	m, n := a[0].Shape.(*shape.Mesh), b[0].Shape.(*shape.Mesh)
	xmin, xmax, ymin, ymax, zmin, zmax := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	moves := [4]float64{}
	counts := [4]int{}
	for i, p := range m.Vertices {
		xmin = math.Min(xmin, p.X)
		xmax = math.Max(xmax, p.X)
		ymin = math.Min(ymin, p.Y)
		ymax = math.Max(ymax, p.Y)
		zmin = math.Min(zmin, p.Z)
		zmax = math.Max(zmax, p.Z)
		if math.Abs(p.Z) > .40 {
			idx := 0
			if p.X < -.27 {
				idx += 2
			}
			if p.Z < 0 {
				idx++
			}
			counts[idx]++
			moves[idx] = math.Max(moves[idx], n.Vertices[i].Sub(p).Len()*3)
		}
	}
	for i := range moves {
		if counts[i] == 0 || moves[i] < .05 {
			t.Fatalf("quadrant %d count %d movement %.3f m", i, counts[i], moves[i])
		}
	}
	t.Logf("rest skin bounds x %.3f..%.3f m length %.3f; y %.3f..%.3f; z %.3f..%.3f; paddle quadrant samples %v max displacements %v m", xmin*3, xmax*3, (xmax-xmin)*3, ymin*3, ymax*3, zmin*3, zmax*3, counts, moves)
}
