# Authored Alfa bubble trail and cavitation study

There are two distinct models. `manufacture.NewCavitationCloud` creates
large, explicit gas-sphere geometry for close views. A separate set-level
`fizzes` entry can opt its sub-resolution density field into the CPU's bounded
clear-water/cloud composition with `clear-water: true`.

Despite the legacy constructor name, the sphere set is not propeller vapour
cavitation. `medium.Fizz` is an authored rising residual-mist field. The
evidence, missing Alfa operating inputs, and proposed phase-locked vapor
filament/sheet controls are documented in
[alfa-propeller-cavitation.md](alfa-propeller-cavitation.md). Neither current
field models pressure-driven inception or collapse.

`manufacture.NewCavitationCloud(config, pose)` returns a reproducible set of
map-space sphere centers. The constructor is bounded to 512 spheres, 100 m/s,
10 seconds, 500 m of trail, 100 m of rise, 50 m of lateral spread, 5 m radii,
and map coordinates within the documented world envelope. The screw-to-cloud
gap is the named `ScrewOffsetM` value (0.8 m in the proof recipe); `ScrewUpM`
places particles at the chosen screw axis above the keel.

The seeded longitudinal distribution represents a trail chosen from speed and
duration. Its buoyant rise and irregular spread are authored positions. It is
not a cavitation, wake, bubble-dynamics, or fluid-transport simulation. The
spheres are unsuitable as a dense mist representation: the comparison renders
show them as beads.

## Bounded participating cloud

`clear-water: true` wraps the existing finite `medium.Fizz` density field in
`medium.ClearCloud`. The volume's `colour` supplies its single-scattering
albedo; its density supplies local cloud extinction. Setting colour to black
leaves cloud extinction on while disabling the cloud's scattering source.

Omitting the volume disables both.

Named clear-water extinction is independent
and is applied between each cloud sample, so cloud-added radiance receives the
water attenuation from its sample to the eye.

For sky illumination, author
`ocean.clear-cloud-sky-samples` together with an infinite mean-plane ocean
continuation. That route refracts air-side sky directions at the mean plane,
applies water Beer attenuation and local-cloud extinction on the incoming path,
and applies water/cloud extinction on the outgoing camera path.

It is a bounded
single-scattering approximation; it does not resolve wave-scale refraction or
multiple scattering.

The CPU-only opt-in requires each visible volume in a named-clear-water world
to be a `ClearCloud` no wider than 500 scene units per axis.

It can combine
named-water single scattering only with the explicit ocean diffuse-sky source
route and `clear-cloud-sky-samples`; water source samples use incoming water
Beer and cloud visibility, and outgoing source is attenuated through the
interleaved water/cloud segments. Unmarked volumes, other transparent source
boundaries, and gravitational fields remain refused.

GPU rendering explicitly
refuses this mixed path until it implements the same segment ordering.

The
model is a bounded, single-scattering density approximation; it does not
resolve bubbles, phase change, fluid motion, or hydrodynamic cavitation.

A
60 m actual lunar draw with both water scattering and cloud source still
reaches an independently tracked infinite-mean-plane boundary miss, so not
every scene source path succeeds.

`game run cavitation-cloud` invokes the Puzzlemouth `cloud` verb and writes a
YAML `spheres:` fragment to stdout.

CLI values are declared by Puzzlemouth;
there is no second flag parser. Use the fragment only in a proof scene, and
record the exact generated fragment with the render inputs.

## Explicit mixed-water sampling budget

`world.ocean.clear-cloud-water-step-mode` selects `per-gap` (the default,
including omission) or `ray-spacing`. The legacy mode repeats the water
medium's full in-scatter step budget in every gap between local cloud samples.

The opt-in mode instead derives cell metres from the capped full camera ray:
`min(ray metres, max in-scatter metres) / in-scatter steps`. Each gap receives
`max(1, ceil(min(gap metres, max in-scatter metres) / cell metres))` samples.

The medium identity, physical coefficients, exact Beer transmittance, cloud
sampling and incoming source transport do not change. Gap boundaries can add
cells beyond the original full-ray budget. This is a configurable quadrature
quality/performance tradeoff, not a bit-exact internal optimization: sharp
shadows or source variation may need a higher authored water step budget.
