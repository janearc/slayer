#ifdef SLAYER_NATIVE_SHADOWS
#include <metal_raytracing>
using namespace metal::raytracing;
#endif

// Metal tracing kernels corresponding to the CPU renderer. Functions retain
// related names so the implementations can be compared, using float where
// the CPU uses double. Go-side validation rejects unsupported scene features.
//
// Buffer structs such as Header, Sphere, and Tree are declared once in Go,
// beside their encoders. prelude() generates their Metal declarations before
// this source, keeping host and device layouts in sync.

static float3 field_in(constant float4 *f,float3 p) {
 float3 d=(p-f[1].xyz)/f[0].w;
 return float3(dot(d,f[2].xyz),-dot(d,f[4].xyz),dot(d,f[3].xyz));
}

// Device-side scene buffers and shared parameters.
struct World {
	constant Header *header;
	device const Sphere *spheres;
	device const Triangle *triangles;
	device const Node *nodes;
	device const Material *materials;
	device const Lamp *lamps;
	device const Volume *volumes;
	device const float *baked;
	device const Tree *trees;
	device const Instance *instances;
	device const Node *tops;
	device const float4 *sky;
	device const Bend *bends;
	device const float *rays;
	device const float *boxes;
	device const Field *fields;
	device const float *heights;
#ifdef SLAYER_NATIVE_MESHES
	device const int *firsts;
#endif
#ifdef SLAYER_NATIVE_SHADOWS
	acceleration_structure<instancing> accelerated;
#endif
	thread uint4 *native_counts;
	// shared is fast light's share of the sky's looks for this sample:
	// the first look it takes, how far apart its looks are, and how many
	// it takes. Nought looks, as every initializer leaves it, is them all.
	int3 shared;
    constant float4 *gravity;
    float4 momentum;
    float energy;
    int cuts;
};

// nowhere is the finite miss-depth sentinel shared through structure.go.
// Fast-math compilation does not give reliable infinity semantics, so the
// CPU converts this sentinel to trace’s +Inf after readback.

// A hit records distance, material, and surface normal. lift offsets smooth
// mesh hits toward the reconstructed curved surface for lighting; other
// shapes use zero. local and scale retain the object-space texture position
// and footprint information for CPU material evaluation.
struct Hit {
	float distance;
	int material;
	float3 normal;
	float3 lift;
	float3 local;
	float scale;
	// Internal hit provenance, not part of a host/device buffer layout.
	int sphere_index = -1;
};

// sphere_hit uses the stable quadratic form shared with globe.Hit and
// shape.Sphere.Hit: return the nearest root within near and far, or a
// negative miss distance. Roots are q and c/q; c is formed from the origin’s
// height above the surface and its distance from the opposite side.
//
// The discriminant uses radius and closest approach instead of b*b-c to
// avoid cancellation for distant spheres. With float32, the direct form can
// shift limb hits by the same order as the ray-origin offset. See Haines
// et al., “Precision Improvements for Ray/Sphere Intersection,” Ray Tracing
// Gems, chapter 7 (2019).
static float sphere_hit(float3 at, float3 way, Sphere sphere, float near,
                        float far) {
	if (sphere.info.z == 1) {
		float facing = dot(way, sphere.centre.xyz);
		if (facing == 0) return -1;
		float distance = (sphere.centre.w - dot(at, sphere.centre.xyz)) / facing;
		return distance > near && distance < far ? distance : -1;
	}
	float3 from = at - sphere.centre.xyz;
	float radius = sphere.centre.w;
	float along = dot(from, way);
	float away = length(from);
	float gap = (away - radius) * (away + radius);
	float closest = length(from - along * way);
	float disc = (radius - closest) * (radius + closest);
	if (disc < 0) {
		return -1;
	}
	float q = -(along + copysign(sqrt(disc), along));
	float first = q, second = q != 0 ? gap / q : 0;
	if (first > second) {
		float swap = first;
		first = second, second = swap;
	}
	if (first > near && first < far) {
		return first;
	}
	if (second > near && second < far) {
		return second;
	}
	return -1;
}

// edge_area uses the same compensated product difference as mesh/meet.go.
// Explicit fused operations retain the residual of the rounded product.
static float edge_area(float2 one, float2 two) {
	float rounded = one.y * two.x;
	float error = fma(-one.y, two.x, rounded);
	float main = fma(one.x, two.y, -rounded);
	return main + error;
}

// ray_corner translates and shears a vertex into the dominant ray axis.
// Shared vertices take identical arithmetic in adjoining triangles.
static float3 ray_corner(float3 corner, float3 at, int x, int y, int z,
                         float3 shear) {
	float3 local = corner - at;
	return float3(fma(-shear.x, local[z], local[x]),
	              fma(-shear.y, local[z], local[y]), local[z] * shear.z);
}

// triangle_met follows Woop, Benthin and Wald's ray-aligned triangle solve
// (JCGT 2013, https://jcgt.org/published/0002/01/05/), as the CPU mesh path does.
// It avoids the poorly conditioned distance quotient of long, thin faces.
// This float-only variant retains the existing grazing rejection and edge
// slack; it does not claim the paper's double-fallback watertight guarantee.
static float triangle_met(float3 at, float3 way, Triangle face, float near,
                          float far, thread float2 &weights) {
	float3 first = face.corner_b.xyz - face.corner_a.xyz;
	float3 second = face.corner_c.xyz - face.corner_a.xyz;
	float det = dot(first, cross(way, second));
	if (abs(det) <= 1e-7f * length(first) * length(second) * length(way)) {
		return -1;
	}
	int z = 0;
	if (abs(way.y) > abs(way[z])) { z = 1; }
	if (abs(way.z) > abs(way[z])) { z = 2; }
	int x = (z + 1) % 3, y = (x + 1) % 3;
	if (way[z] < 0) {
		int swap = x; x = y; y = swap;
	}
	float3 shear = float3(way[x] / way[z], way[y] / way[z], 1 / way[z]);
	float3 a = ray_corner(face.corner_a.xyz, at, x, y, z, shear);
	float3 b = ray_corner(face.corner_b.xyz, at, x, y, z, shear);
	float3 c = ray_corner(face.corner_c.xyz, at, x, y, z, shear);
	float for_a = edge_area(c.xy, b.xy);
	float for_b = edge_area(a.xy, c.xy);
	float for_c = edge_area(b.xy, a.xy);
	float whole = for_a + for_b + for_c;
	if (whole == 0) { return -1; }
	float toward_first = for_b / whole;
	float toward_second = for_c / whole;
	float slack = 1e-6f;
	if (toward_first < -slack || toward_second < -slack ||
	    toward_first + toward_second > 1 + slack) {
		return -1;
	}
	float distance = fma(for_a, a.z, fma(for_b, b.z, for_c * c.z)) / whole;
	weights = float2(toward_first, toward_second);
	return distance > near && distance < far ? distance : -1;
}

static float triangle_hit(float3 at, float3 way, Triangle face, float near,
                          float far) {
	float2 weights;
	return triangle_met(at, way, face, near, far, weights);
}

// bent interpolates vertex normals using barycentric weights, matching the
// smooth mesh path. A zero blend falls back to the flat normal. Hanika’s
// shadow-terminator offset raises the hit toward each vertex tangent plane,
// weighted by that vertex’s contribution: for slayer's own mesh, as its
// tracer lifts one. A shape.Mesh shaded smooth is bent and not lifted, as
// its tracer shades it, and says so in its first normal's spare number.
static Hit bent(Bend bend, Triangle face, float3 point, float2 weights,
                Hit flat) {
	float3 weighed = float3(1 - weights.x - weights.y, weights);
	float3 normals[3] = {bend.normal_a.xyz, bend.normal_b.xyz,
	                     bend.normal_c.xyz};
	float3 corners[3] = {face.corner_a.xyz, face.corner_b.xyz,
	                     face.corner_c.xyz};
	float3 blend = float3(0), lift = float3(0);
	for (int k = 0; k < 3; k++) {
		blend += normals[k] * weighed[k];
		float under = min(0.0f, dot(point - corners[k], normals[k]));
		lift -= normals[k] * (weighed[k] * under);
	}
	if (length(blend) > 0) {
		flat.normal = normalize(blend);
	}
	flat.lift = bend.normal_a.w == 0 ? lift : float3(0);
	return flat;
}

// ring returns one base-ring corner from its index and the side count.
static float4 ring(int side, int sides, float radius) {
	float turn = 2 * M_PI_F * float(side) / float(sides);
	return float4(cos(turn) * radius, sin(turn) * radius, 0, 0);
}

// cone_hit intersects the cone generated by manufacture.Cone in local z-up
// coordinates. Each side is a triangle from the base edge to the apex.
// It returns the nearest distance and normal, or a negative miss distance.
// Triangles are generated during traversal rather than stored.
static Hit cone_hit(float3 at, float3 way, float height, float radius,
                    int sides, float near, float far) {
	Triangle face;
	face.corner_c = float4(0, 0, height, 0);
	Hit found = {-1, -1, float3(0)};
	int nearest = 0;
	for (int side = 0; side < sides; side++) {
		face.corner_a = ring(side, sides, radius);
		face.corner_b = ring(side + 1, sides, radius);
		float distance = triangle_hit(at, way, face, near, far);
		if (distance >= 0) {
			found.distance = distance;
			nearest = side;
			far = distance;
		}
	}
	if (found.distance >= 0) {
		float3 corner = ring(nearest, sides, radius).xyz;
		float3 next = ring(nearest + 1, sides, radius).xyz;
		found.normal = normalize(cross(next - corner,
		                               float3(0, 0, height) - corner));
	}
	return found;
}

// overlaps tests the branch bounding sphere between near and far,
// allowing an entire conifer branch to be culled at once.
static bool overlaps(float3 at, float3 way, float3 centre, float radius,
                     float near, float far) {
	float3 from = at - centre;
	float closest = -dot(from, way);
	float disc = closest * closest - (dot(from, from) - radius * radius);
	if (disc < 0) {
		return false;
	}
	float root = sqrt(disc);
	return closest + root > near && closest - root < far;
}

// tree_in transforms a world point to tree-local coordinates.
static float3 tree_in(Tree tree, float3 point) {
	float3 from = point - tree.origin.xyz;
	return float3(dot(from, tree.x.xyz), dot(from, tree.y.xyz),
	              dot(from, tree.z.xyz)) / tree.origin.w;
}

// tree_turned transforms a world direction to tree-local coordinates.
static float3 tree_turned(Tree tree, float3 way) {
	return float3(dot(way, tree.x.xyz), dot(way, tree.y.xyz),
	              dot(way, tree.z.xyz));
}

// tree_out transforms a tree-local direction to world coordinates.
static float3 tree_out(Tree tree, float3 way) {
	return tree.x.xyz * way.x + tree.y.xyz * way.y + tree.z.xyz * way.z;
}

// Pending conifer traversal state: cone transform and remaining depth.
struct Branch { float3 base; int depth; };

// conifer_hit traverses the procedural tree generated by manufacture.Conifer.
// In local coordinates, each cone tip carries 3 + depth child cones with
// increasing outward lean. Cone size and descendant bounds depend only on
// depth, enabling branch-level culling without stored triangles.
static Hit conifer_hit(float3 at, float3 way, Tree tree, float near,
                       float far) {
	int deepest = min(tree.info.x, most_conifer_depth);
	float heights[most_conifer_depth + 1], radii[most_conifer_depth + 1];
	float reaches[most_conifer_depth + 1];
	float grow = 0.6f * tree.shape.w, thin = 0.5f * tree.shape.z;
	for (int level = 0; level <= deepest; level++) {
		heights[level] = tree.shape.x * pow(grow, float(level));
		radii[level] = tree.shape.y * pow(thin, float(level));
	}
	for (int level = deepest; level >= 0; level--) {
		float own = length(float2(heights[level], radii[level]));
		float grown = level == deepest ? 0 :
		              heights[level] * 1.12f + reaches[level + 1];
		reaches[level] = max(own, grown) * 1.001f;
	}
	Branch stack[conifer_stack];
	int waiting = 0;
	stack[waiting++] = {float3(0), 0};
	Hit found = {-1, -1, float3(0)};
	while (waiting > 0) {
		Branch node = stack[--waiting];
		int level = node.depth;
		if (!overlaps(at, way, node.base, reaches[level], near, far)) {
			continue;
		}
		Hit hit = cone_hit(at - node.base, way, heights[level],
		                   radii[level], tree.info.y, near, far);
		if (hit.distance >= 0) {
			found = hit;
			found.material = tree.info.z;
			far = hit.distance;
		}
		if (level == deepest) {
			continue;
		}
		float3 top = node.base + float3(0, 0, heights[level]);
		int children = 3 + level;
		float lean = tree.lean.x + float(level) * 0.15f;
		float out = heights[level] * 0.4f * 0.3f;
		float level_off = cos(lean), rise = sin(lean);
		float whole = 2 * M_PI_F / float(children);
		for (int child = 0; child < children; child++) {
			float turn = whole * float(child);
			float3 away = float3(level_off * cos(turn),
			                     level_off * sin(turn), rise);
			stack[waiting++] = {top + away * out, level + 1};
		}
	}
	return found;
}

// unit_of matches manufacture’s seeded draw: SplitMix64 over a seed and
// three indices. Go retains 53 fractional bits; this kernel uses the top 24.
// The shared seed and indices reproduce the same branch choices.
static float unit_of(ulong seed, int a, int b, int c) {
	ulong z = seed + 0x9e3779b97f4a7c15UL * ulong(a + 1) +
	          0xbf58476d1ce4e5b9UL * ulong(b + 1) +
	          0x94d049bb133111ebUL * ulong(c + 1);
	z ^= z >> 30;
	z *= 0xbf58476d1ce4e5b9UL;
	z ^= z >> 27;
	z *= 0x94d049bb133111ebUL;
	z ^= z >> 31;
	return float(z >> 40) / 16777216.0f;
}

// spread maps the deterministic draw to the interval from -1 to 1.
static float spread(ulong seed, int a, int b, int c) {
	return 2 * unit_of(seed, a, b, c) - 1;
}

// whorled_seed reconstructs the conifer seed from its two halves.
static ulong whorled_seed(Whorled tree) {
	return (ulong(uint(tree.seed.y)) << 32) | ulong(uint(tree.seed.x));
}

// limb_count matches manufacture.BranchCount: round(perwhorl*(1-0.55*t)),
// with a minimum of three and t=(whorl+0.5)/whorls. Rewriting the taper as
// (40*w-11*(2*i+1))/(40*w) keeps counting in integer arithmetic and rounds
// halves upward like Go’s Round, avoiding float-induced branch differences.
static int limb_count(Whorled tree, int whorl) {
	int whorls = tree.count.x, per = tree.count.y;
	int top = per * (40 * whorls - 11 * (2 * whorl + 1));
	return max(3, (2 * top + 40 * whorls) / (80 * whorls));
}

// Whorled branch parameters: attachment, initial direction, length,
// and base thickness.
struct Limb { float3 base; float3 dir; float length; float radius; };

