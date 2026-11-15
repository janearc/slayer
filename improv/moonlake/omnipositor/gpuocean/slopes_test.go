//go:build draw

package gpuocean

import (
	"math"
	"testing"

	"github.com/janearc/slayer/oceantex"
)

func TestSlopesOfFlatFields(t *testing.T) {
	layers := make([]WaveLayer, 3)
	for index, extent := range []float64{4000, 2000, 1000} {
		zero := func() []float64 { return make([]float64, 4) }
		layers[index] = WaveLayer{Field: &oceantex.Field{N: 2, Extent: extent, SlopeX: zero(), SlopeY: zero(), PushX: zero(), PushY: zero(), StretchXX: zero(), StretchXY: zero(), StretchYY: zero()}, ResolvedAlong: 0.1, ResolvedAcross: 0.2}
	}
	ray := MarchRay{Ray: oceantex.Ray{At: oceantex.Vec{Z: 10}, Dir: oceantex.Vec{Z: -1}}, PixelAngle: 0.01}
	got, err := Slopes(layers, []MarchRay{ray}, []MarchResult{{Distance: 10}})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].X != 0 || got[0].Y != 0 || math.Abs(float64(got[0].ResolvedAlong)-0.3) > 1e-5 || math.Abs(float64(got[0].ResolvedAcross)-0.6) > 1e-5 {
		t.Fatalf("slopes: %+v", got[0])
	}
}

func TestSlopesRejectIncompleteFields(t *testing.T) {
	if _, err := Slopes([]WaveLayer{{}, {}, {}}, []MarchRay{{}}, []MarchResult{{}}); err == nil {
		t.Fatal("incomplete fields accepted")
	}
}
