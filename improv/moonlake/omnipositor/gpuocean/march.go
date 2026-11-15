package gpuocean

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/oceantex"
)

type MarchLayer struct {
	Field *oceantex.Field
	Reach float64
}
type MarchRay struct {
	Ray        oceantex.Ray
	PixelAngle float64
}
type MarchResult struct {
	Distance  float32
	Sky, Flat bool
}

// March finds where each ray meets the waves of the three layers and returns,
// for each, the distance along the ray and two flags. Sky is set for a ray
// that rises clear of the waves. Flat is set for a falling ray whose pixel
// footprint is at least twice the first layer's texel there. It refuses other
// than three layers, layers not ordered from largest texel to smallest, a
// height beyond its layer's Reach, and a ray that is not unit or not above 0.
func March(layers []MarchLayer, rays []MarchRay) ([]MarchResult, error) {
	if len(layers) != 3 || len(rays) > MaxSamples {
		return nil, errors.New("gpuocean: march requires three layers and a bounded ray batch")
	}
	grid := 0
	totalReach := 0.0
	previousTexel := math.Inf(1)
	for _, layer := range layers {
		field := layer.Field
		if field == nil || field.N < 2 || field.N > 1024 || field.N > int(^uint(0)>>1)/field.N || !rayFinite(field.Extent) || field.Extent <= 0 || !rayFinite(layer.Reach) || layer.Reach < 0 {
			return nil, errors.New("gpuocean: invalid march layer")
		}
		texel := field.Extent / float64(field.N)
		if texel > previousTexel {
			return nil, errors.New("gpuocean: march layers must be ordered from largest to smallest texel")
		}
		previousTexel = texel
		count := field.N * field.N
		if len(field.Height) != count || len(field.PushX) != count || len(field.PushY) != count {
			return nil, errors.New("gpuocean: incomplete march layer")
		}
		for _, values := range [][]float64{field.Height, field.PushX, field.PushY} {
			for _, value := range values {
				if !rayFinite(value) || math.Abs(value) > 1e12 {
					return nil, errors.New("gpuocean: invalid march sample")
				}
			}
		}
		for _, height := range field.Height {
			if math.Abs(height) > layer.Reach || math.Abs(float64(float32(height))) > float64(float32(layer.Reach)) {
				return nil, errors.New("gpuocean: wave reach does not bound field heights")
			}
		}
		if field.Extent < 1e-10 || field.Extent > 1e12 || (layer.Reach > 0 && float32(layer.Reach) == 0) {
			return nil, errors.New("gpuocean: march extent or reach outside supported range")
		}
		grid += count * 3
		totalReach += float64(float32(layer.Reach))
	}
	if !rayFinite(totalReach) || totalReach > 1e12 {
		return nil, errors.New("gpuocean: invalid wave reach")
	}
	for _, sample := range rays {
		for _, value := range []float64{sample.Ray.At.X, sample.Ray.At.Y, sample.Ray.At.Z, sample.Ray.Dir.X, sample.Ray.Dir.Y, sample.Ray.Dir.Z, sample.PixelAngle} {
			if !rayFinite(value) || math.Abs(value) > 1e12 {
				return nil, errors.New("gpuocean: invalid march ray")
			}
		}
		for _, layer := range layers {
			if math.Abs(sample.Ray.At.X/layer.Field.Extent) > 1<<20 || math.Abs(sample.Ray.At.Y/layer.Field.Extent) > 1<<20 {
				return nil, errors.New("gpuocean: march origin exceeds periodic precision range")
			}
		}
		length := sample.Ray.Dir.X*sample.Ray.Dir.X + sample.Ray.Dir.Y*sample.Ray.Dir.Y + sample.Ray.Dir.Z*sample.Ray.Dir.Z
		if math.Abs(length-1) > 1e-5 || sample.Ray.At.Z <= 0 || sample.PixelAngle <= 0 || float32(sample.PixelAngle) == 0 {
			return nil, errors.New("gpuocean: march requires unit rays above sea and representable positive pixel angle")
		}
	}
	if len(rays) == 0 {
		return []MarchResult{}, nil
	}
	session, err := metal.OpenSession(marchSource(), "march")
	if err != nil {
		return nil, fmt.Errorf("gpuocean: open march: %w", err)
	}
	defer session.Close()
	fields, err := session.Buffer(grid)
	if err != nil {
		return nil, err
	}
	input, err := session.Buffer(len(rays) * 7)
	if err != nil {
		return nil, err
	}
	output, err := session.Buffer(len(rays) * 2)
	if err != nil {
		return nil, err
	}
	params := make([]byte, 40)
	reach32 := float32(totalReach)
	if float64(reach32) < totalReach {
		reach32 = math.Nextafter32(reach32, float32(math.Inf(1)))
	}
	binary.LittleEndian.PutUint32(params, math.Float32bits(reach32))
	offset := 0
	for index, layer := range layers {
		field := layer.Field
		binary.LittleEndian.PutUint32(params[4+index*12:], uint32(field.N))
		binary.LittleEndian.PutUint32(params[8+index*12:], math.Float32bits(float32(field.Extent)))
		binary.LittleEndian.PutUint32(params[12+index*12:], uint32(offset))
		for _, values := range [][]float64{field.Height, field.PushX, field.PushY} {
			for _, value := range values {
				fields.Numbers()[offset] = float32(value)
				offset++
			}
		}
	}
	for index, sample := range rays {
		values := []float64{sample.Ray.At.X, sample.Ray.At.Y, sample.Ray.At.Z, sample.Ray.Dir.X, sample.Ray.Dir.Y, sample.Ray.Dir.Z, sample.PixelAngle}
		for component, value := range values {
			input.Numbers()[index*7+component] = float32(value)
		}
	}
	if err := session.Dispatch("march", len(rays), 1, params, fields, input, output); err != nil {
		return nil, err
	}
	if err := session.Commit(); err != nil {
		return nil, err
	}
	result := make([]MarchResult, len(rays))
	for index := range result {
		distance, status := output.Numbers()[index*2], output.Numbers()[index*2+1]
		if !rayFinite(float64(distance)) || distance < 0 || (status != 0 && status != 1 && status != 2) {
			return nil, errors.New("gpuocean: invalid march result")
		}
		result[index] = MarchResult{Distance: distance, Sky: status == 1, Flat: status == 2}
	}
	return result, nil
}

