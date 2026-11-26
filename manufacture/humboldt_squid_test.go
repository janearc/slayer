package manufacture

import (
	"bytes"
	"fmt"
	"github.com/janearc/slayer/manufacture/animals/molluscs/squid"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"sync"
	"testing"
)

// squidWithLength builds through the same checked external profile path used by scenes.
func squidWithLength(length, phase float64) ([]thing.Thing, error) {
	old := []byte("mantle-metres: 0.8")
	raw := bytes.Replace(squidProfilesRaw, old, []byte(fmt.Sprintf("mantle-metres: %g", length)), 1)
	if bytes.Equal(raw, squidProfilesRaw) && length != .8 {
		return nil, fmt.Errorf("missing mantle fixture")
	}
	book, err := ReadSquidProfiles("squid-test.yaml", raw)
	if err != nil {
		return nil, err
	}
	return book.NewSquid("humboldt-squid", phase)
}

var humboldtSquidRest = sync.OnceValues(func() (*shape.Mesh, error) {
	return squid.BuildRest(squid.Anatomy{MantleRadius: .255, HeadWidth: .21, FinSpan: .25, ArmLength: 1, TentacleLength: 1.48})
})

func TestNewHumboldtSquidCorrespondence(t *testing.T) {
	a, err := NewHumboldtSquid(0)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewHumboldtSquid(math.Pi / 2)
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

func TestNewHumboldtSquidRestTopology(t *testing.T) {
	m, err := humboldtSquidRest()
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

func TestNewHumboldtSquidRestRelativeEdgeRange(t *testing.T) {
	raw, err := humboldtSquidRest()
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
		v, e := NewHumboldtSquid(2 * math.Pi * float64(k) / 8)
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

func TestHumboldtSquidRejectsNonfinitePhases(t *testing.T) {
	for _, phase := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := NewHumboldtSquid(phase); err == nil {
			t.Fatalf("accepted nonfinite phase %v", phase)
		}
	}
}

func TestSquidSizeScalesSkinAndDetails(t *testing.T) {
	small, err := squidWithLength(.8, .3)
	if err != nil {
		t.Fatal(err)
	}
	large, err := squidWithLength(1.6, .3)
	if err != nil {
		t.Fatal(err)
	}
	if len(small) != len(large) {
		t.Fatalf("piece count changed: %d and %d", len(small), len(large))
	}
	a := small[0].Shape.(*mesh.Mesh).Points()
	b := large[0].Shape.(*mesh.Mesh).Points()
	if len(a) != len(b) {
		t.Fatal("skin topology changed with size")
	}
	for i := range a {
		if b[i].Sub(a[i].Scale(2)).Len() > 1e-12 {
			t.Fatalf("skin point %d did not double", i)
		}
	}
	for i := 1; i < len(small); i++ {
		if large[i].At(0).Origin.Sub(small[i].At(0).Origin.Scale(2)).Len() > 1e-12 {
			t.Fatalf("detail %d did not move with skin", i)
		}
		a, b := small[i].Shape.Bounds(), large[i].Shape.Bounds()
		if b.Min.Sub(a.Min.Scale(2)).Len() > 1e-12 || b.Max.Sub(a.Max.Scale(2)).Len() > 1e-12 {
			t.Fatalf("detail %d did not double", i)
		}
	}
}

func TestSquidRejectsInvalidSize(t *testing.T) {
	for _, size := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := squidWithLength(size, 0); err == nil {
			t.Fatalf("accepted mantle length %v", size)
		}
	}
}

// The original squid's 3×3×3 Bernstein cage can be summed independently:
// the first moment of each quadratic Bernstein basis is 2*t-1. Compare every
// vertex to that closed form, rather than invoking the constructor's pose.
func TestHumboldtSquidKeepsOriginalSkinPose(t *testing.T) {
	raw, err := humboldtSquidRest()
	if err != nil {
		t.Fatal(err)
	}
	clamp := func(x float64) float64 { return math.Max(0, math.Min(1, x)) }
	for _, phase := range []float64{0, .3, 1.2} {
		made, err := NewHumboldtSquid(phase)
		if err != nil {
			t.Fatal(err)
		}
		points := made[0].Shape.(*mesh.Mesh).Points()
		if len(points) != len(raw.Vertices) {
			t.Fatalf("skin point count %d, want %d", len(points), len(raw.Vertices))
		}
		for i, p := range raw.Vertices {
			x := clamp((p.X + 1.7) / 3.5)
			y := clamp((p.Y + .55) / 1.1)
			z := clamp((p.Z + .65) / 1.3)
			weights := [3]float64{(1 - x) * (1 - x), 2 * x * (1 - x), x * x}
			sineY, sineZ := 0.0, 0.0
			for n, w := range weights {
				offset := float64(n - 1)
				sineY += w * math.Sin(phase+.55*offset)
				sineZ += w * math.Sin(phase+.45*offset)
			}
			original := p.Add(maths.V(.015*(2*x-1), .11*(2*y-1)*sineY, .085*(2*z-1)*sineZ)).Scale(.5)
			if points[i].Sub(original).Len() > 1e-11 {
				t.Fatalf("phase %v, point %d: got %v, want %v", phase, i, points[i], original)
			}
		}
	}
}
