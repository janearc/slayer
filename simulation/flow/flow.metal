#include <metal_stdlib>
using namespace metal;

struct FlowStep { uint width; uint height; float dt; uint stage; };

// Every buffer is row-major, complex interleaved float2. Real-space
// products explicitly take real parts after an inverse transform.
kernel void flow_copy(
 device const float2 *input [[buffer(0)]],
 device float2 *output [[buffer(1)]],
 constant FlowStep &grid [[buffer(2)]],
 uint2 at [[thread_position_in_grid]]) {
 if(at.x<grid.width && at.y<grid.height) {
  uint i=at.y*grid.width+at.x;output[i]=input[i];
 }
}

// Coefficients store physical kx, ky, viscous decay and the retained mask.
kernel void flow_curl(
 device const float2 *u [[buffer(0)]],
 device const float2 *v [[buffer(1)]],
 device const float4 *coefficient [[buffer(2)]],
 device float2 *omega [[buffer(3)]],
 constant FlowStep &grid [[buffer(4)]],
 uint2 at [[thread_position_in_grid]]) {
 if(at.x>=grid.width || at.y>=grid.height) { return; }
 uint i=at.y*grid.width+at.x;float4 k=coefficient[i];
 float2 derivative=k.x*v[i]-k.y*u[i];
 omega[i]=k.w*float2(-derivative.y,derivative.x);
}

kernel void flow_rotational(
 device const float2 *u [[buffer(0)]],
 device const float2 *v [[buffer(1)]],
 device const float2 *omega [[buffer(2)]],
 device float2 *rhsU [[buffer(3)]],
 device float2 *rhsV [[buffer(4)]],
 constant FlowStep &grid [[buffer(5)]],
 uint2 at [[thread_position_in_grid]]) {
 if(at.x>=grid.width || at.y>=grid.height) { return; }
 uint i=at.y*grid.width+at.x;
 rhsU[i]=float2(v[i].x*omega[i].x,0);
 rhsV[i]=float2(-u[i].x*omega[i].x,0);
}

kernel void flow_project_rhs(
 device const float2 *u [[buffer(0)]],
 device const float2 *v [[buffer(1)]],
 device const float4 *coefficient [[buffer(2)]],
 device float2 *rhsU [[buffer(3)]],
 device float2 *rhsV [[buffer(4)]],
 constant FlowStep &grid [[buffer(5)]],
 uint2 at [[thread_position_in_grid]]) {
 if(at.x>=grid.width || at.y>=grid.height) { return; }
 uint i=at.y*grid.width+at.x;float4 k=coefficient[i];
 if(i==0 || k.w==0) { rhsU[i]=0;rhsV[i]=0;return; }
 float2 a=rhsU[i]-k.z*u[i],b=rhsV[i]-k.z*v[i];
 float2 longitudinal=(k.x*a+k.y*b)/(k.x*k.x+k.y*k.y);
 rhsU[i]=a-k.x*longitudinal;rhsV[i]=b-k.y*longitudinal;
}

// RK4 weights are 1,2,2,1. Each dispatch reads one completed RHS before
// replacing the stage. The original accepted state remains untouched.
kernel void flow_rk(
 device const float2 *baseU [[buffer(0)]],
 device const float2 *baseV [[buffer(1)]],
 device const float2 *rhsU [[buffer(2)]],
 device const float2 *rhsV [[buffer(3)]],
 device float2 *sumU [[buffer(4)]],
 device float2 *sumV [[buffer(5)]],
 device float2 *nextU [[buffer(6)]],
 device float2 *nextV [[buffer(7)]],
 constant FlowStep &grid [[buffer(8)]],
 uint2 at [[thread_position_in_grid]]) {
 if(at.x>=grid.width || at.y>=grid.height) { return; }
 uint i=at.y*grid.width+at.x;
 if(grid.stage==0) {
  sumU[i]=rhsU[i];sumV[i]=rhsV[i];
  nextU[i]=baseU[i]+(grid.dt*0.5f)*rhsU[i];
  nextV[i]=baseV[i]+(grid.dt*0.5f)*rhsV[i];
 } else if(grid.stage<3) {
  sumU[i]+=2.0f*rhsU[i];sumV[i]+=2.0f*rhsV[i];
  float fraction=grid.stage==1 ? 0.5f : 1.0f;
  nextU[i]=baseU[i]+(grid.dt*fraction)*rhsU[i];
  nextV[i]=baseV[i]+(grid.dt*fraction)*rhsV[i];
 } else {
  nextU[i]=baseU[i]+(grid.dt/6.0f)*(sumU[i]+rhsU[i]);
  nextV[i]=baseV[i]+(grid.dt/6.0f)*(sumV[i]+rhsV[i]);
 }
}
