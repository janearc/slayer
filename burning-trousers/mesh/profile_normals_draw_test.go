//go:build draw

package mesh_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// The hit at (1/4, 1/4) is half of the first corner and a quarter of
// each other corner. Opposing the expected weights is a visible error.
func TestSmoothNormalWitnessAtKnownBarycentricPoint(t *testing.T) {
	points := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}
	normals := []maths.Vec{maths.XAxis, maths.YAxis, maths.ZAxis}
	part, err := mesh.New(points, [][3]int{{0, 1, 2}}, normals, true)
	if err != nil {
		t.Fatal(err)
	}
	met, ok := part.Meet(maths.Ray{At: maths.V(0.25, 0.25, 1), Dir: maths.ZAxis.Neg()}, 0, 2)
	if !ok {
		t.Fatal("known ray missed the triangle")
	}
	want := maths.V(0.5, 0.25, 0.25).Unit()
	if got := part.NormalAt(met); got.Sub(want).Len() > 1e-12 {
		t.Fatalf("normal %v, want %v", got, want)
	}
	wrong := maths.V(0.25, 0.5, 0.25).Unit()
	if wrong.Sub(want).Len() < 0.1 {
		t.Fatal("the witness does not distinguish swapped weights")
	}
}

// checkProfileSmoothNormals checks independently calculated sampled hits.
func checkProfileSmoothNormals(t *testing.T, things []thing.Thing) {
	t.Helper()
	seen := map[*mesh.Mesh]bool{}
	for _, each := range things {
		var parts []*mesh.Mesh
		switch held := each.Shape.(type) {
		case *mesh.Mesh:
			parts = append(parts, held)
		case *shape.MeshGroup:
			for _, member := range held.Members() {
				if part, ok := member.Mesh.(*mesh.Mesh); ok {
					parts = append(parts, part)
				}
			}
		}
		for _, part := range parts {
			if !seen[part] && part.Smooth() {
				checkSmoothNormals(t, part)
				seen[part] = true
			}
		}
	}
}

// checkSmoothNormals meets well-conditioned faces near their centers. The
// expected barycentric weights are calculated from the actual hit point,
// independently of the weights returned by the production intersector.
func checkSmoothNormals(t *testing.T, part *mesh.Mesh) {
	t.Helper()
	points, faces := part.Points(), part.Faces()
	box := part.Bounds()
	hair := 1e-6 * box.Max.Sub(box.Min).Len()
	step := max(1, len(faces)/64)
	checked := 0
	for i := 0; i < len(faces) && checked < 12; i += step {
		face := faces[i]
		a, b, c := points[face[0]], points[face[1]], points[face[2]]
		one, two := b.Sub(a), c.Sub(a)
		d00, d01, d11 := one.Dot(one), one.Dot(two), two.Dot(two)
		denom := d00*d11 - d01*d01
		if !(denom > 1e-8*d00*d11) {
			continue
		}
		normal := one.Cross(two).Unit()
		center := a.Add(b).Add(c).Scale(1.0 / 3)
		ray := maths.Ray{At: center.Add(normal.Scale(hair)), Dir: normal.Neg()}
		met, ok := part.Meet(ray, 0, 1.001*hair)
		if !ok || met.Face() != i {
			continue
		}
		p := ray.Point(met.Along()).Sub(a)
		d20, d21 := p.Dot(one), p.Dot(two)
		second := (d11*d20 - d01*d21) / denom
		third := (d00*d21 - d01*d20) / denom
		normals, _ := part.CornerNormals(i)
		want := normals[0].Scale(1 - second - third).Add(normals[1].Scale(second)).Add(normals[2].Scale(third))
		if want.Len() == 0 {
			continue
		}
		want = want.Unit()
		got := part.NormalAt(met)
		if delta := got.Sub(want).Len(); math.IsNaN(delta) || delta > 1e-7 {
			t.Errorf("face %d: normal %v, independently interpolated %v", i, got, want)
		}
		checked++
	}
	if checked == 0 {
		t.Error("no smooth face could be sampled independently")
	}
}
