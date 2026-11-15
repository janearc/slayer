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

// BackHaze adds mist lit by up to two lamps to front and returns it as a new
// image. Pixels where trees is set are left alone, and a lamp does not light a
// pixel it cannot see past a tree. A lamp with a level under 0.01 or behind
// the camera is skipped; with both skipped, it returns a copy of front. It
// refuses bad sizes, non-finite values and an over-budget frame.
func BackHaze(front lights.Image, trees []bool, input lights.BackHazeInput) (lights.Image, error) {
	width, height := front.Width, front.Height
	if !dimensions(width, height) || len(trees) != width*height {
		return lights.Image{}, errors.New("gpulights: invalid back haze dimensions")
	}
	if err := validateImage(front, width, height); err != nil {
		return lights.Image{}, err
	}
	values := []float64{input.Gain, input.Seconds, input.Moon[0], input.Moon[1], input.Moon[2], input.Levels[0], input.Levels[1], input.Camera.Focal, input.Camera.Width, input.Camera.Height}
	for _, value := range values {
		if !finite(value) || math.Abs(value) > 1e7 {
			return lights.Image{}, errors.New("gpulights: invalid back haze parameter")
		}
	}
	var positions [4]float64
	var flags [2]uint32
	for index, position := range input.Positions {
		if input.Levels[index] < 0.01 {
			continue
		}
		x, y, ahead := input.Camera.Project(position)
		if !ahead {
			continue
		}
		if !finite(x) || !finite(y) || math.Abs(x) > 1e7 || math.Abs(y) > 1e7 {
			return lights.Image{}, errors.New("gpulights: invalid back haze projection")
		}
		positions[index*2], positions[index*2+1] = x, y
		flags[index] = 1
	}
	if flags[0] == 0 && flags[1] == 0 {
		return lights.Image{Width: width, Height: height, Light: append([]float64(nil), front.Light...)}, nil
	}
	params := make([]byte, 52)
	binary.LittleEndian.PutUint32(params, uint32(width))
	binary.LittleEndian.PutUint32(params[4:], uint32(height))
	binary.LittleEndian.PutUint32(params[8:], flags[0])
	binary.LittleEndian.PutUint32(params[12:], flags[1])
	lum := (input.Moon[0] + input.Moon[1] + input.Moon[2]) / 3
	for index, value := range []float64{positions[0], positions[1], positions[2], positions[3], input.Levels[0], input.Levels[1], input.Gain, input.Seconds, lum} {
		if !finite(value) || math.Abs(value) > math.MaxFloat32 {
			return lights.Image{}, errors.New("gpulights: back haze parameter outside float32 range")
		}
		binary.LittleEndian.PutUint32(params[16+index*4:], math.Float32bits(float32(value)))
	}
	if int64(width)*int64(height)*16 > DefaultMaxBytes {
		return lights.Image{}, errors.New("gpulights: back haze allocation exceeds budget")
	}
	session, err := metal.OpenSession(backHazeSource(), "back_haze")
	if err != nil {
		return lights.Image{}, fmt.Errorf("gpulights: open back haze: %w", err)
	}
	defer session.Close()
	image, err := session.Buffer(width * height * 3)
	if err != nil {
		return lights.Image{}, err
	}
	mask, err := session.Buffer(width * height)
	if err != nil {
		return lights.Image{}, err
	}
	if err := uploadImage(image.Numbers(), front); err != nil {
		return lights.Image{}, err
	}
	uploadMask(mask.Numbers(), trees)
	if err := session.Dispatch("back_haze", width, height, params, image, mask); err != nil {
		return lights.Image{}, err
	}
	if err := session.Commit(); err != nil {
		return lights.Image{}, err
	}
	if err := validateFiniteBuffer("back haze", image.Numbers()); err != nil {
		return lights.Image{}, err
	}
	return decodeImage(image.Numbers(), width, height), nil
}

// backHazeSource is the Metal source for the back_haze kernel. It reuses the
// noise helpers from strobeSource, cut out by text position, and returns an
// empty string if it cannot find the cut points.
func backHazeSource() string {
	base := strobeSource()
	start := strings.Index(base, "struct Params")
	end := strings.Index(base, "float haze(")
	if start < 0 || end < 0 {
		return ""
	}
	base = base[:start] + base[strings.Index(base, "float lattice("):end]
	var source strings.Builder
	source.WriteString(base)
	for _, v := range []struct {
		name  string
		value float64
	}{{"hazeX", 240}, {"hazeY", 90}, {"widthScale", 1280}, {"driftX", 0.15}, {"driftY", 0.05}, {"hazeReach", 0.25}, {"magentaGreen", 0.25}, {"magentaBlue", 0.85}} {
		fmt.Fprintf(&source, "constant float %s = %.17ef;\n", v.name, v.value)
	}
	for _, v := range []struct {
		name  string
		value uint64
	}{{"hazeSeed", 23}, {"hazeJitter", 41}, {"hazeSteps", 32}} {
		fmt.Fprintf(&source, "constant ulong %s = %dul;\n", v.name, v.value)
	}
	source.WriteString(`struct Params { uint width; uint height; uint first; uint second; float x0; float y0; float x1; float y1; float level0; float level1; float gain; float seconds; float luminance; };
kernel void back_haze(device float *front [[buffer(0)]],device const float *trees [[buffer(1)]],constant Params &p [[buffer(2)]],uint2 point [[thread_position_in_grid]]) {
    if(point.x>=p.width||point.y>=p.height) return; uint pixel=point.y*p.width+point.x; if(trees[pixel]>zero) return; float scale=float(p.width)/widthScale; float u=float(point.x)/(hazeX*scale)+driftX*p.seconds; float v=float(point.y)/(hazeY*scale)-driftY*p.seconds; float qx=fbm(u+time1*p.seconds,v,hazeSeed),qy=fbm(u+shiftX-time2*p.seconds,v+shiftY,hazeSeed+1); float mist=base+strength*smoothValue(wispsLow,wispsHigh,fbm(u+warpX*qx,v+warpY*qy,hazeSeed+2)); float3 colour=float3(zero);
    for(uint side=0;side<lamps;side++) { if((side==0&&p.first==0)||(side==1&&p.second==0)) continue; float sx=side==0?p.x0:p.x1,sy=side==0?p.y0:p.y1,level=side==0?p.level0:p.level1,dx=sx-float(point.x),dy=sy-float(point.y),gathered=zero,jitter=lattice(long(point.x),long(point.y),hazeJitter); for(uint step=0;step<hazeSteps;step++) { float f=(float(step)+jitter)/float(hazeSteps); int x=int(float(point.x)+dx*f),y=int(float(point.y)+dy*f); if(x>=0&&y>=0&&x<int(p.width)&&y<int(p.height)&&trees[uint(y)*p.width+uint(x)]>zero) break; gathered+=one/float(hazeSteps); } float reach=hazeReach*float(p.width); float glow=p.gain*p.luminance*level*gathered*mist/(one+(dx*dx+dy*dy)/(reach*reach)); colour+=glow*(side==0?float3(one):float3(one,magentaGreen,magentaBlue)); }
    uint at=pixel*3; front[at]+=colour.x; front[at+1]+=colour.y; front[at+2]+=colour.z;
}`)
	return source.String()
}
