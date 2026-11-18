# Analytic rotor-tip density in Metal

The generic `medium.PropellerCavitation` field is evaluated directly in the
Metal volume pipeline. This is the density primitive milestone; it does not
claim GPU parity for the complete underwater scene.

One fixed27word card carries the authored blade count, rotor frame, tip radius,
angular velocity, slipstream speed, compact core, finite recovery extent, density,
phase clock and advected fragmentation controls. The seed retains all64bits.

The shared `Volume` record supplies placement, AABB span, material and source
lighting. There is no density grid, particle mesh allocation or Alfa-specific
shader. Arbitrarily painted/textured variants retain the existing boxed path.
The model remains an authored optical-density population, not a bubble-size
distribution, pressure-derived inception or fluid/collapse solve.

The GPU refuses this field outside the validated lunar water route.

The later [shared transport](gpu-water-propeller.md) and
[exact calm infinite-plane](gpu-calm-infinite.md) subsets execute that combined
route on device; general wavy water and sky quadrature remain unsupported.
The kernel uses float32 coordinates and phase; precision at very large local
coordinates or long clocks needs separate characterization before production.
