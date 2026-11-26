package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"math"
	"testing"
)

func TestHatzDensitySampledSkinAndMotion(t *testing.T) {
	at := maths.NewSubspace(maths.Vec{}, maths.Identity, 1)
	parts, err := hatzDensityStudy(0, false, at)
	if err != nil {
		t.Fatal(err)
	}
	ref := parts[0].Shape.(*mesh.Mesh)
	r := ref.Points()
	faces := ref.Faces()
	graph := make([][]int, len(r))
	for _, f := range faces {
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
	if count != len(r) {
		t.Fatalf("connected vertices %d/%d", count, len(r))
	}
	lo, hi := math.Inf(1), 0.
	tiplo, tiphi := math.Inf(1), math.Inf(-1)
	vmin, vmax := math.Inf(1), 0.
	var first []maths.Vec
	for frame := 0; frame < 13; frame++ {
		p, err := HatzDensityStudy(2*math.Pi*float64(frame)/12, at)
		if err != nil {
			t.Fatalf("phase %d: %v", frame, err)
		}
		m := p[0].Shape.(*mesh.Mesh)
		points := m.Points()
		if !m.Closed() || len(points) != len(r) {
			t.Fatalf("phase %d: open or changed topology", frame)
		}
		if frame == 0 {
			first = points
		}
		if frame == 12 {
			for i := range points {
				if points[i].Sub(first[i]).Len() > 1e-8 {
					t.Fatal("loop discontinuity")
				}
			}
		}
		volume := 0.
		for _, f := range faces {
			volume += points[f[0]].Dot(points[f[1]].Cross(points[f[2]])) / 6
			for k := 0; k < 3; k++ {
				a, b := f[k], f[(k+1)%3]
				d := r[a].Sub(r[b]).Len()
				if d > 1e-8 {
					x := points[a].Sub(points[b]).Len() / d
					lo = math.Min(lo, x)
					hi = math.Max(hi, x)
				}
			}
		}
		vmin = math.Min(vmin, math.Abs(volume))
		vmax = math.Max(vmax, math.Abs(volume))
		for _, v := range points {
			if math.IsNaN(v.X) || math.IsNaN(v.Y) || math.IsNaN(v.Z) || math.IsInf(v.X, 0) || math.IsInf(v.Y, 0) || math.IsInf(v.Z, 0) {
				t.Fatal("nonfinite vertex")
			}
			if math.Abs(v.Z) > 3.7 {
				tiplo = math.Min(tiplo, v.Y)
				tiphi = math.Max(tiphi, v.Y)
			}
		}
	}
	t.Logf("13 phases: outer wing y %.3f..%.3f m; edge ratios %.4f..%.4f; abs volume %.4f..%.4f m3; vertices %d", tiplo, tiphi, lo, hi, vmin, vmax, count)
	if tiphi-tiplo < .5 {
		t.Fatal("insufficient outer-wing motion")
	}
	if _, err := HatzDensityStudy(math.NaN(), at); err == nil {
		t.Fatal("accepted NaN")
	}
}
