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

// Strobes adds the light of two lamps to sky, as seen through input.Camera. A
// lamp is lit if input.White or input.Magenta forces it on, or, if that is
// below zero, when a draw for the frame (12 to the second) is under 0.28. It
// returns the image and half a unit for each lit lamp, even one out of view,
// or a copy of sky if none in view. It refuses bad masks, non-finite values,
// an unsupported shoreline or time, and an over-budget frame.
func Strobes(sky lights.Image, input lights.StrobeInput) (lights.Image, float64, error) {
	width, height := sky.Width, sky.Height
	if !dimensions(width, height) || len(input.Open) != width*height || len(input.Near) != width*height || len(input.Rim) != width {
		return lights.Image{}, 0, errors.New("gpulights: invalid strobe masks")
	}
	if err := validateImage(sky, width, height); err != nil {
		return lights.Image{}, 0, err
	}
	values := []float64{input.Azimuth, input.Top, input.Bottom, input.Shore, input.Seconds, input.Moon[0], input.Moon[1], input.Moon[2], input.FarTree.X, input.FarTree.Y, input.FarTree.Z, input.Camera.Focal, input.Camera.Width, input.Camera.Height, input.Camera.Forward.X, input.Camera.Forward.Y, input.Camera.Forward.Z, input.Camera.Right.X, input.Camera.Right.Y, input.Camera.Right.Z, input.Camera.Up.X, input.Camera.Up.Y, input.Camera.Up.Z}
	for _, value := range values {
		if !finite(value) || math.Abs(value) > 1e7 {
			return lights.Image{}, 0, errors.New("gpulights: invalid strobe parameter")
		}
	}
	if input.Top <= input.Bottom || input.Shore <= input.Bottom || input.Shore >= input.Top || math.Abs(input.Seconds) > 1e5 {
		return lights.Image{}, 0, errors.New("gpulights: unsupported strobe geometry or time")
	}
	frame := int64(math.Round(input.Seconds * 12))
	positions := [2]lights.Vec{input.FarTree.Add(lights.Toward(input.Azimuth, 0).Scale(160)).Add(lights.Vec{Z: 20}), input.FarTree.Add(lights.Toward(input.Azimuth, 0).Scale(220)).Add(lights.Toward(input.Azimuth-90, 0).Scale(260)).Add(lights.Vec{Z: 30})}
	forced := [2]int{input.White, input.Magenta}
	total := 0.0
	var projected [4]float64
	var flags [2]uint32
	for index := range positions {
		on := lights.Lattice(frame, 0, uint64(index+1)) < 0.28
		if forced[index] == 0 || forced[index] < 0 && !on {
			continue
		}
		total += 0.5
		x, y, ahead := input.Camera.Project(positions[index])
		if !ahead {
			continue
		}
		if !finite(x) || !finite(y) || math.Abs(x) > 1e7 || math.Abs(y) > 1e7 {
			return lights.Image{}, 0, errors.New("gpulights: strobe projection outside supported range")
		}
		projected[index*2], projected[index*2+1] = x, y
		flags[index] = 1
	}
	if flags[0] == 0 && flags[1] == 0 {
		return lights.Image{Width: width, Height: height, Light: append([]float64(nil), sky.Light...)}, total, nil
	}
	params := make([]byte, 48)
	binary.LittleEndian.PutUint32(params, uint32(width))
	binary.LittleEndian.PutUint32(params[4:], uint32(height))
	binary.LittleEndian.PutUint32(params[8:], flags[0])
	binary.LittleEndian.PutUint32(params[12:], flags[1])
	shoreRow := (input.Top - input.Shore) / (input.Top - input.Bottom) * float64(height)
	luminance := (input.Moon[0] + input.Moon[1] + input.Moon[2]) / 3
	for index, value := range []float64{projected[0], projected[1], projected[2], projected[3], shoreRow, input.Seconds, luminance, 0} {
		if !finite(value) || math.Abs(value) > math.MaxFloat32 {
			return lights.Image{}, 0, errors.New("gpulights: strobe parameter outside float32 range")
		}
		binary.LittleEndian.PutUint32(params[16+index*4:], math.Float32bits(float32(value)))
	}
	if int64(width)*int64(height)*20+int64(width)*4 > DefaultMaxBytes {
		return lights.Image{}, 0, errors.New("gpulights: strobe allocation exceeds budget")
	}
	session, err := metal.OpenSession(strobeSource(), "strobe")
	if err != nil {
		return lights.Image{}, 0, fmt.Errorf("gpulights: open strobe: %w", err)
	}
	defer session.Close()
	image, err := session.Buffer(width * height * 3)
	if err != nil {
		return lights.Image{}, 0, err
	}
	open, err := session.Buffer(width * height)
	if err != nil {
		return lights.Image{}, 0, err
	}
	near, err := session.Buffer(width * height)
	if err != nil {
		return lights.Image{}, 0, err
	}
	rim, err := session.Buffer(width)
	if err != nil {
		return lights.Image{}, 0, err
	}
	if err := uploadImage(image.Numbers(), sky); err != nil {
		return lights.Image{}, 0, err
	}
	uploadMask(open.Numbers(), input.Open)
	uploadMask(near.Numbers(), input.Near)
	for index, value := range input.Rim {
		rim.Numbers()[index] = float32(value)
	}
	if err := session.Dispatch("strobe", width, height, params, image, open, near, rim); err != nil {
		return lights.Image{}, 0, err
	}
	if err := session.Commit(); err != nil {
		return lights.Image{}, 0, err
	}
	if err := validateFiniteBuffer("strobe", image.Numbers()); err != nil {
		return lights.Image{}, 0, err
	}
	return decodeImage(image.Numbers(), width, height), total, nil
}

