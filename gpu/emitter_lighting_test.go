//go:build darwin

package gpu

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/thing"
	"strings"
	"testing"
)

// emitterProbe counts diffuse-light queries on the real surfaced function.
// The lighting integral is replaced by a measured constant and counter so
// no scene traversal can hide a query that should never have happened.
func emitterProbe(t *testing.T) string {
	t.Helper()
	source := sourceWith(1)
	begin := strings.Index(source, "static float3 falling(")
	if begin < 0 {
		t.Fatal("lighting function not found")
	}
	end := strings.Index(source[begin:], "// mirrors is")
	if end < 0 {
		t.Fatal("lighting function not found")
	}
	end += begin
	source = source[:begin] + `
static float3 falling(World world,float3 point,float3 normal,int leaving_sphere){
 device float *counter=(device float *)world.boxes;
 counter[0]+=1;
 return float3(2);
}
` + source[end:]
	return source + `
kernel void emission_probe(device const Material *materials [[buffer(0)]],
 device float *out [[buffer(1)]]) {
 World world={};world.materials=materials;world.boxes=out;out[0]=0;
 Hit hit={1,0,float3(0,1,0),float3(0),float3(0),1};
 Seen seen=surfaced(world,hit,float3(0),float3(0,1,0));
 out[1]=seen.light.x;out[2]=seen.met.colour.x;out[3]=seen.met.glow.x;
}
`
}
func TestPureEmissionDoesNotQueryDiffuseLighting(t *testing.T) {
	kernel, err := metal.Open(emitterProbe(t), "emission_probe")
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	for _, test := range []struct {
		name                       string
		material                   thing.Material
		calls, light, paint, shine float32
	}{
		{"emitter", thing.Material{Glow: maths.Grey(4)}, 0, 4, 0, 0},
		{"dark", thing.Material{}, 0, 0, 0, 0},
		{"reflecting emitter", thing.Material{Colour: maths.Grey(.5), Glow: maths.Grey(4)}, 1, 5, 0, 0},
		{"reflectance texture", thing.Material{Texture: banded{}}, 1, 0, 2, 0},
		{"emission texture", thing.Material{Shine: banded{}}, 0, 0, 0, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			encoded := &scene{materialOf: map[*thing.Material]int{}}
			encoded.surface(&test.material)
			got, err := kernel.Grid(1, 1, 1, 4, [][]byte{encoded.materials.bytes})
			if err != nil {
				t.Fatal(err)
			}
			want := []float32{test.calls, test.light, test.paint, test.shine}
			for i, v := range want {
				if got[i] != v {
					t.Fatal("diffuse query/emission contract", got, want)
				}
			}
		})
	}
}