// marchSource is the Metal source for the march kernel. It steps each ray down
// to the waves, at most 400 times, weighting each layer by how well its texels
// suit the footprint of the pixel at that distance.
func marchSource() string {
	var source strings.Builder
	source.WriteString("#include <metal_stdlib>\nusing namespace metal;\n")
	for _, number := range []struct {
		name  string
		value float64
	}{{"zero", 0}, {"one", 1}, {"two", 2}, {"floorView", 0.01}, {"near", 0.002}, {"slopeBound", 1.5}, {"chop", 1}, {"maxTiles", 1048576}} {
		fmt.Fprintf(&source, "constant float %s = %.17ef;\n", number.name, number.value)
	}
	fmt.Fprintf(&source, "constant uint maxSteps = %du;\n", 400)
	fmt.Fprintf(&source, "constant uint infinityBits = %du;\n", uint32(0x7f800000))
	source.WriteString(`struct Layer { uint n; float extent; uint offset; }; struct Params { float reach; Layer layers[3]; };
float sampleGrid(device const float *data, constant Layer &l, uint part, float2 p) {
    if (any(abs(p / l.extent) > float2(maxTiles))) return as_type<float>(infinityBits);
    float2 wrapped = fmod(p, float2(l.extent));
    float2 q = wrapped / l.extent * float(l.n); float2 base = floor(q); float2 f = q-base; int n = int(l.n);
    int x = ((int(base.x)%n)+n)%n; int y = ((int(base.y)%n)+n)%n; int xx=(x+1)%n; int yy=(y+1)%n;
    uint offset=l.offset+part*l.n*l.n; float a=data[offset+y*n+x]*(one-f.x)+data[offset+y*n+xx]*f.x; float b=data[offset+yy*n+x]*(one-f.x)+data[offset+yy*n+xx]*f.x; return a*(one-f.y)+b*f.y;
}
float share(constant Layer &l, float footprint) { return clamp(two-footprint/(l.extent/float(l.n)), zero, one); }
kernel void march(device const float *fields [[buffer(0)]], device const float *rays [[buffer(1)]], device float *out [[buffer(2)]], constant Params &p [[buffer(3)]], uint id [[thread_position_in_grid]]) {
    uint base=id*7; float3 origin=float3(rays[base],rays[base+1],rays[base+2]); float3 ray=float3(rays[base+3],rays[base+4],rays[base+5]); float pixel=rays[base+6];
    float graze=sqrt(max(abs(ray.z),floorView)); float level=length(ray.xy); float distance=zero; float status=zero;
    if (origin.z > p.reach) { if (ray.z >= zero) { out[id*2]=zero; out[id*2+1]=one; return; } distance=(origin.z-p.reach)/-ray.z; }
    if (!isfinite(distance)) { out[id*2]=distance; out[id*2+1]=zero; return; }
    for (uint step=0; step<maxSteps; step++) {
        float footprint=distance*pixel/graze; float3 at=origin+ray*distance;
        if (!isfinite(footprint) || !all(isfinite(at))) { out[id*2]=distance; out[id*2+1]=-one; return; }
        if (share(p.layers[0],footprint) == zero) { status=ray.z>=zero ? one : two; break; }
        float2 push=float2(zero); for (uint layer=0; layer<3; layer++) { float w=share(p.layers[layer],footprint); if (w>zero) { push.x+=w*sampleGrid(fields,p.layers[layer],1,at.xy); push.y+=w*sampleGrid(fields,p.layers[layer],2,at.xy); } }
        if (!all(isfinite(push))) { out[id*2]=distance; out[id*2+1]=-one; return; }
        float2 position=at.xy-chop*push; float height=zero; for (uint layer=0; layer<3; layer++) { float w=share(p.layers[layer],footprint); if (w>zero) height+=w*sampleGrid(fields,p.layers[layer],0,position); }
        if (!isfinite(height)) { out[id*2]=distance; out[id*2+1]=-one; return; }
        float gap=at.z-height; if (gap<=near*max(distance,one)) break;
        if (ray.z>zero && at.z>p.reach) { status=one; break; }
        float closing=slopeBound*level-ray.z; if (closing<=zero) { status=one; break; } distance+=gap/closing;
    }
    out[id*2]=distance; out[id*2+1]=status;
}`)
	return source.String()
}
