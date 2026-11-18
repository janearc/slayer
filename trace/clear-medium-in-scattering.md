# First-order clear-medium in-scattering (CPU)

`thing.NewClearMedium` remains the existing extinction-only path. Its supplied
total extinction has no recoverable absorption/scattering split and it still
only attenuates the unscattered beam.

The new Go API `thing.NewClearMediumWithScattering` accepts separate
caller-supplied RGB absorption and scattering coefficients, their phase
parameter, metres per scene unit, and an explicit integration budget.
Coefficients are per metre at 610, 550, and 465 nm. No measured water values
or presets are included.

For a finite active camera segment, the tracer adds a first-order source term
to the already attenuated surface result:

`L = L_surface exp(-sigma_t d) + integral[ sigma_s p L_i exp(-sigma_t s) ds ]`

where `sigma_t = sigma_a + sigma_s`. Incident lamp irradiance and phase are
evaluated at a midpoint per cell. The outgoing exponential is integrated
analytically within each cell using `-expm1(-sigma_t * cellLength) / sigma_t`;
the configurable midpoint count affects spatial variation in incident light,
not this outgoing integral.

This agrees with `gas.VisibleTable.SingleScatter` for constant incident
illumination. Phase uses `scene/gas.Gas.PhaseAt`: zero means Rayleigh, not
isotropic Henyey–Greenstein at `g=0`; nonzero values are HG asymmetry and must
remain strictly between -1 and 1.

Directionless ambient lamps are refused: the current lamp interface supplies
surface-weighted ambient strength, not an incident angular distribution for a
volume source term. Built-in floods fail preflight; a custom emitter that proves
directionless at a sample causes the render to return an error after the
bounded draw.

The general incident-light approximation is deliberately narrow: each lamp's
irradiance is evaluated at the sample and opaque geometry is checked for
shadow, but light is treated as **unattenuated and unrefracted** on its way
from lamp to sample. It refuses participating media when the world has a
transparent surface, sky illumination, separate haze, volume transport, or
gravitational fields. It also refuses more than 16 lamps.

There is one separate ocean route, enabled only when the same named scattering
medium is attached to a visible, fully clear supported ocean boundary. When
`direct-sun: mean-plane` is authored, each midpoint evaluates the tagged direct
sun after the existing Snell/Fresnel crossing and water-path Beer attenuation.
The existing projected area factor is retained. Opaque blockers are tested on
both sides of the interface. The ground-route legacy fill remains separate.

`ClearMedium.WithDiffuseSkyInscatter(samples)` returns a configured copy that
adds a bounded CPU-only diffuse sky source. It samples `samples` equal-solid-
angle directions over the air-side upper hemisphere at each existing camera
midpoint.

Each sky direction is refracted through the authored horizontal mean plane;
source-side water Beer attenuation uses the actual distance to the staged
boundary, and opaque blockers are checked in water and air.

The sky is treated as incident radiance at the air side of the interface. For
the air-angle quadrature, Fresnel transmission and the refractive
radiance/solid angle factors combine as `T*cos_air/cos_water`: radiance scales
by `(n_water/n_air)^2`, while Snell maps solid angle by
`(n_air/n_water)^2*cos_air/cos_water`. The canceled `n²` factors must not be
applied again.

This option can be used without `direct-sun: mean-plane`: in that sky-only
mode, the sky is the only source integrated into the participating medium, while
authored lamps keep their existing surface-lighting behavior.

It requires `world.Sky` and an actual `InfiniteMeanContinuation`; finite
footprints cannot guarantee a boundary hit for every sampled sky direction and
are refused. It is a horizontal mean-plane approximation, not wave-resolved
refraction, caustics, or sky transport through a separately modeled air volume.
Other transparent boundaries, media, haze, volumes and fields remain refused.

This adds first-order direct-sun scattering when mean-plane sun is explicitly
selected, with an independent opt-in diffuse-sky source. It does not integrate water scattering
more than once, scattering in air, or GPU transport. These restrictions apply
only when scattering is positive; legacy extinction-only media remain
unchanged.

Work is bounded independently by `steps` (1–64 camera-path samples) and
`maxDistanceMetres` (the maximum segment distance integrated for source light).
For a longer finite ray or an unbounded sky miss, source integration stops at
that distance from the camera, while extinction of the background continues
over the complete camera path. A zero scattering coefficient disables source
integration even if a budget was supplied.

Finite inputs can still create an unrepresentable source (for example, a very
bright lamp times a large scattering contribution) or a world-coordinate step
that overflows when converting the metre budget. The tracer checks sample
coordinates, source arithmetic, and final float32 pixel range. Such renders
return an explicit error rather than a frame containing NaN or infinity.

The focused CPU tests compare a directional source against both an independent
Beer–Lambert source integral and the existing `VisibleTable.SingleScatter`
primitive.

They also cover phase-zero semantics, zero-scattering bit identity, separate
step/path limits, invalid budgets and coefficients, ambient-source refusals,
extreme finite numeric inputs, and fail-closed unsupported scene inputs.

The authored `ocean.diffuse-sky-samples` set key opts in to that sky-only or
combined source path. `initial-media` must name the exact water boundary medium,
including its optical coefficients, sampling budget and unit scale. This is an
opt-in transport approximation, not an acceptance claim for the underwater
Hawaii shot.