// limb_of computes branch k of whorl i from its parameters and seed, matching
// manufacture.BranchOf without stored geometry. Lower branches are longer
// and droop; upper branches are shorter and rise. Successive whorls rotate
// by the golden angle.
static Limb limb_of(Whorled tree, int whorl, int k) {
	float tall = tree.shape.x, trunk_foot = tree.shape.y;
	float start = tree.shape.z, reach = tree.shape.w;
	ulong seed = whorled_seed(tree);
	float t = (float(whorl) + 0.5f) / float(tree.count.x);
	float height = tall * (start + (0.97f - start) * t);
	int count = limb_count(tree, whorl);
	float golden = M_PI_F * (3 - sqrt(5.0f));
	float round_one = 2 * M_PI_F / float(count);
	float turn = float(whorl) * golden + round_one * float(k) +
	             0.35f * spread(seed, whorl, k, 0) * round_one;
	float down = (100 - 38 * t + 6 * spread(seed, whorl, k, 1)) *
	             M_PI_F / 180;
	Limb limb;
	limb.base = float3(0, 0, height);
	limb.dir = float3(sin(down) * cos(turn), sin(down) * sin(turn),
	                  cos(down));
	limb.length = reach * tall * (1 - 0.88f * t) *
	              (1 + 0.15f * spread(seed, whorl, k, 2));
	limb.radius = max(0.35f * trunk_foot * (1 - height / tall),
	                  0.012f * tall);
	return limb;
}

// slab returns the ray interval between two planes along one axis. A
// parallel ray is either wholly inside or misses, following maths.Box.Hit
// without creating infinity under fast math.
static float2 slab(float at, float way, float low, float high) {
	if (way == 0) {
		return at >= low && at <= high ? float2(-nowhere, nowhere)
		                               : float2(nowhere, -nowhere);
	}
	float one = (low - at) / way, other = (high - at) / way;
	return float2(min(one, other), max(one, other));
}

// box_hit tests whether a ray reaches the box before far. Widen the exit
// by 1+2*gamma(3) for float arithmetic so rounding does not cull a face
// lying on the box boundary.
static bool box_hit(float3 at, float3 way, Node box, float far) {
	// a box inside out on any axis holds nothing, and no ray enters it;
	// the slabs alone would order its two faces and let a ray in
	if (any(box.low.xyz > box.high.xyz)) {
		return false;
	}
	float2 across = slab(at.x, way.x, box.low.x, box.high.x);
	float2 up = slab(at.y, way.y, box.low.y, box.high.y);
	float2 deep = slab(at.z, way.z, box.low.z, box.high.z);
	float enter = max(max(across.x, up.x), max(deep.x, 0.0f));
	float leave = min(min(across.y, up.y), min(deep.y, far));
	return enter <= leave * 1.0000008f;
}

// faces_hit walks one mesh's tree, from its root, for the nearest face
// before far, in the mesh's own space; with any, it stops at the first
// face, for a shadow. the hit's material is nought for a face met, and
// its normal and lift are the face's in the mesh's space: blended and
// lifted for a face shaded smooth, which a shadow does not ask.
static Hit faces_near(World world, int root, float3 at, float3 way,
                     float near, float far, bool any) {
	Hit found = {far, -1, float3(0), float3(0)};
	Triangle met;
	float2 weights_met;
	// the tree splits at the middle face, so its depth is at most
	// log2(faces / 4) and a little: tree_stack, from go, holds any tree
	// that fits in memory
	int stack[tree_stack];
	int depth = 0;
	stack[depth++] = root;
	while (depth > 0) {
		Node box = world.nodes[stack[--depth]];
		if (!box_hit(at, way, box, found.distance)) {
			continue;
		}
		if (box.info.y == 0) {
			stack[depth++] = box.info.x;
			stack[depth++] = box.info.x + 1;
			continue;
		}
		for (int index = 0; index < box.info.y; index++) {
			Triangle face = world.triangles[box.info.x + index];
			float2 weights;
			float distance = triangle_met(at, way, face, near,
			                              found.distance, weights);
			if (distance < 0) {
				continue;
			}
			float3 corner = face.corner_a.xyz;
			float3 flat = cross(face.corner_b.xyz - corner,
			                    face.corner_c.xyz - corner);
			found = {distance, 0, normalize(flat), float3(0)};
			if (any) {
				return found;
			}
			met = face, weights_met = weights;
		}
	}
	if (found.material < 0) {
		return found;
	}
	float3 point = fma(way, found.distance, at);
	found.local = point;
	int bend = int(met.corner_a.w);
	if (bend >= 0) {
		found = bent(world.bends[bend], met, point, weights_met, found);
	}
	return found;
}

// before_mesh finds a point just before the mesh's enclosing sphere, with a
// authored roundoff pad on its radius (about two parts per million over
// the circumsphere). This is not a watertight error bound for arbitrary
// coordinates. Distant origins otherwise lose fine-leaf barycentric detail.
static float before_mesh(World world, int root, float3 at, float3 way) {
	Node box = world.nodes[root];
	float speed = length(way);
	float3 centre = (box.low.xyz + box.high.xyz) * 0.5f;
	float radius = length(box.high.xyz - box.low.xyz) * 0.500001f;
	return max(0.0f, (dot(centre - at, way) / speed - radius) / speed);
}

// into transforms a world point to an instance’s local coordinates.
static float into_row(float4 row, float3 point) {
	return fma(row.x, point.x,
	           fma(row.y, point.y, fma(row.z, point.z, row.w)));
}

static float3 into(Instance placed, float3 point) {
	return float3(into_row(placed.into_x, point), into_row(placed.into_y, point),
	              into_row(placed.into_z, point));
}

// into_way transforms direction without normalizing, preserving the
// ray’s distance parameter across coordinate spaces.
static float3 into_way(Instance placed, float3 way) {
	return fma(float3(placed.into_x.x, placed.into_y.x, placed.into_z.x), way.x,
	           fma(float3(placed.into_x.y, placed.into_y.y, placed.into_z.y), way.y,
	               float3(placed.into_x.z, placed.into_y.z, placed.into_z.z) * way.z));
}

// instance_faces_hit rebases in the parent before rotating into a tiny mesh.
// This also keeps rounded rotation coefficients from acting on a distant
// origin. Recompute the local origin after the shift; keep ray distances in
// the caller's parameter throughout.
static Hit instance_faces_hit(World world, Instance placed, float3 at,
                              float3 way, float near, float far, bool any) {
	float3 local_at = into(placed, at), local_way = into_way(placed, way);
	float shift = before_mesh(world, placed.info.x, local_at, local_way);
	local_at = into(placed, fma(way, shift, at));
	Hit met = faces_near(world, placed.info.x, local_at, local_way,
	                     near - shift, far - shift, any);
	if (met.material >= 0) { met.distance += shift; }
	return met;
}

// out_of transforms an instance-local normal to world space.
static float3 out_of(Instance placed, float3 normal) {
	return normalize(placed.out_x.xyz * normal.x + placed.out_y.xyz * normal.y +
	                 placed.out_z.xyz * normal.z);
}

// scale_of recovers instance scale from its world-to-local transform.
static float scale_of(float4 into_x) { return 1 / length(into_x.xyz); }

// moved_out transforms a local displacement to world space, applying
// both orientation and scale.
static float3 moved_out(Instance placed, float3 move) {
	return (placed.out_x.xyz * move.x + placed.out_y.xyz * move.y +
	        placed.out_z.xyz * move.z) * scale_of(placed.into_x);
}

// group_hit walks one shared tree of local mesh instances. Members are
// meshes, so traversal has a fixed second instance level and no recursion.
static Hit group_hit(World world, int root, float3 at, float3 way,
                     float near, float far, bool any) {
	Hit found = {far, -1, float3(0)};
	int stack[tree_stack];
	int depth = 0;
	stack[depth++] = root;
	while (depth > 0) {
		Node box = world.tops[stack[--depth]];
		if (!box_hit(at, way, box, found.distance)) { continue; }
		if (box.info.y == 0) {
			stack[depth++] = box.info.x;
			stack[depth++] = box.info.x + 1;
			continue;
		}
		for (int index = 0; index < box.info.y; index++) {
			Instance child = world.instances[box.info.x + index];
			Hit met = instance_faces_hit(world, child, at, way,
			                            near, found.distance, any);
			if (met.material < 0) { continue; }
			found = {met.distance, 0, out_of(child, met.normal),
			         moved_out(child, met.lift), fma(way, met.distance, at), 1};
			if (any) { return found; }
		}
	}
	return found;
}

// Forward declaration for a refused native shadow candidate.
static Hit meshes_software(World world, float3 at, float3 way, float near,
                          float far, bool any);

#if defined(SLAYER_NATIVE_SHADOWS) || defined(SLAYER_NATIVE_MESHES)
// native_clip shortens a segment to the sphere that holds every standing
// mesh, so the hardware query never starts far from the structure it
// walks. It is false when the segment between near and far never enters
// the sphere, and then no query is made. Otherwise start and stop are
// the distances along the way, from at, where the query begins and ends:
// the sphere's entry and exit, each moved out by a margin, and the ends
// of the segment where they are inside it. The margin is a share of the
// sphere's radius and the rounding of the arithmetic at the ray's distance
// from the sphere, both named in the prelude. The perpendicular distance
// is found by a cross product, which stays exact where a difference of
// squares would lose the whole distance at a far origin.
static bool native_clip(float4 bounds, float3 at, float3 way, float near,
                        float far, thread float &start, thread float &stop) {
	float3 to = bounds.xyz - at;
	float margin = bounds.w * clip_share + length(to) * clip_rounding;
	float reach = bounds.w + margin;
	float across = length(cross(to, way));
	if (across > reach) { return false; }
	float along = dot(to, way);
	float half_chord = sqrt(max(beam_zero, reach * reach - across * across));
	start = max(max(beam_zero, near), along - half_chord);
	stop = min(far, along + half_chord);
	return stop > start;
}
#endif

#ifdef SLAYER_NATIVE_SHADOWS
// Refine the candidate standing with the same predicate as ordinary meshes.
// Hardware any-hit alone can report a near-surface self hit on thin supports.
static Hit native_shadow_standing(World world, Instance placed,
                                 float3 at, float3 way, float far) {
	if (placed.info.w != 0) {
		return group_hit(world, placed.info.w - 1, into(placed, at),
		                 into_way(placed, way), 0, far, true);
	}
	return instance_faces_hit(world, placed, at, way, 0, far, true);
}

// Native any-hit proposes one opaque standing. Analytic shapes remain in blocked.
static Hit native_shadow(World world, float3 at, float3 way, float far) {
	Hit found = {far, -1, float3(0)};
	if (world.header->tops.x == 0 || !(far > 0)) { return found; }
	float start, stop;
	if (!native_clip(world.header->bounds, at, way, beam_zero, far, start, stop)) {
		return found;
	}
	ray r;
	r.origin = fma(way, start, at);
	r.direction = way;
	r.min_distance = 0;
	r.max_distance = stop - start;
	intersector<instancing, max_levels<3>> query;
	query.assume_geometry_type(geometry_type::triangle);
	query.accept_any_intersection(true);
	auto met = query.intersect(r, world.accelerated, 2u);
	if (met.type != intersection_type::triangle) { return found; }
	Instance placed = world.instances[int(met.instance_id[0])];
	Hit checked = native_shadow_standing(world, placed, at, way, far);
	if (checked.material >= 0) {
		found.material = 0;
		return found;
	}
	// A refused candidate does not prove that the rest of the ray is clear.
	return meshes_software(world, at, way, 0, far, true);
}
#endif

// meshes_software walks the tree of standing meshes for the nearest face
// before far, each mesh walked in its own space; with any, it stops at
// the first face of a mesh that is not clear, for a shadow.
static Hit meshes_software(World world, float3 at, float3 way, float near,
                      float far, bool any) {
	Hit found = {far, -1, float3(0)};
	// no meshes: the tree is a root with no children, which must not be
	// walked, since a root entered is read as a branch whose children
	// are itself and a node past the end. the go side also makes that
	// root a box inside out, by finite numbers, so no ray enters it
	if (world.header->tops.x == 0) {
		return found;
	}
	int stack[tree_stack];
	int depth = 0;
	stack[depth++] = 0;
	while (depth > 0) {
		Node box = world.tops[stack[--depth]];
		if (!box_hit(at, way, box, found.distance)) {
			continue;
		}
		if (box.info.y == 0) {
			stack[depth++] = box.info.x;
			stack[depth++] = box.info.x + 1;
			continue;
		}
		for (int index = 0; index < box.info.y; index++) {
			Instance placed = world.instances[box.info.x + index];
			if (any && placed.info.z != 0) {
				continue;
			}
			float3 local_at = into(placed, at);
			float3 local_way = into_way(placed, way);
			Hit met;
			if (placed.info.w != 0) {
				met = group_hit(world, placed.info.w - 1, local_at,
				                local_way, near, found.distance, any);
			} else {
				met = instance_faces_hit(world, placed, at, way,
				                         near, found.distance, any);
			}
			if (met.material < 0) {
				continue;
			}
			found = {met.distance, placed.info.y,
			         out_of(placed, met.normal),
			         moved_out(placed, met.lift), met.local,
			         scale_of(placed.into_x)};
			if (any) {
				return found;
			}
		}
	}
	return found;
}

#ifdef SLAYER_NATIVE_MESHES
// Refine one native candidate in the same nearby coordinates used by the
// software path. Distant origins otherwise lose the width of thin needles.
static Hit native_face(World world, Instance placed, int triangle,
                       float3 at, float3 way, float near, float far) {
	float3 local_at = into(placed, at), local_way = into_way(placed, way);
	float shift = before_mesh(world, placed.info.x, local_at, local_way);
	local_at = into(placed, fma(way, shift, at));
	Triangle face = world.triangles[triangle];
	float2 weights;
	float distance = triangle_met(local_at, local_way, face,
                               near - shift, far - shift, weights);
	Hit found = {far, -1, float3(0)};
	if (distance < 0) { return found; }
	float3 normal = normalize(cross(face.corner_b.xyz - face.corner_a.xyz,
                                face.corner_c.xyz - face.corner_a.xyz));
	float3 point = fma(local_way, distance, local_at);
	found = {distance, 0, normal, float3(0), point, 1};
	int bend = int(face.corner_a.w);
	if (bend >= 0) { found = bent(world.bends[bend], face, point, weights, found); }
	found.distance += shift;
	return found;
}

// Native IDs name the sorted world standing and, for a group, its sorted
// local member. Apple WWDC2023/10128 documents the nested result ID array.
static Hit native_first(World world, float3 at, float3 way,
                        float near, float far) {
	if (world.native_counts) { world.native_counts->x++; }
	Hit miss = {far, -1, float3(0)};
	if (world.header->tops.x == 0 || !(far > near)) { return miss; }
	float start, stop;
	if (!native_clip(world.header->bounds, at, way, near, far, start, stop)) {
		return miss;
	}
	ray r;
	r.origin = fma(way, start, at);
	r.direction = way;
	r.min_distance = beam_zero;
	r.max_distance = stop - start;
	intersector<instancing, max_levels<3>> query;
	query.assume_geometry_type(geometry_type::triangle);
	query.accept_any_intersection(false);
	auto candidate = query.intersect(r, world.accelerated, 1u);
	if (candidate.type != intersection_type::triangle) { return miss; }
	if (world.native_counts) { world.native_counts->y++; }
	int root = int(candidate.instance_id[0]);
	Instance placed = world.instances[root];
	Hit met;
	if (placed.info.w != 0) {
		int leaf = world.firsts[root] + int(candidate.instance_id[1]);
		Instance child = world.instances[leaf];
		float3 local_at = into(placed, at), local_way = into_way(placed, way);
		int triangle = world.firsts[leaf] + int(candidate.primitive_id);
		met = native_face(world, child, triangle,
                          local_at, local_way, near, far);
		met.normal = out_of(child, met.normal);
		met.lift = moved_out(child, met.lift);
		met.local = fma(local_way, met.distance, local_at);
	} else {
		int triangle = world.firsts[root] + int(candidate.primitive_id);
		met = native_face(world, placed, triangle,
                          at, way, near, far);
	}
	if (met.material < 0) {
		if (world.native_counts) { world.native_counts->w++; }
		return meshes_software(world, at, way, near, far, false);
	}
	if (world.native_counts) { world.native_counts->z++; }
	return {met.distance, placed.info.y, out_of(placed, met.normal),
         moved_out(placed, met.lift), met.local, scale_of(placed.into_x)};
}
#endif

