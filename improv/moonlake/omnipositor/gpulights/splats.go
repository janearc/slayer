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

type Splat struct {
	X, Y, Sigma, Amount float64
	Colour              [3]float64
	SkyOnly             bool
}

// Splats adds each splat, a round spot of light of width Sigma pixels and
// total Amount, to sky and front, and returns both as new images. A SkyOnly
// splat goes to the sky, only where open is set. Any other goes to the sky
// above shoreRow and to the front from shoreRow down. It refuses bad sizes,
// non-finite values, a Sigma under 1e-10, over 1<<20 splats, and an
// over-budget frame.
func Splats(sky, front lights.Image, open []bool, shoreRow float64, splats []Splat) (lights.Image, lights.Image, error) {
	width, height := sky.Width, sky.Height
	if !dimensions(width, height) || len(open) != width*height || len(splats) > 1<<20 {
		return lights.Image{}, lights.Image{}, errors.New("gpulights: invalid splat dimensions or count")
	}
	if err := validateImage(sky, width, height); err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	if err := validateImage(front, width, height); err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	if !finite(shoreRow) || math.Abs(shoreRow) > math.MaxFloat32 {
		return lights.Image{}, lights.Image{}, errors.New("gpulights: invalid splat shoreline")
	}
	for _, splat := range splats {
		for _, value := range []float64{splat.X, splat.Y, splat.Sigma, splat.Amount, splat.Colour[0], splat.Colour[1], splat.Colour[2]} {
			if !finite(value) || math.Abs(value) > 1e7 {
				return lights.Image{}, lights.Image{}, errors.New("gpulights: invalid splat")
			}
		}
		if splat.Sigma < 1e-10 || float32(splat.Sigma) == 0 {
			return lights.Image{}, lights.Image{}, errors.New("gpulights: invalid splat sigma")
		}
	}
	if len(splats) == 0 {
		return lights.Image{Width: width, Height: height, Light: append([]float64(nil), sky.Light...)}, lights.Image{Width: width, Height: height, Light: append([]float64(nil), front.Light...)}, nil
	}
	tiles, err := binSplats(width, height, splats)
	if err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	if int64(width)*int64(height)*28+int64(len(splats))*32+int64(len(tiles.offsets)+len(tiles.indices))*4 > DefaultMaxBytes {
		return lights.Image{}, lights.Image{}, errors.New("gpulights: splat allocation exceeds budget")
	}
	session, err := metal.OpenSession(splatsSource(), "splats")
	if err != nil {
		return lights.Image{}, lights.Image{}, fmt.Errorf("gpulights: open splats: %w", err)
	}
	defer session.Close()
	skyBuffer, err := session.Buffer(width * height * 3)
	if err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	frontBuffer, err := session.Buffer(width * height * 3)
	if err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	maskBuffer, err := session.Buffer(width * height)
	if err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	splatBuffer, err := session.Buffer(len(splats) * 8)
	if err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	offsetBuffer, err := session.Buffer(len(tiles.offsets))
	if err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	indexBuffer, err := session.Buffer(max(1, len(tiles.indices)))
	if err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	for index, value := range tiles.offsets {
		offsetBuffer.Numbers()[index] = math.Float32frombits(value)
	}
	for index, value := range tiles.indices {
		indexBuffer.Numbers()[index] = math.Float32frombits(value)
	}
	if err := uploadImage(skyBuffer.Numbers(), sky); err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	if err := uploadImage(frontBuffer.Numbers(), front); err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	uploadMask(maskBuffer.Numbers(), open)
	for index, splat := range splats {
		base := index * 8
		values := []float64{splat.X, splat.Y, splat.Sigma, splat.Amount, splat.Colour[0], splat.Colour[1], splat.Colour[2]}
		for component, value := range values {
			splatBuffer.Numbers()[base+component] = float32(value)
		}
		if splat.SkyOnly {
			splatBuffer.Numbers()[base+7] = 1
		}
	}
	params := make([]byte, 16)
	binary.LittleEndian.PutUint32(params, uint32(width))
	binary.LittleEndian.PutUint32(params[4:], uint32(height))
	binary.LittleEndian.PutUint32(params[8:], uint32(tiles.columns))
	binary.LittleEndian.PutUint32(params[12:], math.Float32bits(float32(shoreRow)))
	if err := session.Dispatch("splats", width, height, params, skyBuffer, frontBuffer, maskBuffer, splatBuffer, offsetBuffer, indexBuffer); err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	if err := session.Commit(); err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	if err := validateFiniteBuffer("splat sky", skyBuffer.Numbers()); err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	if err := validateFiniteBuffer("splat front", frontBuffer.Numbers()); err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	return decodeImage(skyBuffer.Numbers(), width, height), decodeImage(frontBuffer.Numbers(), width, height), nil
}

// splatsSource is the Metal source for the splats kernel, which adds the
// splats that touch each pixel's tile to the sky or the front.
func splatsSource() string {
	var source strings.Builder
	source.WriteString("#include <metal_stdlib>\nusing namespace metal;\n")
	for _, number := range []struct {
		name  string
		value float64
	}{{"zero", 0}, {"one", 1}, {"two", 2}, {"oneHalf", 0.5}, {"pi", math.Pi}, {"skyRadius", 2.5}, {"fullRadius", 3}} {
		fmt.Fprintf(&source, "constant float %s = %.17ef;\n", number.name, number.value)
	}
	fmt.Fprintf(&source, "constant uint tileSize = %du;\n", splatTileSize)
	source.WriteString(`struct Params { uint width; uint height; uint columns; float shore; };
kernel void splats(device float *sky [[buffer(0)]],device float *front [[buffer(1)]],device const float *open [[buffer(2)]],device const float *splats [[buffer(3)]],device const uint *offsets [[buffer(4)]],device const uint *indices [[buffer(5)]],constant Params &p [[buffer(6)]],uint2 point [[thread_position_in_grid]]) {
    if(point.x>=p.width || point.y>=p.height) return; uint pixel=point.y*p.width+point.x,at=pixel*3; float3 skyValue=float3(sky[at],sky[at+1],sky[at+2]),frontValue=float3(front[at],front[at+1],front[at+2]);
    uint tile=(point.y/tileSize)*p.columns+point.x/tileSize;
    for(uint entry=offsets[tile];entry<offsets[tile+1];entry++) { uint base=indices[entry]*8; float x=splats[base],y=splats[base+1],sigma=splats[base+2],amount=splats[base+3]; bool skyOnly=splats[base+7]>zero; int radius=int(ceil((skyOnly?skyRadius:fullRadius)*sigma)); int cx=int(x),cy=int(y); if(abs(int(point.x)-cx)>radius || abs(int(point.y)-cy)>radius || (skyOnly && open[pixel]<=zero)) continue; float dx=float(point.x)+oneHalf-x,dy=float(point.y)+oneHalf-y; float weight=amount/(two*pi*sigma*sigma)*exp(-(dx*dx+dy*dy)/(two*sigma*sigma)); float3 value=weight*float3(splats[base+4],splats[base+5],splats[base+6]); if(skyOnly || float(point.y)<p.shore) skyValue+=value; else frontValue+=value; }
    sky[at]=skyValue.x; sky[at+1]=skyValue.y; sky[at+2]=skyValue.z; front[at]=frontValue.x; front[at+1]=frontValue.y; front[at+2]=frontValue.z;
}`)
	return source.String()
}
