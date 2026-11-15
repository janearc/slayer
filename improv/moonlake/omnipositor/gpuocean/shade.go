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

type ShadeSample struct {
	Eye   oceantex.Vec
	Slope SurfaceSlope
	Sky   [3]float32
}

// Shade returns the colour of the sea for each sample: the sky by its Fresnel
// weight, the water's own colour, and sun glint, mixed with foam by the
// constants' foam share. It refuses more than MaxSamples samples, constants
// out of range, an eye that is not unit length, and a negative resolved
// variance.
func Shade(constants oceantex.LightingConstants, samples []ShadeSample) ([][3]float32, error) {
	if len(samples) > MaxSamples {
		return nil, errors.New("gpuocean: shade sample limit exceeded")
	}
	values := []float64{constants.Sun.X, constants.Sun.Y, constants.Sun.Z, constants.Sunlight[0], constants.Sunlight[1], constants.Sunlight[2], constants.Water[0], constants.Water[1], constants.Water[2], constants.Index, constants.Along, constants.Across, constants.WindSin, constants.WindCos, constants.Foam, constants.Down, constants.SlopeFloor, constants.ViewFloor, constants.GlintGeometry, constants.GlintTilt, constants.FoamReflectance}
	for _, value := range values {
		if !rayFinite(value) || math.Abs(value) > 1e10 {
			return nil, errors.New("gpuocean: invalid lighting constants")
		}
	}
	sunLength := constants.Sun.X*constants.Sun.X + constants.Sun.Y*constants.Sun.Y + constants.Sun.Z*constants.Sun.Z
	if math.Abs(sunLength-1) > 1e-5 || constants.Sun.Z <= 0 || constants.Index < 1e-10 || constants.Index > 1e10 || constants.SlopeFloor < 1e-10 || constants.ViewFloor < 1e-10 || constants.GlintGeometry <= 0 || constants.GlintTilt <= 0 || constants.Foam < 0 || constants.Foam > 1 {
		return nil, errors.New("gpuocean: unsupported lighting constants")
	}
	for _, sample := range samples {
		length := sample.Eye.X*sample.Eye.X + sample.Eye.Y*sample.Eye.Y + sample.Eye.Z*sample.Eye.Z
		if !rayFinite(length) || math.Abs(length-1) > 1e-5 {
			return nil, errors.New("gpuocean: shade eye must be unit length")
		}
		for _, value := range []float64{float64(sample.Slope.X), float64(sample.Slope.Y), float64(sample.Slope.ResolvedAlong), float64(sample.Slope.ResolvedAcross), float64(sample.Sky[0]), float64(sample.Sky[1]), float64(sample.Sky[2])} {
			if !rayFinite(value) || math.Abs(value) > 1e10 {
				return nil, errors.New("gpuocean: invalid shade sample")
			}
		}
		if sample.Slope.ResolvedAlong < 0 || sample.Slope.ResolvedAcross < 0 {
			return nil, errors.New("gpuocean: negative resolved variance")
		}
	}
	if len(samples) == 0 {
		return [][3]float32{}, nil
	}
	params := make([]byte, len(values)*4)
	for index, value := range values {
		binary.LittleEndian.PutUint32(params[index*4:], math.Float32bits(float32(value)))
	}
	session, err := metal.OpenSession(shadeSource(), "shade_ocean")
	if err != nil {
		return nil, fmt.Errorf("gpuocean: open shade: %w", err)
	}
	defer session.Close()
	input, err := session.Buffer(len(samples) * 10)
	if err != nil {
		return nil, err
	}
	output, err := session.Buffer(len(samples) * 3)
	if err != nil {
		return nil, err
	}
	for index, sample := range samples {
		values := []float64{sample.Eye.X, sample.Eye.Y, sample.Eye.Z, float64(sample.Slope.X), float64(sample.Slope.Y), float64(sample.Slope.ResolvedAlong), float64(sample.Slope.ResolvedAcross), float64(sample.Sky[0]), float64(sample.Sky[1]), float64(sample.Sky[2])}
		for component, value := range values {
			input.Numbers()[index*10+component] = float32(value)
		}
	}
	if err := session.Dispatch("shade_ocean", len(samples), 1, params, input, output); err != nil {
		return nil, err
	}
	if err := session.Commit(); err != nil {
		return nil, err
	}
	result := make([][3]float32, len(samples))
	for index := range result {
		for channel := 0; channel < 3; channel++ {
			value := output.Numbers()[index*3+channel]
			if !rayFinite(float64(value)) {
				return nil, errors.New("gpuocean: nonfinite ocean radiance")
			}
			result[index][channel] = value
		}
	}
	return result, nil
}

