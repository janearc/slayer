#include <metal_stdlib>
using namespace metal;

// The host sends four little-endian words for each parameter block.
struct SpectralFFTPermutation {
    uint width;
    uint height;
    uint widthBits;
    uint heightBits;
};

struct SpectralFFTStage {
    uint width;
    uint height;
    uint span;
    uint inverse;
};

struct SpectralFFTScale {
    uint width;
    uint height;
    float scale;
    uint padding;
};

// Both axes are permuted before either separable transform runs.
kernel void spectral_fft_permute(
    device const float2 *input [[buffer(0)]],
    device float2 *output [[buffer(1)]],
    constant SpectralFFTPermutation &grid [[buffer(2)]],
    uint2 at [[thread_position_in_grid]]) {
    if (at.x >= grid.width || at.y >= grid.height) {
        return;
    }
    uint x = reverse_bits(at.x) >> (32 - grid.widthBits);
    uint y = reverse_bits(at.y) >> (32 - grid.heightBits);
    output[at.y * grid.width + at.x] = input[y * grid.width + x];
}

// Twiddles hold the negative exponent. Conjugation selects the inverse.
static float2 spectral_fft_product(float2 value, float2 turn,
                                   uint inverse) {
    if (inverse != 0) {
        turn.y = -turn.y;
    }
    return float2(value.x * turn.x - value.y * turn.y,
                  value.x * turn.y + value.y * turn.x);
}

// A thread writes both outputs of one butterfly to a distinct buffer.
kernel void spectral_fft_row(
    device const float2 *input [[buffer(0)]],
    device float2 *output [[buffer(1)]],
    device const float2 *twiddles [[buffer(2)]],
    constant SpectralFFTStage &grid [[buffer(3)]],
    uint2 at [[thread_position_in_grid]]) {
    if (at.x >= grid.width / 2 || at.y >= grid.height) {
        return;
    }
    uint halfSpan = grid.span / 2;
    uint lane = at.x % halfSpan;
    uint first = at.y * grid.width + (at.x / halfSpan) * grid.span + lane;
    uint second = first + halfSpan;
    float2 near = input[first];
    float2 far = spectral_fft_product(input[second],
        twiddles[lane * (grid.width / grid.span)], grid.inverse);
    output[first] = near + far;
    output[second] = near - far;
}

// Column twiddles follow the width/2 row twiddles in the same buffer.
kernel void spectral_fft_column(
    device const float2 *input [[buffer(0)]],
    device float2 *output [[buffer(1)]],
    device const float2 *twiddles [[buffer(2)]],
    constant SpectralFFTStage &grid [[buffer(3)]],
    uint2 at [[thread_position_in_grid]]) {
    if (at.x >= grid.width || at.y >= grid.height / 2) {
        return;
    }
    uint halfSpan = grid.span / 2;
    uint lane = at.y % halfSpan;
    uint first = ((at.y / halfSpan) * grid.span + lane) * grid.width + at.x;
    uint second = first + halfSpan * grid.width;
    uint turn = grid.width / 2 + lane * (grid.height / grid.span);
    float2 near = input[first];
    float2 far = spectral_fft_product(input[second], twiddles[turn],
                                      grid.inverse);
    output[first] = near + far;
    output[second] = near - far;
}

// Each inverse cell is scaled once after both axes have completed.
kernel void spectral_fft_scale(
    device float2 *values [[buffer(0)]],
    constant SpectralFFTScale &grid [[buffer(1)]],
    uint2 at [[thread_position_in_grid]]) {
    if (at.x < grid.width && at.y < grid.height) {
        values[at.y * grid.width + at.x] *= grid.scale;
    }
}
