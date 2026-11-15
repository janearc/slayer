package gpulights

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/metal"
)

// Fire sums the fire frames the plan picks into one new image. Each layer adds
// its frame times its amount, except where CloseTrees is set. Then the sum's
// luminance, after a gain of two to the power Stops, is rolled off smoothly
// toward a ceiling of Top times the strike. It refuses more than 64 layers, a
// missing frame, a ceiling or gain it cannot hold in float32, and an
// over-budget frame.
func Fire(config lights.FireConfig, input lights.FireInput) (lights.Image, error) {
	width, height := input.Width, input.Height
	if !dimensions(width, height) || input.CloseTrees != nil && len(input.CloseTrees) != width*height {
		return lights.Image{}, errors.New("gpulights: invalid fire dimensions")
	}
	counts := [3]int{input.Frames[0].Count, input.Frames[1].Count, input.Frames[2].Count}
	layers, err := lights.PlanFire(config, input.Cues, counts)
	if err != nil {
		return lights.Image{}, err
	}
	if len(layers) > 64 {
		return lights.Image{}, errors.New("gpulights: fire layer limit exceeded")
	}
	gain := math.Exp2(config.Stops)
	top := config.Top * lights.FireStrike(config, input.Cues)
	if !finite(gain) || gain <= 0 || gain > math.MaxFloat32 || !finite(top) || top <= 0 || top > math.MaxFloat32 || float32(top) == 0 {
		return lights.Image{}, errors.New("gpulights: unsupported fire shoulder")
	}
	pixels := width * height
	if int64(pixels)*int64(12*len(layers)+16) > DefaultMaxBytes {
		return lights.Image{}, errors.New("gpulights: fire allocation exceeds budget")
	}
	for _, layer := range layers {
		if layer.Sequence < lights.FireSteady || layer.Sequence > lights.FireBurst {
			return lights.Image{}, errors.New("gpulights: invalid fire sequence")
		}
		pass, ok := input.Frames[layer.Sequence].Images[layer.Index]
		if !ok {
			return lights.Image{}, fmt.Errorf("gpulights: missing fire sequence %d frame %d", layer.Sequence, layer.Index)
		}
		if err := validateImage(pass, width, height); err != nil {
			return lights.Image{}, err
		}
		if !finite(layer.Amount) || math.Abs(layer.Amount) > math.MaxFloat32 {
			return lights.Image{}, errors.New("gpulights: invalid fire amount")
		}
	}
	session, err := metal.OpenSession(fireSource(), "fire_add", "fire_shoulder")
	if err != nil {
		return lights.Image{}, fmt.Errorf("gpulights: open fire: %w", err)
	}
	defer session.Close()
	output, err := session.Buffer(pixels * 3)
	if err != nil {
		return lights.Image{}, err
	}
	mask, err := session.Buffer(pixels)
	if err != nil {
		return lights.Image{}, err
	}
	if input.CloseTrees != nil {
		uploadMask(mask.Numbers(), input.CloseTrees)
	}
	for _, layer := range layers {
		pass := input.Frames[layer.Sequence].Images[layer.Index]
		buffer, err := session.Buffer(pixels * 3)
		if err != nil {
			return lights.Image{}, err
		}
		if err := uploadImage(buffer.Numbers(), pass); err != nil {
			return lights.Image{}, err
		}
		params := make([]byte, 12)
		binary.LittleEndian.PutUint32(params, uint32(width))
		binary.LittleEndian.PutUint32(params[4:], uint32(height))
		binary.LittleEndian.PutUint32(params[8:], math.Float32bits(float32(layer.Amount)))
		if err := session.Dispatch("fire_add", width, height, params, output, buffer, mask); err != nil {
			return lights.Image{}, err
		}
	}
	params := make([]byte, 16)
	binary.LittleEndian.PutUint32(params, uint32(width))
	binary.LittleEndian.PutUint32(params[4:], uint32(height))
	binary.LittleEndian.PutUint32(params[8:], math.Float32bits(float32(top)))
	binary.LittleEndian.PutUint32(params[12:], math.Float32bits(float32(gain)))
	if err := session.Dispatch("fire_shoulder", width, height, params, output); err != nil {
		return lights.Image{}, err
	}
	if err := session.Commit(); err != nil {
		return lights.Image{}, err
	}
	if err := validateFiniteBuffer("fire", output.Numbers()); err != nil {
		return lights.Image{}, err
	}
	return decodeImage(output.Numbers(), width, height), nil
}

// fireSource is the Metal source for the fire_add and fire_shoulder kernels. It
// defines its own expm1, which Metal does not have.
func fireSource() string {
	var source strings.Builder
	source.WriteString("#include <metal_stdlib>\nusing namespace metal;\n")
	for _, number := range []struct {
		name  string
		value float64
	}{{"zero", 0}, {"one", 1}, {"red", 0.2126}, {"green", 0.7152}, {"blue", 0.0722}, {"expSeries", 0.25}, {"expTerm2", 1.0 / 2}, {"expTerm3", 1.0 / 6}, {"expTerm4", 1.0 / 24}, {"expTerm5", 1.0 / 120}, {"expTerm6", 1.0 / 720}, {"expTerm7", 1.0 / 5040}} {
		fmt.Fprintf(&source, "constant float %s = %.17ef;\n", number.name, number.value)
	}
	// Metal has no expm1. Below a quarter, e to the x less one is its own
	// series to the seventh power, which is right to parts in a thousand
	// million; from a quarter up, the plain difference loses under a part
	// in a million.
	source.WriteString(`struct AddParams { uint width; uint height; float amount; }; struct ShoulderParams { uint width; uint height; float top; float gain; };
float expm1Value(float x) { if(abs(x)<expSeries) return x*(one+x*(expTerm2+x*(expTerm3+x*(expTerm4+x*(expTerm5+x*(expTerm6+x*expTerm7)))))); return exp(x)-one; }
kernel void fire_add(device float *out [[buffer(0)]],device const float *frame [[buffer(1)]],device const float *trees [[buffer(2)]],constant AddParams &p [[buffer(3)]],uint2 point [[thread_position_in_grid]]) {
    if(point.x>=p.width || point.y>=p.height) return; uint pixel=point.y*p.width+point.x; if(trees[pixel]>zero) return; uint at=pixel*3; out[at]+=p.amount*frame[at]; out[at+1]+=p.amount*frame[at+1]; out[at+2]+=p.amount*frame[at+2];
}
kernel void fire_shoulder(device float *out [[buffer(0)]],constant ShoulderParams &p [[buffer(1)]],uint2 point [[thread_position_in_grid]]) {
    if(point.x>=p.width || point.y>=p.height) return; uint at=(point.y*p.width+point.x)*3; float3 fire=float3(out[at],out[at+1],out[at+2]); float y=dot(fire,float3(red,green,blue))*p.gain; if(isinf(y) && y>zero) { fire=float3(p.top/p.gain); } else if(y>zero) { float scale=-p.top*expm1Value(-y/p.top)/y; fire*=scale; } out[at]=fire.x; out[at+1]=fire.y; out[at+2]=fire.z;
}`)
	return source.String()
}
