// A ring the kernel asks itself: the annulus of planet's Ring and the share
// of its mean depth that stands at a point of its plane, as Ring.share has
// it, from a card of words the go side writes (ring.go). Each function
// names the one in planet it follows.

// ring_word is a word of a ring's card read as the whole number it holds.
static uint ring_word(device const float *card, int field) {
	return ((device const uint *)card)[field];
}

// ring_count is a word of the card that holds a count.
static int ring_count(device const float *card, int field) {
	return int(ring_word(card, field));
}

// ring_seed is a seed the card holds in two words, low then high.
static ulong ring_seed(device const float *card, int low) {
	return (ulong(ring_word(card, low + seed_high_word)) << seed_half_bits) | ulong(ring_word(card, low));
}

// ring_hash3 is maths/noise's Hash3: a number from nought up to one for a
// seed and three whole numbers, cut to a float's twenty-four bits.
static float ring_hash3(ulong seed, long i, long j, long k) {
	ulong z_part = mixh(ulong(k) + golden);
	ulong y_part = mixh(ulong(j) + z_part);
	ulong whole = mixh(seed ^ mixh(ulong(i) + y_part));
	return float(whole >> hash_drop) * ring_unit;
}

// ring_eased is planet's eased: smoothstep's ease over a cell.
static float ring_eased(float t) { return t * t * (smooth_three - smooth_two * t); }

// ring_taken is one less e to the minus a depth, kept exact for a depth so
// small that the subtraction would lose it all: a series for those, the
// difference for the rest.
static float ring_taken(float depth) {
	if (depth < beam_half / beam_five) {
		return depth * (beam_one - depth / beam_two * (beam_one - depth / beam_three *
		       (beam_one - depth / beam_four * (beam_one - depth / beam_five))));
	}
	return beam_one - exp(-depth);
}

// ringlet_share is ringlets.share: the depth rising and falling by radius,
// in layers of seeded noise, each finer and weaker than the last.
static float ringlet_share(device const float *card, float reach) {
	float strength = card[ring_ringlet_strength];
	if (strength == beam_zero) { return beam_one; }
	ulong seed = ring_seed(card, ring_ringlet_seed_low);
	float size = beam_one / card[ring_ringlet_width], weight = beam_one, sum = beam_zero;
	float across = card[ring_ringlet_across];
	for (int i = whole_zero; i < ring_count(card, ring_ringlet_layers); i++) {
		sum += weight * gradient(seed + ulong(i), size * reach, across, across);
		size *= card[ring_ringlet_finer];
		weight *= card[ring_ringlet_weaker];
	}
	return max(beam_zero, beam_one + strength * sum);
}

// stretch_read is Baked.read: a level's light kept at a radius, straight
// between the middles of its bins.
static float stretch_read(device const float *card, int level, float radius) {
	int size = ring_count(card, ring_stretch_bins);
	int start = ring_count(card, ring_stretch_first);
	for (int each = whole_zero; each < level; each++) {
		start += size;
		size = (size + whole_one) / whole_two;
	}
	float width = card[ring_stretch_step] * float(whole_one << level);
	float along = (radius - card[ring_stretch_inner]) / width - beam_half;
	int k = int(floor(along));
	float toward = along - float(k);
	float low = card[start + clamp(k, whole_zero, size - whole_one)];
	float high = card[start + clamp(k + whole_one, whole_zero, size - whole_one)];
	return low * (beam_one - toward) + high * toward;
}

// stretch_seen is Baked.Seen: the depth across so many metres of the profile
// about a radius of it, read from the level as wide as the stretch, between
// the two levels either side; nought outside the profile.
static float stretch_seen(device const float *card, float radius,
                          float across) {
	float inner = card[ring_stretch_inner], step = card[ring_stretch_step];
	int bins = ring_count(card, ring_stretch_bins);
	if (radius < inner || radius >= inner + step * float(bins)) { return beam_zero; }
	float level = beam_zero;
	if (across > step) { level = log2(across / step); }
	int top = ring_count(card, ring_stretch_levels) - whole_one;
	level = min(level, float(top));
	int below = int(level);
	int above = min(below + whole_one, top);
	float toward = level - float(below);
	float kept = stretch_read(card, below, radius) * (beam_one - toward) +
	             stretch_read(card, above, radius) * toward;
	return -log(max(kept, ring_floor));
}

// stretch_share is measured.share: the profile's depth where this ring's
// reach is laid, over the ring's own.
static float stretch_share(device const float *card, float reach,
                           float finest) {
	float wide = card[ring_stretch_wide];
	if (wide == beam_zero) { return beam_one; }
	float there = card[ring_stretch_from] + (reach - card[ring_inner]) * wide;
	return stretch_seen(card, there, finest * wide) / card[ring_depth];
}

