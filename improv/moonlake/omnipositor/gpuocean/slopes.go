package gpuocean

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/metal"
)

type SurfaceSlope struct {
	X, Y, ResolvedAlong, ResolvedAcross float32
}

// Slopes returns the surface slope at each ray's hit, and the resolved
// variances, which are the sum over layers of share squared times the layer's
// own values. A ray whose hit is Sky or Flat gets zeros. It refuses other than
// three layers, layers not ordered from largest texel to smallest, rays and
// hits of different counts, and a ray that is not unit or not above 0.
func Slopes(layers []WaveLayer, rays []MarchRay, hits []MarchResult) ([]SurfaceSlope, error) {
	if len(layers) != 3 || len(rays) != len(hits) || len(rays) > MaxSamples {
		return nil, errors.New("gpuocean: invalid slope request")
	}
	gridCount := 0
	previous := math.Inf(1)
	for _, layer := range layers {
		field := layer.Field
		if field == nil || field.N < 2 || field.N > 1024 || field.Extent < 1e-10 || field.Extent > 1e12 || !rayFinite(field.Extent) || !rayFinite(layer.ResolvedAlong) || !rayFinite(layer.ResolvedAcross) || layer.ResolvedAlong < 0 || layer.ResolvedAcross < 0 || layer.ResolvedAlong > 1e12 || layer.ResolvedAcross > 1e12 {
			return nil, errors.New("gpuocean: invalid slope layer")
		}
		if field.Extent/float64(field.N) > previous {
			return nil, errors.New("gpuocean: unordered slope layers")
		}
		previous = field.Extent / float64(field.N)
		count := field.N * field.N
		for _, values := range [][]float64{field.SlopeX, field.SlopeY, field.PushX, field.PushY, field.StretchXX, field.StretchXY, field.StretchYY} {
			if len(values) != count {
				return nil, errors.New("gpuocean: incomplete slope layer")
			}
			for _, value := range values {
				if !rayFinite(value) || math.Abs(value) > 1e12 {
					return nil, errors.New("gpuocean: invalid slope field")
				}
			}
		}
		gridCount += count * 7
	}
	for index, ray := range rays {
		if !rayFinite(float64(hits[index].Distance)) || hits[index].Distance < 0 || hits[index].Sky && hits[index].Flat || !rayFinite(ray.PixelAngle) || ray.PixelAngle <= 0 || float32(ray.PixelAngle) == 0 || ray.PixelAngle > 1e12 {
			return nil, errors.New("gpuocean: invalid slope ray")
		}
		for _, value := range []float64{ray.Ray.At.X, ray.Ray.At.Y, ray.Ray.At.Z, ray.Ray.Dir.X, ray.Ray.Dir.Y, ray.Ray.Dir.Z} {
			if !rayFinite(value) || math.Abs(value) > 1e12 {
				return nil, errors.New("gpuocean: invalid slope ray")
			}
		}
		length := ray.Ray.Dir.X*ray.Ray.Dir.X + ray.Ray.Dir.Y*ray.Ray.Dir.Y + ray.Ray.Dir.Z*ray.Ray.Dir.Z
		if math.Abs(length-1) > 1e-5 || ray.Ray.At.Z <= 0 {
			return nil, errors.New("gpuocean: slope requires unit ray above sea")
		}
	}
	if len(rays) == 0 {
		return []SurfaceSlope{}, nil
	}
	session, err := metal.OpenSession(slopesSource(), "surface_slopes")
	if err != nil {
		return nil, fmt.Errorf("gpuocean: open slopes: %w", err)
	}
	defer session.Close()
	fields, err := session.Buffer(gridCount)
	if err != nil {
		return nil, err
	}
	input, err := session.Buffer(len(rays) * 9)
	if err != nil {
		return nil, err
	}
	output, err := session.Buffer(len(rays) * 4)
	if err != nil {
		return nil, err
	}
	params := make([]byte, 60)
	offset := 0
	for index, layer := range layers {
		field := layer.Field
		base := index * 20
		binary.LittleEndian.PutUint32(params[base:], uint32(field.N))
		binary.LittleEndian.PutUint32(params[base+4:], math.Float32bits(float32(field.Extent)))
		binary.LittleEndian.PutUint32(params[base+8:], uint32(offset))
		binary.LittleEndian.PutUint32(params[base+12:], math.Float32bits(float32(layer.ResolvedAlong)))
		binary.LittleEndian.PutUint32(params[base+16:], math.Float32bits(float32(layer.ResolvedAcross)))
		for _, values := range [][]float64{field.SlopeX, field.SlopeY, field.PushX, field.PushY, field.StretchXX, field.StretchXY, field.StretchYY} {
			for _, value := range values {
				fields.Numbers()[offset] = float32(value)
				offset++
			}
		}
	}
	for index, ray := range rays {
		status := 0.0
		if hits[index].Sky || hits[index].Flat {
			status = 1
		}
		values := []float64{ray.Ray.At.X, ray.Ray.At.Y, ray.Ray.At.Z, ray.Ray.Dir.X, ray.Ray.Dir.Y, ray.Ray.Dir.Z, ray.PixelAngle, float64(hits[index].Distance), status}
		for component, value := range values {
			input.Numbers()[index*9+component] = float32(value)
		}
	}
	if err := session.Dispatch("surface_slopes", len(rays), 1, params, fields, input, output); err != nil {
		return nil, err
	}
	if err := session.Commit(); err != nil {
		return nil, err
	}
	result := make([]SurfaceSlope, len(rays))
	for index := range result {
		values := output.Numbers()[index*4 : index*4+4]
		for _, value := range values {
			if !rayFinite(float64(value)) {
				return nil, errors.New("gpuocean: nonfinite surface slope")
			}
		}
		result[index] = SurfaceSlope{X: values[0], Y: values[1], ResolvedAlong: values[2], ResolvedAcross: values[3]}
	}
	return result, nil
}

