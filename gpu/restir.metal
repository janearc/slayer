//// prelude
// restir's share of trace.metal: doc/restir.md. Everything here is
// compiled only in the restir mode; no other mode sees it.
//
// The rays' input, in the restir mode, is floats: a control block of
// restir_ctrl floats, then the light tree, the G-buffers and the
// reservoirs. gpu/restir.go lays them out; the control block says where.
constexpr constant int restir_ctrl = 64;
constexpr constant int nodeFloatsK = 12;
// the control block holds whole numbers as their bits, which a float
// cannot hold exactly past 2^24, and a layout runs past that
static int rc_int(device const float *rays, int slot) {
	return as_type<int>(rays[slot]);
}
// control block slots
constexpr constant int rc_frame = 0, rc_candidates = 1, rc_temporal = 2,
	rc_spatial = 3, rc_radius = 4, rc_cap = 5, rc_nodes = 6,
	rc_skip_sky = 7, rc_width = 8, rc_height = 9, rc_tree = 10,
	rc_gbuf = 11, rc_prev_gbuf = 12, rc_res_a = 13, rc_res_b = 14,
	rc_prev_res = 15, rc_eye = 16;
constexpr constant int rc_visible = 40;
// restir_pixel is a sample's number among the whole picture's
static int restir_pixel(device const float *rays, uint2 pixel) {
	return int(pixel.y) * rc_int(rays, rc_width) + int(pixel.x);
}

// pcg32's output step: a hash of a 32 bit state to a 32 bit number
static uint restir_hash(uint v) {
	uint state = v * 747796405u + 2891336653u;
	uint word = ((state >> ((state >> 28u) + 4u)) ^ state) * 277803737u;
	return (word >> 22u) ^ word;
}
struct Restir_rng { uint state; };
static float restir_next(thread Restir_rng &rng) {
	rng.state = restir_hash(rng.state);
	return float(rng.state >> 8) * (1.0f / 16777216.0f);
}

// tree_weight is gpu/restir_tree.go's weigh: a node's worth as a proposal
static float tree_weight(device const float *node, float3 point,
                         float3 normal) {
	float3 low = float3(node[0], node[1], node[2]);
	float3 high = float3(node[4], node[5], node[6]);
	bool behind = true;
	for (int corner = 0; corner < 8; corner++) {
		float3 edge = float3((corner & 1) ? high.x : low.x,
		                     (corner & 2) ? high.y : low.y,
		                     (corner & 4) ? high.z : low.z);
		if (dot(normal, edge - point) > 0) { behind = false; break; }
	}
	if (behind) { return 0; }
	float3 gap = max(max(low - point, point - high), float3(0));
	return node[3] / max(dot(gap, gap), 1e-4f);
}

// tree_sample is restir_tree.go's sample: a lamp and its chance, or
// minus one for none
static int tree_sample(device const float *tree, float3 point, float3 normal,
                       float u, thread float &chance) {
	int at = 0;
	chance = 1;
	for (int depth = 0; depth < 64; depth++) {
		device const float *node = tree + at * nodeFloatsK;
		int lamp = int(node[9]);
		if (lamp >= 0) { return lamp; }
		int left = int(node[7]), right = int(node[8]);
		float wl = tree_weight(tree + left * nodeFloatsK, point, normal);
		float wr = tree_weight(tree + right * nodeFloatsK, point, normal);
		float total = wl + wr;
		if (!(total > 0)) { chance = 0; return -1; }
		float share = wl / total;
		if (u < share) { at = left; chance *= share; u = u / share; }
		else { at = right; chance *= 1 - share; u = (u - share) / (1 - share); }
	}
	chance = 0;
	return -1;
}

// restir_phat is the target: the lamp's light at the point, times the
// cosine, as its luminance, without a shadow
static float restir_phat(World world, int lamp, float3 point,
                         float3 normal) {
	if (lamp < 0) { return 0; }
	Reached reached = lamp_light(world.lamps[lamp], point);
	if (all(reached.toward == 0)) { return 0; }
	return luminance(reached.light) * max(0.0f, dot(normal, reached.toward));
}

