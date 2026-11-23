package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"math"
	"testing"
)

func TestPliosaurBaseline(t *testing.T) {
	place := maths.NewSubspace(maths.Vec{}, maths.Identity, 1)
	a, e := PliosaurStudy(0, place)
	if e != nil {
		t.Fatal(e)
	}
	b, e := PliosaurStudy(math.Pi/2, place)
	if e != nil {
		t.Fatal(e)
	}
	c, e := PliosaurStudy(2*math.Pi, place)
	if e != nil {
		t.Fatal(e)
	}
	if len(a) != 55 {
		t.Fatalf("got %d pieces", len(a))
	}
	maxmove, loop := 0., 0.
	for j := 0; j < 5; j++ {
		m, ok := a[j].Shape.(*mesh.Mesh)
		if !ok || !m.Closed() {
			t.Fatalf("piece %d not closed mesh", j)
		}
		n := b[j].Shape.(*mesh.Mesh)
		z := c[j].Shape.(*mesh.Mesh)
		pa, pb, pc := m.Points(), n.Points(), z.Points()
		for i := range pa {
			maxmove = math.Max(maxmove, pb[i].Sub(pa[i]).Len())
			loop = math.Max(loop, pc[i].Sub(pa[i]).Len())
		}
	}
	if maxmove < .05 || loop > 1e-8 {
		t.Fatalf("movement %g loop %g", maxmove, loop)
	}
	t.Logf("five separate closed skin components; max stroke displacement %.3f m; loop error %.3g m", maxmove, loop)
}

func TestPliosaurBaselineHeadAndVolume(t *testing.T) {
	parts, e := PliosaurStudy(0, maths.NewSubspace(maths.Vec{}, maths.Identity, 1))
	if e != nil {
		t.Fatal(e)
	}
	m := parts[0].Shape.(*mesh.Mesh)
	points := m.Points()
	xmin, xmax, zmin, zmax := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, p := range points {
		if p.X >= 2 {
			xmin = math.Min(xmin, p.X)
			xmax = math.Max(xmax, p.X)
			zmin = math.Min(zmin, p.Z)
			zmax = math.Max(zmax, p.Z)
		}
	}
	vol := 0.
	for _, f := range m.Faces() {
		a, b, c := points[f[0]], points[f[1]], points[f[2]]
		vol += a.Dot(b.Cross(c)) / 6
	}
	t.Logf("head-region surface bbox %.3f x %.3f m (x>=2; not fossil landmarks), trunk volume %.3f m3", xmax-xmin, zmax-zmin, math.Abs(vol))
}
