// The optional advection entry shares state and the legacy coefficient prefix.
// Its additional values are precomputed nonnegative step weights.
struct AdvectedSpec {
	BedSpec base;
	float centre, diffuse, west, east, north, south, burn;
};

// Mandel et al., equation(1), https://arxiv.org/pdf/0709.0086, printed page5.
// First-order upwind advection is this implementation's discretization choice.
kernel void fuel_step_advected(device const float2 *in [[buffer(0)]],
	device float2 *out [[buffer(1)]], constant AdvectedSpec &p [[buffer(2)]],
	uint2 at [[thread_position_in_grid]]) {
	if (at.x >= p.base.width || at.y >= p.base.depth) { return; }
	uint x = at.x, z = at.y, index = z*p.base.width+x;
	float2 cell = in[index];
	float left = in[z*p.base.width+(x>0 ? x-1 : 0)].x;
	float right = in[z*p.base.width+min(x+1,p.base.width-1)].x;
	float up = in[(z>0 ? z-1 : 0)*p.base.width+x].x;
	float down = in[min(z+1,p.base.depth-1)*p.base.width+x].x;
	float heat = p.centre*cell.x + p.diffuse*(left+right+up+down);
	if (x == 0) { left = 0; }
	if (x == p.base.width-1) { right = 0; }
	if (z == 0) { up = 0; }
	if (z == p.base.depth-1) { down = 0; }
	heat += p.west*left + p.east*right + p.north*up + p.south*down;
	float rate = cell.x>0 ? min(1.0f, exp(-p.base.activation/cell.x)) : 0;
	float fuel = cell.y*(1-p.burn*rate);
	float burned = cell.y-fuel;
	out[index] = float2(heat+p.base.heating/p.base.consumption*burned, fuel);
}
