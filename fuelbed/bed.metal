#include <metal_stdlib>
using namespace metal;
struct BedSpec { uint width,depth; float dt,diffusion,heating,activation,cooling,consumption; };
// Mandel et al., equations (1-2), https://arxiv.org/abs/0709.0086 .
// Double buffering keeps all neighbours on the same time slice.
kernel void fuel_step(device const float2 *in [[buffer(0)]],
 device float2 *out [[buffer(1)]],constant BedSpec &p [[buffer(2)]],
 uint2 at [[thread_position_in_grid]]) {
 if(at.x>=p.width||at.y>=p.depth)return;
 uint x=at.x,z=at.y,i=z*p.width+x;
 float2 c=in[i];
 float l=in[z*p.width+(x>0?x-1:0)].x;
 float r=in[z*p.width+min(x+1,p.width-1)].x;
 float u=in[(z>0?z-1:0)*p.width+x].x;
 float d=in[min(z+1,p.depth-1)*p.width+x].x;
 float reaction=c.x>0?c.y*exp(-p.activation/c.x):0;
 float burned=p.dt*p.consumption*reaction;
 float heat=p.diffusion*(l+r+u+d-4*c.x)-p.heating*p.cooling*c.x;
 out[i]=float2(c.x+p.dt*heat+p.heating/p.consumption*burned,c.y-burned);
}
