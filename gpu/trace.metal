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
#ifdef SLAYER_NATIVE_MESHES
	device const int *firsts;
#endif
#ifdef SLAYER_NATIVE_SHADOWS
	acceleration_structure<instancing> accelerated;
#endif
	thread uint4 *native_counts;
};

// nowhere is the finite miss-depth sentinel shared through structure.go.
// Fast-math compilation does not give reliable infinity semantics, so the
// CPU converts this sentinel to render/trace’s +Inf after readback.

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
// weighted by that vertex’s contribution.
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
	flat.lift = lift;
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
	ray r;
	r.origin = at;
	r.direction = way;
	r.min_distance = 0;
	r.max_distance = far;
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
	ray r;
	r.origin = at;
	r.direction = way;
	r.min_distance = max(0.0f, near);
	r.max_distance = far;
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

// first is render/trace's: the nearest thing a ray meets past near, sphere or
// face; a material of -1 is nothing.
static Hit first(World world, float3 at, float3 way, float near,
                 int leaving_sphere) {
	Hit found = meshes_hit(world, at, way, near, nowhere, false);
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
		         normalize(point - sphere.centre.xyz), float3(0), local,
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

// blocked is render/trace's: whether anything not clear stands between a point
// and a lamp far along the ray.
// A ray leaving its own convex sphere outward cannot meet that sphere
// again: |p+t*d-c|^2 = r^2 + 2*t*dot(p-c,d) + t^2*|d|^2 > r^2.
// Excluding only that primitive avoids false self-hits from planetary float32
// rounding. Inward rays and every other occluder retain the normal tests.
static bool blocked(World world, float3 at, float3 way, float far,
                    int leaving_sphere) {
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

// lamp_along is how much of a lamp's light goes out a way, a unit
// long: all of it for a lamp with no width, and for one kept to a
// cone, all of it within the width of the cone's way, fading to none
// across its soft edge. it is emit's Along, and maths.Smooth.
static float lamp_along(Lamp lamp, float3 out) {
	float width = lamp.toward.w;
	if (width <= 0) {
		return 1;
	}
	float off = acos(clamp(dot(out, lamp.toward.xyz), -1.0f, 1.0f));
	float soft = lamp.strength.w;
	if (off <= width) {
		return 1;
	}
	if (off >= width + soft) {
		return 0;
	}
	float through = (off - width) / soft;
	return 1 - through * through * (3 - 2 * through);
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

// lamp_light agrees with emit.Lit, emit.Volume, emit.Beam and emit.Distant.
// Finite sources keep the centre-ray approximation for shadows.
struct Reached { float3 toward; float far; float3 light; };
static Reached lamp_light(Lamp lamp, float3 point) {
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

// density is the volume's density at a point in its own space: a boxed
// medium's from its grid; a puff's read from its baked grid, blended
// between the eight cells round the point, when it has one, and asked of
// the noise when it does not. outside the grid's box or cube there is no
// density, as outside a puff's ball.
static float density(World world, Volume volume, float3 local) {
	if (volume.info.z == boxed_kind) {
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
	if (volume.info.z == boxed_kind) {
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
		float2 crossing = span(volume, at, way, 0, far / volume.origin.w);
		if (crossing.y <= crossing.x) {
			continue;
		}
		float stride = (crossing.y - crossing.x) / shades;
		float wobble = 2 / (stride * volume.origin.w);
		float jitter = 0.5f + 0.5f * gradient(77, point.x * wobble,
		                                      point.y * wobble,
		                                      point.z * wobble);
		float thick = 0;
		for (int sample = 0; sample < shades; sample++) {
			float along = crossing.x + (sample + jitter) * stride;
			float3 sample_at = at + way * along;
			thick += density(world, volume, sample_at) * stride;
		}
		depth += volume.tint.xyz * thick;
	}
	return exp(-depth);
}

// scattered is render/trace's: the light a bit of a volume sends toward
// the eye, what it scatters of every lamp's light that reaches it, evenly,
// and its glow; a boxed medium's paint and shine read from its grids at
// the point of its own space, where it has them.
static float3 scattered_direct(World world, Volume volume, float3 point,
                        float3 local) {
	Material made = world.materials[volume.info.w];
	float3 colour = made.colour.xyz, sum = made.glow.xyz;
	int3 low;
	float3 part;
	if (volume.info.z == boxed_kind &&
	    boxed_part(volume, local, low, part)) {
		if (volume.shone.x >= 0) {
			sum = boxed_at(world, volume, volume.shone.x, 3, low, part);
		}
		if (volume.shone.y >= 0) {
			colour = boxed_at(world, volume, volume.shone.y, 3, low, part);
		}
	}
	// A medium with no scattering sends only its own emission.
	// Skip light selection, geometry shadows and transmission in this case.
	if (all(colour == float3(0))) { return sum; }
	Ask asks[lamps_asked];
	int count = asked(world, point, float3(0), asks);
	for (int index = 0; index < count; index++) {
		Lamp lamp = world.lamps[asks[index].lamp];
		Reached reached = lamp_light(lamp, point);
		if (all(reached.toward == 0)) {
			continue;
		}
		if (blocked(world, point, reached.toward, reached.far)) {
			continue;
		}
		float3 reach = world.header->tuning.z *
		               clear(world, point, reached.toward, reached.far);
		sum += colour * reached.light * reach * asks[index].counts;
	}
	return sum;
}

// The optional grid caches the isotropic source term, not density or opacity.
static float3 scattered(World world, Volume volume, float3 point,
                        float3 local) {
 if (volume.lighting.w < 0) { return scattered_direct(world,volume,point,local); }
 Volume grid=volume;
 grid.grid=volume.lighting;
 int3 low; float3 part;
 if (!boxed_part(grid,local,low,part)) {return float3(0);}
 return boxed_at(world,grid,grid.grid.w,3,low,part);
}

// a ray's crossing of one volume: where it enters and leaves, in the
// world's distances, and which volume
struct Crossing { float enter, leave; int volume; };

// crossings is render/trace's: every volume the ray passes through before far,
// nearest entry first. at most most_volumes: the go side refuses a world
// with more volumes than that, rather than drawing it without some.
static int crossings(World world, float3 at, float3 way, float far,
                     thread Crossing *found) {
	int count = 0;
	int volumes = min(world.header->more.x, most_volumes);
	for (int index = 0; index < volumes; index++) {
		Volume volume = world.volumes[index];
		float scale = volume.origin.w;
		float2 crossing = span(volume, inside(volume, at),
		                       turned(volume, way), 0,
		                       min(far, nowhere) / scale);
		if (crossing.y <= crossing.x) {
			continue;
		}
		Crossing next = {crossing.x * scale, crossing.y * scale, index};
		int place = count++;
		while (place > 0 && found[place - 1].enter > next.enter) {
			found[place] = found[place - 1];
			place--;
		}
		found[place] = next;
	}
	return count;
}

// shows is render/trace's: whether enough light still gets through to go on
static bool shows(float3 keep, float faint) {
	return keep.x > faint || keep.y > faint || keep.z > faint;
}

// through is render/trace's throughGain for a painted volume: what the ray
// gathers from the volumes before far, step by step, front to back,
// overlapping crossings' steps taken in turn by how far along each is.
struct Through { float3 keep, added; };

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

static Through through(World world, float3 at, float3 way, float far) {
	Through gathered = {float3(1), float3(0)};
	Crossing found[most_volumes];
	int count = crossings(world, at, way, far, found);
	int next[most_volumes] = {};
	int parts[most_volumes] = {};
	BeamMarch refine = beam_march(world, at, way, far);
	int strides = world.header->more.y;
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
		float thick = density(world, volume, local);
		if (thick <= 0) {
			continue;
		}
		// each channel as trace's absorbed: its tint's times for a
		// tinted medium, alike for the rest
		float3 absorb = 1 - exp(-thick * stride / volume.origin.w *
		                        volume.tint.xyz);
		float3 light = scattered(world, volume, point, local);
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
	const int looks = 16;
	const float golden_turn = 2.39996322972865332f;
	float sign = normal.z < 0 ? -1.0f : 1.0f;
	float a = -1 / (sign + normal.z);
	float b = normal.x * normal.y * a;
	float3 first = float3(1 + sign * normal.x * normal.x * a, sign * b,
	                      -sign * normal.x);
	float3 second = float3(b, sign + normal.y * normal.y * a, -normal.y);
	float3 from = lifted(world, point, normal);
	float3 sum = float3(0);
	for (int look = 0; look < looks; look++) {
		float share = (look + 0.5f) / looks;
		float out = sqrt(share), turn = golden_turn * look;
		float3 way = first * (out * cos(turn)) +
		             second * (out * sin(turn)) + normal * sqrt(1 - share);
		if (blocked(world, from, way, nowhere, leaving_sphere)) {
			continue;
		}
		sum += sky_at(world, way) * clear(world, from, way, nowhere);
	}
	return sum / looks;
}

// falling is the light that falls on a matte surface, for its colour to
// be taken by: every lamp's that reaches it past the things and the
// volumes, over pi, and the sky's. render/trace's surfaceAt is the surface's
// glow and its colour times this, without the mirror.
static float3 falling(World world, float3 point, float3 normal,
                      int leaving_sphere) {
	float3 sum = skylit(world, point, normal, leaving_sphere);
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
		if (blocked(world, from, reached.toward, reached.far,
		            leaving_sphere)) {
			continue;
		}
		float3 reach = square * clear(world, from, reached.toward,
		                              reached.far);
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
	Painted met, mirrored;
};

// to_sky is a ray that met nothing: all of the sky's light counts
static Seen to_sky(float3 way) {
	return {float3(0), 0, nowhere, float3(1), way, unpainted, unpainted};
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
	return {light, 1, hit.distance, float3(0), float3(0), place, unpainted};
}

// behind is render/trace's end of shade: the volumes in front of what the ray
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
	return seen;
}

// facing is a hit's normal turned toward the eye, as render/trace turns it
static float3 facing(Hit hit, float3 way) {
	return dot(hit.normal, way) > 0 ? -hit.normal : hit.normal;
}

// shade_plain is render/trace's shade at its last bounce: what a ray meets,
// lit, and the volumes in front, with no mirror.
static Seen shade_plain(World world, float3 at, float3 way,
                        int leaving_sphere) {
	Hit hit = first(world, at, way, 0, leaving_sphere);
	if (hit.material < 0) {
		return behind(world, at, way, to_sky(way));
	}
	float3 point = fma(way, hit.distance, at);
	Seen seen = surfaced(world, hit, point, facing(hit, way));
	return behind(world, at, way, seen);
}

// shade is render/trace's shade at the first bounce: as shade_plain, and a
// glossy surface mirrors what shade_plain sees along the mirrored ray.
// metal has no recursion, so the bounce is a second function.
static Seen shade(World world, float3 at, float3 way) {
	Hit hit = first(world, at, way, 0);
	if (hit.material < 0) {
		return behind(world, at, way, to_sky(way));
	}
	float3 point = fma(way, hit.distance, at);
	float3 normal = facing(hit, way);
	Material made = world.materials[hit.material];
	Seen seen = surfaced(world, hit, point, normal);
	float share = mirrors(made, -dot(normal, way));
	if (share > 0) {
		float3 bounce = way - normal * 2 * dot(way, normal);
		float3 mirrored = normalize(bounce);
		float3 from = lifted(world, point, normal);
		Seen bounced = shade_plain(world, from, mirrored, hit.sphere_index);
		seen.light = seen.light * (1 - share) + bounced.light * share;
		seen.weight = bounced.weight * share;
		seen.sky = bounced.sky;
		seen.met = dimmed(seen.met, float3(1 - share));
		seen.mirrored = dimmed(bounced.met, float3(share));
	}
	return behind(world, at, way, seen);
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
 device const float *boxes [[buffer(14)]],

#ifdef SLAYER_NATIVE_MESHES
#define SLAYER_SAMPLE_BUFFERS \
 device const int *firsts [[buffer(15)]], \
 device float *out [[buffer(16)]], \
 constant uint2 &origin [[buffer(17)]],
#define SLAYER_FIRSTS_VALUE , firsts
#define SLAYER_ACCELERATION_INDEX 18
#else
#define SLAYER_SAMPLE_BUFFERS \
 device float *out [[buffer(15)]], \
 constant uint2 &origin [[buffer(16)]],
#define SLAYER_FIRSTS_VALUE
#define SLAYER_ACCELERATION_INDEX 17
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
  trees, instances, tops, sky, bends, rays, boxes SLAYER_FIRSTS_VALUE \
  SLAYER_ACCELERATION_VALUE, nullptr}

// trace draws one sample: twelve numbers, the light, the cover, the
// sky's weight, the depth, and the way into the sky; and eleven more for
// each record the scene hands back to be painted.
kernel void trace(SLAYER_TRACE_ARGUMENTS) {
	World world = SLAYER_WORLD_INITIALIZER;
	uint4 native_counts(0);
	world.native_counts = &native_counts;
	uint2 pixel = uint2(at.x, at.y + origin.y);
	float3 from = header->origin.xyz, way = look(header, pixel);
	if (header->look.x != 0) {
		// the ray the cpu made through the lens, for this sample
		device const float *ray = rays + (pixel.y * grid.x + pixel.x) * 6;
		from = float3(ray[0], ray[1], ray[2]);
		way = float3(ray[3], ray[4], ray[5]);
	}
	Seen seen = shade(world, from, way);
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
 float3 value=scattered_direct(world,volume,point,local);
 int first=(at.y*grid.x+at.x)*3;
 out[first]=value.x;out[first+1]=value.y;out[first+2]=value.z;
}
