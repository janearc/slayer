package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"math"
	"testing"
)

func TestHatz2BSampledShellAndMotion(t *testing.T) {
	at := maths.NewSubspace(maths.Vec{}, maths.Identity, 1)
	ref, err := HatzFeedback2B(0, at)
	if err != nil {
		t.Fatal(err)
	}
	lo, hi := math.Inf(1), 0.
	tiplo, tiphi := math.Inf(1), math.Inf(-1)
	for frame := 0; frame < 13; frame++ {
		pieces, err := HatzFeedback2B(2*math.Pi*float64(frame)/12, at)
		if err != nil {
			t.Fatalf("frame %d: %v", frame, err)
		}
		for i, p := range pieces {
			if mesh, ok := p.Shape.(*shape.Mesh); ok {
				if !mesh.Closed() {
					t.Fatalf("frame %d shell %d open", frame, i)
				}
				for _, v := range mesh.Vertices {
					if math.IsNaN(v.X) || math.IsNaN(v.Y) || math.IsNaN(v.Z) || math.IsInf(v.X, 0) || math.IsInf(v.Y, 0) || math.IsInf(v.Z, 0) {
						t.Fatalf("frame %d: nonfinite", frame)
					}
				}
			}
		}
		for _, i := range []int{1, 3} {
			m := pieces[i].Shape.(*shape.Mesh)
			r := ref[i].Shape.(*shape.Mesh)
			if len(m.Vertices) != len(r.Vertices) {
				t.Fatal("changed topology")
			}
			for _, f := range m.Faces {
				for k := 0; k < 3; k++ {
					a, b := f[k], f[(k+1)%3]
					d := r.Vertices[a].Sub(r.Vertices[b]).Len()
					if d > 1e-8 {
						q := m.Vertices[a].Sub(m.Vertices[b]).Len() / d
						lo = math.Min(lo, q)
						hi = math.Max(hi, q)
					}
				}
			}
			for _, v := range m.Vertices {
				if math.Abs(v.Z) > 3.7 {
					tiplo = math.Min(tiplo, v.Y)
					tiphi = math.Max(tiphi, v.Y)
				}
			}
		}
	}
	t.Logf("13 phases: outer wing y %.3f..%.3f m; edge ratios %.4f..%.4f", tiplo, tiphi, lo, hi)
	if tiphi-tiplo < 2 {
		t.Fatal("insufficient visible wing motion")
	}
	if _, err := HatzFeedback2B(math.NaN(), at); err == nil {
		t.Fatal("accepted NaN")
	}
}