// strobeSource is the Metal source for the strobe kernel. It also holds the
// noise helpers that backHazeSource and beamSource cut out and reuse.
func strobeSource() string {
	var source strings.Builder
	source.WriteString("#include <metal_stdlib>\nusing namespace metal;\n")
	for _, v := range []struct {
		name  string
		value float64
	}{{"zero", 0}, {"one", 1}, {"two", 2}, {"three", 3}, {"oneHalf", 0.5}, {"latticeScale", 1.0 / 9007199254740992}, {"time1", 0.09}, {"time2", 0.07}, {"time3", 0.1}, {"time4", 0.03}, {"shiftX", 5.2}, {"shiftY", 1.3}, {"warpX", 1.9}, {"warpY", 0.8}, {"octaveScale", 2.03}, {"norm", 0.96875}, {"base", 0.35}, {"strength", 0.65}, {"wispsLow", 0.35}, {"wispsHigh", 0.75}, {"behindDistance", 40}, {"behindScale", 90}, {"heightScale", 720}, {"lakeRise", 30}, {"lakeDecay", 1.5}, {"noiseX", 220}, {"noiseY", 60}, {"reachScale", 420.0 / 1280}, {"glowGain", 1.4}, {"rimMargin", 6}, {"occluded", 0.08}, {"magentaG", 0.25}, {"magentaB", 0.85}} {
		fmt.Fprintf(&source, "constant float %s = %.17ef;\n", v.name, v.value)
	}
	for _, v := range []struct {
		name  string
		value uint64
	}{{"hashX", 0x9E3779B97F4A7C15}, {"hashY", 0xC2B2AE3D27D4EB4F}, {"hashSeed", 0x165667B19E3779F9}, {"hashMix", 0xff51afd7ed558ccd}, {"octaveStep", 7919}, {"jitterSeed", 99}, {"baseSeed", 7}} {
		fmt.Fprintf(&source, "constant ulong %s = %dul;\n", v.name, v.value)
	}
	fmt.Fprintf(&source, "constant uint strobeSteps = %du;\n", 48)
	for _, value := range []struct {
		name  string
		value uint64
	}{{"hashShift", 33}, {"precisionShift", 11}, {"octaves", 5}, {"lamps", 2}} {
		fmt.Fprintf(&source, "constant uint %s = %du;\n", value.name, value.value)
	}
	source.WriteString(`struct Params { uint width; uint height; uint first; uint second; float x0; float y0; float x1; float y1; float shore; float seconds; float luminance; float padding; };
float lattice(long x,long y,ulong seed) { ulong h=ulong(x)*hashX ^ ulong(y)*hashY ^ seed*hashSeed; h^=h>>hashShift; h*=hashMix; h^=h>>hashShift; return float(h>>precisionShift)*latticeScale; }
float smoothValue(float a,float b,float x) { float t=clamp((x-a)/(b-a),zero,one); return t*t*(three-two*t); }
float noise(float u,float v,ulong seed) { float x=floor(u),y=floor(v),fx=u-x,fy=v-y; float sx=fx*fx*(three-two*fx),sy=fy*fy*(three-two*fy); long ix=long(x),iy=long(y); float a=lattice(ix,iy,seed),b=lattice(ix+1,iy,seed),c=lattice(ix,iy+1,seed),d=lattice(ix+1,iy+1,seed); return a+(b-a)*sx+(c-a)*sy+(a-b-c+d)*sx*sy; }
float fbm(float u,float v,ulong seed) { float sum=zero,amp=oneHalf; for(uint octave=0;octave<octaves;octave++) { sum+=amp*noise(u,v,seed+ulong(octave)*octaveStep); u*=octaveScale; v*=octaveScale; amp*=oneHalf; } return sum/norm; }
float curled(float u,float v,float seconds) { float qx=fbm(u+time1*seconds,v,baseSeed); float qy=fbm(u+shiftX-time2*seconds,v+shiftY,baseSeed+1); return fbm(u+warpX*qx,v+warpY*qy,baseSeed+2); }
float haze(float x,float y,float rim,float shore,float height,float seconds) { float k=height/heightScale,above=rim-y,behind=zero,lake=zero; if(above>-behindDistance*k) behind=exp(-max(zero,above)/(behindScale*k))*smoothValue(-behindDistance*k,zero,above); if(y>shore-lakeRise*k) lake=smoothValue(shore-lakeRise*k,shore,y)*(oneHalf+oneHalf*exp(-(y-shore)/(height-shore)*lakeDecay)); float u=x/(noiseX*k)+time3*seconds,v=y/(noiseY*k)-time4*seconds; return (behind+lake)*(base+strength*smoothValue(wispsLow,wispsHigh,curled(u,v,seconds))); }
kernel void strobe(device float *sky [[buffer(0)]],device const float *open [[buffer(1)]],device const float *nearMask [[buffer(2)]],device const float *rim [[buffer(3)]],constant Params &p [[buffer(4)]],uint2 point [[thread_position_in_grid]]) {
    if(point.x>=p.width || point.y>=p.height) return; uint pixel=point.y*p.width+point.x; float jitter=lattice(long(point.x),long(point.y),jitterSeed); float h=haze(float(point.x),float(point.y),rim[point.x],p.shore,float(p.height),p.seconds); float reach=reachScale*float(p.width); float3 colour=float3(zero);
    for(uint lamp=0;lamp<lamps;lamp++) { if((lamp==0 && p.first==0)||(lamp==1 && p.second==0)) continue; float sx=lamp==0?p.x0:p.x1,sy=lamp==0?p.y0:p.y1; float dx=sx-float(point.x),dy=sy-float(point.y),gathered=zero; for(uint step=0;step<strobeSteps;step++) { float f=(float(step)+jitter)/float(strobeSteps); int x=int(float(point.x)+dx*f),y=int(float(point.y)+dy*f); if(x<0 || y<0 || x>=int(p.width) || y>=int(p.height)) continue; uint at=uint(y)*p.width+uint(x); if(open[at]<=zero && nearMask[at]<=zero) break; gathered+=one/float(strobeSteps); } float d2=(dx*dx+dy*dy)/(reach*reach); float glow=glowGain*gathered*h/(one+d2); if(open[pixel]<=zero && float(point.y)<rim[point.x]+rimMargin) glow*=occluded; colour+=glow*(lamp==0?float3(one):float3(one,magentaG,magentaB)); }
    uint at=pixel*3; sky[at]+=colour.x*p.luminance; sky[at+1]+=colour.y*p.luminance; sky[at+2]+=colour.z*p.luminance;
}`)
	return source.String()
}