// falling_restir is falling for a primary hit, lit by its reservoir's lamp
// with a real shadow ray, W over pi times what arrives
static float3 falling_restir(World world, float3 point, float3 normal,
                             int leaving_sphere) {
	float3 sum = float3(0);
	if (rc_int(world.rays, rc_skip_sky) == 0) {
		sum = skylit(world, point, normal, leaving_sphere);
	}
	device const float *r = world.rays + world.restir_at;
	int index = int(r[0]);
	float W = r[1];
	if (index < 0 || !(W > 0)) { return sum; }
	float3 from = lifted(world, point, normal);
	Lamp lamp = world.lamps[index];
	Reached reached = lamp_light(lamp, point);
	float square = dot(normal, reached.toward);
	if (square <= 0) { return sum; }
	float3 reach = square * unhidden(world, lamp, from, reached.toward,
	                                 reached.far, leaving_sphere);
	return sum + reached.light * reach * (W / M_PI_F);
}
//// kernels
// restir_view is where a world point is in the last frame's picture,
// by the eye the control block keeps: the inverse of look
static bool restir_view(device const float *rays, float3 point,
                        thread int2 &pixel) {
	device const float *e = rays + rc_eye;
	float3 origin = float3(e[0], e[1], e[2]);
	float3 right = float3(e[3], e[4], e[5]);
	float3 up = float3(e[6], e[7], e[8]);
	float3 back = float3(e[9], e[10], e[11]);
	float4 lens = float4(e[12], e[13], e[14], e[15]);
	float3 v = point - origin;
	float depth = -dot(v, back);
	if (!(depth > 0)) { return false; }
	float across = dot(v, right) / depth, rise = dot(v, up) / depth;
	float x = (across * lens.z / lens.x + lens.z - 1) * 0.5f;
	float y = (lens.w - 1 - rise * lens.w * lens.y / lens.x) * 0.5f;
	pixel = int2(floor(x + 0.5f), floor(y + 0.5f));
	return pixel.x >= 0 && pixel.y >= 0 && pixel.x < int(lens.z) &&
	       pixel.y < int(lens.w);
}

// alike is whether another sample's G-buffer is of the same surface
static bool restir_alike(device const float *mine, device const float *theirs) {
	if (!(theirs[7] > 0) || !(mine[7] > 0)) { return false; }
	if (abs(theirs[7] - mine[7]) > 0.1f * mine[7]) { return false; }
	return dot(float3(mine[3], mine[4], mine[5]),
	           float3(theirs[3], theirs[4], theirs[5])) > 0.9f;
}

// restir_gbuffer is the primary hit of a sample, as shade finds it: the
// point lit from (lifted by its hit's own lift), its normal, the sphere it
// leaves, and its depth, which is nought for a ray that hits nothing
kernel void restir_gbuffer(SLAYER_TRACE_ARGUMENTS) {
	World world = SLAYER_WORLD_INITIALIZER;
	uint4 native_counts(0);
	world.native_counts = &native_counts;
	uint2 pixel = uint2(at.x, at.y + origin.y);
	float3 from = header->origin.xyz, way = look(header, pixel);
	Hit hit = first(world, from, way, 0);
	device float *g = out + (at.y * grid.x + at.x) * 8;
	if (hit.material < 0) {
		for (int i = 0; i < 8; i++) { g[i] = 0; }
		return;
	}
	float3 point = fma(way, hit.distance, from);
	Material made = world.materials[hit.material];
	float3 normal = crinkled(made, point, facing(hit, way), -way,
		pixel_at(world, hit.distance * length(way)));
	float3 lit = point + hit.lift;
	g[0] = lit.x, g[1] = lit.y, g[2] = lit.z;
	g[3] = normal.x, g[4] = normal.y, g[5] = normal.z;
	g[6] = float(hit.sphere_index);
	g[7] = hit.distance;
}

// restir_write puts a reservoir where a pass keeps its output
static void restir_write(device float *r, int lamp, float W, float M) {
	r[0] = float(lamp), r[1] = W, r[2] = M, r[3] = 0;
}

