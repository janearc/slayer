package molluscs

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func molluscTriangle() Spec {
	return Spec{
		Rest:  &shape.Mesh{Vertices: []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}, Faces: [][3]int{{0, 1, 2}}},
		Scale: 2, Pose: func(p maths.Vec) maths.Vec { return p.Scale(2).Add(maths.V(0, 0, 3)) },
		Skin: &thing.Material{Name: "test skin"},
	}
}

func TestMolluscBuildsAnIndependentPosedSoftBody(t *testing.T) {
	spec := molluscTriangle()
	made, err := NewMollusc(spec)
	if err != nil {
		t.Fatal(err)
	}
	body := made.Body()
	skin, ok := body.Shape.(*mesh.Mesh)
	if !ok {
		t.Fatalf("body shape %T", body.Shape)
	}
	points := skin.Points()
	if got, want := points[1], maths.V(2, 0, 3); got != want {
		t.Fatalf("posed point %v, want %v", got, want)
	}
	if got, want := made.Nearest(maths.V(.9, 0, 0)), maths.V(2, 0, 3); got != want {
		t.Fatalf("anchor %v, want %v", got, want)
	}
	spec.Rest.Vertices[1] = maths.V(100, 0, 0)
	if got := made.Nearest(maths.V(.9, 0, 0)); got != maths.V(2, 0, 3) {
		t.Fatalf("source mutation changed anchor: %v", got)
	}
	if body.Material != spec.Skin {
		t.Fatal("skin material was not used")
	}
}

func TestMolluscRejectsInvalidInputs(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Spec)
	}{
		{"missing rest", func(s *Spec) { s.Rest = nil }},
		{"missing pose", func(s *Spec) { s.Pose = nil }},
		{"missing skin", func(s *Spec) { s.Skin = nil }},
		{"zero scale", func(s *Spec) { s.Scale = 0 }},
		{"infinite scale", func(s *Spec) { s.Scale = math.Inf(1) }},
		{"bad face", func(s *Spec) { s.Rest.Faces[0][2] = 3 }},
		{"negative face", func(s *Spec) { s.Rest.Faces[0][2] = -1 }},
		{"nonfinite pose", func(s *Spec) { s.Pose = func(maths.Vec) maths.Vec { return maths.V(math.NaN(), 0, 0) } }},
		{"degenerate face", func(s *Spec) { s.Rest.Vertices[2] = s.Rest.Vertices[1] }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := molluscTriangle()
			tc.change(&spec)
			if _, err := NewMollusc(spec); err == nil {
				t.Fatal("accepted invalid mollusc")
			}
		})
	}
}
