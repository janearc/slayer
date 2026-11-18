#include <metal_stdlib>
using namespace metal;

struct WaveGrid { uint width; uint height; };

// Real coefficients act independently on both complex components. The two
// outputs share their immutable Cauchy inputs without any in-place overlap.
kernel void wave_combine(
 device const float2 *initialValue [[buffer(0)]],
 device const float2 *initialRate [[buffer(1)]],
 device const float4 *mode [[buffer(2)]],
 device float2 *value [[buffer(3)]],
 device float2 *rate [[buffer(4)]],
 constant WaveGrid &grid [[buffer(5)]],
 uint2 at [[thread_position_in_grid]]) {
 if(at.x>=grid.width || at.y>=grid.height) { return; }
 uint i=at.y*grid.width+at.x;
 float4 factor=mode[i];
 float2 q=initialValue[i],v=initialRate[i];
 value[i]=factor.x*q+factor.y*v;
 rate[i]=factor.z*q+factor.x*v;
}

// Preserve an inverse result before a second inverse reuses shared scratch.
kernel void wave_copy(
 device const float2 *input [[buffer(0)]],
 device float2 *output [[buffer(1)]],
 constant WaveGrid &grid [[buffer(2)]],
 uint2 at [[thread_position_in_grid]]) {
 if(at.x<grid.width && at.y<grid.height) {
  uint i=at.y*grid.width+at.x;output[i]=input[i];
 }
}
