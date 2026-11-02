// Metal implementations of the corresponding Go weather functions.
// Calculations use a unit-radius sphere to keep float32 magnitudes near
// one. weather.go supplies the reference; gpu_test.go compares each stage.

#include <metal_stdlib>
using namespace metal;

// sky contains grid dimensions, planetary spin, atmospheric time scales,
// hyperviscosity and hyperdiffusion on the unit sphere, plus the current
// step and Runge–Kutta weights.
struct sky {
	uint across;
	uint rows;
	uint bits;
	uint unused;
	float limit;
	float rise;
	float run;
	float spin;
	float drag;
	float relax;
	float viscosity;
	float diffusion;
	float step;
	float keep;
	float weigh;
	float filtered;
	float upwelling;
	float fresh;
};

static float latitude(constant sky &s, int j) {
	return -s.limit + (float(j) + 0.5f) * s.rise;
}

static uint at(constant sky &s, int i, int j) {
	int n = int(s.across);
	i %= n;
	if (i < 0) {
		i += n;
	}
	return uint(j) * s.across + uint(i);
}

// walled: past a wall, the negative of the row inside, as Grid.walled
static float walled(constant sky &s, device const float *f, int i, int j) {
	if (j < 0) {
		return -f[at(s, i, -1 - j)];
	}
	if (j >= int(s.rows)) {
		return -f[at(s, i, 2 * int(s.rows) - 1 - j)];
	}
	return f[at(s, i, j)];
}

// mirrored: past a wall, the row inside, as Grid.mirrored
static float mirrored(constant sky &s, device const float *f, int i, int j) {
	if (j < 0) {
		return f[at(s, i, -1 - j)];
	}
	if (j >= int(s.rows)) {
		return f[at(s, i, 2 * int(s.rows) - 1 - j)];
	}
	return f[at(s, i, j)];
}

// laplace is Grid.laplacian, on a sphere of radius one, of a mirrored
// field
static float laplace(constant sky &s, device const float *f, int i, int j) {
	float up = mirrored(s, f, i, j + 1), down = mirrored(s, f, i, j - 1);
	float here = f[at(s, i, j)];
	float north = cos(latitude(s, j) + s.rise / 2);
	float south = cos(latitude(s, j) - s.rise / 2);
	float middle = cos(latitude(s, j));
	float across = (north * (up - here) - south * (here - down)) /
	               (middle * s.rise * s.rise);
	float round = (f[at(s, i + 1, j)] - 2 * here + f[at(s, i - 1, j)]) /
	              (middle * middle * s.run * s.run);
	return across + round;
}

kernel void laplacian(device const float *in [[buffer(0)]],
                      device float *out [[buffer(1)]],
                      constant sky &s [[buffer(2)]],
                      uint2 cell [[thread_position_in_grid]]) {
	out[at(s, cell.x, cell.y)] = laplace(s, in, cell.x, cell.y);
}

// carried is Grid.jacobian of a walled stream function and a field,
// mirrored, plus an added term by row, on a sphere of radius one
static float carried(constant sky &s, device const float *psi,
                     device const float *q, float spun, int i, int j) {
	#define P(a, b) walled(s, psi, i + (a), j + (b))
	#define Q(a, b) (mirrored(s, q, i + (a), j + (b)) + \
	                 spun * sin(latitude(s, j + (b))))
	float plain = (P(1, 0) - P(-1, 0)) * (Q(0, 1) - Q(0, -1)) -
	              (P(0, 1) - P(0, -1)) * (Q(1, 0) - Q(-1, 0));
	float bent = P(1, 0) * (Q(1, 1) - Q(1, -1)) -
	             P(-1, 0) * (Q(-1, 1) - Q(-1, -1)) -
	             P(0, 1) * (Q(1, 1) - Q(-1, 1)) +
	             P(0, -1) * (Q(1, -1) - Q(-1, -1));
	float bending = Q(0, 1) * (P(1, 1) - P(-1, 1)) -
	                Q(0, -1) * (P(1, -1) - P(-1, -1)) -
	                Q(1, 0) * (P(1, 1) - P(1, -1)) +
	                Q(-1, 0) * (P(-1, 1) - P(-1, -1));
	#undef P
	#undef Q
	return (plain + bent + bending) /
	       (12 * s.run * s.rise * cos(latitude(s, j)));
}