// ring_finest is vantage.finest: the finest a ray from the eye tells apart at
// a point of the ring's plane, in metres across the orbit; nought with no
// eye.
static float ring_finest(device const float *card, float x, float z) {
	float3 eye = float3(card[ring_eye_x], card[ring_eye_y], card[ring_eye_z]);
	float3 place = float3(x, beam_zero, z);
	float3 toward = place - eye;
	float distance = length(toward), reach = length(place);
	float ray = card[ring_eye_ray];
	if (ray == beam_zero || distance == beam_zero || reach == beam_zero) { return beam_zero; }
	float slant = dot(place / reach, toward / distance);
	float squash = max(beam_one / ring_stretch, sqrt(max(beam_zero, beam_one - slant * slant)));
	return ray * distance / squash;
}

// grain_cells is planet's cells: worley's distance to the nearest of a seeded
// point in each cell round about, turned so a clump stands thick.
static float grain_cells(ulong seed, float x, float y, float z) {
	float3 at = float3(x, y, z);
	float nearest = beam_one;
	for (int i = -whole_one; i <= whole_one; i++) {
		for (int k = -whole_one; k <= whole_one; k++) {
			for (int depth = -whole_one; depth <= whole_one; depth++) {
				int3 cell = int3(floor(at)) + int3(i, k, depth);
				float3 point = float3(cell);
				for (int axis = whole_zero; axis < whole_three; axis++) {
					point[axis] += ring_hash3(seed + ulong(axis), long(cell.x),
					                          long(cell.y), long(cell.z));
				}
				nearest = min(nearest, length(at - point));
			}
		}
	}
	return beam_one - beam_two * nearest;
}

// grain_speckle is planet's speckle: a fresh value in every cell, eased to
// its neighbours', its cells counted along x and round the circle y and z lie
// on.
static float grain_speckle(ulong seed, float x, float y, float z) {
	float around = max(beam_one, round(beam_two * M_PI_F * length(float2(y, z))));
	float turn = (atan2(z, y) + M_PI_F) / (beam_two * M_PI_F) * around;
	float reach = floor(x), round_to = floor(turn);
	float inward = ring_eased(x - reach), outward = ring_eased(turn - round_to);
	float values[whole_two][whole_two];
	for (int i = whole_zero; i < whole_two; i++) {
		for (int k = whole_zero; k < whole_two; k++) {
			float cell = fmod(round_to + float(k), around);
			values[i][k] = ring_hash3(seed, long(reach + float(i)), long(cell),
			                          long(around));
		}
	}
	float near = values[whole_zero][whole_zero] +
	             (values[whole_zero][whole_one] - values[whole_zero][whole_zero]) * outward;
	float far = values[whole_one][whole_zero] +
	            (values[whole_one][whole_one] - values[whole_one][whole_zero]) * outward;
	return beam_two * (near + (far - near) * inward) - beam_one;
}

// grain_pattern is the pattern of a kind, a noise through space at a scale
// of one.
static float grain_pattern(int kind, ulong seed, float x, float y, float z) {
	if (kind == ring_cells) { return grain_cells(seed, x, y, z); }
	if (kind == ring_speckle) { return grain_speckle(seed, x, y, z); }
	float noised = gradient(seed, x, y, z);
	return kind == ring_ridges ? beam_one - beam_two * abs(noised) : noised;
}

// grain_layered is grains.layered: the pattern in layers at a point of the
// plane, a layer for each doubling of size from the largest, each weighed by
// the card's weights; and how much of it is hidden, finer than the finest a
// ray there sees. the sum is in x, the hidden in y.
static float2 grain_layered(device const float *card, float x, float z,
                            float finest) {
	float reach = length(float2(x, z)), round_by = atan2(z, x);
	float along = reach / card[ring_grain_shear];
	float size = card[ring_grain_largest], unseen = beam_zero, sum = beam_zero;
	ulong seed = ring_seed(card, ring_grain_seed_low);
	int kind = ring_count(card, ring_grain_pattern);
	int first = ring_count(card, ring_grain_first);
	for (int i = whole_zero; i < ring_count(card, ring_grain_layers); i++) {
		float strength = card[first + i], seen = beam_one;
		if (finest > beam_zero) { seen = max(beam_zero, min(beam_one, size / finest - beam_one)); }
		unseen += (beam_one - seen * seen) * strength * strength;
		if (seen > beam_zero) {
			sum += seen * strength * grain_pattern(kind, seed + ulong(i),
			       reach / size, along * cos(round_by) / size,
			       along * sin(round_by) / size);
		}
		size /= beam_two;
	}
	return float2(sum, sqrt(unseen) * card[ring_grain_spread]);
}

