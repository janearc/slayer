package manufacture

import (
	"github.com/janearc/slayer/scene/shape"
	"math"
	"sort"
	"testing"
)

func TestPurussaurusStudy(t *testing.T) {
	a, e := PurussaurusStudy(0, 0, 0)
	if e != nil {
		t.Fatal(e)
	}
	b, e := PurussaurusStudy(math.Pi/2, .3, 1)
	if e != nil {
		t.Fatal(e)
	}
	c, e := PurussaurusStudy(2*math.Pi, 0, 0)
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

func TestPurussaurusStudyShapeAndStrain(t *testing.T) {
	a, e := PurussaurusStudy(0, 0, 0)
	if e != nil {
		t.Fatal(e)
	}
	b, e := PurussaurusStudy(math.Pi/2, .3, 1)
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

func TestPurussaurusStudyEdgeDistribution(t *testing.T) {
	a, e := PurussaurusStudy(0, 0, 0)
	if e != nil {
		t.Fatal(e)
	}
	b, e := PurussaurusStudy(math.Pi/2, .3, 1)
	if e != nil {
		t.Fatal(e)
	}
	m, n := a[0].Shape.(*shape.Mesh), b[0].Shape.(*shape.Mesh)
	changes := make([]float64, 0, 3*len(m.Faces))
	for _, f := range m.Faces {
		for i := 0; i < 3; i++ {
			u, v := f[i], f[(i+1)%3]
			l := m.Vertices[u].Sub(m.Vertices[v]).Len()
			q := n.Vertices[u].Sub(n.Vertices[v]).Len()
			changes = append(changes, math.Abs(q/l-1))
		}
	}
	sort.Float64s(changes)
	t.Logf("absolute edge-length change p50 %.3f%% p95 %.3f%% p99 %.3f%% max %.3f%%; repeated triangle edges counted", 100*changes[len(changes)/2], 100*changes[len(changes)*95/100], 100*changes[len(changes)*99/100], 100*changes[len(changes)-1])
}

func TestPurussaurusStudyNoTopologyHandle(t *testing.T) {
	p, e := PurussaurusStudy(0, 0, 0)
	if e != nil {
		t.Fatal(e)
	}
	m := p[0].Shape.(*shape.Mesh)
	// For the checked single closed triangular component, E=3F/2.
	chi := len(m.Vertices) - len(m.Faces)/2
	if chi != 2 {
		t.Fatalf("unexpected Euler characteristic %d; possible handle or orphan", chi)
	}
}

func TestPurussaurusStudyToothAttachments(t *testing.T) {
	a, e := PurussaurusStudy(0, 0, 0)
	if e != nil {
		t.Fatal(e)
	}
	b, e := PurussaurusStudy(math.Pi/2, .3, 1)
	if e != nil {
		t.Fatal(e)
	}
	skin := a[0].Shape.(*shape.Mesh)
	maxGap, maxMotion := 0., 0.
	for _, i := range []int{3, 4, 5, 6, 7, 10, 11, 12, 13, 14} {
		t0 := a[i].Shape.(*shape.Mesh)
		t1 := b[i].Shape.(*shape.Mesh)
		root := t0.Vertices[len(t0.Vertices)-2]
		near := math.Inf(1)
		for _, v := range skin.Vertices {
			near = math.Min(near, v.Sub(root).Len())
		}
		maxGap = math.Max(maxGap, near)
		for j := range t0.Vertices {
			maxMotion = math.Max(maxMotion, t0.Vertices[j].Sub(t1.Vertices[j]).Len())
		}
	}
	t.Logf("ten illustrative teeth: worst root-to-nearest-skin-vertex %.4f m; max rigid-tooth point drift %.3g m", maxGap*2.8, maxMotion*2.8)
	if maxGap*2.8 > .20 || maxMotion*2.8 > 1e-8 {
		t.Fatalf("teeth detached or unexpectedly deformed")
	}
}

func TestPurussaurusStudyBoundsAndGround(t *testing.T) {
	a, e := PurussaurusStudy(0, 0, 0)
	if e != nil {
		t.Fatal(e)
	}
	g, e := PurussaurusStudy(0, 0, 1)
	if e != nil {
		t.Fatal(e)
	}
	m := a[0].Shape.(*shape.Mesh)
	n := g[0].Shape.(*shape.Mesh)
	xmin, xmax, ymin, ymax, zmin, zmax := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, p := range m.Vertices {
		xmin = math.Min(xmin, p.X)
		xmax = math.Max(xmax, p.X)
		ymin = math.Min(ymin, p.Y)
		ymax = math.Max(ymax, p.Y)
		zmin = math.Min(zmin, p.Z)
		zmax = math.Max(zmax, p.Z)
	}
	groundMin := math.Inf(1)
	for _, p := range n.Vertices {
		groundMin = math.Min(groundMin, p.Y*2.8)
	}
	t.Logf("rest skin bounds x %.3f..%.3f m (length %.3f), y %.3f..%.3f m, z %.3f..%.3f m; ground skin min y %.3f m, plane y -0.820 m, signed clearance %.3f m", xmin*2.8, xmax*2.8, (xmax-xmin)*2.8, ymin*2.8, ymax*2.8, zmin*2.8, zmax*2.8, groundMin, groundMin+.82)
}