// shadeSource is the Metal source for the shade_ocean kernel. Its comment says
// why erfc is worked by two routes.
func shadeSource() string {
	var source strings.Builder
	source.WriteString("#include <metal_stdlib>\nusing namespace metal;\n")
	for _, number := range []struct {
		name  string
		value float64
	}{{"zero", 0}, {"one", 1}, {"two", 2}, {"oneHalf", 0.5}, {"pi", math.Pi}, {"hiddenCutoff", 10}, {"erfSwitch", 1}} {
		fmt.Fprintf(&source, "constant float %s = %.17ef;\n", number.name, number.value)
	}
	fmt.Fprintf(&source, "constant uint infinityBits = %du;\n", uint32(0x7f800000))
	// Metal has no erfc. What hidden wants is half of e to the minus v
	// squared over v root pi, less erfc of v, and the two nearly cancel
	// for a large v. So from one up it is worked without the
	// subtraction, from erfc's continued fraction, sixty terms from the
	// tail; under one, where nothing cancels, erf is its series of
	// positive terms, thirty of them.
	fmt.Fprintf(&source, "constant uint erfFraction = %du;\nconstant uint erfSeries = %du;\n", 60, 30)
	source.WriteString(`struct Params { packed_float3 sun; packed_float3 sunlight; packed_float3 water; float index; float along; float across; float windSin; float windCos; float foam; float down; float slopeFloor; float viewFloor; float glintGeometry; float glintTilt; float foamReflectance; };
float fresnelValue(float cosine,float index) { cosine=clamp(cosine,zero,one); float sine2=(one-cosine*cosine)/(index*index); if(sine2>=one) return one; float inside=sqrt(one-sine2); float a=(cosine-index*inside)/(cosine+index*inside); float b=(index*cosine-inside)/(index*cosine+inside); return (a*a+b*b)*oneHalf; }
float spread(float2 direction,constant Params &p) { float size=length(direction); if(size==zero) return p.along; float a=(direction.x*p.windCos+direction.y*p.windSin)/size; float b=(-direction.x*p.windSin+direction.y*p.windCos)/size; return p.along*a*a+p.across*b*b; }
float shadowed(float v) { float tail=exp(-v*v)/(v*sqrt(pi)); if(v>=erfSwitch) { float q=one/(two*v*v),t=zero; for(uint k=erfFraction;k>0;k--) t=float(k)*q/(one+t); return tail*t/(one+t)*oneHalf; } float term=v,sum=v; for(uint n=0;n<erfSeries;n++) { term*=two*v*v/float(2*n+3); sum+=term; } float erfValue=two/sqrt(pi)*exp(-v*v)*sum; return (tail-(one-erfValue))*oneHalf; }
float hidden(float cosine,float variance) { if(cosine<=zero) return as_type<float>(infinityBits); if(cosine>=one || variance<=zero) return zero; float cot=cosine/sqrt(one-cosine*cosine); float v=cot/sqrt(two*variance); if(v>hiddenCutoff) return zero; return shadowed(v); }
kernel void shade_ocean(device const float *input [[buffer(0)]],device float *output [[buffer(1)]],constant Params &p [[buffer(2)]],uint id [[thread_position_in_grid]]) {
    uint base=id*10; float3 eye=float3(input[base],input[base+1],input[base+2]); float slopeX=input[base+3],slopeY=input[base+4]; float varAlong=max(p.along-input[base+5],p.slopeFloor); float varAcross=max(p.across-input[base+6],p.slopeFloor); float3 sky=float3(input[base+7],input[base+8],input[base+9]); float3 sun=float3(p.sun); float3 seen=float3(zero);
    float3 sum=sun+eye; float magnitude=length(sum); float3 halfway=float3(zero); if(magnitude>zero) halfway=sum/magnitude;
    if(halfway.z>zero) { float dx=-halfway.x/halfway.z-slopeX,dy=-halfway.y/halfway.z-slopeY; float a=dx*p.windCos+dy*p.windSin,b=-dx*p.windSin+dy*p.windCos; float chance=exp(-(a*a/varAlong+b*b/varAcross)*oneHalf)/(two*pi*sqrt(varAlong*varAcross)); float visibility=one/((one+hidden(eye.z,spread(eye.xy,p)))*(one+hidden(sun.z,spread(sun.xy,p)))); float glint=fresnelValue(dot(sun,halfway),p.index)*chance*visibility/(p.glintGeometry*max(eye.z,p.viewFloor)*pow(halfway.z,p.glintTilt)); seen+=float3(p.sunlight)*glint; }
    float3 normal=normalize(float3(-slopeX,-slopeY,one)); float cosine=max(dot(normal,eye),zero); seen+=sky*fresnelValue(cosine,p.index); seen+=float3(p.water)*(p.down/pi); float3 white=float3(p.sunlight)*(p.foamReflectance*p.down/pi); seen=seen*(one-p.foam)+white*p.foam;
    uint dest=id*3; output[dest]=seen.x; output[dest+1]=seen.y; output[dest+2]=seen.z;
}`)
	return source.String()
}
