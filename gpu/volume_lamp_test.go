//go:build darwin

package gpu

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/thing"
)

const volumeLampProbe = `
kernel void volume_lamp_probe(device const Lamp *lamps [[buffer(0)]],
 device const float4 *points [[buffer(1)]],
 device float *out [[buffer(2)]],
 constant uint2 &origin [[buffer(3)]],
 uint2 at [[thread_position_in_grid]]) {
 uint i = origin.x + at.x;
 Reached reached = lamp_light(lamps[0], points[i].xyz);
 out[7*i] = reached.toward.x;
 out[7*i+1] = reached.toward.y;
 out[7*i+2] = reached.toward.z;
 out[7*i+3] = reached.far;
 out[7*i+4] = reached.light.x;
 out[7*i+5] = reached.light.y;
 out[7*i+6] = reached.light.z;
}
`

func TestFiniteVolumeLampOnGPUAgreesWithCPU(t *testing.T) {
	lamp, err := emit.NewVolume(emit.Lit{At: maths.V(1, 2, 3), Strength: maths.RGB(1, 2, 3)}, measure.Metres(2))
	if err != nil {
		t.Fatal(err)
	}
	encoded := newScene()
	encoded.lampsOf(&thing.World{Lamps: []thing.Lamp{lamp}})
	kernel, err := metal.Open(source()+volumeLampProbe, "volume_lamp_probe")
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	var points packer
	distances := []float64{0, .01, .1, .5, 1, 1.99999, 2, 2.00001, 4, 8, 40}
	for _, distance := range distances {
		points.row(structure{"Point", []field{floatField("at", "")}},
			values{"at": point(lamp.Point().At.Sub(maths.YAxis.Scale(distance)))})
	}
	got, err := kernel.Grid(len(distances), 1, 1, 7, [][]byte{encoded.lamps.bytes, points.bytes})
	if err != nil {
		t.Fatal(err)
	}
	for i, distance := range distances {
		way, far, light := lamp.Light(lamp.Point().At.Sub(maths.YAxis.Scale(distance)), maths.YAxis)
		want := []float64{way.X, way.Y, way.Z, far, light.R, light.G, light.B}
		for c, value := range want {
			actual := float64(got[7*i+c])
			if math.IsNaN(actual) || math.IsInf(actual, 0) || math.Abs(actual-value) > 1e-5*max(1, math.Abs(value)) {
				t.Fatalf("distance %g channel %d: GPU %g CPU %g", distance, c, actual, value)
			}
		}
	}
}
