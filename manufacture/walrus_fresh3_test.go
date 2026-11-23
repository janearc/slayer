package manufacture

import (
	"github.com/janearc/slayer/scene/shape"
	"math"
	"testing"
)

func TestWalrusFresh3Construction(t *testing.T) {
	for _, p := range []struct{ yaw, phase float64 }{{0, 0}, {.8, 0}, {.28, math.Pi / 2}} {
		if _, e := WalrusFresh3(p.yaw, p.phase); e != nil {
			t.Fatal(e)
		}
	}
}
func TestWalrusFresh3ConnectedAndMotion(t *testing.T) {
	rest, e := WalrusFresh3(0, 0)
	if e != nil {
		t.Fatal(e)
	}
	turn, e := WalrusFresh3(.8, 0)
	if e != nil {
		t.Fatal(e)
	}
	swim, e := WalrusFresh3(.28, math.Pi/2)
	if e != nil {
		t.Fatal(e)
	}
	loop, e := WalrusFresh3(0, 2*math.Pi)
	if e != nil {
		t.Fatal(e)
	}
	a := rest[0].Shape.(*shape.Mesh)
	b := turn[0].Shape.(*shape.Mesh)
	c := swim[0].Shape.(*shape.Mesh)
	d := loop[0].Shape.(*shape.Mesh)
	if !a.Closed() || !b.Closed() || !c.Closed() {
		t.Fatal("open shell")
	}
	adj := make([][]int, len(a.Vertices))
	for _, f := range a.Faces {
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
	for _, edges := range adj {
		if len(edges) > 0 {
			active++
		}
	}
	if count != active {
		t.Fatalf("connected %d of %d face-referenced vertices", count, active)
	}
	t.Logf("face-referenced vertices=%d; unused vertices=%d", active, len(a.Vertices)-active)
	volume := func(m *shape.Mesh) float64 {
		v := 0.
		for _, f := range m.Faces {
			x, y, z := m.Vertices[f[0]], m.Vertices[f[1]], m.Vertices[f[2]]
			v += x.Dot(y.Cross(z)) / 6
		}
		return math.Abs(v)
	}
	v0, v1 := volume(a), volume(b)
	maxMove, loopErr := 0., 0.
	for i := range a.Vertices {
		maxMove = math.Max(maxMove, c.Vertices[i].Sub(a.Vertices[i]).Len())
		loopErr = math.Max(loopErr, d.Vertices[i].Sub(a.Vertices[i]).Len())
	}
	if maxMove < .05 || loopErr > 1e-8 {
		t.Fatalf("movement %f loop error %g", maxMove, loopErr)
	}
	t.Logf("one connected closed skin: %d vertices, %d faces, volume rest %.6f m3 and turn %.6f m3 (change %.2f%%), max swim displacement %.3f m, loop error %.3g m", len(a.Vertices), len(a.Faces), v0, v1, 100*(v1-v0)/v0, maxMove, loopErr)
}