// freshening is Weather.freshening
static float freshening(constant sky &s, float zeta, int j) {
	float cyclonic = zeta / (2 * s.spin);
	if (latitude(s, j) < 0) {
		cyclonic = -cyclonic;
	}
	return clamp(-cyclonic * s.upwelling, 0.0f, 1.0f) / s.fresh;
}

// means is Grid.means, a row to a group, its sum halved down in the
// group's memory
kernel void means(device const float *field [[buffer(0)]],
                  device float *out [[buffer(1)]],
                  constant sky &s [[buffer(2)]],
                  uint row [[threadgroup_position_in_grid]],
                  uint t [[thread_position_in_threadgroup]]) {
	threadgroup float held[1024];
	uint n = s.across;
	held[t] = field[row * n + t] + field[row * n + t + n / 2];
	threadgroup_barrier(mem_flags::mem_threadgroup);
	for (uint span = n / 4; span > 0; span >>= 1) {
		if (t < span) {
			held[t] += held[t + span];
		}
		threadgroup_barrier(mem_flags::mem_threadgroup);
	}
	if (t == 0) {
		out[row] = held[0] / float(n);
	}
}

// tendency is Weather.changing, cell by cell, given the stream function,
// the laplacians it smooths by, and the vorticity's means by row
kernel void tendency(device const float *psi [[buffer(0)]],
                     device const float *zeta [[buffer(1)]],
                     device const float *origin [[buffer(2)]],
                     device const float *age [[buffer(3)]],
                     device const float *thrice [[buffer(4)]],
                     device const float *once [[buffer(5)]],
                     device const float *older [[buffer(6)]],
                     device const float *mean [[buffer(7)]],
                     device float *spun [[buffer(8)]],
                     device float *drawn [[buffer(9)]],
                     device float *aged [[buffer(10)]],
                     constant sky &s [[buffer(11)]],
                     uint2 cell [[thread_position_in_grid]]) {
	int i = cell.x, j = cell.y;
	uint k = at(s, i, j);
	spun[k] = -carried(s, psi, zeta, 2 * s.spin, i, j) - zeta[k] / s.drag -
	          s.viscosity * laplace(s, thrice, i, j);
	drawn[k] = -carried(s, psi, origin, 0, i, j) -
	           (origin[k] - latitude(s, j)) / s.relax -
	           s.diffusion * laplace(s, once, i, j);
	aged[k] = -carried(s, psi, age, 0, i, j) + 1 -
	          age[k] * freshening(s, mean[j], j) -
	          s.diffusion * laplace(s, older, i, j);
}

// combine is one stage of Weather.step: keep of the step's start and
// weigh of the stage carried on by its change
kernel void combine(device const float *start [[buffer(0)]],
                    device const float *stage [[buffer(1)]],
                    device const float *change [[buffer(2)]],
                    device float *out [[buffer(3)]],
                    constant sky &s [[buffer(4)]],
                    uint k [[thread_position_in_grid]]) {
	out[k] = s.keep * start[k] + s.weigh * (stage[k] + s.step * change[k]);
}

// transform is fft over a row held in the group's memory, n of 2^bits,
// by n/2 threads, each one butterfly a stage
static void transform(threadgroup float2 *a, uint n, uint bits, uint t,
                      float sign) {
	for (uint k = t; k < n; k += n / 2) {
		uint r = reverse_bits(k) >> (32 - bits);
		if (k < r) {
			float2 held = a[k];
			a[k] = a[r];
			a[r] = held;
		}
	}
	threadgroup_barrier(mem_flags::mem_threadgroup);
	for (uint size = 2; size <= n; size <<= 1) {
		uint span = size / 2;
		uint low = (t / span) * size + t % span, high = low + span;
		float turn = sign * 2 * M_PI_F * float(t % span) / float(size);
		float2 w = float2(precise::cos(turn), precise::sin(turn));
		float2 b = a[high], x = a[low];
		float2 wb = float2(b.x * w.x - b.y * w.y, b.x * w.y + b.y * w.x);
		a[low] = x + wb;
		a[high] = x - wb;
		threadgroup_barrier(mem_flags::mem_threadgroup);
	}
}

