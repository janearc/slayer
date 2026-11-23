package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"math"
	"testing"
)

func TestHatzTwoSampledGeometryAndMotion(t *testing.T) {
	at := maths.NewSubspace(maths.Vec{}, maths.Identity, 1)
	minY, maxY := math.Inf(1), math.Inf(-1)
	maxAttach := 0.
	lo, hi := math.Inf(1), 0.
	reference, err := HatzFeedbackTwo(0, at)
	if err != nil {
		t.Fatal(err)
	}
	for frame := 0; frame < 13; frame++ {
		phase := 2 * math.Pi * float64(frame) / 12
		pieces, err := HatzFeedbackTwo(phase, at)
		if err != nil {
			t.Fatalf("phase %d: %v", frame, err)
		}
		for i, p := range pieces {
			if mesh, ok := p.Shape.(*shape.Mesh); ok {
				if !mesh.Closed() {
					t.Fatalf("phase %d part %d open", frame, i)
				}
				for _, v := range mesh.Vertices {
					if math.IsNaN(v.X) || math.IsNaN(v.Y) || math.IsNaN(v.Z) || math.IsInf(v.X, 0) || math.IsInf(v.Y, 0) || math.IsInf(v.Z, 0) {
						t.Fatalf("phase %d nonfinite", frame)
					}
				}
			}
		}
		for k, side := range []float64{-1, 1} {
			j := hatzTwoJoints(phase, side)
			for i, l := range []float64{.86, 1.04, 1.13, 1.48} {
				if math.Abs(j[i+1].Sub(j[i]).Len()-l) > 1e-9 {
					t.Fatalf("segment %d length drift", i)
				}
			}
			minY = math.Min(minY, j[4].Y)
			maxY = math.Max(maxY, j[4].Y)
			mesh := pieces[1+2*k].Shape.(*shape.Mesh)
			ref := reference[1+2*k].Shape.(*shape.Mesh)
			half := len(mesh.Vertices) / 2
			for i, v := range j {
				mid := mesh.Vertices[i].Add(mesh.Vertices[i+half]).Scale(.5)
				maxAttach = math.Max(maxAttach, mid.Sub(v).Len())
			}
			for _, f := range mesh.Faces {
				for e := 0; e < 3; e++ {
					a, b := f[e], f[(e+1)%3]
					d := ref.Vertices[a].Sub(ref.Vertices[b]).Len()
					if d > 1e-9 {
						ratio := mesh.Vertices[a].Sub(mesh.Vertices[b]).Len() / d
						lo = math.Min(lo, ratio)
						hi = math.Max(hi, ratio)
					}
				}
			}
		}
	}
	if maxY-minY < 2 {
		t.Fatalf("insufficient tip range %.3f", maxY-minY)
	}
	if maxAttach > 1e-9 {
		t.Fatalf("joint not on shell boundary midpoint: %.9f", maxAttach)
	}
	t.Logf("13 phases: tip y %.3f..%.3f m; joint boundary midpoint error %.9f m; edge ratio %.4f..%.4f", minY, maxY, maxAttach, lo, hi)
	if _, err := HatzFeedbackTwo(math.NaN(), at); err == nil {
		t.Fatal("accepted NaN")
	}
}
