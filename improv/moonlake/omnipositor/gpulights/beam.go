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

// Beam adds a cone of light and its halo and fog, standing on the water
// config.Out out from the origin, to sky and to front, and returns both as new
// images. Pixels where trees is set are left alone. The front changes only
// where water is set. It refuses bad sizes, non-finite values, a Radius or
// Apex not above zero, a camera of another size, and an over-budget frame.
func Beam(sky, front lights.Image, config lights.UFOConfig, input lights.UFOInput, trees, water []bool) (lights.Image, lights.Image, error) {
	width, height := sky.Width, sky.Height
	if !dimensions(width, height) || trees != nil && len(trees) != width*height || water != nil && len(water) != width*height {
		return lights.Image{}, lights.Image{}, errors.New("gpulights: invalid beam dimensions")
	}
	if err := validateImage(sky, width, height); err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	if err := validateImage(front, width, height); err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	axis := lights.Toward(config.Azimuth, 0).Scale(config.Out)
	apex := axis.Add(lights.Vec{Z: config.WaterZ + config.Apex})
	footX, footY, _ := input.Camera.Project(axis.Add(lights.Vec{Z: config.WaterZ}))
	headX, headY, _ := input.Camera.Project(apex)
	edgeX, _, _ := input.Camera.Project(axis.Add(lights.Vec{Z: config.WaterZ}).Add(lights.Toward(config.Azimuth+90, 0).Scale(config.Radius)))
	footHalf := math.Max(1, math.Abs(edgeX-footX))
	lum := (input.Moon[0] + input.Moon[1] + input.Moon[2]) / 3
	values := []float64{config.Out, config.Radius, config.Apex, config.Gain, config.Halo, config.Azimuth, config.WaterZ, input.Seconds, input.Cues.Overhead, lum, axis.X, axis.Y, apex.Z, footX, footY, headX, headY, footHalf, input.Camera.Focal, input.Camera.Width, input.Camera.Height, input.Camera.Forward.X, input.Camera.Forward.Y, input.Camera.Forward.Z, input.Camera.Right.X, input.Camera.Right.Y, input.Camera.Right.Z, input.Camera.Up.X, input.Camera.Up.Y, input.Camera.Up.Z}
	for _, value := range values {
		if !finite(value) || math.Abs(value) > 1e7 {
			return lights.Image{}, lights.Image{}, errors.New("gpulights: invalid beam parameter")
		}
	}
	if config.Radius <= 0 || config.Apex <= 0 || float32(config.Radius) == 0 || float32(config.Apex) == 0 || input.Camera.Focal <= 0 || float32(input.Camera.Focal) == 0 || input.Camera.Width != float64(width) || input.Camera.Height != float64(height) || math.Abs(input.Seconds) > 1e5 {
		return lights.Image{}, lights.Image{}, errors.New("gpulights: unsupported beam geometry")
	}
	if int64(width)*int64(height)*32 > DefaultMaxBytes {
		return lights.Image{}, lights.Image{}, errors.New("gpulights: beam allocation exceeds budget")
	}
	packed := []float64{config.Radius, config.Apex, config.Gain, config.Halo, config.WaterZ, input.Seconds, input.Cues.Overhead, lum, axis.X, axis.Y, apex.Z, footX, footY, headX, headY, footHalf, input.Camera.Focal, input.Camera.Forward.X, input.Camera.Forward.Y, input.Camera.Forward.Z, input.Camera.Right.X, input.Camera.Right.Y, input.Camera.Right.Z, input.Camera.Up.X, input.Camera.Up.Y, input.Camera.Up.Z}
	params := make([]byte, 8+len(packed)*4)
	binary.LittleEndian.PutUint32(params, uint32(width))
	binary.LittleEndian.PutUint32(params[4:], uint32(height))
	for index, value := range packed {
		binary.LittleEndian.PutUint32(params[8+index*4:], math.Float32bits(float32(value)))
	}
	session, err := metal.OpenSession(beamSource(), "beam")
	if err != nil {
		return lights.Image{}, lights.Image{}, fmt.Errorf("gpulights: open beam: %w", err)
	}
	defer session.Close()
	skyBuf, err := session.Buffer(width * height * 3)
	if err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	frontBuf, err := session.Buffer(width * height * 3)
	if err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	treeBuf, err := session.Buffer(width * height)
	if err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	waterBuf, err := session.Buffer(width * height)
	if err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	if err := uploadImage(skyBuf.Numbers(), sky); err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	if err := uploadImage(frontBuf.Numbers(), front); err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	if trees != nil {
		uploadMask(treeBuf.Numbers(), trees)
	}
	if water != nil {
		uploadMask(waterBuf.Numbers(), water)
	}
	if err := session.Dispatch("beam", width, height, params, skyBuf, frontBuf, treeBuf, waterBuf); err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	if err := session.Commit(); err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	if err := validateFiniteBuffer("beam sky", skyBuf.Numbers()); err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	if err := validateFiniteBuffer("beam front", frontBuf.Numbers()); err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	return decodeImage(skyBuf.Numbers(), width, height), decodeImage(frontBuf.Numbers(), width, height), nil
}

