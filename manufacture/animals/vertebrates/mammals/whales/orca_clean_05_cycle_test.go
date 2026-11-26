package whales

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

func TestOrcaClean05TwelvePhaseStrainSurvey(t *testing.T) {
	at := maths.NewSubspace(maths.Vec{}, maths.Identity, 1)
	first, err := orcaSwimming(0, at)
	if err != nil {
		t.Fatal(err)
	}
	base := first[0].Shape.(*mesh.Mesh)
	p0 := base.Points()
	faces := base.Faces()
	minRatio, maxRatio := math.Inf(1), 0.
	minVolume, maxVolume := math.Inf(1), math.Inf(-1)
	for frame := 0; frame < 12; frame++ {
		candidate, err := orcaSwimming(2*math.Pi*float64(frame)/12, at)
		if err != nil {
			t.Fatal(err)
		}
		skin := candidate[0].Shape.(*mesh.Mesh)
		pts := skin.Points()
		volume := 0.
		if !skin.Closed() || len(pts) != len(p0) {
			t.Fatalf("frame %d altered topology", frame)
		}
		for _, f := range faces {
			volume += pts[f[0]].Dot(pts[f[1]].Cross(pts[f[2]])) / 6
			for j := 0; j < 3; j++ {
				a, b := f[j], f[(j+1)%3]
				r := pts[a].Sub(pts[b]).Len() / p0[a].Sub(p0[b]).Len()
				minRatio = math.Min(minRatio, r)
				maxRatio = math.Max(maxRatio, r)
			}
		}
		minVolume = math.Min(minVolume, volume)
		maxVolume = math.Max(maxVolume, volume)
	}
	t.Logf("12 sampled phases: edge-length ratio %.5f–%.5f; signed-volume range %.6f–%.6f m³ (range %.3f%% of maximum)", minRatio, maxRatio, minVolume, maxVolume, 100*(maxVolume-minVolume)/maxVolume)
}
