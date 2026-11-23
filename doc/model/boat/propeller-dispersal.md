# Authored propeller trail dispersal

The existing generic propeller density field now supports two independent
optional approximations: advected centreline meander and radial spreading.
These do not predict vortex instability, vapor collapse, gas persistence or
bubble sizes from pressure. The existing pressure deficit/recovery inputs
still independently limit the active length, so spreading cannot prolong it.

`wander-m` bounds each of two transverse displacement coordinates. Its seed
is the existing `fragment-seed`; separate blade and channel coordinates give
continuous independent displacements. Two-octave gradient noise is sampled
at the convected coordinate `(distance-slipstream*time)/wander-scale-m`.
Each noise channel is explicitly clamped to ±1 for a rigorous support bound.

A smoothstep ramp from zero to one over the first `wander-scale-m` anchors
the centreline and its displacement derivative to the rotor tip. This is not
new random noise per frame. The same material packet carries its meander.

`spread-start-m` locates the beginning of radial spreading; `spread-per-m`
grows the effective Gaussian width per downstream metre beyond it. Each
filament's density is multiplied by `(original-core/effective-width)^2`,
preserving its cross-sectional density integral before the independent
pressure, fragmentation and endpoint masks. Overlapping broadened filaments
can read as a dilute cloud.

This is an optical population envelope, not an
individual physical bubble radius or a conserved gas-flow model. It requires
no extra particle meshes or CPU voxel bake. Existing residual-gas Fizz remains
a separate, optionally authored medium.

```yaml
wander-m: 1.2
wander-scale-m: 3
spread-start-m: 4
spread-per-m: 0.12
```

All four options default to zero. Nonzero wander requires a scale0.01..100m;
amplitude is bounded0..10m, spread start0..200m and rate0..0.5m/m. These are
authored numerical work/extent limits. At zero wander and zero spread the
existing density operations remain unchanged. No exposure, emission, rotor
RPM or shutter changes accompany these controls.

The finite support uses the old tip radius plus `sqrt(2)*wander-m` plus
three times the maximum broadened core. That transverse support is projected
onto each world axis, including skew rotors. Metal additionally refuses
nonzero float32-subnormal parameters, core-square underflow, and meander
noise coordinates beyond the documented numerical2^20-cell budget.

The Metal card is append-only: existing indices0..26 remain unchanged;
indices27..30 are `wander`, `wander_scale`, `spread_start`, `spread`.
The fixed card is now31words/124bytes, including disabled fields. This is an
ABI size change; raw cards are not byte-identical to the old108byte layout.
Unchanged defaults refer to authored scene behavior/density/rendered pixels.

Run `game run check-propeller-dispersal` for constructor/schema, anchored
and convected displacement, approximate area-integral dilution and support
checks. `game run check-propeller-dispersal-metal` runs actual Metal density
and volume controls, including legacy and dispersed multi-blade fields at
multiple clocks. Tests include nonzero and exact-zero samples.

Ordinary no-optics kernels retain the exact legacy shader source and its
original 27 constants; the strict source-hash baseline remains unchanged.

Named optical kernels include the extended implementation. A standalone
field with nonzero wander or spread selects an independent extended shader
variant without adding named-medium buffers. Its 31-word card is the same
as the optical path's card. Standalone controls are not silently ignored.
The frozen legacy shader text is retained solely for this compatibility
contract, not as another scene constructor.

The isolated standalone volume-image control disables the independent legacy
fragment mask and retains the general normalized-mean bound of 1e-5.

A separate automated coupled image retains fragmentation, wander and spread
at 97 by 65 with one sample per pixel. Its independently reviewed, fixture-
specific contract requires normalized mean error at most 2e-5 **and** maximum
absolute RGB error at most 3e-6. Finite output, coverage, depth and the existing
normalized-largest bound remain checked. No global tolerance is changed.

The Apple M3 Max review measured normalized mean 1.2110e-5 and maximum
absolute RGB error 1.6093e-6 for that coupled fixture. The previously recorded
6.2389e-6 value was normalized-largest error, not absolute RGB error.

Float32 card, coordinate, noise and trigonometric evaluation accumulate error
relative to CPU doubles. The chosen hard Gaussian support at three standard
deviations is discontinuous and can amplify tiny positional differences near
its boundary; review at sample sides two through four found rare larger
absolute outliers.

The local one-sample fixture contract is not a universal
precision guarantee or a physical validation of cavitation. Multi-blade,
multi-clock density probes and real-scene comparisons remain separate checks.