// beamSource is the Metal source for the beam kernel. It reuses the noise
// helpers from strobeSource, and its own comment says why a ray that never
// meets the water is a far distance and a flag, not infinity.
func beamSource() string {
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
	}{{"coneEpsilon", 1e-12}, {"haloWidth", 0.9}, {"haloDecay", 0.5}, {"haloCut", 4}, {"haloX", 60.0 / 1280}, {"haloY", 25.0 / 1280}, {"haloTimeX", 0.2}, {"haloTimeY", 0.1}, {"haloBase", 0.4}, {"haloWisp", 0.6}, {"haloLow", 0.3}, {"haloHigh", 0.75}, {"fogHeight", 90}, {"fogX", 45}, {"fogZ", 18}, {"fogTimeX", 0.6}, {"fogTimeY", 0.2}, {"fogTimeZ", 0.1}, {"fogBase", 0.03}, {"fogWisp", 0.97}, {"fogLow", 0.45}, {"fogHigh", 0.7}, {"shell2", 1.07}, {"shell3", 1.16}, {"shellGain2", 0.5}, {"shellGain3", 0.3}, {"lightBase", 0.35}, {"lightFog", 6}, {"footOuter", 1.3}, {"footInner", 0.9}, {"footGain", 60}, {"tintR", 0.7}, {"tintG", 0.8}, {"innerR", 0.82}, {"innerG", 0.9}, {"shell2G", 0.12}, {"shell2B", 0.08}, {"shell3R", 0.55}, {"shell3G", 0.15}, {"sampleA", 0.2}, {"sampleB", 0.5}, {"sampleC", 0.8}, {"farAway", 1e30}} {
		fmt.Fprintf(&source, "constant float %s = %.17ef;\n", v.name, v.value)
	}
	for _, v := range []struct {
		name  string
		value uint64
	}{{"beamSeed", 29}, {"fogSeed", 17}, {"noiseYFactor", 7349}, {"noiseZFactor", 104729}, {"fogOctaves", 4}, {"infinityBits", 0x7f800000}} {
		fmt.Fprintf(&source, "constant ulong %s = %dul;\n", v.name, v.value)
	}
	// Metal compiles with fast math, which is free to assume no number is
	// infinite: a test for infinity may be folded away. So "never" here is
	// farAway, a finite distance past anything in the scene, and whether a
	// ray meets the water is kept as its own flag instead of being read
	// back out of an infinite distance.
	source.WriteString(`struct Params { uint width; uint height; float radius; float apexHeight; float gain; float halo; float waterZ; float seconds; float overhead; float lum; float axisX; float axisY; float apexZ; float footX; float footY; float headX; float headY; float footHalf; float focal; packed_float3 forward; packed_float3 right; packed_float3 up; };
float lerpValue(float a,float b,float t) { return a+(b-a)*t; }
float noise3(float u,float v,float w,ulong seed) { float x=floor(u),y=floor(v),z=floor(w),fx=u-x,fy=v-y,fz=w-z; float sx=fx*fx*(three-two*fx),sy=fy*fy*(three-two*fy),sz=fz*fz*(three-two*fz); long ix=long(x),iy=long(y),iz=long(z); long row=long(noiseYFactor),low=iy+iz*long(noiseZFactor),high=iy+(iz+1)*long(noiseZFactor); float a=lattice(ix,low,seed),b=lattice(ix+1,low,seed),c=lattice(ix,low+row,seed),d=lattice(ix+1,low+row,seed); float e=lattice(ix,high,seed),f=lattice(ix+1,high,seed),g=lattice(ix,high+row,seed),h=lattice(ix+1,high+row,seed); return lerpValue(lerpValue(lerpValue(a,b,sx),lerpValue(c,d,sx),sy),lerpValue(lerpValue(e,f,sx),lerpValue(g,h,sx),sy),sz); }
float fog(float3 point,float seconds,float water) { float above=point.z-water; if(above<zero) return zero; float u=point.x/fogX+fogTimeX*seconds,v=point.y/fogX-fogTimeY*seconds,w=point.z/fogZ+fogTimeZ*seconds,sum=zero,amp=oneHalf,norm3=zero; for(ulong octave=0;octave<fogOctaves;octave++) { sum+=amp*noise3(u,v,w,fogSeed+octave*octaveStep); norm3+=amp; u*=octaveScale; v*=octaveScale; w*=octaveScale; amp*=oneHalf; } return exp(-above/fogHeight)*(fogBase+fogWisp*smoothValue(fogLow,fogHigh,sum/norm3)); }
bool chord(float3 d,float3 apex,float slope,float low,thread float &inside,thread float &outside) { float k2=slope*slope,a=d.x*d.x+d.y*d.y-k2*d.z*d.z,b=-two*(d.x*apex.x+d.y*apex.y)+two*k2*apex.z*d.z,c=apex.x*apex.x+apex.y*apex.y-k2*apex.z*apex.z; float infinity=farAway; float2 first=float2(-infinity,infinity),second=float2(zero); bool hasFirst=false,hasSecond=false; if(abs(a)<coneEpsilon) { if(b==zero) return false; float t=-c/b; first=b>zero?float2(-infinity,t):float2(t,infinity); hasFirst=true; } else { float disc=b*b-two*two*a*c; if(disc<zero) { if(a<zero) { first=float2(-infinity,infinity); hasFirst=true; } } else { float root=sqrt(disc),t1=(-b-root)/(two*a),t2=(-b+root)/(two*a); if(t1>t2) { float swap=t1;t1=t2;t2=swap; } if(a>zero) { first=float2(t1,t2); hasFirst=true; } else { first=float2(-infinity,t1);second=float2(t2,infinity);hasFirst=true;hasSecond=true; } } } float lo=zero,hi=infinity; if(d.z>zero) { lo=max(zero,low/d.z);hi=apex.z/d.z; } else if(d.z<zero) { lo=max(zero,apex.z/d.z);hi=low/d.z; } else if(low>zero||apex.z<zero) return false; if(hasFirst) { inside=max(first.x,lo);outside=min(first.y,hi);if(outside>inside)return true; } if(hasSecond) { inside=max(second.x,lo);outside=min(second.y,hi);if(outside>inside)return true; } return false; }
kernel void beam(device float *sky [[buffer(0)]],device float *front [[buffer(1)]],device const float *trees [[buffer(2)]],device const float *water [[buffer(3)]],constant Params &p [[buffer(4)]],uint2 point [[thread_position_in_grid]]) {
    if(point.x>=p.width||point.y>=p.height)return; uint pixel=point.y*p.width+point.x; if(trees[pixel]>zero)return; uint at=pixel*3; float3 result=float3(zero); float below=float(point.y)-p.footY;
    if(below<haloCut*p.footHalf) { float fade=below>zero?exp(-below/(haloDecay*p.footHalf)):one; float fraction=p.headY!=p.footY?max(zero,(float(point.y)-p.footY)/(p.headY-p.footY)):zero; float axis=p.footX+fraction*(p.headX-p.footX),halfWidth=max(one,p.footHalf*(one-fraction)),across=abs(float(point.x)-axis)/halfWidth-one; if(below>zero&&across<zero) across=zero; if(across>zero||below>zero) { float u=float(point.x)/(haloX*float(p.width))+haloTimeX*p.seconds,v=float(point.y)/(haloY*float(p.width))-haloTimeY*p.seconds; float qx=fbm(u+time1*p.seconds,v,beamSeed),qy=fbm(u+shiftX-time2*p.seconds,v+shiftY,beamSeed+1); float wisps=haloBase+haloWisp*smoothValue(haloLow,haloHigh,fbm(u+warpX*qx,v+warpY*qy,beamSeed+2)); float halo=p.halo*p.gain*p.overhead*p.lum*two*p.radius*(one-fraction)*wisps*exp(-across/haloWidth)*fade; result+=halo*float3(tintR,tintG,one); } }
    float3 d=normalize(float3(p.forward)+float3(p.right)*((float(point.x)+oneHalf-float(p.width)*oneHalf)/p.focal)+float3(p.up)*(-(float(point.y)+oneHalf-float(p.height)*oneHalf)/p.focal)); bool wet=d.z<zero; float hitWater=wet?p.waterZ/d.z:farAway; float3 apex=float3(p.axisX,p.axisY,p.apexZ); float inner=zero; float slopes[3]={one,shell2,shell3}; float gains[3]={one,shellGain2,shellGain3}; float3 colours[3]={float3(innerR,innerG,one),float3(one,shell2G,shell2B),float3(shell3R,shell3G,one)}; float samples[3]={sampleA,sampleB,sampleC};
    for(uint layer=0;layer<3;layer++) { float entry,exit; if(!chord(d,apex,p.radius/p.apexHeight*slopes[layer],p.waterZ,entry,exit)||entry>=hitWater)continue; exit=min(exit,hitWater); float mist=zero; for(uint sample=0;sample<3;sample++) mist+=(lightBase+lightFog*fog(d*(entry+samples[sample]*(exit-entry)),p.seconds,p.waterZ))/three; float chordLength=exit-entry,shell=chordLength-inner; inner=chordLength; if(shell>zero)result+=p.gain*p.overhead*p.lum*mist*shell*gains[layer]*colours[layer]; }
    if(wet) { float3 hit=d*hitWater; float distance=length(hit.xy-float2(p.axisX,p.axisY)); if(distance<p.radius*footOuter) result+=p.gain*p.overhead*p.lum*footGain*smoothValue(p.radius*footOuter,p.radius*footInner,distance)*colours[0]; }
    sky[at]+=result.x;sky[at+1]+=result.y;sky[at+2]+=result.z; if(water[pixel]>zero) { front[at]+=result.x;front[at+1]+=result.y;front[at+2]+=result.z; }
}`)
	return source.String()
}
