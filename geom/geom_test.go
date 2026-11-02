package geom

import (
	"math"
	"testing"
)

func TestVec3Add(t *testing.T) {
	a := Vec3{1, 2, 3}
	b := Vec3{4, 5, 6}
	got := a.Add(b)
	want := Vec3{5, 7, 9}
	if got != want {
		t.Errorf("Add(%v, %v) = %v, want %v", a, b, got, want)
	}
}

func TestVec3Sub(t *testing.T) {
	a := Vec3{5, 7, 9}
	b := Vec3{1, 2, 3}
	got := a.Sub(b)
	want := Vec3{4, 5, 6}
	if got != want {
		t.Errorf("Sub(%v, %v) = %v, want %v", a, b, got, want)
	}
}

func TestVec3Scale(t *testing.T) {
	v := Vec3{1, 2, 3}
	got := v.Scale(2)
	want := Vec3{2, 4, 6}
	if got != want {
		t.Errorf("Scale(%v, 2) = %v, want %v", v, got, want)
	}
}

func TestVec3Dot(t *testing.T) {
	a := Vec3{1, 2, 3}
	b := Vec3{4, 5, 6}
	got := a.Dot(b)
	want := 1.0*4 + 2.0*5 + 3.0*6
	if got != want {
		t.Errorf("Dot(%v, %v) = %v, want %v", a, b, got, want)
	}
}

func TestVec3Cross(t *testing.T) {
	a := Vec3{1, 0, 0}
	b := Vec3{0, 1, 0}
	got := a.Cross(b)
	want := Vec3{0, 0, 1}
	if got != want {
		t.Errorf("Cross(%v, %v) = %v, want %v", a, b, got, want)
	}
}

func TestVec3Length(t *testing.T) {
	v := Vec3{3, 4, 0}
	got := v.Length()
	want := 5.0
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("Length(%v) = %v, want %v", v, got, want)
	}
}

func TestVec3Normalize(t *testing.T) {
	v := Vec3{3, 4, 0}
	got := v.Normalize()
	len := got.Length()
	if math.Abs(len-1.0) > 1e-9 {
		t.Errorf("Normalize(%v) has length %v, want 1", v, len)
	}
	wantX, wantY := 0.6, 0.8
	if math.Abs(got[0]-wantX) > 1e-9 || math.Abs(got[1]-wantY) > 1e-9 {
		t.Errorf("Normalize(%v) = %v, want ~[0.6 0.8 0]", v, got)
	}
}

func TestNormal(t *testing.T) {
	v0 := Vec3{0, 0, 0}
	v1 := Vec3{1, 0, 0}
	v2 := Vec3{0, 1, 0}
	got := Normal(v0, v1, v2)
	want := Vec3{0, 0, 1}
	if math.Abs(got[0]-want[0]) > 1e-9 ||
		math.Abs(got[1]-want[1]) > 1e-9 ||
		math.Abs(got[2]-want[2]) > 1e-9 {
		t.Errorf("Normal(%v, %v, %v) = %v, want %v", v0, v1, v2, got, want)
	}
}

func TestBoundsOfTriangles(t *testing.T) {
	triangles := []Triangle{
		{V: [3]Vec3{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}}},
		{V: [3]Vec3{{2, 2, 2}, {3, 2, 2}, {2, 3, 2}}},
	}
	got := BoundsOfTriangles(triangles)
	if got.Min != (Vec3{0, 0, 0}) || got.Max != (Vec3{3, 3, 2}) {
		t.Errorf("BoundsOfTriangles() = [%v, %v], want [[0 0 0], [3 3 2]]",
			got.Min, got.Max)
	}
}

func TestBoundsSize(t *testing.T) {
	b := Box{Min: Vec3{0, 0, 0}, Max: Vec3{2, 3, 4}}
	got := b.Size()
	want := Vec3{2, 3, 4}
	if got != want {
		t.Errorf("Box.Size() = %v, want %v", got, want)
	}
}

func TestEmptyBox(t *testing.T) {
	b := EmptyBox()
	p := Vec3{1, 2, 3}
	b = b.Fit(p)
	if b.Min != p || b.Max != p {
		t.Errorf("EmptyBox.Fit() = [%v, %v], want [[1 2 3], [1 2 3]]",
			b.Min, b.Max)
	}
}