// Keep software available for comparison and a refused candidate's refinement.
static Hit meshes_hit(World world, float3 at, float3 way, float near,
                      float far, bool any) {
#ifdef SLAYER_NATIVE_SHADOWS
	if (any) { return native_shadow(world, at, way, far); }
#endif
#ifdef SLAYER_NATIVE_MESHES
	return native_first(world, at, way, near, far);
#else
	return meshes_software(world, at, way, near, far, any);
#endif
}

// first is trace's: the nearest thing a ray meets past near, sphere or
// face; a material of -1 is nothing.
// field_in and field_turned take the world into a height field's own
// space, and field_out brings a direction back, as a tree's do.
static float3 field_in(Field field, float3 point) {
	float3 from = point - field.origin.xyz;
	return float3(dot(from, field.x.xyz), dot(from, field.y.xyz),
	              dot(from, field.z.xyz)) / field.origin.w;
}

static float3 field_turned(Field field, float3 way) {
	return float3(dot(way, field.x.xyz), dot(way, field.y.xyz),
	              dot(way, field.z.xyz));
}

static float3 field_out(Field field, float3 way) {
	return way.x * field.x.xyz + way.y * field.y.xyz +
	       way.z * field.z.xyz;
}

// field_level is the field's height over a place, on the straight
// lines between its four nearest samples, held to the square:
// shape.HeightField's height, number for number.
static float field_level(World world, Field field, float x, float z) {
	int n = field.data.y;
	float step = field.size.x / (n - 1);
	float last = float(n - 1);
	float gx = clamp((x + field.size.x / 2) / step, 0.0f, last);
	float gz = clamp((z + field.size.x / 2) / step, 0.0f, last);
	int column = min(int(gx), n - 2);
	int row = min(int(gz), n - 2);
	float tx = gx - column, tz = gz - row;
	device const float *sample = world.heights + field.data.x +
	                             row * n + column;
	float low = sample[0] * (1 - tx) + sample[1] * tx;
	float upper = sample[n] * (1 - tx) + sample[n + 1] * tx;
	return low * (1 - tz) + upper * tz;
}

// field_normal is which way the field faces over a place: the cell's
// own slopes, as shape.HeightField.Normal takes them.
static float3 field_normal(World world, Field field, float x, float z) {
	int n = field.data.y;
	float step = field.size.x / (n - 1);
	float last = float(n - 1);
	float gx = clamp((x + field.size.x / 2) / step, 0.0f, last);
	float gz = clamp((z + field.size.x / 2) / step, 0.0f, last);
	int column = min(int(gx), n - 2);
	int row = min(int(gz), n - 2);
	float tx = gx - column, tz = gz - row;
	device const float *sample = world.heights + field.data.x +
	                             row * n + column;
	float low_x = (sample[1] - sample[0]) / step;
	float high_x = (sample[n + 1] - sample[n]) / step;
	float low_z = (sample[n] - sample[0]) / step;
	float high_z = (sample[n + 1] - sample[1]) / step;
	float dx = low_x * (1 - tz) + high_x * tz;
	float dz = low_z * (1 - tx) + high_z * tx;
	return normalize(float3(-dx, 1, -dz));
}

// field_hit is how far along a local ray it meets the field, by the
// march trace takes over shape.HeightField: into the field's
// box, then steps of the flattened vertical gap, which cannot pass
// through the surface, until the gap is within touch of it. Minus
// one is a miss.
static float field_hit(World world, Field field, float3 at, float3 way,
                       float near, float far) {
	float wide = field.size.x / 2; // half is a type to metal
	float3 low = float3(-wide, field.size.y, -wide);
	float3 high = float3(wide, field.size.z, wide);
	float3 enters = (low - at) / way;
	float3 leaves = (high - at) / way;
	float3 least = min(enters, leaves), most = max(enters, leaves);
	float enter = max(max(least.x, least.y), max(least.z, near));
	float leave = min(min(most.x, most.y), min(most.z, far));
	if (leave < enter) {
		return -1;
	}
	// how big the box is where floats are concerned, as trace's reach
	float reach = max(length(high - low), max(length(low), length(high)));
	float along = enter;
	for (int step = 0; step < march_steps && along <= leave; step++) {
		float3 p = at + way * along;
		float gap = (p.y - field_level(world, field, p.x, p.z)) *
		            field.size.w;
		if (fabs(gap) < march_touch * max(reach, length(p))) {
			return along;
		}
		along += fabs(gap);
	}
	return -1;
}

static Hit first(World world, float3 at, float3 way, float near,
                 int leaving_sphere) {
	Hit found = meshes_hit(world, at, way, near, nowhere, false);
	for (int index = 0; index < world.header->look.z; index++) {
		Field field = world.fields[index];
		float scale = field.origin.w;
		float3 local_at = field_in(field, at);
		float3 local_way = field_turned(field, way);
		float met = field_hit(world, field, local_at, local_way,
		                      near / scale, found.distance / scale);
		if (met < 0) {
			continue;
		}
		float3 local = local_at + local_way * met;
		float3 faces = field_normal(world, field, local.x, local.z);
		found = {met * scale, field.data.z,
		         normalize(field_out(field, faces)), float3(0), local,
		         scale};
	}
	for (int index = 0; index < world.header->counts.x; index++) {
		Sphere sphere = world.spheres[index];
		if (index == leaving_sphere &&
		    dot(at - sphere.centre.xyz, way) > 0) {
			continue;
		}
		float distance = sphere_hit(at, way, sphere, near,
		                            found.distance);
		if (distance < 0) {
			continue;
		}
		float3 point = at + way * distance;
		float4 into = float4(point, 1);
		float3 local = float3(dot(sphere.into_x, into),
		                      dot(sphere.into_y, into),
		                      dot(sphere.into_z, into));
		found = {distance, sphere.info.x,
		         (sphere.info.z == 1 ? sphere.centre.xyz :
		          normalize(point - sphere.centre.xyz)), float3(0), local,
		         scale_of(sphere.into_x)};
		found.sphere_index = index;
	}
	for (int index = 0; index < world.header->more.w; index++) {
		Tree tree = world.trees[index];
		float scale = tree.origin.w;
		float3 local_at = tree_in(tree, at);
		float3 local_way = tree_turned(tree, way);
		Hit grown = conifer_hit(local_at, local_way, tree, near / scale,
		                        found.distance / scale);
		if (grown.distance >= 0) {
			found = {grown.distance * scale, grown.material,
			         tree_out(tree, grown.normal), float3(0),
			         local_at + local_way * grown.distance, scale};
		}
	}
	return found;
}

static Hit first(World world, float3 at, float3 way, float near) {
	return first(world, at, way, near, -1);
}

// Curved segments stop at their end; ordinary first keeps the RC's refinements.
static Hit first_limited(World world, float3 at, float3 way, float near, float far) {
    Hit hit = first(world, at, way, near);
    if (hit.distance > far) {
        return {far, -1, float3(0), float3(0), float3(0), 1};
    }
    return hit;
}

// fields_block is whether any height field not clear stands on the way
// to a lamp, for blocked.
static bool fields_block(World world, float3 at, float3 way, float far) {
	for (int index = 0; index < world.header->look.z; index++) {
		Field field = world.fields[index];
		if (field.data.w != 0) {
			continue;
		}
		float scale = field.origin.w;
		if (field_hit(world, field, field_in(field, at),
		              field_turned(field, way), 0, far / scale) >= 0) {
			return true;
		}
	}
	return false;
}

// blocked is trace's: whether anything not clear stands between a point
// and a lamp far along the ray.
// A ray leaving its own convex sphere outward cannot meet that sphere
// again: |p+t*d-c|^2 = r^2 + 2*t*dot(p-c,d) + t^2*|d|^2 > r^2.
// Excluding only that primitive avoids false self-hits from planetary float32
// rounding. Inward rays and every other occluder retain the normal tests.
static bool blocked(World world, float3 at, float3 way, float far,
                    int leaving_sphere) {
	if (fields_block(world, at, way, far)) {
		return true;
	}
	for (int index = 0; index < world.header->counts.x; index++) {
		Sphere sphere = world.spheres[index];
		if (index == leaving_sphere &&
		    dot(at - sphere.centre.xyz, way) > 0) {
			continue;
		}
		if (sphere.info.y == 0 &&
		    sphere_hit(at, way, sphere, 0, far) >= 0) {
			return true;
		}
	}
	for (int index = 0; index < world.header->more.w; index++) {
		Tree tree = world.trees[index];
		float scale = tree.origin.w;
		float3 local_at = tree_in(tree, at);
		float3 local_way = tree_turned(tree, way);
		Hit grown = conifer_hit(local_at, local_way, tree, 0,
		                        far / scale);
		if (grown.distance >= 0) {
			return true;
		}
	}
	return meshes_hit(world, at, way, 0, far, true).material >= 0;
}

static bool blocked(World world, float3 at, float3 way, float far) {
	return blocked(world, at, way, far, -1);
}

// cone_share is how much of a cone's light goes out a way, a unit long:
// all of it within the width of the cone's pole, fading to none across
// its soft edge. it is emit's Along past its width, and maths.Smooth.
static float cone_share(float width, float soft, float3 pole, float3 out) {
	float off = acos(clamp(dot(out, pole), -beam_one, beam_one));
	if (off <= width) {
		return 1;
	}
	if (off >= width + soft) {
		return 0;
	}
	float through = (off - width) / soft;
	return 1 - through * through * (3 - 2 * through);
}

// lamp_along is how much of a lamp's light goes out a way, a unit
// long: all of it for a lamp with no width, and for one kept to a
// cone, its cone_share. it is emit's Along.
static float lamp_along(Lamp lamp, float3 out) {
	float width = lamp.toward.w;
	if (width <= beam_zero) {
		return beam_one;
	}
	return cone_share(width, lamp.strength.w, lamp.toward.xyz, out);
}

// The volume average of inverse squared distance over a uniform ball.
// These are the same stable near/far series used by emit.Volume.
static float volume_falloff(float distance, float radius) {
 float q = distance / radius;
 if (q <= 0.25f) {
  float u = q*q;
  return 3/(radius*radius) *
   (1-u*(1.0f/3+u*(1.0f/15+u*(1.0f/35+u/63))));
 }
 if (q >= 4) {
  float u = (radius/distance)*(radius/distance);
  return (1+u*(1.0f/5+u*(3.0f/35+u/21))) / (distance*distance);
 }
 if (q == 1) return 1.5f/(radius*radius);
 return 1.5f/(radius*radius) *
  (1 + (1-q*q)/(2*q)*log((1+q)/abs(1-q)));
}

// is_flood is whether a lamp is a film.Flood, which lights from everywhere:
// it has no way, and no shadow.
static bool is_flood(Lamp lamp) { return lamp.second.w == lamp_flood; }

// lamp_light agrees with emit.Lit, emit.Volume, emit.Beam, emit.Distant,
// emit.Beams and film.Flood. Finite sources keep the centre-ray
// approximation for shadows. seen is the way a surface faces, or the way
// back along a ray in a medium, and is read only by a flood, whose light
// is its strength weighed by how that way faces its dome; a seen of
// nought is a way facing it neither toward nor away, as film.Flood's is.
struct Reached { float3 toward; float far; float3 light; };
static Reached lamp_light(Lamp lamp, float3 point, float3 seen) {
 if (is_flood(lamp)) {
  float3 facing = any(seen != beam_zero) ? normalize(seen) : seen;
  float share = (flood_whole + dot(facing, lamp.at.xyz)) * flood_half;
  return {float3(beam_zero), nowhere, lamp.strength.xyz * share};
 }
 if (lamp.second.w == lamp_polar) {
  float3 toward = lamp.at.xyz - point;
  float distance = length(toward);
  if (distance <= lamp.at.w) {
   return {float3(beam_zero), beam_zero, float3(beam_zero)};
  }
  float3 in = toward / distance;
  float share = cone_share(lamp.toward.w, lamp.strength.w, lamp.toward.xyz, -in) +
                cone_share(lamp.toward.w, lamp.strength.w, lamp.second.xyz, -in);
  float falling = share / (distance * distance);
  return {in, distance - lamp.at.w, lamp.strength.xyz * falling};
 }
 if (lamp.extent.y > beam_zero) {
  float3 delta = point - lamp.at.xyz;
  float along = dot(delta, lamp.toward.xyz);
  if (!isfinite(along) || along <= beam_zero || along >= lamp.extent.w) {
   return {float3(beam_zero), beam_zero, float3(beam_zero)};
  }
  float width = lamp.extent.y + along * lamp.extent.z;
  float off = length(delta - along * lamp.toward.xyz) / width;
  float ratio = lamp.extent.y / width;
  float weight = ratio * ratio * exp(lamp.extent.x * off * off);
  if (!(weight > beam_zero)) {
   return {float3(beam_zero), beam_zero, float3(beam_zero)};
  }
  return {-lamp.toward.xyz, along, lamp.strength.xyz * weight};
 }
 if (lamp.at.w < 0) {
  return {lamp.at.xyz, nowhere, lamp.strength.xyz};
 }
 float3 toward = lamp.at.xyz - point;
 float distance = length(toward);
 if (lamp.extent.x > 0) {
  float3 in = distance > 0 ? toward / distance : float3(0,1,0);
  float falling = lamp_along(lamp, -in) * volume_falloff(distance, lamp.extent.x);
  return {in, distance, lamp.strength.xyz * falling};
 }
 if (distance <= lamp.at.w) {
  return {float3(0), 0, float3(0)};
 }
 float3 in = toward / distance;
 float falling = lamp_along(lamp, -in) / (distance * distance);
 return {in, distance - lamp.at.w, lamp.strength.xyz * falling};
}

// flooded is the light the floods among the lamps give a surface facing
// seen, or a bit of a medium a ray runs away from along seen: each lights
// from everywhere at once, unshadowed, and is asked at every point, as the
// tracer asks every lamp of a scene that has one.
static float3 flooded(World world, float3 seen) {
	float3 sum = float3(beam_zero);
	for (int index = whole_zero; index < world.header->counts.w; index++) {
		Lamp lamp = world.lamps[index];
		if (is_flood(lamp)) {
			sum += lamp_light(lamp, float3(beam_zero), seen).light;
		}
	}
	return sum;
}

// a lamp read with no way to face is read facing none: there is no flood
// among the lamps asked about by their worth, which refuse keeps out.
static Reached lamp_light(Lamp lamp, float3 point) {
 return lamp_light(lamp, point, float3(beam_zero));
}

// an Ask is one lamp a point asks, and how many times over it counts
struct Ask { int lamp; float counts; };

// luminance is maths.Light.Luminance
static float luminance(float3 light) {
	return dot(light, float3(0.2126f, 0.7152f, 0.0722f));
}

// place_share is trace's placeShare: a share between nought and one a
// point's place decides, its three numbers' bits stirred by
// splitmix64's finish. the tracer stirs a double's bits and this a
// float's, so the two pick alike in the mean, not lamp for lamp.
static float place_share(float3 point) {
	ulong stirred = 0x9e3779b97f4a7c15UL;
	for (int part = 0; part < 3; part++) {
		stirred ^= ulong(as_type<uint>(point[part]));
		stirred ^= stirred >> 30;
		stirred *= 0xbf58476d1ce4e5b9UL;
		stirred ^= stirred >> 27;
		stirred *= 0x94d049bb133111ebUL;
		stirred ^= stirred >> 31;
	}
	return float(stirred >> 40) / 16777216.0f;
}

