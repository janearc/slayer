//go:build draw && darwin

package gpu

import (
	"encoding/binary"
	"errors"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"

	"math"
	"testing"
	"time"
)

const probePropeller = `
kernel void probe_propeller(device const packed_float3 *points [[buffer(0)]], device float *out [[buffer(1)]], device const float *card [[buffer(2)]], uint id [[thread_position_in_grid]]){out[id]=propeller_at(card,float3(points[id]));}
`

func TestPropellerMetalDensityMatchesCPU(t *testing.T) {
	kernel, err := metal.Open(source()+probePropeller, "probe_propeller")
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no GPU")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	for _, blades := range []int{2, 5, 16} {
		for _, seconds := range []float64{0, .37} {
			field := propellerFixture(t, blades).Tick(time.Duration(seconds * float64(time.Second))).(medium.PropellerCavitation)
			words, err := propellerCardOf(field)
			if err != nil {
				t.Fatal(err)
			}
			bytes := make([]byte, len(words)*4)
			for i, w := range words {
				binary.LittleEndian.PutUint32(bytes[i*4:], w)
			}
			points := make([][3]float32, 0, 12000)
			for z := 0; z < 60; z++ {
				for a := 0; a < 100; a++ {
					theta := 2 * math.Pi * float64(a) / 100
					s := float64(z) * .1
					for _, r := range []float64{.92, 1.07} {
						points = append(points, [3]float32{float32(r * math.Cos(theta)), float32(r * math.Sin(theta)), float32(s)})
					}
				}
			}
			got, err := kernel.Points(points, bytes)
			if err != nil {
				t.Fatal(err)
			}
			worst, lit, zeros := 0.0, 0, 0
			for i, p := range points {
				want := field.Density(maths.V(float64(p[0]), float64(p[1]), float64(p[2])))
				difference := math.Abs(float64(got[i]) - want)
				worst = math.Max(worst, difference)
				if want > .01 {
					lit++
				}
				if want == 0 {
					zeros++
				}
			}
			t.Logf("blades%d time%g worst absolute density%g lit%d zero%d", blades, seconds, worst, lit, zeros)
			if worst > 2e-4 || lit < 100 || zeros < 100 {
				t.Fatalf("density mismatch/vacuous probe: %g lit%d zero%d", worst, lit, zeros)
			}
		}
	}
}

func TestPropellerMetalVolumeImageMatchesCPU(t *testing.T) {
	for _, blades := range []int{2, 5} {
		world := thing.NewWorld()
		world.AddVolume(thing.Volume{Medium: propellerFixture(t, blades), Material: &thing.Material{Glow: maths.RGB(.5, .7, 1)}})
		origin := maths.V(4, 3, 9)
		eye := maths.Subspace{Origin: origin, Orientation: maths.LookAt(origin, maths.V(0, 0, 2.5))}
		againstAt(t, world, eye, 97, 65, 1)
	}
}
