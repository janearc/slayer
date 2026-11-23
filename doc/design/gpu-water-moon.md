# Finite calm-water Moon transport on Metal

The GPU now executes the explicit `direct-moon: mean-plane` source for a
finite, horizontal, axis-aligned flat HeightField. It copies the direction
and irradiance of the exact staged `emit.Distant` Moon; it does not introduce
another lunar brightness model. The existing scene switch selects the route.

`trace.PlanOceanMoon` validates the supported subset before encoding. Three
optical-only float4 records on the named water medium carry the air direction
and water IOR, source irradiance and lamp identity, and finite interface
centre and extent. Ordinary scenes retain their exact prior shader source.

The kernel refracts the incoming direction using Snell's law, finds the mean
interface crossing, applies unpolarised Fresnel transmission and the CPU's
projected-irradiance factor, and integrates incoming Beer attenuation in metres.
Both source legs test opaque visibility while excluding only the tagged water
interface. Surface diffuse and glint contributions use the same refracted
source below water; the water boundary itself retains its existing response.

Water single scattering samples this source at each existing midpoint. Zero
phase asymmetry retains Slayer's Rayleigh convention; other values use HG.
Outgoing attenuation uses the same analytic cell integral as CPU. Metal lacks
`expm1`, so optical thickness at most 0.01 uses its five-term expansion, whose
omitted term is below float32 roundoff; the zero-extinction limit is the cell
length. Larger optical thickness uses the exponential expression directly.

General continuations, displaced water and sky quadrature remain refused.
The later [mixed transport](gpu-water-propeller.md) and
[exact calm infinite-plane](gpu-calm-infinite.md) subsets extend this route
with explicit validation; they do not enable arbitrary clouds or waves.

A path that leaves the finite source footprint reports diagnostic bit20 in
the existing exact-integer sample lane; every band finisher rejects it instead
of producing dark water. The native four-bit counters retain their meanings.
