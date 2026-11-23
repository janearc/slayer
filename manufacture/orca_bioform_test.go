package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"math"
	"testing"
)

func bioVolume(m *mesh.Mesh) float64 {
	p := m.Points()
	v := 0.0
	for _, f := range m.Faces() {
		v += p[f[0]].Dot(p[f[1]].Cross(p[f[2]]))
	}
	return v / 6
}
func TestOrcaBioformClosedSkinAndMotion(t *testing.T) {
	at := maths.NewSubspace(maths.Vec{}, maths.Identity, 1)
	var volumes [3]float64
	for i, stroke := range []float64{-1, 0, 1} {
		parts, err := OrcaBioform(stroke, at)
		if err != nil {
			t.Fatalf("pose %g: %v", stroke, err)
		}
		if len(parts) != 3 {
			t.Fatalf("got %d pieces, want body plus two separately closed flukes", len(parts))
		}
		for j, part := range parts {
			m, ok := part.Shape.(*mesh.Mesh)
			if !ok || !m.Closed() {
				t.Fatalf("pose %g part %d is not closed", stroke, j)
			}
			if bioVolume(m) <= 0 {
				t.Fatalf("pose %g part %d is inside out", stroke, j)
			}
		}
		body := parts[0].Shape.(*mesh.Mesh)
		volumes[i] = bioVolume(body)
	}
	for _, v := range []float64{volumes[0], volumes[2]} {
		if math.Abs(v/volumes[1]-1) > .08 {
			t.Fatalf("body volume changed by more than 8%%: %v", volumes)
		}
	}
	head, tail := maths.V(2, 0, 0), maths.V(-3, 0, 0)
	if orcaBioPose(head, 1) != orcaBioPose(head, -1) {
		t.Fatal("head moved in rear-body stroke")
	}
	if orcaBioPose(tail, 1).Y <= orcaBioPose(tail, -1).Y {
		t.Fatal("tail failed to reverse")
	}
	for _, bad := range []float64{math.NaN(), math.Inf(1), -1.01, 1.01} {
		if _, err := OrcaBioform(bad, at); err == nil {
			t.Fatalf("accepted pose %g", bad)
		}
	}
}
func TestOrcaBioformMarkingsAndEyes(t *testing.T) {
	coat := orcaBioCoat{}
	patch := coat.At(maths.V(1.85, .16, .64)).Luminance()
	back := coat.At(maths.V(1.0, .5, .5)).Luminance()
	if patch < 3*back {
		t.Fatalf("postocular patch %g is not lighter than back %g", patch, back)
	}
	left, right := OrcaBioformEye(-1), OrcaBioformEye(1)
	if left.X != right.X || left.Y != right.Y || left.Z != -right.Z {
		t.Fatalf("eyes not mirrored: %v %v", left, right)
	}
	if left.Z >= 0 || right.Z <= 0 {
		t.Fatal("eyes on wrong sides")
	}
}