// forward is each row of a field turned into its wavenumbers, a row to
// a group, as streamOf begins
kernel void forward(device const float *in [[buffer(0)]],
                    device float2 *modes [[buffer(1)]],
                    constant sky &s [[buffer(2)]],
                    uint row [[threadgroup_position_in_grid]],
                    uint t [[thread_position_in_threadgroup]]) {
	threadgroup float2 a[2048];
	uint n = s.across;
	a[t] = float2(in[row * n + t], 0);
	a[t + n / 2] = float2(in[row * n + t + n / 2], 0);
	threadgroup_barrier(mem_flags::mem_threadgroup);
	transform(a, n, s.bits, t, -1);
	modes[row * n + t] = a[t];
	modes[row * n + t + n / 2] = a[t + n / 2];
}

// back is each row of wavenumbers turned back into its field, scaled,
// as streamOf ends
kernel void back(device const float2 *modes [[buffer(0)]],
                 device float *out [[buffer(1)]],
                 constant sky &s [[buffer(2)]],
                 uint row [[threadgroup_position_in_grid]],
                 uint t [[thread_position_in_threadgroup]]) {
	threadgroup float2 a[2048];
	uint n = s.across;
	a[t] = modes[row * n + t];
	a[t + n / 2] = modes[row * n + t + n / 2];
	threadgroup_barrier(mem_flags::mem_threadgroup);
	transform(a, n, s.bits, t, 1);
	out[row * n + t] = a[t].x / float(n);
	out[row * n + t + n / 2] = a[t + n / 2].x / float(n);
}

// sweep is Grid.sweep and thomas for one wavenumber a thread, its rows
// solved in place
kernel void sweep(device float2 *modes [[buffer(0)]],
                  device float *ahead [[buffer(1)]],
                  constant sky &s [[buffer(2)]],
                  uint k [[thread_position_in_grid]]) {
	uint n = s.across, rows = s.rows;
	int m = int(k) > int(n / 2) ? int(k) - int(n) : int(k);
	float curve = 2 * (1 - precise::cos(float(m) * s.run)) /
	              (s.run * s.run);
	float pivot = 0;
	float below = 0;
	for (uint j = 0; j < rows; j++) {
		float c = cos(latitude(s, j));
		float south = cos(latitude(s, j) - s.rise / 2) /
		              (c * s.rise * s.rise);
		float north = cos(latitude(s, j) + s.rise / 2) /
		              (c * s.rise * s.rise);
		float middle = -south - north - curve / (c * c);
		if (j == 0) {
			middle -= south;
		}
		if (j == rows - 1) {
			middle -= north;
		}
		below = j == 0 ? 0 : south;
		float before = j == 0 ? 0 : ahead[(j - 1) * n + k];
		pivot = middle - below * before;
		ahead[j * n + k] = north / pivot;
		float2 earlier = j == 0 ? float2(0) : modes[(j - 1) * n + k];
		modes[j * n + k] = (modes[j * n + k] - below * earlier) / pivot;
	}
	for (int j = int(rows) - 2; j >= 0; j--) {
		modes[j * n + k] -= ahead[j * n + k] * modes[(j + 1) * n + k];
	}
}

// filter is Weather.filter for a row past filtered, in place, a row to
// a group
kernel void filter(device float *field [[buffer(0)]],
                   constant sky &s [[buffer(1)]],
                   uint row [[threadgroup_position_in_grid]],
                   uint t [[thread_position_in_threadgroup]]) {
	float c = cos(latitude(s, row));
	if (c >= s.filtered) {
		return;
	}
	threadgroup float2 a[2048];
	uint n = s.across;
	a[t] = float2(field[row * n + t], 0);
	a[t + n / 2] = float2(field[row * n + t + n / 2], 0);
	threadgroup_barrier(mem_flags::mem_threadgroup);
	transform(a, n, s.bits, t, -1);
	float most = float(n / 2) * c / s.filtered;
	for (uint k = t; k < n; k += n / 2) {
		int m = int(k) > int(n / 2) ? int(k) - int(n) : int(k);
		if (abs(float(m)) > most) {
			a[k] = float2(0);
		}
	}
	threadgroup_barrier(mem_flags::mem_threadgroup);
	transform(a, n, s.bits, t, 1);
	field[row * n + t] = a[t].x / float(n);
	field[row * n + t + n / 2] = a[t + n / 2].x / float(n);
}
