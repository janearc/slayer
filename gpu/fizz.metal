// The existing medium.Fizz population density, evaluated without a grid.
// The coordinate frame is supplied by the ordinary Volume placement.
static float fizz_at(device const float *card, float3 at) {
 float radius = card[fizz_radius], height = card[fizz_height];
 float thick = card[fizz_thick];
 if (radius <= 0 || height <= 0 || thick <= 0) { return 0; }
 float up = at.y / height;
 if (up < 0 || up > 1) { return 0; }
 float wide = radius * (1 + (card[fizz_spread] - 1) * up);
 float out = length(at.xz) / wide;
 if (out > 1) { return 0; }
 float density = thick * (1 - wake_smooth(0.0f, 1.0f, up)) *
                 (1 - wake_smooth(0.0f, 1.0f, out));
 if (card[fizz_churn] <= 0) { return density; }
 device const uint *whole = reinterpret_cast<device const uint *>(card);
 ulong seed = (ulong(whole[fizz_seed_high]) << 32) |
              ulong(whole[fizz_seed_low]);
 float3 churned = at;
 churned.y -= card[fizz_advected];
 float scale = radius / 2;
 float n = fractal(seed, churned.x / scale, churned.y / scale,
                   churned.z / scale, 4);
 return density * (1 - min(card[fizz_churn], 1.0f) * (.5f + .5f * n));
}
