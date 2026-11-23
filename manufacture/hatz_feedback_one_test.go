package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"math"
	"testing"
)

func TestHatzOneSampledGeometryAndMotion(t *testing.T) {
	at := maths.NewSubspace(maths.Vec{}, maths.Identity, 1)
	lo, hi := math.Inf(1), math.Inf(-1)
	worst := 0.
	for frame := 0; frame < 13; frame++ {
		phase := 2 * math.Pi * float64(frame) / 12
		pieces, err := HatzFeedbackOne(phase, at)
		if err != nil {
			t.Fatalf("frame %d: %v", frame, err)
		}
		for k, p := range pieces {
			if mesh, ok := p.Shape.(*shape.Mesh); ok {
				if !mesh.Closed() {
					t.Fatalf("frame %d piece %d open", frame, k)
				}
				for _, v := range mesh.Vertices {
					if math.IsNaN(v.X) || math.IsNaN(v.Y) || math.IsNaN(v.Z) || math.IsInf(v.X, 0) || math.IsInf(v.Y, 0) || math.IsInf(v.Z, 0) {
						t.Fatalf("frame %d: nonfinite vertex", frame)
					}
				}
			}
		}
		for _, s := range []float64{-1, 1} {
			j := hatzOneWingJoints(phase, s)
			for i, l := range []float64{.85, 1.05, 1.15, 1.45} {
				if math.Abs(j[i+1].Sub(j[i]).Len()-l) > 1e-9 {
					t.Fatalf("joint %d length changed", i)
				}
			}
			lo = math.Min(lo, j[4].Y)
			hi = math.Max(hi, j[4].Y)
			m, err := hatzOneMembrane(j, s)
			if err != nil {
				t.Fatal(err)
			}
			for _, v := range j {
				min := math.Inf(1)
				for _, p := range m.Vertices {
					min = math.Min(min, v.Sub(p).Len())
				}
				worst = math.Max(worst, min)
			}
		}
	}
	if hi-lo < 2 {
		t.Fatalf("tip displacement %.3f m is too small for visible proof", hi-lo)
	}
	if worst > .18 {
		t.Fatalf("maximum nearest sampled membrane vertex to wing joint %.3f m", worst)
	}
	t.Logf("13 phases: wing tip vertical range %.3f..%.3f m; nearest membrane-to-joint vertex maximum %.4f m", lo, hi, worst)
	if _, err := HatzFeedbackOne(math.NaN(), at); err == nil {
		t.Fatal("accepted nonfinite phase")
	}
}
