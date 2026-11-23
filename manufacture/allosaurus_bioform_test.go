package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"math"
	"testing"
)

func TestAllosaurusPlantedFootWorldContact(t *testing.T) {
	for _, side := range []float64{-1, 1} {
		for _, start := range []float64{0, 2} {
			if side > 0 {
				start += 1
			}
			_, _, _, first := allosaurusLeg(start, side)
			anchor := first.X + start*(.56/1.2)
			for _, dt := range []float64{.2, .5, 1.0, 1.19} {
				_, _, _, foot := allosaurusLeg(start+dt, side)
				worldX := foot.X + (start+dt)*(.56/1.2)
				if math.Abs(worldX-anchor) > 1e-10 {
					t.Fatalf("side %g planted x %.9f != %.9f", side, worldX, anchor)
				}
				if math.Abs(foot.Y-.095) > 1e-10 {
					t.Fatalf("side %g stance foot raised: %g", side, foot.Y)
				}
			}
		}
	}
	_, _, _, swing := allosaurusLeg(1.6, -1)
	if swing.Y <= .095 {
		t.Fatal("swing foot did not lift")
	}
}
func TestAllosaurusRejectsNonfiniteTime(t *testing.T) {
	for _, bad := range []float64{math.NaN(), math.Inf(1)} {
		if _, err := AllosaurusStudy(bad, maths.Local()); err == nil {
			t.Fatalf("accepted %g", bad)
		}
	}
}

// The scalar volume and centroid below are for the body mesh only under a
// uniform-density assumption. They do not establish whole-animal balance.
func TestAllosaurusClosedBodyVolumeAndDeformation(t *testing.T) {
	var volume [3]float64
	for i, at := range []float64{0, .5, 1.5} {
		body, err := allosaurusBody(at)
		if err != nil {
			t.Fatal(err)
		}
		if !body.Closed() {
			t.Fatalf("body at %g is not closed", at)
		}
		points := body.Points()
		moment := maths.Vec{}
		signed := 0.0
		for _, f := range body.Faces() {
			a, b, c := points[f[0]], points[f[1]], points[f[2]]
			v := a.Dot(b.Cross(c))
			signed += v
			moment = moment.Add(a.Add(b).Add(c).Scale(v / 4))
		}
		if signed <= 0 {
			t.Fatalf("body at %g wound inside out", at)
		}
		volume[i] = signed / 6
		center := moment.Scale(1 / signed)
		t.Logf("t=%.1f body-only volume %.6f m³, centroid %v", at, volume[i], center)
	}
	for _, v := range volume[1:] {
		if math.Abs(v/volume[0]-1) > .08 {
			t.Fatalf("body-volume change above 8%%: %v", volume)
		}
	}
	a, b := allosaurusDeform(maths.V(-3, 1.6, 0), .5), allosaurusDeform(maths.V(-3, 1.6, 0), 1.5)
	if a.Z*b.Z >= 0 {
		t.Fatalf("tail poses did not reverse: %v %v", a, b)
	}
}

func TestAllosaurusSeparateMeshPiecesClosed(t *testing.T) {
	for _, at := range []float64{0, .5, 1.5} {
		pieces, err := AllosaurusStudy(at, maths.Local())
		if err != nil {
			t.Fatal(err)
		}
		for i, p := range pieces {
			if m, ok := p.Shape.(*mesh.Mesh); ok && !m.Closed() {
				t.Fatalf("piece %d at %g is not closed", i, at)
			}
		}
	}
}
