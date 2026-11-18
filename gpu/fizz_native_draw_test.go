//go:build draw && darwin

package gpu

import (
	"encoding/binary"
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/medium"
)

const probeFizzNative = `
kernel void probe_fizz_native(
 device const packed_float3 *points [[buffer(0)]],
 device float *out [[buffer(1)]],
 device const float *card [[buffer(2)]],
 uint id [[thread_position_in_grid]]) {
 out[id] = fizz_at(card, float3(points[id]));
}
`

func TestFizzNativeMetalDensityMatchesExistingConstructor(t *testing.T) {
	kernel, err := metal.Open(
		sourceWithOpticalState(0, probeVolumes)+probeFizzNative,
		"probe_fizz_native")
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	for _, churn := range []float64{0, .8} {
		for _, seconds := range []float64{0, .7} {
			f := residualFizz()
			f.Thick = .8
			f.Churn = churn
			f = f.Tick(time.Duration(seconds * float64(time.Second))).(medium.Fizz)
			words, err := fizzCardOf(f)
			if err != nil {
				t.Fatal(err)
			}
			raw := make([]byte, len(words)*4)
			for i, w := range words {
				binary.LittleEndian.PutUint32(raw[4*i:], w)
			}
			var points [][3]float32
			for k := 0; k < 4000; k++ {
				a := float64(k) * .61803398875 * 2 * math.Pi
				r := 1.4 * float64(k%71) / 70
				y := 22*float64(k%97)/96 - 1
				points = append(points, [3]float32{float32(r * math.Cos(a)), float32(y), float32(r * math.Sin(a))})
			}
			got, err := kernel.Points(points, raw)
			if err != nil {
				t.Fatal(err)
			}
			worst, lit, zero := 0.0, 0, 0
			for i, p := range points {
				want := f.Density(maths.V(float64(p[0]), float64(p[1]), float64(p[2])))
				worst = math.Max(worst, math.Abs(float64(got[i])-want))
				if want > .001 {
					lit++
				}
				if want == 0 {
					zero++
				}
			}
			t.Logf("churn%g time%g maxdensityerror%g lit%d zero%d", churn, seconds, worst, lit, zero)
			if worst > 3e-5 || lit < 500 || zero < 500 {
				t.Fatalf("density mismatch/vacuous: %g lit%d zero%d", worst, lit, zero)
			}
		}
	}
}