// asked is trace's asked: the lamps a point asks, each with how many
// times over it counts. with lamps_asked lamps or fewer, every lamp,
// once. with more, lamps_asked picks spread evenly along the lamps'
// worths laid end to end, from a start the point's place decides, each
// pick counting its share of all the worth over its lamp's own. a lamp
// is worth its light's luminance, times how squarely it lands for a
// surface facing normal; a normal of nought takes light every way, as a
// bit of a volume does.
static int asked(World world, float3 point, float3 normal,
                 thread Ask *asks) {
	int lamps = world.header->counts.w;
	if (lamps <= lamps_asked) {
		for (int index = 0; index < lamps; index++) {
			asks[index] = {index, 1};
		}
		return lamps;
	}
	bool facing = any(normal != 0);
	float total = 0;
	for (int index = 0; index < lamps; index++) {
		Reached reached = lamp_light(world.lamps[index], point);
		float square = facing ? max(0.0f, dot(normal, reached.toward)) : 1;
		total += luminance(reached.light) * square;
	}
	if (total <= 0) {
		return 0;
	}
	float step = total / lamps_asked;
	float next = place_share(point) * step, laid = 0;
	int taken = 0, made = 0;
	for (int index = 0; index < lamps && taken < lamps_asked; index++) {
		Reached reached = lamp_light(world.lamps[index], point);
		float square = facing ? max(0.0f, dot(normal, reached.toward)) : 1;
		float worth = luminance(reached.light) * square;
		if (worth <= 0) {
			continue;
		}
		laid += worth;
		int picks = 0;
		while (next < laid && taken < lamps_asked) {
			picks++, taken++;
			next += step;
		}
		if (picks > 0) {
			asks[made++] = {index, float(picks) * step / worth};
		}
	}
	return made;
}

// inside is a world point in a volume's own space, as maths.Subspace.In
static float3 inside(Volume volume, float3 point) {
	float3 from = point - volume.origin.xyz;
	return float3(dot(from, volume.x.xyz), dot(from, volume.y.xyz),
	              dot(from, volume.z.xyz)) / volume.origin.w;
}

// turned is a world direction in a volume's own space
static float3 turned(Volume volume, float3 way) {
	return float3(dot(way, volume.x.xyz), dot(way, volume.y.xyz),
	              dot(way, volume.z.xyz));
}

// live is medium.Puff, made ragged by Wispy's noise when it is wispy,
// asked at a point: the truth the bake is made from
static float live(Volume volume, float3 local) {
	float radius = volume.puff.x;
	if (radius <= 0) {
		return 0;
	}
	float puffed = volume.puff.y *
	               (1 - smoothstep(0.0f, 1.0f, length(local) / radius));
	if (puffed == 0 || volume.info.z == 0) {
		return puffed;
	}
	float scale = volume.puff.z > 0 ? volume.puff.z : 1;
	ulong high = ulong(uint(volume.info.y));
	ulong seed = (high << 32) | ulong(uint(volume.info.x));
	float3 at = local / scale;
	float wisp = fractal(seed, at.x, at.y, at.z, 4);
	float thin = clamp(volume.puff.w, 0.0f, 1.0f);
	return puffed * (1 - thin * (0.5f + 0.5f * wisp));
}

// cell is one of a baked grid's cells, clamped to the grid
static float cell(World world, Volume volume, int3 at) {
	int side = volume.baked.y;
	int3 held = clamp(at, int3(0), int3(side - 1));
	int index = (held.z * side + held.y) * side + held.x;
	return world.baked[volume.baked.x + index];
}

// slice is the grid blended across one layer of cells, from the cell at
// low, by how far between cells the point is along x and y
static float slice(World world, Volume volume, int3 low, float2 part) {
	float near_y = mix(cell(world, volume, low),
	                   cell(world, volume, low + int3(1, 0, 0)), part.x);
	float far_y = mix(cell(world, volume, low + int3(0, 1, 0)),
	                  cell(world, volume, low + int3(1, 1, 0)), part.x);
	return mix(near_y, far_y, part.y);
}

// boxed_part is where a point of a boxed medium's own space falls in its
// grid, in cells from the middle of the first, and whether it is inside
// the box at all
static bool boxed_part(Volume volume, float3 local, thread int3 &low,
                       thread float3 &part) {
	float3 lowest = volume.low.xyz, highest = volume.high.xyz;
	if (any(local < lowest) || any(local > highest)) {
		return false;
	}
	float3 cells = float3(volume.grid.xyz);
	float3 place = (local - lowest) / (highest - lowest) * cells - 0.5f;
	low = int3(floor(place));
	part = place - float3(low);
	return true;
}

// boxed_cell is a boxed medium's numbers at one cell, so many a cell from
// first, clamped to the grid
static float3 boxed_cell(World world, Volume volume, int first, int each,
                         int3 at) {
	int3 held = clamp(at, int3(0), volume.grid.xyz - 1);
	int cell = (held.z * volume.grid.y + held.y) * volume.grid.x + held.x;
	device const float *numbers = world.boxes + first + cell * each;
	return each == 1 ? float3(numbers[0], 0, 0)
	                 : float3(numbers[0], numbers[1], numbers[2]);
}

// boxed_at is a boxed medium's numbers at a point, blended between the
// eight cells round it: its density, one a cell, or a light, three
static float3 boxed_at(World world, Volume volume, int first, int each,
                       int3 low, float3 part) {
	float3 corners[2][2][2];
	for (int z = 0; z < 2; z++) {
		for (int y = 0; y < 2; y++) {
			for (int x = 0; x < 2; x++) {
				corners[z][y][x] = boxed_cell(world, volume, first, each,
				                              low + int3(x, y, z));
			}
		}
	}
	float3 near = mix(mix(corners[0][0][0], corners[0][0][1], part.x),
	                  mix(corners[0][1][0], corners[0][1][1], part.x),
	                  part.y);
	float3 far = mix(mix(corners[1][0][0], corners[1][0][1], part.x),
	                 mix(corners[1][1][0], corners[1][1][1], part.x),
	                 part.y);
	return mix(near, far, part.z);
}

// shell_density shares medium.ShellWeather's radial parabola and prescribed
// advected patches. It is evaluated at march points, never a planet-sized grid.
static float shell_density(Volume volume, float3 local) {
	float height = length(local) - volume.low.x;
	if (height <= volume.low.y || height >= volume.low.z || volume.puff.w <= 0) {
		return 0;
	}
	float h = (height - volume.low.y) / (volume.low.z - volume.low.y);
	float weight = 4 * h * (1 - h);
	if (volume.puff.w < 1) {
		ulong seed = (ulong(uint(volume.info.y)) << 32) | ulong(uint(volume.info.x));
		float3 at = (local - volume.high.xyz) / volume.puff.z;
		float patch = 0.5f + 0.5f * fractal(seed, at.x, at.y, at.z, volume.baked.x);
		float threshold = 1 - volume.puff.w;
		weight *= smoothstep(threshold - volume.high.w, threshold + volume.high.w, patch);
	}
	return volume.puff.y * weight;
}

// density reads a shell directly, a boxed medium from its grid, and a puff
// from its baked cells when available or from its original field otherwise.
// Bounds and occupied shell intervals keep the march outside empty regions.
static float density(World world, Volume volume, float3 local) {
	if (volume.info.z == ring_kind) {
		return ring_density(world.boxes + volume.grid.w, local);
	}
	if (volume.info.z == wake_kind) {
		return wake_density(world.boxes + volume.grid.w, local);
	}
	if (volume.info.z == shell_kind) {
		return shell_density(volume, local);
	}
	if (volume.info.z == boxed_kind) {
		if (volume.info.x==2) {
			float h=volume.puff.z/2,r=length(local.xz);
			if (h<=0 || volume.puff.y<=volume.puff.x || r<volume.puff.x || r>volume.puff.y || abs(local.y)>h) return 0;
			float face=clamp((h-abs(local.y))/(.1f*h),0.0f,1.0f);
			return (volume.puff.w>0 ? volume.puff.w : 1)*face*face*(3-2*face);
		}
		if (volume.info.x == 1) {
			float radius = volume.puff.x, depth = volume.puff.y;
			if (radius <= 0 || depth <= 0) {
				return 0;
			}
			float below = radius - length(local);
			float along = clamp(below / depth, 0.0f, 1.0f);
			return volume.puff.z * along * along * (3 - 2 * along);
		}
		int3 low;
		float3 part;
		if (!boxed_part(volume, local, low, part)) {
			return 0;
		}
		return boxed_at(world, volume, volume.grid.w, 1, low, part).x;
	}
	int side = volume.baked.y;
	if (side == 0) {
		return live(volume, local);
	}
	float radius = volume.puff.x;
	if (any(abs(local) > radius)) {
		return 0;
	}
	float3 place = (local + radius) / (2 * radius) * side - 0.5f;
	int3 low = int3(floor(place));
	float3 part = place - float3(low);
	return mix(slice(world, volume, low, part.xy),
	           slice(world, volume, low + int3(0, 0, 1), part.xy), part.z);
}

// span is where a ray in a volume's own space crosses it, from near to
// far: a puff's ball, as shape.Sphere, or a boxed medium's box, as
// maths.Box.Hit, which trace spans a medium with no Span of its own by
static float2 span(Volume volume, float3 at, float3 way, float near,
                   float far) {
	if (volume.info.z==boxed_kind && volume.info.x==2) {
		float h=volume.puff.z/2;
		float2 up=slab(at.y,way.y,-h,h);
		float enter=max(near,up.x),leave=min(far,up.y);
		float a=dot(way.xz,way.xz);
		if (a>0) {
			float b=dot(at.xz,way.xz)/a;
			float c=(dot(at.xz,at.xz)-volume.puff.y*volume.puff.y)/a;
			float d=b*b-c;
			if (d<0) return float2(1,0);
			enter=max(enter,-b-sqrt(d));leave=min(leave,-b+sqrt(d));
		} else if (dot(at.xz,at.xz)>volume.puff.y*volume.puff.y) return float2(1,0);
		return float2(enter,leave);
	}
	if ((volume.info.z == boxed_kind && volume.info.x != 1) || volume.info.z == ring_kind) {
		float3 low = volume.low.xyz, high = volume.high.xyz;
		float2 across = slab(at.x, way.x, low.x, high.x);
		float2 up = slab(at.y, way.y, low.y, high.y);
		float2 deep = slab(at.z, way.z, low.z, high.z);
		float enter = max(max(across.x, up.x), max(deep.x, near));
		float leave = min(min(across.y, up.y), min(deep.y, far));
		return float2(enter, leave);
	}
	float radius = volume.puff.x;
	float closest = -dot(at, way);
	float disc = closest * closest - (dot(at, at) - radius * radius);
	if (disc < 0) {
		return float2(1, 0);
	}
	float root = sqrt(disc);
	return float2(max(closest - root, near), min(closest + root, far));
}

// shell_ball uses a factored distance term and paired quadratic roots. Keeping
// the near root as c/q avoids subtracting two planetary distances at entry.
static float2 shell_ball(float radius, float3 at, float3 way, float near, float far) {
	float distance = length(at);
	float c = (distance - radius) * (distance + radius);
	float closest = -dot(at, way);
	float disc = fma(closest, closest, -c);
	if (disc < 0) { return float2(1, 0); }
	float root = sqrt(disc);
	float q = closest + copysign(root, closest);
	float other = q == 0 ? closest : c / q;
	return float2(max(min(q, other), near), min(max(q, other), far));
}

// spans keeps the two occupied legs of a hollow shell separate. Ordinary
// volumes retain the original span calculation and one integration interval.
static int spans(Volume volume, float3 at, float3 way, float near, float far,
                 thread float2 *parts) {
	parts[0] = volume.info.z == shell_kind ?
		shell_ball(volume.puff.x, at, way, near, far) : span(volume, at, way, near, far);
	if (parts[0].y <= parts[0].x) { return 0; }
	if (volume.info.z != shell_kind) { return 1; }
	float2 outer = parts[0];
	float2 inner = shell_ball(volume.low.x + volume.low.y, at, way, outer.x, outer.y);
	if (inner.y <= inner.x) { return 1; }
	int count = 0;
	if (inner.x > outer.x) { parts[count++] = float2(outer.x, inner.x); }
	if (outer.y > inner.y) { parts[count++] = float2(inner.y, outer.y); }
	return count;
}

// clear is how much of a lamp far along toward reaches a point through
// every volume, each channel: a short march toward the lamp that stops
// at it, since what lies beyond it is behind the light and takes none;
// jittered by a smooth noise so the steps leave no tiers of shadow, two
// turns of it to a step in the world, whatever the volume's scale; each
// volume's depth its tint's times in each channel, as trace's
// paintedClear has it.
static float3 clear(World world, float3 point, float3 toward, float far) {
	float3 depth = 0;
	int shades = world.header->more.z;
	for (int index = 0; index < world.header->more.x; index++) {
		Volume volume = world.volumes[index];
		float3 at = inside(volume, point), way = turned(volume, toward);
		float thick = 0;
		float2 parts[2];
		int count = spans(volume, at, way, 0, far / volume.origin.w, parts);
		for (int leg = 0; leg < count; leg++) {
			float2 crossing = parts[leg];
			device const float *card = world.boxes + volume.grid.w;
			if (volume.info.z == ring_kind &&
			    ring_flat(card, at + way * crossing.x, at + way * crossing.y)) {
				float middle = (crossing.x + crossing.y) / beam_two;
				thick += density(world, volume, at + way * middle) *
				         (crossing.y - crossing.x);
				continue;
			}
			float stride = (crossing.y - crossing.x) / shades;
			float wobble = 2 / (stride * volume.origin.w);
			float jitter = 0.5f + 0.5f * gradient(77, point.x * wobble,
			                                      point.y * wobble,
			                                      point.z * wobble);
			for (int sample = 0; sample < shades; sample++) {
				float along = crossing.x + (sample + jitter) * stride;
				float3 sample_at = at + way * along;
				thick += density(world, volume, sample_at) * stride;
			}
		}
		depth += volume.tint.xyz * thick;
	}
	return exp(-depth);
}

// The sky table is defined below transport; this declaration lets the volume
// source sample the same radiance table as surfaces without moving its code.
static float3 sky_at(World world, float3 way);

// volume_sky_at pairs trace.volumeSkyAt. Equal-solid-angle antipodal samples
// cancel p=1/(4*pi) with the sphere's sampling density. There is no surface
// cosine. This is single scattering, with opaque shadows and medium extinction.
static float3 volume_sky_direct(World world, float3 point) {
	int count = world.header->environment.x;
	if (count <= 0) { return float3(0); }
	float3 sum(0);
	int pairs = count/2;
	for (int index=0;index<pairs;index++) {
		float y=(index+0.5f)/pairs;
		float radius=sqrt(1-y*y);
		float angle=index*2.39996322972865332f;
		float3 paired(radius*cos(angle),y,radius*sin(angle));
		for (int side=0;side<2;side++) {
			float3 way=side==0 ? paired : -paired;
			float3 incoming=sky_at(world,way);
			if (all(incoming==float3(0)) || blocked(world,point,way,nowhere)) { continue; }
			sum+=incoming*clear(world,point,way,nowhere);
		}
	}
	return sum/count;
}

