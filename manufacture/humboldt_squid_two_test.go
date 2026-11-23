package manufacture

import (
	"github.com/janearc/slayer/mesh"
	"math"
	"testing"
)

func TestHumboldtSquidTwoCorrespondence(t *testing.T) {
	a, err := HumboldtSquidTwo(0)
	if err != nil {
		t.Fatal(err)
	}
	b, err := HumboldtSquidTwo(math.Pi / 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != len(b) || len(a) < 2 {
		t.Fatalf("pieces: %d %d", len(a), len(b))
	}
	max := 0.
	meshes := 0
	for i := range a {
		x, ok := a[i].Shape.(*mesh.Mesh)
		if !ok {
			continue
		}
		y, ok := b[i].Shape.(*mesh.Mesh)
		if !ok {
			t.Fatal("mesh type changed")
		}
		p, q := x.Points(), y.Points()
		if len(p) != len(q) || x.FaceCount() != y.FaceCount() {
			t.Fatal("lost correspondence")
		}
		meshes++
		for j := range p {
			max = math.Max(max, p[j].Sub(q[j]).Len())
		}
	}
	if max < .025 {
		t.Fatalf("no visible movement: %.5f", max)
	}
	t.Logf("%d indexed mesh pieces, maximum movement %.5f m", meshes, max)
}

func TestHumboldtSquidTwoRestTopology(t *testing.T) {
	m, err := squidTwoRest()
	if err != nil {
		t.Fatal(err)
	}
	edges := map[[2]int]int{}
	adj := make([][]int, len(m.Vertices))
	used := map[int]bool{}
	for _, f := range m.Faces {
		for k := 0; k < 3; k++ {
			a, b := f[k], f[(k+1)%3]
			used[a] = true
			used[b] = true
			adj[a] = append(adj[a], b)
			adj[b] = append(adj[b], a)
			if a > b {
				a, b = b, a
			}
			edges[[2]int{a, b}]++
		}
	}
	bad := 0
	for _, n := range edges {
		if n != 2 {
			bad++
		}
	}
	seen := map[int]bool{}
	components := 0
	for v := range used {
		if seen[v] {
			continue
		}
		components++
		stack := []int{v}
		seen[v] = true
		for len(stack) > 0 {
			n := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			for _, w := range adj[n] {
				if !seen[w] {
					seen[w] = true
					stack = append(stack, w)
				}
			}
		}
	}
	t.Logf("vertices=%d referenced=%d faces=%d nonmanifold-or-boundary=%d components=%d", len(m.Vertices), len(used), len(m.Faces), bad, components)
	if bad != 0 {
		t.Fatalf("%d edges not incident to exactly two faces", bad)
	}
	if components != 1 {
		t.Fatalf("got %d disconnected components", components)
	}
}

func TestHumboldtSquidTwoRestRelativeEdgeRange(t *testing.T) {
	raw, err := squidTwoRest()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[[2]int]bool{}
	edges := [][2]int{}
	for _, f := range raw.Faces {
		for k := 0; k < 3; k++ {
			a, b := f[k], f[(k+1)%3]
			if a > b {
				a, b = b, a
			}
			e := [2]int{a, b}
			if !seen[e] {
				seen[e] = true
				edges = append(edges, e)
			}
		}
	}
	lo, hi := math.Inf(1), 0.
	for k := 0; k < 8; k++ {
		v, e := HumboldtSquidTwo(2 * math.Pi * float64(k) / 8)
		if e != nil {
			t.Fatal(e)
		}
		p := v[0].Shape.(*mesh.Mesh).Points()
		for _, edge := range edges {
			a, b := edge[0], edge[1]
			d := raw.Vertices[a].Sub(raw.Vertices[b]).Len() * .5
			if d < 1e-9 {
				continue
			}
			q := p[a].Sub(p[b]).Len() / d
			lo = math.Min(lo, q)
			hi = math.Max(hi, q)
		}
	}
	t.Logf("8 phases, %d mesh edges, min/max rest-relative edge ratio %.5f / %.5f", len(edges), lo, hi)
	if math.IsNaN(lo) || math.IsInf(lo, 0) || math.IsNaN(hi) || math.IsInf(hi, 0) {
		t.Fatal("nonfinite deformation")
	}
}
