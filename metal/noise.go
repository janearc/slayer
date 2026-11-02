//go:build darwin

package metal

import "encoding/binary"

// NoiseLibrary is slayer's fractal noise (maths/noise) in the Metal shading
// language: the same splitmix64 hash, so a seed gives the same field, and
// Perlin's improved noise over it.
//
// The hash is exact, since Metal has 64-bit integers; the arithmetic is
// 32-bit float where maths/noise's is 64, so the two agree closely and not
// bit for bit. The test measures how closely.
const NoiseLibrary = `
#include <metal_stdlib>
using namespace metal;

constant ulong golden = 0x9e3779b97f4a7c15UL;

static inline ulong mixh(ulong z) {
	z ^= z >> 30;
	z *= 0xbf58476d1ce4e5b9UL;
	z ^= z >> 27;
	z *= 0x94d049bb133111ebUL;
	return z ^ (z >> 31);
}

static inline float fade(float fraction) {
	return fraction * fraction * fraction *
	       (fraction * (fraction * 6 - 15) + 10);
}

static inline float lerpf(float t, float a, float b) { return a + t * (b - a); }

// twelve is a hash % 12 from its two 32-bit halves, since 2^32 % 12 is 4:
// the same remainder, without the 64-bit division the gpu emulates
static inline uint twelve(ulong hashed) {
	uint high = uint(hashed >> 32), low = uint(hashed);
	return ((high % 12) * 4 + low % 12) % 12;
}

static inline float grad(ulong hashed, float x, float y, float z) {
	switch (twelve(hashed)) {
	case 0: return x + y;
	case 1: return -x + y;
	case 2: return x - y;
	case 3: return -x - y;
	case 4: return x + z;
	case 5: return -x + z;
	case 6: return x - z;
	case 7: return -x - z;
	case 8: return y + z;
	case 9: return -y + z;
	case 10: return y - z;
	}
	return -y - z;
}

// gradient is perlin's noise over maths/noise's hash at the cube's eight
// corners. a corner's hash is mixh(seed ^ mixh(x + mixh(y + mixh(z +
// golden)))), as there, so the inner mixes are shared: the z mixes
// are two, the y mixes four, and only the outer ones are eight.
static inline float gradient(ulong seed, float x, float y, float z) {
	float fx = floor(x), fy = floor(y), fz = floor(z);
	long ix = long(fx), iy = long(fy), iz = long(fz);
	x -= fx; y -= fy; z -= fz;
	float u = fade(x), v = fade(y), w = fade(z);
	// near and far along each axis: the cube's two corners on it
	ulong z_near = mixh(ulong(iz) + golden);
	ulong z_far = mixh(ulong(iz + 1) + golden);
	ulong y_near_z_near = mixh(ulong(iy) + z_near);
	ulong y_far_z_near = mixh(ulong(iy + 1) + z_near);
	ulong y_near_z_far = mixh(ulong(iy) + z_far);
	ulong y_far_z_far = mixh(ulong(iy + 1) + z_far);
	ulong x_near = ulong(ix), x_far = ulong(ix + 1);
	ulong corner000 = mixh(seed ^ mixh(x_near + y_near_z_near));
	ulong corner100 = mixh(seed ^ mixh(x_far + y_near_z_near));
	ulong corner010 = mixh(seed ^ mixh(x_near + y_far_z_near));
	ulong corner110 = mixh(seed ^ mixh(x_far + y_far_z_near));
	ulong corner001 = mixh(seed ^ mixh(x_near + y_near_z_far));
	ulong corner101 = mixh(seed ^ mixh(x_far + y_near_z_far));
	ulong corner011 = mixh(seed ^ mixh(x_near + y_far_z_far));
	ulong corner111 = mixh(seed ^ mixh(x_far + y_far_z_far));
	float g000 = grad(corner000, x, y, z);
	float g100 = grad(corner100, x - 1, y, z);
	float g010 = grad(corner010, x, y - 1, z);
	float g110 = grad(corner110, x - 1, y - 1, z);
	float g001 = grad(corner001, x, y, z - 1);
	float g101 = grad(corner101, x - 1, y, z - 1);
	float g011 = grad(corner011, x, y - 1, z - 1);
	float g111 = grad(corner111, x - 1, y - 1, z - 1);
	return lerpf(w,
		lerpf(v, lerpf(u, g000, g100), lerpf(u, g010, g110)),
		lerpf(v, lerpf(u, g001, g101), lerpf(u, g011, g111)));
}

static inline float fractal(ulong seed, float x, float y, float z,
                            int octaves) {
	float sum = 0, amp = 1, norm = 0;
	for (int o = 0; o < octaves; o++) {
		ulong s = seed + ulong(o) * 0x632be59bd9b4e019UL;
		sum += amp * gradient(s, x, y, z);
		norm += amp;
		amp *= 0.5;
		x *= 2.03; y *= 2.03; z *= 2.03;
	}
	return norm == 0 ? 0 : sum / norm;
}

struct params {
	ulong seed;
	int octaves;
	int pad;
};

`

// noiseSource is the library with a kernel that asks it at points.
const noiseSource = NoiseLibrary + `
kernel void fractal_points(device const packed_float3 *points [[buffer(0)]],
                           device float *out [[buffer(1)]],
                           constant params &p [[buffer(2)]],
                           uint id [[thread_position_in_grid]]) {
	float3 q = float3(points[id]);
	out[id] = fractal(p.seed, q.x, q.y, q.z, p.octaves);
}
`

// Noise is slayer's fractal noise (maths/noise) on the GPU.
type Noise struct{ k *Kernel }

// OpenNoise compiles the noise kernel.
func OpenNoise() (*Noise, error) {
	k, err := Open(noiseSource, "fractal_points")
	if err != nil {
		return nil, err
	}
	return &Noise{k: k}, nil
}

// Device is the GPU the kernel runs on.
func (n *Noise) Device() string { return n.k.Device() }

// Fractal is noise.Fractal at every point, all at once.
func (n *Noise) Fractal(seed uint64, octaves int,
	points [][3]float32) ([]float32, error) {
	asked := make([]byte, 16)
	binary.LittleEndian.PutUint64(asked[0:], seed)
	binary.LittleEndian.PutUint32(asked[8:], uint32(int32(octaves)))
	return n.k.Points(points, asked)
}

// Close lets the kernel go.
func (n *Noise) Close() { n.k.Close() }