// grain_share is grains.share: where the pattern stands above its threshold
// there is matter, packed so the ring keeps its mean depth, and below it
// none, the edge widened by what the ray cannot tell apart.
static float grain_share(device const float *card, float x, float z,
                         float finest) {
	if (ring_count(card, ring_grain_pattern) == whole_zero) { return beam_one; }
	float2 layered = grain_layered(card, x, z, finest);
	float width = card[ring_grain_softness] +
	              card[ring_grain_widths] * layered.y;
	float edge = max(beam_zero, min(beam_one, (layered.x - card[ring_grain_threshold]) /
	                           width + beam_half));
	return ring_eased(edge) / card[ring_grain_filled];
}

// moonlet_edged is planet's edged: zero inside one, one past one and the
// card's edge more, smoothly between.
static float moonlet_edged(device const float *card, float apart) {
	float share = clamp((apart - beam_one) / card[ring_moonlet_edge], beam_zero, beam_one);
	return share * share * (smooth_three - smooth_two * share);
}

// moonlet_cleared is Ring.cleared: how much of the ring the moonlets leave
// at a point of its plane.
static float moonlet_cleared(device const float *card, float x, float z) {
	int count = ring_count(card, ring_moonlet_count);
	if (count == whole_zero) { return beam_one; }
	float reach = length(float2(x, z)), angle = atan2(z, x), left = beam_one;
	device const float *each = card + ring_count(card, ring_moonlet_first);
	for (int i = whole_zero; i < count; i++, each += moonlet_words) {
		float out = abs(reach - each[moonlet_reach]) / each[moonlet_gap];
		if (each[moonlet_opens] != beam_zero) {
			left *= moonlet_edged(card, out);
			continue;
		}
		float behind = angle - each[moonlet_angle];
		float along = (behind - beam_two * M_PI_F * rint(behind / (beam_two * M_PI_F))) *
		              each[moonlet_reach];
		float apart = length(float2(out, along / each[moonlet_length]));
		left *= moonlet_edged(card, apart);
	}
	return left;
}

// ring_share is Ring.share: how much of the ring's mean depth stands at a
// point of its plane, its ringlets', measured, grains' and what its moonlets
// leave, together.
static float ring_share(device const float *card, float x, float z) {
	float finest = ring_finest(card, x, z), reach = length(float2(x, z));
	return ringlet_share(card, reach) * stretch_share(card, reach, finest) *
	       grain_share(card, x, z, finest) * moonlet_cleared(card, x, z);
}

// ring_density is a ring's slab's density at a point of its own space: its
// share inside the slab and the annulus, and nought outside; a glow slab's
// is its faintness of that.
static float ring_density(device const float *card, float3 local) {
	float reach = length(float2(local.x, local.z));
	if (abs(local.y) > card[ring_half] || reach < card[ring_inner] ||
	    reach >= card[ring_outer]) {
		return beam_zero;
	}
	float share = ring_share(card, local.x, local.z);
	return ring_word(card, ring_glow) != whole_zero ? card[ring_glow_faint] * share
	                                       : share;
}

// ring_shine is GlowShine.At: the light a metre of a glow slab gives off at
// a point of its own space over its density there: the x-rays falling there,
// the share the ring's depth stops along their slant, given back by its
// phosphors.
static float3 ring_shine(device const float *card, float3 local) {
	float reach = length(float2(local.x, local.z));
	float share = ring_share(card, local.x, local.z);
	float way_out = reach - card[ring_glow_radius], lift = card[ring_glow_lift];
	if (!(way_out > beam_zero)) { return float3(beam_zero); }
	float square = lift * lift + way_out * way_out;
	float sine = lift / sqrt(square);
	if (!(share > beam_zero) || !(sine > beam_zero)) { return float3(beam_zero); }
	float stopped = ring_taken(card[ring_depth] * share / sine);
	float given = card[ring_glow_source] * stopped / (reach * square);
	float3 colour = float3(card[ring_glow_red], card[ring_glow_green],
	                       card[ring_glow_blue]);
	return colour * (given / (card[ring_glow_faint] * share));
}

// ring_flat is whether a stretch of a ray in a ring's own space, from a start
// to an end, both in its slab, has one density all along it: it runs less
// than a fraction of the finest scale the ring carries across the plane, and
// stays inside the annulus, whose rims are where its density steps. The
// reach of a straight stretch is least where it is nearest the middle and
// most at an end.
static bool ring_flat(device const float *card, float3 start, float3 end) {
	float2 from = start.xz, run = end.xz - start.xz;
	if (!(length(run) < ring_flat_part * card[ring_scale])) { return false; }
	float along = dot(run, run);
	float nearest = along > beam_zero ? clamp(-dot(from, run) / along, beam_zero,
	                                  beam_one) : beam_zero;
	float closest = length(from + run * nearest);
	return closest >= card[ring_inner] && length(from) < card[ring_outer] &&
	       length(end.xz) < card[ring_outer];
}
