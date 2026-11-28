// march_pcg is the PCG hash of O'Neill 2014 (pcg-random.org), the output
// function of a 32-bit state, word for word as trace's pcgHash has it.
static inline uint march_pcg(uint value) {
	uint state = value * 747796405u + 2891336453u;
	uint word = ((state >> ((state >> 28u) + 4u)) ^ state) * 277803737u;
	return (word >> 22u) ^ word;
}

// march_shift is where in each step the ray of a fine pixel takes its
// samples, a share of the step from nought up to under one: the top 24
// bits of the hash of the pixel's two integers over 2^24, which a float
// holds exactly, so this is trace's MarchShift to the last bit. Every
// ray has its own, so a coarse march is fine noise and not arcs where the
// steps line up from pixel to pixel. It is kept 2^-12 of a step from
// either end, as trace's shiftEdge says, so no sample sits where a
// float32's rounding and a float64's put it either side of a medium's edge.
static inline float march_shift(uint2 fine) {
	float share = float(march_pcg(fine.x ^ march_pcg(fine.y)) >> 8u) *
	              (1.0f / 16777216.0f);
	return clamp(share, 1.0f / 4096.0f, 1.0f - 1.0f / 4096.0f);
}
