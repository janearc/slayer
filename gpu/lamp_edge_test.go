//go:build draw && darwin

package gpu

import (
	"math"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/thing"
)

// lampEdgeKernel asks the kernel's own lamp_light for the red light of the
// first lamp at each point.
const lampEdgeKernel = `
kernel void lamp_edge_probe(device const float *points [[buffer(0)]],
	device float *out [[buffer(1)]],
	device const Lamp *lamps [[buffer(2)]],
	uint i [[thread_position_in_grid]]) {
	Reached got = lamp_light(lamps[0], float3(points[3 * i],
		points[3 * i + 1], points[3 * i + 2]));
	out[i] = got.light.x;
}
`

// A point lamp's light has a soft edge: within a float32's rounding of
// Beyond, the Metal kernel and the CPU give the light alike to within what
// the edge's slope times that rounding allows, a few thousandths of the
// whole where a step gave all of it or none; and well past Beyond, the
// light is what it was, the inverse square, to a float32's precision.
func TestAPointLampsEdgeIsSoftAndTheSameOnBothSides(t *testing.T) {
	const beyond = 1.02
	lit := emit.Lit{Beyond: beyond, Strength: maths.RGB(9, 8, 7)}
	world := thing.NewWorld()
	world.Lamp(lit)
	encoded := &scene{}
	encoded.lampsOf(world)
	kernel, err := metal.Open(sourceWith(1, probeVolumes)+lampEdgeKernel,
		"lamp_edge_probe")
	if err != nil {
		t.Skipf("no Metal device to ask: %v", err)
	}
	defer kernel.Close()
	distances := []float64{beyond * (1 - 2*emit.EdgeShare),
		beyond * (1 - emit.EdgeShare), beyond - 1e-5, beyond - 1e-6, beyond,
		beyond + 1e-6, beyond + 1e-5, beyond * 1.001, 2, 10}
	var points [][3]float32
	for _, each := range distances {
		points = append(points, [3]float32{float32(each), 0, 0})
	}
	got, err := kernel.Points(points, encoded.lamps.bytes)
	if err != nil {
		t.Fatal(err)
	}
	for index := range distances {
		at := maths.V(float64(points[index][0]), 0, 0)
		_, _, light := lit.Light(at, maths.Vec{})
		whole := lit.Strength.R / (at.X * at.X)
		// the edge's slope, a smoothstep's 3/2 over its thickness, times a
		// float32's rounding of the distance, as a share of the whole
		bound := 1.5 / (beyond * emit.EdgeShare) * 2.5e-7 * at.X
		if apart := math.Abs(float64(got[index])-float64(light.R)) / whole; apart > math.Max(bound, 1e-5) {
			t.Errorf("distance %.7f: the kernel's light %v, the CPU's %v, "+
				"%.3g of the whole apart; the edge allows %.3g", at.X,
				got[index], light.R, apart, bound)
		}
	}
}

// Anti: beyond its edge the light is the inverse square, as it was when
// the edge was a step: at Beyond and at a thousandth past it, whole.
func TestAPointLampsLightPastItsEdgeIsUnchanged(t *testing.T) {
	const beyond = 1.02
	lit := emit.Lit{Beyond: beyond, Strength: maths.RGB(9, 8, 7)}
	for _, distance := range []float64{beyond, beyond * 1.001, 3} {
		_, far, light := lit.Light(maths.V(distance, 0, 0), maths.Vec{})
		want := lit.Strength.R / (distance * distance)
		if math.Abs(float64(light.R)-want) > 1e-12*want {
			t.Errorf("light at %v is %v, want the inverse square %v", distance,
				light.R, want)
		}
		if wantFar := distance - beyond; math.Abs(far-wantFar) > 1e-12 {
			t.Errorf("far at %v is %v, want %v", distance, far, wantFar)
		}
	}
}