// volume_sky_at uses a current-frame incident-source lattice when requested.
// A one-cell smooth boundary blend joins direct integration continuously.
// No density, opacity or explicit lamp is interpolated by this cache.
static float3 volume_sky_at(World world,float3 point) {
	int4 grid=world.header->sky_grid;
	float3 low=world.header->sky_low.xyz,high=world.header->sky_high.xyz;
	if (grid.w<0 || any(point<low) || any(point>high)) {
		return volume_sky_direct(world,point);
	}
	float3 index=(point-low)/(high-low)*float3(grid.xyz-1);
	float3 edge=min(index,float3(grid.xyz-1)-index);
	float blend=smoothstep(0.0f,1.0f,min(edge.x,min(edge.y,edge.z)));
	if (blend<=0) { return volume_sky_direct(world,point); }
	int3 first=clamp(int3(floor(index)),int3(0),grid.xyz-2);
	float3 part=index-float3(first);
	Volume lattice={};
	lattice.grid=grid;
	float3 cached=boxed_at(world,lattice,grid.w,3,first,part);
	if (blend>=1) { return cached; }
	return mix(volume_sky_direct(world,point),cached,blend);
}

// gas_phase is defined with the gases below; a lopsided mist takes it too.
static float gas_phase(float asymmetry, float cosine);

// lopsided is whether a volume that is not a gas scatters by a lobe: a
// mist whose asymmetry its row keeps in scatter.w, nought for the rest.
static bool lopsided(Volume volume) {
	return volume.air.x != gas_on && volume.scatter.w != beam_zero;
}

// Fast light's lamp store (gpu/reach.go says the design). The header's
// lighting says whether there is one: w is one more than where the
// beams' tables begin among the boxes, nought for none; z has a bit for
// each lamp that keeps a grid in every medium that has one.

// reach_stored is whether a step in this volume reads the lamp store:
// the header says there is one, and the volume's row holds its grid.
static bool reach_stored(World world, Volume volume) {
	return world.header->lighting.w > 0 && volume.lighting.w >= 0;
}

// reach_along is a narrow beam's share that gets a way along its axis,
// each channel, blended between the two places of its table round it.
static float3 reach_along(World world, int index, Lamp lamp, float along) {
	device const float *table = world.boxes + (world.header->lighting.w - 1) +
	                            index * reach_beam_cells * 3;
	float last = float(reach_beam_cells - 1);
	float place = clamp(along / lamp.extent.w, 0.0f, 1.0f) * last;
	int low = min(int(place), reach_beam_cells - 2);
	device const float *near = table + low * 3;
	return mix(float3(near[0], near[1], near[2]),
	           float3(near[3], near[4], near[5]), place - float(low));
}

// reach_within is a wide lamp's share that gets to a point of a medium's
// own space, each channel, blended between the eight cells round it of
// the lamp's grid: the grids lie one after another, in the lamps' order.
static float3 reach_within(World world, Volume volume, float3 local,
                           int index) {
	int before = world.header->lighting.z & ((1 << index) - 1);
	Volume grid = volume;
	grid.grid = volume.lighting;
	int cells = grid.grid.x * grid.grid.y * grid.grid.z;
	int first = grid.grid.w + popcount(before) * cells * 3;
	int3 low;
	float3 part;
	if (!boxed_part(grid, local, low, part)) {
		return float3(0);
	}
	return boxed_at(world, grid, first, 3, low, part);
}

// reach_of is the lamp store's answer for one lamp at a step: the share
// of its light that gets there past the things and through the volumes.
static float3 reach_of(World world, Volume volume, float3 local, int index,
                       Lamp lamp, float far) {
	if (lamp.extent.y > beam_zero) {
		return reach_along(world, index, lamp, far);
	}
	return reach_within(world, volume, local, index);
}

// scattered is trace's: the light a bit of a volume sends toward
// the eye, what it scatters of every lamp's light that reaches it, evenly
// or, for a lopsided mist, by its lobe over the even one,
// and its glow; a boxed medium's paint and shine read from its grids at
// the point of its own space, where it has them.
static float3 scattered_direct(World world, Volume volume, float3 point,
                        float3 local, float3 way) {
	Material made = world.materials[volume.info.w];
	float3 colour = made.colour.xyz, sum = made.glow.xyz;
	int3 low;
	float3 part;
	if (volume.info.z == ring_kind &&
	    ring_word(world.boxes + volume.grid.w, ring_glow) != whole_zero) {
		sum = ring_shine(world.boxes + volume.grid.w, local);
	}
	if (volume.info.z == boxed_kind &&
	    boxed_part(volume, local, low, part)) {
		if (volume.shone.x >= 0) {
			sum = boxed_at(world, volume, volume.shone.x, 3, low, part);
		}
		if (volume.shone.y >= 0) {
			colour = boxed_at(world, volume, volume.shone.y, 3, low, part);
		}
	}
	if (volume.thermal.y>1) {
		float r=length(local.xz),inner=volume.thin.x,outer=volume.thin.y;
		sum=float3(0);
		if (r>=inner && r<=outer && outer>inner) {
			float u=(r-inner)/(outer-inner)*(volume.thermal.y-1);
			int i=min(int(u),volume.thermal.y-2);
			device const float *a=world.boxes+volume.thermal.x+i*3;
			sum=mix(float3(a[0],a[1],a[2]),float3(a[3],a[4],a[5]),u-i);
			if (volume.thin.z>0 && r>inner) {
				float phi=atan2(local.z,local.x)+volume.thin.w*log(r/inner)+volume.phase.x*pow(r,-1.5f);
				float across=6*(r-inner)/(outer-inner);
				ulong seed=ulong(uint(volume.thermal.z))|(ulong(uint(volume.thermal.w))<<32);
				float n=fractal(seed,across,2*cos(phi),2*sin(phi),3);
				sum*=max(0.0f,1+volume.thin.z*2*n);
			}
		}
	}
	// A medium with no scattering sends only its own emission.
	// Skip light selection, geometry shadows and transmission in this case.
	if (all(colour == float3(0))) { return sum; }
	float phase = world.header->environment.x > 0 ? 1/(4*M_PI_F) : world.header->tuning.z;
	if (world.header->environment.x > 0) { sum += colour*volume_sky_at(world,point); }
	sum += colour * flooded(world, -way) * phase;
	Ask asks[lamps_asked];
	int count = asked(world, point, float3(0), asks);
	for (int index = 0; index < count; index++) {
		Lamp lamp = world.lamps[asks[index].lamp];
		Reached reached = lamp_light(lamp, point);
		if (all(reached.toward == 0)) {
			continue;
		}
		float3 through;
		if (reach_stored(world, volume)) {
			// fast light: the store answers for the things and the volumes
			through = reach_of(world, volume, local, asks[index].lamp, lamp,
			                   reached.far);
		} else {
			if (blocked(world, point, reached.toward, reached.far)) {
				continue;
			}
			through = clear(world, point, reached.toward, reached.far);
		}
		float3 reach = phase * through;
		if (lopsided(volume)) {
			reach *= gas_four_pi * gas_phase(volume.scatter.w,
			         dot(normalize(reached.toward), way));
		}
		sum += colour * reached.light * reach * asks[index].counts;
	}
	return sum;
}

// The optional grid caches the isotropic source term, not density or opacity;
// a lopsided mist's depends on the way, so it is never taken from a grid.
static float3 scattered(World world, Volume volume, float3 point,
                        float3 local, float3 way) {
 if (volume.lighting.w < 0 || lopsided(volume) ||
     world.header->lighting.w > 0) {
  // no grid; a lopsided mist, whose light depends on the way; or the grid
  // is fast light's lamp store, read lamp by lamp
  return scattered_direct(world,volume,point,local,way);
 }
 Volume grid=volume;
 grid.grid=volume.lighting;
 int3 low; float3 part;
 if (!boxed_part(grid,local,low,part)) {return float3(0);}
 return boxed_at(world,grid,grid.grid.w,3,low,part);
}

// a ray's crossing of one volume: where it enters and leaves, in the
// world's distances, and which volume
struct Crossing { float enter, leave; int volume; };

// crossings is trace's: every volume the ray passes through before far,
// nearest entry first. Each volume has at most two occupied shell legs;
// most_crossings is twice most_volumes, which the scene's own count
// sizes when the source is built.
static int crossings(World world, float3 at, float3 way, float far,
                     thread Crossing *found) {
	int count = 0;
	int volumes = min(world.header->more.x, most_volumes);
	for (int index = 0; index < volumes; index++) {
		Volume volume = world.volumes[index];
		float scale = volume.origin.w;
		float2 parts[2];
		int legs = spans(volume, inside(volume, at), turned(volume, way), 0,
		                 min(far, nowhere) / scale, parts);
		for (int leg = 0; leg < legs; leg++) {
			Crossing next = {parts[leg].x * scale, parts[leg].y * scale, index};
			int place = count++;
			while (place > 0 && found[place - 1].enter > next.enter) {
				found[place] = found[place - 1];
				place--;
			}
			found[place] = next;
		}
	}
	return count;
}

// shows is trace's: whether enough light still gets through to go on
static bool shows(float3 keep, float faint) {
	return keep.x > faint || keep.y > faint || keep.z > faint;
}

// through is trace's throughGain for a painted volume: what the ray
// gathers from the volumes before far, step by step, front to back,
// overlapping crossings' steps taken in turn by how far along each is.
struct Through { float3 keep, added; };

static float field_gain(World world,Volume volume,float3 point,float3 way) {
 if (world.gravity==nullptr) {
  float speed=volume.beam.w;
  if (!(speed>0 && speed<1)) return 1;
  float3 at=inside(volume,point),axis=volume.beam.xyz;
  float3 out=at-axis*dot(at,axis);
  float r=length(out);
  if (r==0) return 1;
  float3 flow=cross(axis,out/r);
  if (volume.flow.y>0) speed=min(.99f,speed*sqrt(volume.flow.y/r));
  float delta=sqrt(1-speed*speed)/(1-speed*dot(flow,normalize(turned(volume,-way))));
  return delta*delta*delta;
 }
 constant float4 *f=world.gravity;
 bool circular=volume.flow.x!=0 && volume.beam.w>0 && volume.beam.w<1;
 float4 emitter=kerr_emitter(field_in(f,point),f[0].x,f[0].y,circular);
 float denominator=dot(world.momentum,emitter);
 if (!isfinite(denominator) || denominator==0) return 0;
 float shift=world.energy/denominator;
 return min(50.0f,shift*shift*shift*shift);
}

// BeamMarch is this ray's conservative refinement region. It changes only
// sampling effort: light outside the region still gets ordinary integration.
struct BeamMarch { float enter, leave, step; };
static BeamMarch beam_march(World world, float3 at, float3 way, float far) {
 BeamMarch result = {far, beam_zero, beam_zero};
 if (world.header->look.y <= beam_zero) { return result; }
 for (int index = beam_zero; index < world.header->counts.w; index++) {
  Lamp lamp = world.lamps[index];
  if (lamp.extent.y <= beam_zero) { continue; }
  float3 end = lamp.at.xyz + lamp.toward.xyz * lamp.extent.w;
  float margin = (lamp.extent.y + lamp.extent.z * lamp.extent.w) *
                 beam_bound_radii;
  float3 low = min(lamp.at.xyz, end) - margin;
  float3 high = max(lamp.at.xyz, end) + margin;
  float2 x = slab(at.x, way.x, low.x, high.x);
  float2 y = slab(at.y, way.y, low.y, high.y);
  float2 z = slab(at.z, way.z, low.z, high.z);
  float enter = max(beam_zero, max(x.x, max(y.x, z.x)));
  float leave = min(far, min(x.y, min(y.y, z.y)));
  if (leave <= enter) { continue; }
  float step = lamp.extent.y / beam_radial_samples;
  if (result.step == beam_zero) {
   result = {enter, leave, step};
  } else {
   result.enter = min(result.enter, enter);
   result.leave = max(result.leave, leave);
   result.step = min(result.step, step);
  }
 }
 return result;
}

static int beam_splits(BeamMarch refine, float enter, float leave) {
 if (refine.step <= beam_zero || leave <= refine.enter ||
     enter >= refine.leave) { return int(beam_one); }
 return int(clamp(ceil((leave-enter)/refine.step), beam_one,
                  float(beam_max_splits)));
}

// gas_phase is scene/gas's PhaseAt: how much of the light a gas scatters
// goes at an angle whose cosine is cosine, per steradian, by rayleigh's
// lobe for an asymmetry of nought and else henyey and greenstein's.
static float gas_phase(float asymmetry, float cosine) {
	if (asymmetry == beam_zero) {
		return gas_rayleigh * (beam_one + cosine * cosine);
	}
	float sides = beam_one + asymmetry * asymmetry -
	              gas_twice * asymmetry * cosine;
	return (beam_one - asymmetry * asymmetry) /
	       (gas_four_pi * pow(sides, gas_lobe_power));
}

// gas_gathered is trace's gasStep's gather: the light a gas sends toward
// the eye from a point, before its scatter and its thickness, along a ray
// that runs way. it is every lamp's light that reaches the point, each
// weighed by the phase of the angle between the lamp's way of travel and
// the ray's, except a lamp from everywhere, which is taken evenly.
static float3 gas_gathered(World world, Volume volume, float3 point,
                           float3 way) {
	float3 sum = flooded(world, -way) * world.header->tuning.z;
	Ask asks[lamps_asked];
	int count = asked(world, point, float3(beam_zero), asks);
	for (int index = whole_zero; index < count; index++) {
		Reached reached = lamp_light(world.lamps[asks[index].lamp], point);
		if (all(reached.toward == beam_zero)) {
			continue;
		}
		if (blocked(world, point, reached.toward, reached.far)) {
			continue;
		}
		float3 reach = clear(world, point, reached.toward, reached.far);
		float cosine = dot(normalize(reached.toward), way);
		sum += reached.light * reach * asks[index].counts *
		       gas_phase(volume.scatter.w, cosine);
	}
	return sum;
}

// gas_step is trace's gasStep: one step of a ray through a gas, length
// long in the medium's units, from the density there. each channel keeps
// what beer's law leaves of it, the extinction at the channel's band; the
// step sends on its gather times the gas's scatter and its length, of
// what still showed.
static void gas_step(World world, Volume volume, float3 point, float3 way,
                     float length, thread Through &gathered) {
	float3 kept = exp(-volume.tint.xyz * length);
	float3 light = gas_gathered(world, volume, point, way) *
	               volume.scatter.xyz * length;
	gathered.added += light * gathered.keep;
	gathered.keep *= kept;
}

// crossing_density is a volume's density at a step's place, local in its own
// space. a ring's, where the whole crossing is flat, is read once at the
// crossing's middle and kept, since every step of it would read the same.
static float crossing_density(World world, Volume volume, Crossing crossing,
                              float3 at, float3 way, float3 local,
                              thread int &known, thread float &kept) {
	if (volume.info.z != ring_kind) {
		return density(world, volume, local);
	}
	if (known == flat_unasked) {
		device const float *card = world.boxes + volume.grid.w;
		float3 start = inside(volume, at + way * crossing.enter);
		float3 end = inside(volume, at + way * crossing.leave);
		known = ring_flat(card, start, end) ? flat_yes : flat_no;
		kept = density(world, volume, (start + end) / beam_two);
	}
	return known == flat_yes ? kept : density(world, volume, local);
}

