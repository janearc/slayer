#include <metal_stdlib>
using namespace metal;

// Float32 counterpart of hole.radius. Mass and spin use the scene's length
// units; normalizing the scene by mass avoids overflow at astronomical scales.
static float kerr_radius(float3 x, float spin) {
 float d = dot(x,x)-spin*spin;
 // On the equatorial disc the exact root is zero. Fast float arithmetic
 // must not turn cancellation here into a tiny radius and enormous force.
 if (x.z==0 && d<=0) return 0;
 return sqrt(max(0.0f,.5f*(d+sqrt(d*d+4*spin*spin*x.z*x.z))));
}

struct KerrRate { float4 x, p; };

static float4 kerr_lower(float3 x,float4 v,float mass,float spin) {
 float r=kerr_radius(x,spin),r2=r*r,a2=spin*spin;
 float denominator=r2*r2+a2*x.z*x.z;
 if (r==0 || denominator==0) return float4(-v.x,v.yzw);
 float3 n=float3((r*x.x+spin*x.y)/(r2+a2),
                 (r*x.y-spin*x.x)/(r2+a2),x.z/r);
 float pull=2*mass*r*r2/denominator*(v.x+dot(n,v.yzw));
 return float4(pull-v.x,v.yzw+pull*n);
}

// Static and circular emitters follow frame.go's normalization. Undefined
// stationary orbits propagate NaN; Slayer maps their gains to zero.
static float4 kerr_emitter(float3 x,float mass,float spin,bool circular) {
 float r=kerr_radius(x,spin);
 float omega=circular ? sqrt(mass)/(r*sqrt(r)+spin*sqrt(mass)) : 0;
 float4 v=float4(1,-omega*x.y,omega*x.x,0);
 return v*rsqrt(-dot(kerr_lower(x,v,mass,spin),v));
}

static float4 kerr_axis(float3 at,float4 seed,float4 observer,
 thread float4 *earlier,int count,float mass,float spin) {
 float4 v=seed+observer*dot(kerr_lower(at,seed,mass,spin),observer);
 for (int i=0;i<count;i++) v-=earlier[i]*dot(kerr_lower(at,v,mass,spin),earlier[i]);
 return v*rsqrt(dot(kerr_lower(at,v,mass,spin),v));
}

// Static observer construction for a ray re-entering the field; the first
// camera seed may instead carry a moving observer, constructed by FrameOf.
static float4 kerr_arriving(float3 at,float3 direction,float mass,float spin) {
 float4 observer=kerr_emitter(at,mass,spin,false);
 float4 axes[3];
 axes[0]=kerr_axis(at,float4(0,1,0,0),observer,axes,0,mass,spin);
 axes[1]=kerr_axis(at,float4(0,0,1,0),observer,axes,1,mass,spin);
 axes[2]=kerr_axis(at,float4(0,0,0,1),observer,axes,2,mass,spin);
 float3 d=normalize(direction);
 return -kerr_lower(at,observer-d.x*axes[0]-d.y*axes[1]-d.z*axes[2],mass,spin);
}

// Hamilton's equations from rates.go. Time is component zero; .yzw is space.
static KerrRate kerr_rate_at(float4 x,float4 p,float mass,float spin,float r) {
 float3 at=x.yzw, momentum=p.yzw;
 float r2=r*r, a2=spin*spin;
 float sum=r2+a2, denominator=r2*r2+a2*at.z*at.z;
 if (r==0 || denominator==0) return {float4(-p.x,momentum),float4(0)};
 float overR=1/r, overSum=1/sum, overD=1/denominator;
 float scalar=2*mass*r*r2*overD;
 float3 vector=float3((r*at.x+spin*at.y)*overSum,
                     (r*at.y-spin*at.x)*overSum,at.z*overR);
 float plane=dot(vector.xy,momentum.xy), axis=vector.z*momentum.z;
 float reach=plane+axis-p.x, pull=scalar*reach;
 float4 velocity=float4(pull-p.x,momentum-pull*vector);
 float gradientPlane=r*r2*overD;
 float gradientZ=at.z*r*sum*overD;
 float scalarRadius=(3*a2*at.z*at.z-r2*r2)*overR*overD;
 float scalarAxis=2*a2*at.z*overD;
 float reachRadius=(dot(at.xy,momentum.xy)-2*r*plane)*overSum-axis*overR;
 float along=.5f*reach*scalarRadius+reachRadius;
 float3 change=float3(
  along*at.x*gradientPlane+(r*momentum.x-spin*momentum.y)*overSum,
  along*at.y*gradientPlane+(r*momentum.y+spin*momentum.x)*overSum,
  along*gradientZ+momentum.z*overR-.5f*reach*scalarAxis);
 return {velocity,float4(0,pull*change)};
}
static KerrRate kerr_rate(float4 x,float4 p,float mass,float spin) {
 return kerr_rate_at(x,p,mass,spin,kerr_radius(x.yzw,spin));
}

// One caller-sized affine interval; stopping and adaptive pacing are separate.
static KerrRate kerr_delta_from(float4 x,float4 p,KerrRate a,
 float mass,float spin,float step) {
 KerrRate b=kerr_rate(x+a.x*(step*.5f),p+a.p*(step*.5f),mass,spin);
 KerrRate c=kerr_rate(x+b.x*(step*.5f),p+b.p*(step*.5f),mass,spin);
 KerrRate d=kerr_rate(x+c.x*step,p+c.p*step,mass,spin);
 return {(a.x+2*b.x+2*c.x+d.x)*(step/6),
         (a.p+2*b.p+2*c.p+d.p)*(step/6)};
}
static KerrRate kerr_delta(float4 x,float4 p,float mass,float spin,float step) {
 return kerr_delta_from(x,p,kerr_rate(x,p,mass,spin),mass,spin,step);
}
static void kerr_step(thread float4 &x, thread float4 &p,
 float mass, float spin, float step) {
 KerrRate change=kerr_delta(x,p,mass,spin,step);
 x+=change.x;p+=change.p;
}

// Compensated updates retain sub-ULP increments across long winding paths.
static void kerr_add(thread float4 &sum,thread float4 &error,float4 change) {
 #pragma clang fp reassociate(off)
 float4 corrected=change-error;
 float4 next=sum+corrected;
 error=(next-sum)-corrected;
 sum=next;
}
static void kerr_step_held_from(thread float4 &x,thread float4 &p,
 thread float4 &xError,thread float4 &pError,KerrRate first,
 float mass,float spin,float step) {
 KerrRate change=kerr_delta_from(x,p,first,mass,spin,step);
 kerr_add(x,xError,change.x);kerr_add(p,pError,change.p);
}
static void kerr_step_held(thread float4 &x,thread float4 &p,
 thread float4 &xError,thread float4 &pError,float mass,float spin,float step) {
 kerr_step_held_from(x,p,xError,pError,kerr_rate(x,p,mass,spin),mass,spin,step);
}
