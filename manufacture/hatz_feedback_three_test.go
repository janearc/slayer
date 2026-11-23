package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"math"
	"testing"
)

func TestHatzThreeSampledSkinAndMotion(t *testing.T) {
	at := maths.NewSubspace(maths.Vec{}, maths.Identity, 1)
	parts, err := HatzFeedbackThree(0, at)
	if err != nil {
		t.Fatal(err)
	}
	ref := parts[0].Shape.(*shape.Mesh)
	graph := make([][]int, len(ref.Vertices))
	for _, f := range ref.Faces {
		for i := 0; i < 3; i++ {
			a, b := f[i], f[(i+1)%3]
			graph[a] = append(graph[a], b)
			graph[b] = append(graph[b], a)
		}
	}
	seen := make([]bool, len(graph))
	seen[0] = true
	q := []int{0}
	count := 0
	for len(q) > 0 {
		v := q[len(q)-1]
		q = q[:len(q)-1]
		count++
		for _, n := range graph[v] {
			if !seen[n] {
				seen[n] = true
				q = append(q, n)
			}
		}
	}
	if count != len(graph) {
		t.Fatalf("connected vertices %d/%d", count, len(graph))
	}
	lo, hi := math.Inf(1), 0.
	tiplo, tiphi := math.Inf(1), math.Inf(-1)
	for frame := 0; frame < 13; frame++ {
		p, err := HatzFeedbackThree(2*math.Pi*float64(frame)/12, at)
		if err != nil {
			t.Fatalf("phase %d: %v", frame, err)
		}
		m := p[0].Shape.(*shape.Mesh)
		if !m.Closed() || len(m.Vertices) != len(ref.Vertices) {
			t.Fatalf("phase %d: open or changed topology", frame)
		}
		for _, f := range m.Faces {
			for k := 0; k < 3; k++ {
				a, b := f[k], f[(k+1)%3]
				d := ref.Vertices[a].Sub(ref.Vertices[b]).Len()
				if d > 1e-8 {
					r := m.Vertices[a].Sub(m.Vertices[b]).Len() / d
					lo = math.Min(lo, r)
					hi = math.Max(hi, r)
				}
			}
		}
		for _, v := range m.Vertices {
			if math.IsNaN(v.X) || math.IsNaN(v.Y) || math.IsNaN(v.Z) || math.IsInf(v.X, 0) || math.IsInf(v.Y, 0) || math.IsInf(v.Z, 0) {
				t.Fatal("nonfinite vertex")
			}
			if math.Abs(v.Z) > 3.7 {
				tiplo = math.Min(tiplo, v.Y)
				tiphi = math.Max(tiphi, v.Y)
			}
		}
	}
	t.Logf("13 phases: wing y %.3f..%.3f m; edge ratios %.4f..%.4f; connected vertices %d", tiplo, tiphi, lo, hi, count)
	if tiphi-tiplo < 2 {
		t.Fatal("insufficient outer-wing motion")
	}
	if _, err := HatzFeedbackThree(math.NaN(), at); err == nil {
		t.Fatal("accepted NaN")
	}
}