static Through through(World world, float3 at, float3 way, float far) {
	Through gathered = {float3(1), float3(0)};
	Crossing found[most_crossings];
	int count = crossings(world, at, way, far, found);
	int next[most_crossings] = {};
	int parts[most_crossings] = {};
	int known[most_crossings] = {};
	float kept[most_crossings] = {};
	BeamMarch refine = beam_march(world, at, way, far);
	int strides = world.cuts > 0 ? world.cuts : world.header->more.y;
	float faint = world.header->tuning.y;
	while (shows(gathered.keep, faint)) {
		int taken = -1;
		float middle = 0, stride = 0;
		int divided = beam_one;
		for (int index = 0; index < count; index++) {
			if (next[index] >= strides) {
				continue;
			}
			Crossing crossing = found[index];
			float crossed = crossing.leave - crossing.enter;
			float size = crossed / strides;
			float begin = crossing.enter + next[index] * size;
			int split = beam_splits(refine, begin, begin+size);
			float middle_of = next[index] +
			                  (parts[index] + beam_half) / split;
			float along = crossing.enter + middle_of * size;
			if (taken < 0 || along < middle) {
				taken = index, middle = along, stride = size / split;
				divided = split;
			}
		}
		if (taken < 0) {
			break;
		}
		parts[taken]++;
		if (parts[taken] >= divided) {
			parts[taken] = beam_zero;
			next[taken]++;
		}
		Volume volume = world.volumes[found[taken].volume];
		float3 point = at + way * middle;
		float3 local = inside(volume, point);
		float thick = crossing_density(world, volume, found[taken], at, way,
		                               local, known[taken], kept[taken]);
		if (thick <= 0) {
			continue;
		}
		if (volume.air.x == gas_on) {
			gas_step(world, volume, point, way,
			         thick * stride / volume.origin.w, gathered);
			continue;
		}
		// each channel as trace's absorbed: its tint's times for a
		// tinted medium, alike for the rest
		float3 absorb = 1 - exp(-thick * stride / volume.origin.w *
		                        volume.tint.xyz);
		if (volume.info.z == ring_kind &&
		    ring_word(world.boxes + volume.grid.w, ring_glow) != whole_zero) {
			absorb = float3(ring_taken(thick * stride / volume.origin.w *
			                           volume.tint.x));
		}
        float3 gainPoint = at + way * ((found[taken].enter + found[taken].leave) * .5f);
        float3 light = scattered(world, volume, point, local, way) *
            field_gain(world, volume, world.gravity == nullptr ? point : gainPoint, way);
		gathered.added += light * gathered.keep * absorb;
		gathered.keep *= 1 - absorb;
	}
	return gathered;
}

// round_of is go's atan2(x, -z), the way round from minus z toward x:
// fast math's atan2 reads a nought below it as minus nought, and went
// round the wrong side by half a turn for a way with no z, as every
// flat floor's first look of the sky has. a way straight toward x or
// away from it is said outright, whatever sign its nought has.
static float round_of(float3 way) {
	if (way.z == 0) {
		return way.x < 0 ? -M_PI_F / 2 : M_PI_F / 2;
	}
	return atan2(way.x, -way.z);
}

// sky_at is the sky's light along a way: the go side's table of it,
// read between its four nearest points, round the horizon wrapping.
static float3 sky_at(World world, float3 way) {
	const int rows = 33, columns = 64;
	float up = asin(clamp(way.y, -1.0f, 1.0f));
	float round = round_of(way);
	round = round < 0 ? round + 2 * M_PI_F : round;
	float row = clamp((up + M_PI_F / 2) / M_PI_F * (rows - 1), 0.0f,
	                  float(rows - 1));
	float column = round / (2 * M_PI_F) * columns;
	int low = min(int(row), rows - 2);
	int left = int(column) % columns, right = (left + 1) % columns;
	float down = row - low, across = column - floor(column);
	float3 near = mix(world.sky[low * columns + left].xyz,
	                  world.sky[low * columns + right].xyz, across);
	float3 far = mix(world.sky[(low + 1) * columns + left].xyz,
	                 world.sky[(low + 1) * columns + right].xyz, across);
	return mix(near, far, down);
}

// lifted is where a ray leaving a surface at a point starts, off it
// along its normal so it does not meet the point it left: the lift's
// share of the world's spread or of the point's distance from the
// origin, whichever is more, as the tracer lifts it (trace/stage.go).
static float3 lifted(World world, float3 point, float3 normal) {
	float far = max(world.header->tuning.w, length(point));
	return point + normal * (world.header->tuning.x * far);
}

// skylit is the tracer's skylight without the surface's colour: the
// mean of the sky's light along sixteen looks round its normal, each
// dimmed by what stands in the way. the looks and the frame they are
// turned into are the tracer's, number for number (trace/skylight.go).
static float3 skylit(World world, float3 point, float3 normal,
                     int leaving_sphere) {
	const int looks = sky_looks;
	const float golden_turn = 2.39996322972865332f;
	// every look, in order; or, for fast light, this sample's share
	int taken = looks, begin = 0, apart = 1;
	if (world.shared.z > 0) {
		begin = world.shared.x, apart = world.shared.y;
		taken = world.shared.z;
	}
	float sign = normal.z < 0 ? -1.0f : 1.0f;
	float a = -1 / (sign + normal.z);
	float b = normal.x * normal.y * a;
	float3 first = float3(1 + sign * normal.x * normal.x * a, sign * b,
	                      -sign * normal.x);
	float3 second = float3(b, sign + normal.y * normal.y * a, -normal.y);
	float3 from = lifted(world, point, normal);
	float3 sum = float3(0);
	for (int turn_of = 0; turn_of < taken; turn_of++) {
		int look = (begin + turn_of * apart) % looks;
		float share = (look + 0.5f) / looks;
		float out = sqrt(share), turn = golden_turn * look;
		float3 way = first * (out * cos(turn)) +
		             second * (out * sin(turn)) + normal * sqrt(1 - share);
		if (blocked(world, from, way, nowhere, leaving_sphere)) {
			continue;
		}
		sum += sky_at(world, way) * clear(world, from, way, nowhere);
	}
	return sum / taken;
}

// sun_spread is film.Sun.Spread: the i-th of eight ways across a sun's
// disc of angular width size about middle, on a spiral turning by the
// golden angle, each at the root of its place along the eight times the
// disc's radius out.
static float3 sun_spread(float3 middle, float size, int i) {
	float3 up = abs(middle.y) >= 0.9f ? float3(1, 0, 0) : float3(0, 1, 0);
	float3 across = normalize(cross(middle, up));
	float3 other = cross(middle, across);
	float out = size / 2 * sqrt((float(i) + 0.5f) / 8);
	float turn = 2.399963229728653f * float(i);
	float3 aside = across * cos(turn) + other * sin(turn);
	return normalize(middle * cos(out) + aside * sin(out));
}

// unhidden is trace's unhidden: how much of a lamp's light gets from a
// point to it, through whatever stands between; for a sun drawn as a
// disc, its extent's x its width, the share unhidden along the eight
// ways across it.
static float3 unhidden(World world, Lamp lamp, float3 from, float3 toward,
                       float far, int leaving_sphere) {
	if (lamp.at.w < 0 && lamp.extent.x > 0) {
		float3 middle = normalize(lamp.at.xyz);
		float3 sum = float3(0);
		for (int i = 0; i < 8; i++) {
			float3 way = sun_spread(middle, lamp.extent.x, i);
			if (!blocked(world, from, way, far, leaving_sphere)) {
				sum += clear(world, from, way, far);
			}
		}
		return sum / 8;
	}
	if (blocked(world, from, toward, far, leaving_sphere)) {
		return float3(0);
	}
	return clear(world, from, toward, far);
}

// falling is the light that falls on a matte surface, for its colour to
// be taken by: every lamp's that reaches it past the things and the
// volumes, over pi, and the sky's. trace's surfaceAt is the surface's
// glow and its colour times this, without the mirror.
static float3 falling(World world, float3 point, float3 normal,
                      int leaving_sphere) {
	float3 sum = skylit(world, point, normal, leaving_sphere);
	sum += flooded(world, normal);
	float3 from = lifted(world, point, normal);
	Ask asks[lamps_asked];
	int count = asked(world, point, normal, asks);
	for (int index = 0; index < count; index++) {
		Lamp lamp = world.lamps[asks[index].lamp];
		Reached reached = lamp_light(lamp, point);
		reached.light *= asks[index].counts;
		float square = dot(normal, reached.toward);
		if (square <= 0) {
			continue;
		}
		float3 reach = square * unhidden(world, lamp, from, reached.toward,
		                                 reached.far, leaving_sphere);
		// matte: its colour over pi times the irradiance, as the cpu's
		sum += reached.light * reach / M_PI_F;
	}
	return sum;
}

// mirrors is thing.Material.Mirrors: the share a glossy surface
// mirrors at a look, by schlick's fresnel.
static float mirrors(Material made, float cosine) {
	float gloss = made.mirror.x;
	if (gloss <= 0) {
		return 0;
	}
	float index = made.mirror.y > 0 ? made.mirror.y : 1.5f;
	float straight = (index - 1) / (index + 1);
	straight *= straight;
	float away = 1 - clamp(cosine, 0.0f, 1.0f);
	return gloss * (straight + (1 - straight) * pow(away, 5.0f));
}

// fresnel is maths.Fresnel: the share of unpolarised light a flat
// boundary reflects, from the cosine of the light's angle to the normal
// and the index of the medium it goes into over the one it comes from.
// Past the critical angle nothing is transmitted and the share is one.
static float fresnel(float cosine, float index) {
	cosine = clamp(cosine, 0.0f, 1.0f);
	float sine2 = (1 - cosine * cosine) / (index * index);
	if (sine2 >= 1) {
		return 1;
	}
	float inside = sqrt(1 - sine2);
	float across = (cosine - index * inside) / (cosine + index * inside);
	float along = (index * cosine - inside) / (index * cosine + inside);
	return (across * across + along * along) / 2;
}

// most_held and slips are trace's mostHeld and slips: how many clear
// media a look can be inside at once, and how many clear surfaces that
// are no boundary it may pass straight through. each is twice what it
// can bend through, as trace picked them.
constexpr constant int most_held = 2 * refractions;
constexpr constant int slips = 2 * refractions;

// Held is trace's held: a clear medium a look is inside, its index and
// its rank among clear things that overlap. things of one index and one
// rank are one medium.
struct Held {
	float index;
	int rank;
};

// Inside is the clear media a look is in, in the order it entered
// them, and how many of them there are: trace's look.inside and
// look.holding.
struct Inside {
	Held held[most_held];
	int holding;
};

// Boundary is trace's boundary: what a clear surface is to a look that
// meets it. relative is the index of what the light goes into over the
// index of what it comes from, which fresnel asks; ratio is the other
// way up, which snell asks. each is one division of the two indices
// and neither is the other's inverse worked again, so a thing in the
// air bends and splits by the very numbers it always did. real is
// whether the surface is a boundary at all, and entering whether the
// look is going into the thing.
struct Boundary {
	float relative;
	float ratio;
	bool real;
	bool entering;
};

// held_of is trace's of: the medium a clear material is.
static Held held_of(Material made) {
	return {made.mirror.y > 0 ? made.mirror.y : 1.5f, made.info.z};
}

// same is whether two media are one: one index and one rank.
static bool same(Held one, Held other) {
	return one.index == other.index && one.rank == other.rank;
}

// medium is trace's look.medium: what the look is in. of the clear
// media it is inside, the one of the highest rank, and among equals the
// one it entered last; inside none, the air, of index one.
static Held medium(Inside inside) {
	Held now = {1.0f, 0};
	for (int at = 0; at < inside.holding; at++) {
		if (at == 0 || inside.held[at].rank >= now.rank) {
			now = inside.held[at];
		}
	}
	return now;
}

// without is trace's look.without: the look no longer inside a medium,
// the rest in their order. it leaves it once.
static Inside without(Inside inside, Held one) {
	int kept = 0;
	bool left = false;
	for (int at = 0; at < inside.holding; at++) {
		if (!same(inside.held[at], one) || left) {
			inside.held[kept] = inside.held[at];
			kept++;
			continue;
		}
		left = true;
	}
	inside.holding = kept;
	return inside;
}

// meets is trace's look.meets: the boundary a clear surface of a thing
// is to a look inside these media. leaving a thing the look is in, the
// surface is a boundary only if that thing was the medium; entering
// one, it is none where a medium the look is already in outranks it.
static Boundary meets(Inside inside, Material made) {
	Held one = held_of(made);
	Held before = medium(inside);
	bool within = false;
	for (int at = 0; at < inside.holding; at++) {
		within = within || same(inside.held[at], one);
	}
	if (within) {
		Held after = medium(without(inside, one));
		return {after.index / before.index, before.index / after.index,
		        same(before, one), false};
	}
	if (inside.holding > 0 && one.rank < before.rank) {
		return {1, 1, false, true};
	}
	return {one.index / before.index, before.index / one.index, true, true};
}

// splits is thing.Material.SplitsBetween: the share a clear surface
// reflects at a look, by fresnel's equations for what is on each side
// of it, relative being the index of what the light goes into over the
// index of what it comes from. All of it past the critical angle, at
// any gloss.
static float splits(Material made, float cosine, float relative) {
	float share = fresnel(cosine, relative);
	if (share >= 1) {
		return 1;
	}
	return clamp(made.mirror.x, 0.0f, 1.0f) * share;
}

// conductor is thing.Conductor: the share a conductor of complex index
// n + ik reflects at a look, fresnel's s and p averaged.
static float conductor(float cosine, float n, float k) {
	float cos2 = cosine * cosine;
	float sin2 = 1 - cos2;
	float n2 = n * n, k2 = k * k;
	float t0 = n2 - k2 - sin2;
	float ab = sqrt(t0 * t0 + 4 * n2 * k2);
	float t1 = ab + cos2;
	float a = sqrt(max(0.0f, (ab + t0) / 2));
	float t2 = 2 * cosine * a;
	float s = (t1 - t2) / (t1 + t2);
	float t3 = cos2 * ab + sin2 * sin2;
	float t4 = t2 * sin2;
	float p = s * (t3 - t4) / (t3 + t4);
	return (s + p) / 2;
}

// dielectric is thing.Dielectric: fresnel's share for a clear surface
// of index n, exactly; all of it past the critical angle.
static float dielectric(float cosine, float n) {
	return fresnel(cosine, n);
}

// reflects is thing.Material.Reflects: what each colour a surface
// mirrors at a look. a metal's by its complex index, under its film when
// it has one; any other's schlick's one share for all three.
static float3 reflects(Material made, float cosine) {
	if (made.film.x <= 0) {
		return float3(mirrors(made, cosine));
	}
	float gloss = min(made.mirror.x, 1.0f);
	if (gloss <= 0) {
		return float3(0);
	}
	float look = clamp(cosine, 0.0f, 1.0f);
	float3 share;
	for (int c = 0; c < 3; c++) {
		float n = made.metal_n[c], k = made.metal_k[c];
		if (made.film.y <= 0) {
			share[c] = gloss * conductor(look, n, k);
			continue;
		}
		float within = made.film.y;
		float inside = sqrt(max(0.0f,
			1 - (1 - look * look) / (within * within)));
		float front = dielectric(look, within);
		float passes = exp(-made.absorbs[c] * made.film.z / inside);
		float twice = passes * passes;
		float below = conductor(inside, n / within, k / within);
		share[c] = gloss * (front + (1 - front) * (1 - front) * below *
			twice / (1 - front * below * twice));
	}
	return share;
}

// crinkled is thing.Crinkle.Crinkled: a wrinkled surface's normal at a
// point, seen along look, tilted against the slope of three octaves of
// noise read a tenth of a wrinkle apart; left as it was where the tilt
// would turn the face from the eye.
static float3 crinkled(Material made, float3 point, float3 normal,
                       float3 look) {
	float size = made.crinkle.x, slope = made.crinkle.y;
	if (!(size > 0) || !(slope > 0)) {
		return normal;
	}
	float3 at = point / size;
	ulong seed = ulong(made.crinkle.z);
	float step = 0.1f;
	float here = fractal(seed, at.x, at.y, at.z, 3);
	float3 rise = float3(fractal(seed, at.x + step, at.y, at.z, 3) - here,
	                     fractal(seed, at.x, at.y + step, at.z, 3) - here,
	                     fractal(seed, at.x, at.y, at.z + step, 3) - here) /
	              step;
	float3 across = rise - normal * dot(rise, normal);
	float3 tilted = normalize(normal - across * slope);
	if (!(dot(tilted, look) > 0.05f) || isnan(tilted.x)) {
		return normal;
	}
	return tilted;
}