// restir_resample: initial candidates from the light tree, one shadow ray
// on the one kept, then the last frame's reservoir joined
kernel void restir_resample(SLAYER_TRACE_ARGUMENTS) {
	World world = SLAYER_WORLD_INITIALIZER;
	uint4 native_counts(0);
	world.native_counts = &native_counts;
	uint2 pixel = uint2(at.x, at.y + origin.y);
	int n = restir_pixel(rays, pixel);
	device float *result = out + (at.y * grid.x + at.x) * 4;
	device const float *g = rays + rc_int(rays, rc_gbuf) + n * 8;
	if (!(g[7] > 0)) { restir_write(result, -1, 0, 0); return; }
	float3 point = float3(g[0], g[1], g[2]), normal = float3(g[3], g[4], g[5]);
	Restir_rng rng = {restir_hash(uint(n) * 9781u + uint(rc_int(rays, rc_frame)) * 6271u + 1u)};
	device const float *tree = rays + rc_int(rays, rc_tree);
	int candidates = rc_int(rays, rc_candidates);
	int kept = -1;
	float wsum = 0, M = 0, keptphat = 0;
	for (int i = 0; i < candidates; i++) {
		float chance;
		int lamp = tree_sample(tree, point, normal, restir_next(rng), chance);
		M += 1;
		if (lamp < 0) { continue; }
		float phat = restir_phat(world, lamp, point, normal);
		float weight = phat / chance;
		wsum += weight;
		if (weight > 0 && restir_next(rng) * wsum < weight) {
			kept = lamp, keptphat = phat;
		}
	}
	// the one kept is shot at once; hidden, it is nothing, as the paper's
	if (kept >= 0 && rc_int(rays, rc_visible) != 0) {
		Reached reached = lamp_light(world.lamps[kept], point);
		float3 from = lifted(world, point, normal);
		if (blocked(world, from, reached.toward, reached.far, int(g[6]))) {
			kept = -1, wsum = 0;
		}
	}
	float W = (kept >= 0 && keptphat > 0 && M > 0) ? wsum / (M * keptphat) : 0;
	if (rc_int(rays, rc_temporal) != 0 && rc_int(rays, rc_frame) > 0) {
		int2 was;
		if (restir_view(rays, point, was)) {
			int m = was.y * rc_int(rays, rc_width) + was.x;
			device const float *old = rays + rc_int(rays, rc_prev_gbuf) + m * 8;
			if (restir_alike(g, old)) {
				device const float *r = rays + rc_int(rays, rc_prev_res) + m * 4;
				int lamp = int(r[0]);
				float theirs = r[2];
				float limit = rays[rc_cap] * max(M, 1.0f);
				theirs = min(theirs, limit);
				if (lamp >= 0 && r[1] > 0) {
					float weight = restir_phat(world, lamp, point, normal) * r[1] * theirs;
					wsum += weight;
					if (weight > 0 && restir_next(rng) * wsum < weight) { kept = lamp; }
				}
				M += theirs;
				keptphat = restir_phat(world, kept, point, normal);
				W = (kept >= 0 && keptphat > 0 && M > 0) ? wsum / (M * keptphat) : 0;
			}
		}
	}
	restir_write(result, W > 0 ? kept : -1, W, M);
}

// restir_spatial: neighbours' reservoirs from restir_resample joined
kernel void restir_spatial(SLAYER_TRACE_ARGUMENTS) {
	World world = SLAYER_WORLD_INITIALIZER;
	uint4 native_counts(0);
	world.native_counts = &native_counts;
	uint2 pixel = uint2(at.x, at.y + origin.y);
	int n = restir_pixel(rays, pixel);
	int width = rc_int(rays, rc_width), height = rc_int(rays, rc_height);
	device float *result = out + (at.y * grid.x + at.x) * 4;
	device const float *g = rays + rc_int(rays, rc_gbuf) + n * 8;
	device const float *mine = rays + rc_int(rays, rc_res_a) + n * 4;
	if (!(g[7] > 0)) { restir_write(result, -1, 0, 0); return; }
	float3 point = float3(g[0], g[1], g[2]), normal = float3(g[3], g[4], g[5]);
	int kept = int(mine[0]);
	float wsum = kept >= 0 ? restir_phat(world, kept, point, normal) * mine[1] * mine[2] : 0;
	float M = mine[2];
	Restir_rng rng = {restir_hash(uint(n) * 12289u + uint(rc_int(rays, rc_frame)) * 3079u + 7u)};
	int taps = rc_int(rays, rc_spatial);
	float radius = rays[rc_radius];
	float own = max(M, 1.0f);
	for (int tap = 0; tap < taps; tap++) {
		float angle = 6.2831853f * restir_next(rng);
		float reach = radius * sqrt(restir_next(rng));
		int2 other = int2(int(pixel.x) + int(round(reach * cos(angle))),
		                  int(pixel.y) + int(round(reach * sin(angle))));
		if (other.x < 0 || other.y < 0 || other.x >= width || other.y >= height ||
		    (other.x == int(pixel.x) && other.y == int(pixel.y))) { continue; }
		int m = other.y * width + other.x;
		device const float *theirs_g = rays + rc_int(rays, rc_gbuf) + m * 8;
		if (!restir_alike(g, theirs_g)) { continue; }
		device const float *r = rays + rc_int(rays, rc_res_a) + m * 4;
		int lamp = int(r[0]);
		float count = min(r[2], rays[rc_cap] * own);
		if (lamp >= 0 && r[1] > 0) {
			float weight = restir_phat(world, lamp, point, normal) * r[1] * count;
			wsum += weight;
			if (weight > 0 && restir_next(rng) * wsum < weight) { kept = lamp; }
		}
		M += count;
	}
	float phat = restir_phat(world, kept, point, normal);
	float W = (kept >= 0 && phat > 0 && M > 0) ? wsum / (M * phat) : 0;
	restir_write(result, W > 0 ? kept : -1, W, M);
}
