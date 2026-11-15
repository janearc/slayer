// A wake the kernel asks itself: the nebula of medium.Wake, ragged gas in a
// ball, thinning to its edge, cleared about its paths and banked along the
// channel's edges, from a card of words the go side writes (wake.go). Each
// function names the one in the medium package it follows. A card is read
// with ring.metal's readers: ring_word, ring_count and ring_seed.

// wake_smooth is maths.Smooth: nought at and below the start, one at and
// above the end, the cubic between. it is not metal's smoothstep, which
// answers differently for a start at or past the end.
static float wake_smooth(float start, float end, float x) {
	if (x <= start) { return beam_zero; }
	if (x >= end) { return beam_one; }
	float along = (x - start) / (end - start);
	return along * along * (smooth_three - smooth_two * along);
}

// wake_fractal is noise.Fractal of the card's seed, at a point of the wake
// over the card's scale, summed over its octaves.
static float wake_fractal(device const float *card, float3 local) {
	ulong seed = ring_seed(card, wake_seed_low);
	float3 at = local / card[wake_scale];
	return fractal(seed, at.x, at.y, at.z, ring_count(card, wake_octaves));
}

// wake_edge is one less the fade toward the ball's edge: one inside its
// fade's start, nought at the edge.
static float wake_edge(device const float *card, float far) {
	float reach = card[wake_reach];
	return beam_one - wake_smooth(card[wake_fade] * reach, reach, far);
}

// wake_away is Wake.fromPaths: how far a point is from the nearest path, the
// circle at its radius in its own plane, each path's space taking the point
// in as Subspace.In does.
static float wake_away(device const float *card, float3 local) {
	float nearest = wake_farthest;
	device const float *each = card + ring_count(card, wake_path_first);
	for (int i = whole_zero; i < ring_count(card, wake_path_count);
	     i++, each += path_words) {
		float3 moved = local - float3(each[path_origin_x], each[path_origin_y],
		                              each[path_origin_z]);
		float3 across = float3(each[path_axis_x_x], each[path_axis_x_y],
		                       each[path_axis_x_z]);
		float3 up = float3(each[path_axis_y_x], each[path_axis_y_y],
		                   each[path_axis_y_z]);
		float3 along = float3(each[path_axis_z_x], each[path_axis_z_y],
		                      each[path_axis_z_z]);
		float3 in = float3(dot(moved, across), dot(moved, up),
		                   dot(moved, along)) * (beam_one / each[path_scale]);
		float round_by = length(in.xz) - each[path_radius];
		nearest = min(nearest, length(float2(round_by, in.y)));
	}
	return nearest;
}

// wake_clearing is the share of the gas the channel leaves a distance d from the
// nearest path: nought within the clear's swept share of it, one beyond it.
static float wake_clearing(device const float *card, float d) {
	float clear = card[wake_clear];
	return wake_smooth(card[wake_swept] * clear, clear, d);
}

// wake_banks is the banks along the channel's edge, a bell about the clear
// of the card's width, as many times the open gas as the card's bank. the
// bell squares what is across it by a product, as a power of a number below
// nought is not a number here.
static float wake_banks(device const float *card, float d) {
	float clear = card[wake_clear];
	float across = (d - clear) / (card[wake_ridge] * clear);
	return card[wake_bank] * exp(-(across * across));
}

// wake_density is Wake.Density at a point of the wake's own space: ragged
// gas cubed and lifted, faded to the edge, cleared and banked by the paths;
// nought outside the ball.
static float wake_density(device const float *card, float3 local) {
	float far = length(local);
	if (far > card[wake_reach]) { return beam_zero; }
	float rag = beam_half + beam_half * wake_fractal(card, local);
	float base = card[wake_thick] * rag * rag * rag * card[wake_lift];
	base *= wake_edge(card, far);
	float d = wake_away(card, local);
	return base * (wake_clearing(card, d) + wake_banks(card, d));
}