// most is the largest of a colour's three: whether it mirrors at all,
// and, for one share in all three, that share.
static float most(float3 colour) {
	return max(colour.x, max(colour.y, colour.z));
}

// shares is trace's colour-wise share at a look: a clear surface that is
// not a metal reflects by fresnel's share for what is on each side of
// it, the same in all three colours, and nothing at all where it is no
// boundary; anything else by reflects.
static float3 shares(Material made, float clearness, float cosine,
                     Boundary at) {
	if (clearness > 0 && made.film.x <= 0) {
		return at.real ? float3(splits(made, cosine, at.relative)) : float3(0);
	}
	return reflects(made, cosine);
}

// smith is thing's smith: the share of a rough surface's tiny faces a
// way of that cosine sees unshadowed.
static float smith(float cosine, float a2) {
	return 2 * cosine / (cosine + sqrt(a2 + (1 - a2) * cosine * cosine));
}

// microfacet is thing.Microfacet: the GGX lobe a lamp's highlight
// takes, per unit of irradiance, before fresnel's share.
static float microfacet(float facing, float lit, float viewed, float alpha) {
	if (!(facing > 0 && lit > 0 && viewed > 0 && alpha > 0)) {
		return 0;
	}
	float a2 = alpha * alpha;
	float spread = facing * facing * (a2 - 1) + 1;
	float faces = a2 / (M_PI_F * spread * spread);
	return faces * smith(lit, a2) * smith(viewed, a2) / (4 * viewed);
}

// glinted is trace's glint: the highlights a surface's lamps leave on
// it, seen along look, each lamp's light that reaches it by the lobe of
// its satin and the share of each colour it mirrors at the half way.
// not the surface's colour.
static float3 glinted(World world, float3 point, float3 normal, float3 look,
                      Material made, int leaving_sphere) {
	float satin = min(made.mirror.w, 1.0f);
	float viewed = dot(normal, look);
	if (!(satin > 0) || !(viewed > 0)) {
		return float3(0);
	}
	float alpha = satin * satin;
	float3 sum = float3(0);
	float3 from = lifted(world, point, normal);
	Ask asks[lamps_asked];
	int count = asked(world, point, normal, asks);
	for (int index = 0; index < count; index++) {
		Lamp lamp = world.lamps[asks[index].lamp];
		Reached reached = lamp_light(lamp, point);
		reached.light *= asks[index].counts;
		float square = dot(normal, reached.toward);
		if (square <= 0) {
			continue;
		}
		float3 half_way = normalize(reached.toward + look);
		float lobe = microfacet(dot(normal, half_way), square, viewed, alpha);
		if (!(lobe > 0)) {
			continue;
		}
		float3 reach = unhidden(world, lamp, from, reached.toward,
		                        reached.far, leaving_sphere);
		sum += reached.light * reach * reflects(made, dot(half_way, look)) *
		       lobe;
	}
	return sum;
}

// snell is trace's refracted: the way a clear surface bends an
// arriving ray, where the normal faces the ray and ratio is the index
// of what the ray comes from over the index of what it goes into. Past
// the critical angle the bend is the reflection itself, and turned
// says so.
static float3 snell(float3 way, float3 normal, float ratio,
                    thread bool *turned) {
	float cosine = -dot(normal, way);
	float squared = ratio * ratio * (1 - cosine * cosine);
	*turned = squared >= 1;
	if (*turned) {
		return normalize(way - normal * 2 * dot(way, normal));
	}
	float along = ratio * cosine - sqrt(1 - squared);
	return normalize(way * ratio + normal * along);
}

// turns is whether snell sends the ray back whole: total internal
// reflection, as trace's refracted says it.
static bool turns(float3 way, float3 normal, float ratio) {
	float cosine = -dot(normal, way);
	return ratio * ratio * (1 - cosine * cosine) >= 1;
}

// Painted is a place a ray met that the cpu paints: its material, where
// in the thing's own space, how far the ray came over the thing's scale,
// for the footprint, and the light its paint and its shine are each to
// be taken by, as far as it reaches the eye. a material of minus one is
// none.
struct Painted {
	int material;
	float3 local;
	float reach;
	float3 colour, glow;
};

// unpainted is no place to paint
constant Painted unpainted = {-1, float3(0), 0, float3(0), float3(0)};

// Seen is what a ray brings back, with the sky left out: the light, how
// much of the pixel is covered, how far, and where the ray went into the
// sky and how much of the sky's light counts, for the cpu to add; and
// the place it met and the place its mirrored ray met, for the cpu to
// paint.
struct Seen {
	float3 light;
	float cover, depth;
	float3 weight, sky;
	Painted met, mirrored, passed;
};

// to_sky is a ray that met nothing: all of the sky's light counts
static Seen to_sky(float3 way) {
	return {float3(0), 0, nowhere, float3(1), way, unpainted, unpainted,
	        unpainted};
}

// dimmed is a place to paint with its light taken by a share
static Painted dimmed(Painted place, float3 by) {
	place.colour *= by, place.glow *= by;
	return place;
}

// surfaced is what a ray brings back from a surface it met, without the
// mirror: its glow and its colour times the light falling on it, or,
// where the material is painted or shining, the place for the cpu to
// paint and the light to paint it by.
static Seen surfaced(World world, Hit hit, float3 point, float3 normal) {
	Material made = world.materials[hit.material];
	bool painted = made.info.x != 0, shining = made.info.y != 0;
	// Pure emission has no diffuse response. In particular, the Moon
	// does not need terrestrial shadow queries at astronomical coordinates.
	float3 fell = float3(0);
	if (painted || any(made.colour.xyz != 0)) {
		fell = falling(world, point + hit.lift, normal, hit.sphere_index);
	}
	float3 light = (painted ? float3(0) : made.colour.xyz * fell) +
	               (shining ? float3(0) : made.glow.xyz);
	Painted place = unpainted;
	if (painted || shining) {
		place = {hit.material, hit.local, hit.distance / hit.scale,
		         painted ? fell : float3(0),
		         shining ? float3(1) : float3(0)};
	}
	return {light, 1, hit.distance, float3(0), float3(0), place, unpainted,
	        unpainted};
}

// behind is trace's end of shade: the volumes in front of what the ray
// met, adding their light over it and dimming it, and the sky with it.
static Seen behind(World world, float3 at, float3 way, Seen seen) {
	if (world.header->more.x == 0) {
		return seen;
	}
	Through gathered = through(world, at, way, seen.depth);
	float3 keep = gathered.keep;
	float most = max(keep.x, max(keep.y, keep.z));
	seen.light = seen.light * gathered.keep + gathered.added;
	seen.cover = 1 - (1 - seen.cover) * most;
	seen.weight *= gathered.keep;
	seen.met = dimmed(seen.met, keep);
	seen.mirrored = dimmed(seen.mirrored, keep);
	seen.passed = dimmed(seen.passed, keep);
	return seen;
}

// facing is a hit's normal turned toward the eye, as trace turns it
static float3 facing(Hit hit, float3 way) {
	return dot(hit.normal, way) > 0 ? -hit.normal : hit.normal;
}

// shade_plain is trace's shade at its last bounce: what a ray meets,
// lit, and the volumes in front, with no mirror.
static Seen shade_plain(World world, float3 at, float3 way,
                        int leaving_sphere) {
	Hit hit = first(world, at, way, 0, leaving_sphere);
	if (hit.material < 0) {
		return behind(world, at, way, to_sky(way));
	}
	float3 point = fma(way, hit.distance, at);
	Material made = world.materials[hit.material];
	float3 normal = crinkled(made, point, facing(hit, way), -way);
	Seen seen = surfaced(world, hit, point, normal);
	seen.light += glinted(world, point + hit.lift, normal, -way, made,
	                      hit.sphere_index);
	return behind(world, at, way, seen);
}

// shade is trace's shade at the first bounce: as shade_plain, and a
// glossy surface mirrors what shade_plain sees along the mirrored ray.
// metal has no recursion, so the bounce is a second function, and a
// clear surface's bends are a loop: fresnel's share mirrors first,
// Through's share of the rest bends by snell through at most
// refractions surfaces, total internal reflection bending back, each
// leg's volumes over what it brings. the legs follow trace's bent
// looks: no mirror off a leg, fresnel's toll paid and not returned. a
// leg into the sky reads the kernel's table, as skylit does; a clear
// surface part way along paints nothing, only the chain's end does.
// primary_behind leaves the foreground transport out of a surface plate.
// Secondary paths still see volumes, including surface shadows.
static Seen primary_behind(World world, float3 at, float3 way, Seen seen) {
#if defined(SLAYER_COMPOSITE_PASS) && SLAYER_COMPOSITE_PASS == 1
	return seen;
#else
	return behind(world, at, way, seen);
#endif
}

static Seen shade(World world, float3 at, float3 way) {
	Hit hit = first(world, at, way, 0);
	if (hit.material < 0) {
		return primary_behind(world, at, way, to_sky(way));
	}
	float3 point = fma(way, hit.distance, at);
	Material made = world.materials[hit.material];
	float3 normal = crinkled(made, point, facing(hit, way), -way);
	Seen seen = surfaced(world, hit, point, normal);
	float clearness = clamp(made.mirror.z, 0.0f, 1.0f);
	// every look starts in the air, inside nothing, as trace's does
	Inside inside;
	inside.holding = 0;
	Boundary met = meets(inside, made);
	float3 share = shares(made, clearness, -dot(normal, way), met);
	if (most(share) > 0 || clearness > 0) {
		float3 keep = (1 - share) * (1 - clearness);
		seen.light *= keep;
		seen.met = dimmed(seen.met, keep);
	}
	// the lamps' highlights, over what it keeps, as trace adds them
	seen.light += glinted(world, point + hit.lift, normal, -way, made,
	                      hit.sphere_index);
	if (most(share) > 0) {
		float3 bounce = way - normal * 2 * dot(way, normal);
		float3 mirrored = normalize(bounce);
		float3 from = lifted(world, point, normal);
		Seen bounced = shade_plain(world, from, mirrored, hit.sphere_index);
		seen.light += bounced.light * share;
		seen.weight = bounced.weight * share;
		seen.sky = bounced.sky;
		seen.mirrored = dimmed(bounced.met, share);
	}
	if (clearness > 0) {
		float3 carry = (1 - share) * clearness;
		// crossed is how many clear surfaces the look has bent through
		// and slipped how many it has passed that were no boundary, as
		// trace counts them; leg_at is what the leg's own surface is to
		// the look, worked out when the look met it
		int crossed = 0, slipped = 0;
		float3 leg_way = way, leg_normal = normal, leg_point = point;
		Material leg_made = made;
		Boundary leg_at = met;
		for (int leg = 0; leg < refractions + slips; leg++) {
			// a boundary bends the look, or past the critical angle
			// turns it back whole; a surface that is none lets it
			// straight on. bent or straight it is inside the thing now
			// if it was not, and out of it if it was; turned back it is
			// inside whatever it was inside
			bool turned = false;
			float3 bent = leg_way;
			if (leg_at.real) {
				bent = snell(leg_way, leg_normal, leg_at.ratio, &turned);
				crossed++;
			} else {
				slipped++;
			}
			if (!turned) {
				Held one = held_of(leg_made);
				if (leg_at.entering) {
					inside.held[inside.holding] = one;
					inside.holding++;
				} else {
					inside = without(inside, one);
				}
			}
			float3 from = lifted(world, leg_point,
			                     turned ? leg_normal : -leg_normal);
			Hit next = first(world, from, bent, 0);
			if (next.material < 0) {
				Through gathered = through(world, from, bent, nowhere);
				seen.light += carry * (gathered.added +
					gathered.keep * sky_at(world, bent));
				break;
			}
			Through gathered = through(world, from, bent, next.distance);
			float3 next_point = fma(bent, next.distance, from);
			Material next_made = world.materials[next.material];
			float3 next_normal = crinkled(next_made, next_point,
				facing(next, bent), -bent);
			float next_clear = clamp(next_made.mirror.z, 0.0f, 1.0f);
			Boundary next_at = meets(inside, next_made);
			float3 next_share = shares(next_made, next_clear,
				-dot(next_normal, bent), next_at);
			// trace's follows: whether the look can go on past this
			// surface. a boundary costs one of its bends, a surface that
			// is none one of its slips, and a thing it enters must have
			// room among the things it is inside
			bool follows = next_clear > 0 &&
				(next_at.real ? crossed < refractions : slipped < slips) &&
				!(next_at.entering && inside.holding >= most_held);
			// a surface that turns the ray back whole passes it on
			// whole, to bend as a reflection at the next leg
			bool next_turns = follows && next_at.real &&
				turns(bent, next_normal, next_at.ratio);
			if (!follows) {
				// where the look cannot go on the surface is opaque, as
				// trace's, and one there that reflects all of what
				// arrives lights as itself
				if (next_clear > 0 && most(next_share) >= 1) {
					next_share = float3(0);
				}
				next_clear = 0;
			}
			Seen deep = surfaced(world, next, next_point, next_normal);
			float3 keep = (1 - next_share) * (1 - next_clear);
			float3 shone = glinted(world, next_point + next.lift,
				next_normal, -bent, next_made, next.sphere_index);
			seen.light += carry * gathered.added +
				carry * gathered.keep * deep.light * keep +
				carry * gathered.keep * shone;
			if (next_clear == 0) {
				seen.passed = dimmed(deep.met,
					carry * gathered.keep * keep);
				break;
			}
			carry *= gathered.keep *
				(next_turns ? 1 : (1 - next_share) * next_clear);
			leg_way = bent;
			leg_normal = next_normal;
			leg_point = next_point;
			leg_made = next_made;
			leg_at = next_at;
		}
	}
	return primary_behind(world, at, way, seen);
}

// record writes a place to paint into a sample's numbers, at a record's
// place among them
static void record(device float *sample, int which, Painted place) {
	device float *at = sample + per_base + per_record * which;
	at[0] = float(place.material);
	at[1] = place.local.x, at[2] = place.local.y, at[3] = place.local.z;
	at[4] = place.reach;
	at[5] = place.colour.x, at[6] = place.colour.y;
	at[7] = place.colour.z;
	at[8] = place.glow.x, at[9] = place.glow.y, at[10] = place.glow.z;
}

// look is film.Pinhole.Ray through film.Look: the ray through a pixel's
// middle, taken out of the eye's space into the world's.
static float3 look(constant Header *header, uint2 pixel) {
	float4 lens = header->lens;
	// Center in integer pixel units before scaling, and normalize once
	// after the orthonormal camera rotation. Fine foliage exposes the
	// cancellation and extra rounding of two normalizations.
	float across = (2 * float(pixel.x) + 1 - lens.z) * (lens.x / lens.z);
	float up = (lens.w - 2 * float(pixel.y) - 1) * (lens.x / (lens.w * lens.y));
	return precise::normalize(fma(header->right.xyz, across,
	                             fma(header->up.xyz, up, -header->back.xyz)));
}

// An exposure: the primary ray of a sample where the shutter is open or
// the lens has an aperture. gpu/exposure.go says the design. Nothing
// below is asked with the shutter closed and no aperture; look makes
// that ray, as it always did.

