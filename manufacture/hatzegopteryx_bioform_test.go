package manufacture

import (
	"github.com/janearc/slayer/maths"
	"math"
	"testing"
)

func TestHatzWingSegmentLengths(t *testing.T) {
	for _, side := range []float64{-1, 1} {
		for i := 0; i <= 10; i++ {
			f := float64(i) / 10
			s, e, w, tip, _ := hatzWingControls(f, 0, side)
			for j, p := range []struct {
				a, b maths.Vec
				want float64
			}{{s, e, 1.35}, {e, w, 1.45}, {w, tip, 2.85}} {
				if got := p.a.Sub(p.b).Len(); math.Abs(got-p.want) > 1e-9 {
					t.Fatalf("side %.0f fold %.1f segment %d length %.9f want %.2f", side, f, j, got, p.want)
				}
			}
		}
	}
}
func TestHatzGroundContactAndInvalidPose(t *testing.T) {
	for _, side := range []float64{-1, 1} {
		_, _, w, _, _ := hatzWingControls(1, 0, side)
		if math.Abs(w.Y-.10) > 1e-9 {
			t.Fatalf("wrist y %g", w.Y)
		}
	}
	for _, bad := range []float64{math.NaN(), math.Inf(1), -.01, 1.01} {
		if _, err := HatzegopteryxStudy(bad, 0, maths.Local()); err == nil {
			t.Fatalf("accepted %g", bad)
		}
	}
}
func TestHatzWingMembraneClosedOverFold(t *testing.T) {
	for _, side := range []float64{-1, 1} {
		for i := 0; i <= 10; i++ {
			f := float64(i) / 10
			s, e, w, tip, inner := hatzWingControls(f, 0, side)
			m, err := hatzWingSurface(s, e, w, tip, inner, f, side)
			if err != nil {
				t.Fatalf("side %g fold %g: %v", side, f, err)
			}
			if !m.Closed() {
				t.Fatalf("side %g fold %g not closed", side, f)
			}
		}
	}
}
func TestHatzMembraneAreaDiagnostic(t *testing.T) {
	for _, fold := range []float64{0, .5, 1} {
		s, e, w, tip, inner := hatzWingControls(fold, 0, 1)
		m, err := hatzWingSurface(s, e, w, tip, inner, fold, 1)
		if err != nil {
			t.Fatal(err)
		}
		p := m.Points()
		area := 0.0
		for i := 0; i < hatzSpan; i++ {
			for j := 0; j < hatzChord; j++ {
				a := p[i*(hatzChord+1)+j]
				b := p[(i+1)*(hatzChord+1)+j]
				c := p[(i+1)*(hatzChord+1)+j+1]
				d := p[i*(hatzChord+1)+j+1]
				area += .5*b.Sub(a).Cross(c.Sub(a)).Len() + .5*c.Sub(a).Cross(d.Sub(a)).Len()
			}
		}
		t.Logf("fold %.1f single-side tessellated area %.6f m²", fold, area)
	}
}

// Diagnostic only: tests non-coplanar proper edge/triangle crossings on one
// membrane surface, skipping faces sharing vertices. It misses coplanar and
// edge-touching contacts and does not test against other animal pieces.
func TestHatzMembraneCrossingDiagnostic(t *testing.T) {
	segmentCross := func(p, q, a, b, c maths.Vec) bool {
		const eps = 1e-8
		dir := q.Sub(p)
		e1, e2 := b.Sub(a), c.Sub(a)
		h := dir.Cross(e2)
		det := e1.Dot(h)
		if math.Abs(det) < eps {
			return false
		}
		inv := 1 / det
		s := p.Sub(a)
		u := inv * s.Dot(h)
		if u <= eps || u >= 1-eps {
			return false
		}
		v := inv * dir.Dot(s.Cross(e1))
		if v <= eps || u+v >= 1-eps {
			return false
		}
		dist := inv * e2.Dot(s.Cross(e1))
		return dist > eps && dist < 1-eps
	}
	for _, fold := range []float64{0, .5, 1} {
		s, e, w, tip, inner := hatzWingControls(fold, 0, 1)
		m, err := hatzWingSurface(s, e, w, tip, inner, fold, 1)
		if err != nil {
			t.Fatal(err)
		}
		p := m.Points()
		all := m.Faces()
		var faces [][3]int
		for i := 0; i < hatzSpan*hatzChord*4; i++ {
			if i%4 < 2 {
				faces = append(faces, all[i])
			}
		}
		hits := 0
		for i, a := range faces {
			for _, b := range faces[i+1:] {
				shared := false
				for _, x := range a {
					for _, y := range b {
						if x == y {
							shared = true
						}
					}
				}
				if shared {
					continue
				}
				hit := false
				for j := 0; j < 3 && !hit; j++ {
					hit = segmentCross(p[a[j]], p[a[(j+1)%3]], p[b[0]], p[b[1]], p[b[2]]) || segmentCross(p[b[j]], p[b[(j+1)%3]], p[a[0]], p[a[1]], p[a[2]])
				}
				if hit {
					hits++
				}
			}
		}
		t.Logf("fold %.1f: %d non-adjacent proper triangle crossings detected", fold, hits)
	}
}