// slopesSource is the Metal source for the surface_slopes kernel, which undoes
// the horizontal push of the waves in three rounds to find the surface slope.
func slopesSource() string {
	var source strings.Builder
	source.WriteString("#include <metal_stdlib>\nusing namespace metal;\n")
	for _, number := range []struct {
		name  string
		value float64
	}{{"zero", 0}, {"one", 1}, {"two", 2}, {"floorView", 0.01}, {"foldFloor", 0.1}, {"chop", 1}, {"maxTiles", 1048576}} {
		fmt.Fprintf(&source, "constant float %s = %.17ef;\n", number.name, number.value)
	}
	fmt.Fprintf(&source, "constant uint infinityBits = %du;\n", uint32(0x7f800000))
	fmt.Fprintf(&source, "constant uint inversionRounds = %du;\n", uint32(3))
	source.WriteString(`struct Layer { uint n; float extent; uint offset; float along; float across; }; struct Params { Layer layers[3]; };
float sampleField(device const float *data, constant Layer &l, uint part, float2 p) {
    if (any(abs(p/l.extent)>float2(maxTiles))) return as_type<float>(infinityBits);
    float2 at=fmod(p,float2(l.extent))/l.extent*float(l.n); float2 base=floor(at); float2 f=at-base; int n=int(l.n); int x=((int(base.x)%n)+n)%n; int y=((int(base.y)%n)+n)%n; int xx=(x+1)%n; int yy=(y+1)%n; uint offset=l.offset+part*l.n*l.n;
    float a=data[offset+y*n+x]*(one-f.x)+data[offset+y*n+xx]*f.x; float b=data[offset+yy*n+x]*(one-f.x)+data[offset+yy*n+xx]*f.x; return a*(one-f.y)+b*f.y;
}
float share(constant Layer &l,float footprint) { return clamp(two-footprint/(l.extent/float(l.n)),zero,one); }
kernel void surface_slopes(device const float *fields [[buffer(0)]],device const float *rays [[buffer(1)]],device float *out [[buffer(2)]],constant Params &p [[buffer(3)]],uint id [[thread_position_in_grid]]) {
    uint base=id*9; uint dest=id*4; if (rays[base+8]>zero) { for (uint c=0;c<4;c++) out[dest+c]=zero; return; }
    float3 origin=float3(rays[base],rays[base+1],rays[base+2]); float3 ray=float3(rays[base+3],rays[base+4],rays[base+5]); float distance=rays[base+7]; float footprint=distance*rays[base+6]/sqrt(max(abs(ray.z),floorView)); float2 target=(origin+ray*distance).xy; float2 position=target;
    if (!isfinite(footprint) || !all(isfinite(target))) { for(uint c=0;c<4;c++) out[dest+c]=as_type<float>(infinityBits); return; }
    for (uint round=0;round<inversionRounds;round++) { float2 push=float2(zero); for(uint layer=0;layer<3;layer++) { float w=share(p.layers[layer],footprint); if(w>zero) { push.x+=w*sampleField(fields,p.layers[layer],2,position); push.y+=w*sampleField(fields,p.layers[layer],3,position); } } if (!all(isfinite(push))) { for(uint c=0;c<4;c++) out[dest+c]=as_type<float>(infinityBits); return; } position=target-chop*push; }
    float gx=zero,gy=zero,xx=zero,xy=zero,yy=zero,along=zero,across=zero;
    for(uint layer=0;layer<3;layer++) { float w=share(p.layers[layer],footprint); if(w==zero) continue; gx+=w*sampleField(fields,p.layers[layer],0,position); gy+=w*sampleField(fields,p.layers[layer],1,position); xx+=w*sampleField(fields,p.layers[layer],4,position); xy+=w*sampleField(fields,p.layers[layer],5,position); yy+=w*sampleField(fields,p.layers[layer],6,position); along+=w*w*p.layers[layer].along; across+=w*w*p.layers[layer].across; }
    float a=one+chop*xx,b=chop*xy,d=one+chop*yy; float determinant=max(a*d-b*b,foldFloor); out[dest]=(d*gx-b*gy)/determinant; out[dest+1]=(a*gy-b*gx)/determinant; out[dest+2]=along; out[dest+3]=across;
}`)
	return source.String()
}