// Eye is where a sample looks from: its place and its three axes.
struct Eye {
	float3 origin;
	float3 right;
	float3 up;
	float3 back;
};

// Primary is a sample's first ray: where it starts and its way.
struct Primary {
	float3 from;
	float3 way;
};

// shutter_time is a sample's time in the open shutter, from nought up to
// one. The samples of a pixel are counted along its rows; each takes one
// of as many equal shares of the shutter, the count turned by a whole
// number of the pixel's own, at a place in the share of the sample's own.
// One division makes the time, of the share's number by their count.
static float shutter_time(constant Header *header, uint2 sample) {
	uint side = uint(header->exposure.z);
	int count = int(side * side);
	uint2 pixel = sample / side, within = sample % side;
	int index = int(within.y * side + within.x);
	ulong seed = ulong(uint(header->exposure.w));
	float drawn = unit_of(seed, int(pixel.x), int(pixel.y), shutter_turn);
	int share = (index + int(drawn * hash_whole)) % count;
	float along = unit_of(seed, int(pixel.x), int(pixel.y),
	                      shutter_first + index);
	return (float(share) + along) / float(count);
}

// eye_at is the eye at a time in the open shutter: its place on the
// straight line between where the shutter opens and where it closes, and
// its axes the two eyes' mixed and made square again. Exact for an eye
// that slides; a small-angle approximation for one that turns.
static Eye eye_at(constant Header *header, float when) {
	Eye eye;
	eye.origin = mix(header->origin.xyz, header->close_origin.xyz, when);
	float3 back = mix(header->back.xyz, header->close_back.xyz, when);
	float3 right = mix(header->right.xyz, header->close_right.xyz, when);
	eye.back = precise::normalize(back);
	eye.right = precise::normalize(right - eye.back * dot(right, eye.back));
	eye.up = cross(eye.back, eye.right);
	return eye;
}

// square_point is lens.squarePoint: where an index of the r2 sequence
// lies along one side of the unit square, from minus one to one. The
// step and the half are shares of the square in 64 bits, so the product
// wraps round the square exactly at any index; a float keeps the top 24.
static float square_point(ulong step, ulong index) {
	ulong along = step * index + r2_half;
	return float(along >> hash_drop) / hash_whole * 2 - 1;
}

// aperture_point is lens.Thin.through: the point on the open aperture a
// sample takes, the r2 sequence's point for its index, row by row across
// the whole picture, carried onto the round opening by the concentric
// map. Each angle is one division, as lens.concentric makes it.
static float3 aperture_point(constant Header *header, uint2 sample) {
	ulong index = ulong(sample.y) * ulong(uint(header->lens.z)) +
	              ulong(sample.x);
	float abscissa = square_point(r2_across, index);
	float ordinate = square_point(r2_up, index);
	if (abscissa == 0 && ordinate == 0) {
		return float3(0);
	}
	float radius = ordinate;
	float turn = quarter_turn - eighth_turn * (abscissa / ordinate);
	if (fabs(abscissa) > fabs(ordinate)) {
		radius = abscissa;
		turn = eighth_turn * (ordinate / abscissa);
	}
	radius *= header->optics.x;
	return float3(radius * precise::cos(turn), radius * precise::sin(turn),
	              0);
}

// exposed is a sample's ray through an open shutter, an aperture, or
// both. In the eye's space a pinhole's ray starts at the eye, as look
// makes it; a thin lens's is lens.Thin.Ray, from its point on the
// aperture toward the point at the focus the sample's middle sees sharp.
// The ray is then taken out of the eye at the sample's time, and made
// one long once, after the turn.
static Primary exposed(constant Header *header, uint2 sample) {
	Eye eye = {header->origin.xyz, header->right.xyz, header->up.xyz,
	           header->back.xyz};
	if (header->exposure.x != 0) {
		eye = eye_at(header, shutter_time(header, sample));
	}
	float4 lens = header->lens;
	float3 start = float3(0);
	float3 toward = float3(
		(2 * float(sample.x) + 1 - lens.z) * (lens.x / lens.z),
		(lens.w - 2 * float(sample.y) - 1) * (lens.x / (lens.w * lens.y)),
		-1);
	if (header->exposure.y != 0) {
		float4 optics = header->optics;
		start = aperture_point(header, sample);
		float3 sharp = float3(
			(2 * float(sample.x) + 1 - lens.z) / (2 * lens.z) * optics.z,
			(lens.w - 2 * float(sample.y) - 1) / (2 * lens.w) * optics.w,
			-optics.y);
		toward = sharp - start;
	}
	Primary made;
	made.from = eye.origin + eye.right * start.x + eye.up * start.y;
	made.way = precise::normalize(eye.right * toward.x +
	                              eye.up * toward.y + eye.back * toward.z);
	return made;
}

// sky_share is a sample's share of the sky's looks, for fast light's
// sky-looks: which look it takes first, how far apart its looks are and
// how many it takes; no looks where the header's lighting asks for all.
// Its x is then the samples along a pixel's side. The looks are dealt
// round a pixel's samples as cards are: with sixteen looks and four
// samples each takes four and the pixel takes every look once. Each
// pixel starts the deal at a look of its own.
static int3 sky_share(constant Header *header, uint2 sample) {
	int side = header->lighting.w > 0 ? header->lighting.x : 0;
	if (side <= 1) {
		return int3(0);
	}
	int count = side * side;
	uint2 pixel = sample / uint(side), within = sample % uint(side);
	int index = int(within.y) * side + int(within.x);
	int dealt = int(place_share(float3(float(pixel.x), float(pixel.y),
	                                   beam_zero)) * float(sky_looks));
	int apart = min(count, sky_looks);
	return int3((dealt + index) % sky_looks, apart,
	            max(sky_looks / count, 1));
}

// Trace and smoke preparation share this scene ABI and initializer. Keep
// their optional native buffers beside the common bindings, in one place.
#define SLAYER_SCENE_BUFFERS \
 constant Header *header [[buffer(0)]], \
 device const Sphere *spheres [[buffer(1)]], \
 device const Triangle *triangles [[buffer(2)]], \
 device const Node *nodes [[buffer(3)]], \
 device const Material *materials [[buffer(4)]], \
 device const Lamp *lamps [[buffer(5)]], \
 device const Volume *volumes [[buffer(6)]], \
 device const float *baked [[buffer(7)]], \
 device const Tree *trees [[buffer(8)]], \
 device const Instance *instances [[buffer(9)]], \
 device const Node *tops [[buffer(10)]], \
 device const float4 *sky [[buffer(11)]], \
 device const Bend *bends [[buffer(12)]], \
 device const float *rays [[buffer(13)]], \
 device const float *boxes [[buffer(14)]], \
 device const Field *fields [[buffer(15)]], \
 device const float *heights [[buffer(16)]],

#ifdef SLAYER_NATIVE_MESHES
#define SLAYER_SAMPLE_BUFFERS \
 device const int *firsts [[buffer(17)]], \
 device float *out [[buffer(18)]], \
 constant uint2 &origin [[buffer(19)]],
#define SLAYER_FIRSTS_VALUE , firsts
#define SLAYER_ACCELERATION_INDEX 20
#else
#define SLAYER_SAMPLE_BUFFERS \
 device float *out [[buffer(17)]], \
 constant uint2 &origin [[buffer(18)]],
#define SLAYER_FIRSTS_VALUE
#define SLAYER_ACCELERATION_INDEX 19
#endif

#ifdef SLAYER_NATIVE_SHADOWS
#define SLAYER_ACCELERATION_BUFFER \
 , acceleration_structure<instancing> accelerated \
   [[buffer(SLAYER_ACCELERATION_INDEX)]]
#define SLAYER_ACCELERATION_VALUE , accelerated
#else
#define SLAYER_ACCELERATION_BUFFER
#define SLAYER_ACCELERATION_VALUE
#endif

#define SLAYER_TRACE_ARGUMENTS \
 SLAYER_SCENE_BUFFERS SLAYER_SAMPLE_BUFFERS \
 uint2 at [[thread_position_in_grid]], \
 uint2 grid [[threads_per_grid]] SLAYER_ACCELERATION_BUFFER

#define SLAYER_WORLD_INITIALIZER \
 {header, spheres, triangles, nodes, materials, lamps, volumes, baked, \
  trees, instances, tops, sky, bends, rays, boxes, fields, heights \
  SLAYER_FIRSTS_VALUE SLAYER_ACCELERATION_VALUE, nullptr}

// trace draws one sample: twelve numbers, the light, the cover, the
// sky's weight, the depth, and the way into the sky; and eleven more for
// each record the scene hands back to be painted.
kernel void trace(SLAYER_TRACE_ARGUMENTS) {
	World world = SLAYER_WORLD_INITIALIZER;
	uint4 native_counts(0);
	world.native_counts = &native_counts;
	uint2 pixel = uint2(at.x, at.y + origin.y);
	world.shared = sky_share(header, pixel);
	float3 from = header->origin.xyz, way = look(header, pixel);
	if (header->look.x != 0) {
		// the ray the cpu made through the lens, for this sample
		device const float *ray = rays + (pixel.y * grid.x + pixel.x) * 6;
		from = float3(ray[0], ray[1], ray[2]);
		way = float3(ray[3], ray[4], ray[5]);
	} else if (header->exposure.x != 0 || header->exposure.y != 0) {
		// the ray made here through an open shutter or an aperture
		Primary made = exposed(header, pixel);
		from = made.from, way = made.way;
	}
#if defined(SLAYER_COMPOSITE_PASS) && SLAYER_COMPOSITE_PASS == 2
	Hit hit = first(world, from, way, 0);
	float depth = hit.material < 0 ? nowhere : hit.distance;
	Through transport = through(world, from, way, depth);
	Seen seen = {transport.added, hit.material < 0 ? 0.0f : 1.0f,
	             depth, transport.keep, float3(0), unpainted,
	             unpainted, unpainted};
#else
	Seen seen = shade(world, from, way);
#endif
	device float *sample = out + (at.y * grid.x + at.x) * per_sample;
	sample[0] = seen.light.x, sample[1] = seen.light.y;
	sample[2] = seen.light.z, sample[3] = seen.cover;
	sample[4] = seen.weight.x, sample[5] = seen.weight.y;
	sample[6] = seen.weight.z, sample[7] = seen.depth;
	sample[8] = seen.sky.x, sample[9] = seen.sky.y;
	sample[10] = seen.sky.z;
	// Four four-bit counters fit exactly in a float's integer range. The
	// tracer has only a primary and one reflected closest query per sample.
	sample[11] = float(native_counts.x + (native_counts.y << 4) +
	                   (native_counts.z << 8) + (native_counts.w << 12));
	if (records > 0) {
		record(sample, 0, seen.met);
	}
	if (records > 1) {
		record(sample, 1, seen.mirrored);
	}
	if (records > 2) {
		record(sample, 2, seen.passed);
	}
}

// bake writes one volume's density into its grid, a cell a thread: the
// grid is side cells along each axis of the cube round the puff's ball,
// and a thread at (x, y) is the cell (x % side, x / side, y).
kernel void bake(device const Volume *volumes [[buffer(0)]],
                 constant int2 &which [[buffer(1)]],
                 device float *out [[buffer(2)]],
                 constant uint2 &origin [[buffer(3)]],
                 uint2 at [[thread_position_in_grid]],
                 uint2 grid [[threads_per_grid]]) {
	Volume volume = volumes[which.x];
	int side = which.y;
	int3 place = int3(at.x % side, at.x / side, at.y + origin.y);
	float radius = volume.puff.x;
	float3 local = (float3(place) + 0.5f) / side * (2 * radius) - radius;
	out[at.y * grid.x + at.x] = live(volume, local);
}

// Prepare the same source term once per smoke lighting cell.
kernel void smoke_light(SLAYER_TRACE_ARGUMENTS) {
	World world = SLAYER_WORLD_INITIALIZER;
 Volume volume=world.volumes[header->lighting.x];
 int3 dims=volume.lighting.xyz;
 int3 cell=int3(at.x%dims.x,at.x/dims.x,at.y+origin.y);
 float3 local=volume.low.xyz+(float3(cell)+0.5f)/float3(dims)*
              (volume.high.xyz-volume.low.xyz);
 float3 point=volume.origin.xyz+volume.origin.w*
              (volume.x.xyz*local.x+volume.y.xyz*local.y+volume.z.xyz*local.z);
 float3 value=scattered_direct(world,volume,point,local,float3(beam_zero));
 int first=(at.y*grid.x+at.x)*3;
 out[first]=value.x;out[first+1]=value.y;out[first+2]=value.z;
}

// reach_grid fills one cell of one wide lamp's grid over one medium, for
// fast light: the share of the lamp that gets to the cell's middle, by
// the same blocked and clear a step of the exact march asks. The header's
// lighting names the volume in x and the lamp in y.
kernel void reach_grid(SLAYER_TRACE_ARGUMENTS) {
	World world = SLAYER_WORLD_INITIALIZER;
	Volume volume = world.volumes[header->lighting.x];
	Lamp lamp = world.lamps[header->lighting.y];
	int3 dims = volume.lighting.xyz;
	int3 cell = int3(at.x % dims.x, at.x / dims.x, at.y + origin.y);
	float3 local = volume.low.xyz + (float3(cell) + 0.5f) / float3(dims) *
	               (volume.high.xyz - volume.low.xyz);
	float3 point = volume.origin.xyz + volume.origin.w *
	               (volume.x.xyz * local.x + volume.y.xyz * local.y +
	                volume.z.xyz * local.z);
	Reached reached = lamp_light(lamp, point);
	float3 value = float3(0);
	if (any(reached.toward != 0) &&
	    !blocked(world, point, reached.toward, reached.far)) {
		value = clear(world, point, reached.toward, reached.far);
	}
	int first = (at.y * grid.x + at.x) * 3;
	out[first] = value.x, out[first + 1] = value.y, out[first + 2] = value.z;
}

// reach_beam fills one place of one narrow beam's table, for fast light:
// the share of the beam that gets so far along its axis, asked back down
// the axis to the beam's start as a step in the beam asks. The header's
// lighting names the lamp in y; the grid is one row of reach_beam_cells.
kernel void reach_beam(SLAYER_TRACE_ARGUMENTS) {
	World world = SLAYER_WORLD_INITIALIZER;
	Lamp lamp = world.lamps[header->lighting.y];
	float along = float(at.x) / float(reach_beam_cells - 1) * lamp.extent.w;
	float3 point = lamp.at.xyz + lamp.toward.xyz * along;
	float3 value = float3(1);
	if (along > 0) {
		value = blocked(world, point, -lamp.toward.xyz, along)
		        ? float3(0) : clear(world, point, -lamp.toward.xyz, along);
	}
	int first = (at.y * grid.x + at.x) * 3;
	out[first] = value.x, out[first + 1] = value.y, out[first + 2] = value.z;
}

// environment_light prepares incident radiance, before multiplying any
// volume's albedo or adding emission. All vertices include the declared bounds.
kernel void environment_light(SLAYER_TRACE_ARGUMENTS) {
	World world=SLAYER_WORLD_INITIALIZER;
	int3 dims=header->sky_grid.xyz;
	int3 node=int3(at.x%dims.x,at.x/dims.x,at.y+origin.y);
	float3 fraction=float3(node)/float3(dims-1);
	float3 point=header->sky_low.xyz+fraction*(header->sky_high.xyz-header->sky_low.xyz);
	float3 value=volume_sky_direct(world,point);
	int first=(at.y*grid.x+at.x)*3;
	out[first]=value.x;out[first+1]=value.y;out[first+2]=value.z;
}
