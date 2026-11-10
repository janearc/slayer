// This is flame.Body's field evaluator in float arithmetic. Its rise constants
// arrive from the same pinned McCaffrey data; the shape and turbulence choices
// below mirror flame/body.go. They are authored appearance, not measurements.
// See doc/fire-recipes.md for sources, model limits and the conformance tests.

static ulong fire_seed(constant FireGrid &p) {
 return (ulong(uint(p.seed.y)) << 32) | ulong(uint(p.seed.x));
}

// Integrate the three centreline-speed regions exactly as flame.Body.travel.
static float fire_travel(constant FireGrid &p, float height) {
 float scale = p.shape.y / p.borders.x;
 float fifth = sqrt(scale);
 float low = p.speeds.w * scale, high = p.borders.x * scale;
 if (height <= 0) { return 0; }
 if (height <= low) { return 2 * sqrt(height) / p.speeds.x; }
 float in_tongues = 2 * sqrt(low) / p.speeds.x +
  (min(height, high) - low) / (p.speeds.y * fifth);
 if (height <= high) { return in_tongues; }
 return in_tongues + 3 / (4 * p.speeds.z * fifth) *
  pow(1 / scale, 1.0f / 3) *
  (pow(height, 4.0f / 3) - pow(high, 4.0f / 3));
}

static float fire_lifted(constant FireGrid &p, float height) {
 float fifth = sqrt(p.shape.y / p.borders.x);
 return (fire_travel(p, height) - p.heat.z) * p.speeds.y * fifth;
}

// Return absorption and temperature together, sharing the same displaced point.
static float2 fire_field(constant FireGrid &p, float3 point) {
 ulong seed = fire_seed(p);
 float size = 0.8f * p.shape.x;
 float risen = fire_lifted(p, point.y);
 float breath = 0.2f * gradient(seed, p.heat.z / p.shape.w, 0.5f, 0.5f);
 float up = point.y / (p.shape.y * (1 + breath));
 if (!(up > 0) || up >= 1) { return float2(0, p.heat.x); }
 float x = point.x + 0.25f * p.shape.x *
  fractal(seed + 1, point.x / size, risen / size, point.z / size, 6);
 float z = point.z + 0.25f * p.shape.x *
  fractal(seed + 2, point.x / size, risen / size, point.z / size, 6);
 float widest = p.shape.x * pow(1 - up, 0.6f);
 float deep = max(0.0f, 1 - (x * x + z * z) / (widest * widest));
 if (deep == 0) { return float2(0, p.heat.x); }
 float tongue = 1;
 float heat = p.heat.x + (p.heat.y - p.heat.x) * sqrt(deep);
 float border=p.offset.w>0?p.offset.w:0.4f;
 if (up > border) {
  float flow = fractal(seed + 3, point.x / size,
   risen / (3 * size), point.z / size, 6);
  float bar = -0.65f + 1.3f * (up - border) / (1-border);
  tongue = clamp((flow - bar) / 0.05f, 0.0f, 1.0f);
   }
 if (up>0.4f || (p.heat.w!=0 && up>border)) {
  float axis = p.heat.w!=0 ? p.heat.y :
   p.heat.x + (p.heat.y - p.heat.x) * 0.4f / up;

  heat = p.heat.x + (axis - p.heat.x) * sqrt(deep);
  float rag_size = size / 2;
  float rag = fractal(seed + 5, point.x / rag_size,
   risen / (3 * rag_size), point.z / rag_size, 6);
  float ragged = clamp((rag - 0.4f) / 0.08f, 0.0f, 1.0f);
  heat -= (heat - p.heat.x) * 0.6f * ragged;
 }
 return float2(p.shape.z * deep * tongue, heat);
}

static float3 fire_glow(constant FireGrid &p, device const float4 *table,
                        float heat) {
 if (!(p.heat.y > p.heat.x)) { return table[0].xyz; }
 float place = clamp((heat - p.heat.x) * (255 / (p.heat.y - p.heat.x)),
                     0.0f, 255.0f);
 int low = min(int(floor(place)), 254);
 float share = place - float(low);
 return table[low].xyz * (1 - share) + table[low + 1].xyz * share;
}

kernel void fire_grid(constant FireGrid &p [[buffer(0)]],
                      constant FireGrid *parts [[buffer(1)]],
                      device const float4 *table [[buffer(2)]],
                      device float *out [[buffer(3)]],
                      constant uint2 &origin [[buffer(4)]],
                      uint2 at [[thread_position_in_grid]],
                      uint2 grid [[threads_per_grid]]) {
 int3 index = int3(at.x % uint(p.cells.x), at.x / uint(p.cells.x),
                   at.y + origin.y);
 float3 point = p.low.xyz + (p.high.xyz - p.low.xyz) *
  ((float3(index) + 0.5f) / float3(p.cells.xyz));
 float absorption = 0;
 float3 emitted = float3(0), single = float3(0);
 for (int i = 0; i < p.seed.z; i++) {
  constant FireGrid &cell = parts[i];
  float3 local = point - cell.offset.xyz;
  float2 field = float2(0, cell.heat.x);
  if (abs(local.x) <= 1.5f * cell.shape.x &&
      abs(local.z) <= 1.5f * cell.shape.x &&
      local.y > 0 && local.y < 1.2f * cell.shape.y) {
   field = fire_field(cell, local);
  }
  float3 glow = fire_glow(cell, table + i * 256, field.y);
  absorption += field.x;
  emitted += glow * field.x;
  single = glow;
 }
 float3 glow = p.seed.w == 0 ? single :
  (absorption > 0 ? emitted / absorption : float3(0));
 uint offset = (at.y * grid.x + at.x) * 4;
 out[offset] = absorption;
 out[offset + 1] = glow.x;
 out[offset + 2] = glow.y;
 out[offset + 3] = glow.z;
}
