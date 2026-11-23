# Ocean water

## Scene and data contract

Set-file coordinates use Slayer’s existing east/north/up convention: x east,
y north, z up. Staging converts them to the renderer’s east/up/south world:
x east, y up, z south.

One world unit is one metre, with mean sea level at authored z=0 and
renderer y=0. Georeferenced terrain is transformed to a nearby local origin
before GPU packing, with datum and source recorded. In renderer coordinates
the surface evaluates a time-dependent height `h(x,z,t)` and an outward
normal pointing toward air; water is below it. Place the complete hull in
this coordinate frame; the ocean and hull intersect geometrically.

Do not cut or hide the underwater hull to simulate contact.

The first CPU attenuation slice has a strict set-material field in both
ecosystem schemas. Put `extinction-per-metre: [red, green, blue]` under a
material's `clear` block. The material key is the stable medium ID, and that
key may also be listed in `camera.initial-media` for an explicit camera start.

The material must carry a nonempty `source` provenance string; plain and quoted
YAML scalars are accepted for this field. Values are RGB inverse metres and
describe unscattered Beer-Lambert extinction only.

Omit the field to retain the
legacy interface-only behavior.

Staging converts the values using the world's
metres-per-unit scale. World-only non-unit scales are supported; terrain-backed
worlds with optical materials currently require `unit: 1` because their
coordinate-scale contract is not consistent.

CPU and opt-in Metal tracing
support named clear-medium extinction and an explicitly authored fixed camera
stack; both refuse coexisting volume/fizz or gravitational fields. A moving
job camera is refused.

This does not implement scattering, underwater image
formation, or direct-light/sun refraction. No measured site coefficients or
preset are supplied.

The staged session also exposes `Session.OceanSpectrum()` and the session and
Door runners report its summary once per staged scene when an ocean is present.

The summary uses `oceantex.Sea.Height()` and, when a peak exists,
`2π/Sea.Peak()`; it labels both as spectrum-model estimates rather than
observations and records the authored surface inputs needed to reproduce the
estimate. Calm inputs report zero significant height and no peak period.

These
values describe the spectrum model: a finite, sampled wave field need not
reproduce the full-spectrum estimate. No-ocean jobs emit no new report line.

This is diagnostic output only; it does not add scattering, alter the wave
field, or establish ocean conditions at a real site.

An explicit `flat-mean.extends-to-infinity: true` option is
available for CPU authored ocean sets. It keeps the sampled core and smooth
collar, then extends the chosen mean level as an infinite flat plane.

Omission
preserves the existing finite footprint and edge behavior.

This is a geometric
far-field approximation: it does not add infinite waves, tiled spectrum,
reflections, or hydrodynamic behavior. Its ray hit solves the plane analytically
outside the collar and traverses the bounded core/collar on CPU. Metal encoding
refuses the option explicitly; CPU/Metal parity is not claimed for it.

Within the sampled core, continuation delegates to the exact per-cell
bilinear HeightField intersection solver.

Conservative advancement is limited
to the smooth collar, split around the core footprint so the earliest collar
or core hit is retained.

The collar remains an authored approximation with bounded
iterative traversal.

## Sources

The existing `oceantex/sources.yaml` records provenance for a refractive-index
fit, pure-water absorption, wave spectra and other parameters. It records
pure-water absorption samples attributed to Pope and Fry, and a seawater
refractive-index relation attributed to Quan and Fry. Those have temperature,
salinity and wavelength limits. These are not site-specific optical
measurements, and pure-water data is not a coastal-water preset.

The [PBRT volume-scattering chapter](https://pbr-book.org/4ed/Volume_Scattering)
and [equation of transfer](https://pbr-book.org/4ed/Light_Transport_II_Volume_Rendering/The_Equation_of_Transfer)
provide the reference separation between transmittance and added in-scattering.
They are explanatory references, not sources of any coefficient.

The NOAA Hawaii CRM2023 Big Island float GeoTIFF covers longitude -156.2 to
-154.7 and latitude 18.8 to 20.4, and supplies land and bathymetry within
that coverage. Preserve the vertical-datum caveat in `doc/island-scene.md`;
it is not an optical water measurement.

The existing code's wind/fetch inputs do not establish local San Onofre or
Hawaii sea conditions. Its sparse, retyped 23-point chlorophyll table is not
a local climatology.

## CPU unscattered clear-medium transport

`thing.NewClearMedium` makes an immutable, named RGB extinction snapshot in
inverse metres, plus an explicit metres-per-world-unit conversion. A
`scene/gas.VisibleTable` can make the snapshot at the CPU tracer's existing
610, 550 and 465 nm channels.

Assign the same descriptor to each transparent
boundary for that interior and, when the camera starts inside it, to the
matching `trace.InitialMedium`. Reusing an ID with different coefficients,
units, refractive index or rank is refused before tracing.

The optical
descriptor is a pointer to private copied values, so many faces can share it
without embedding coefficients in every material or ray stack entry.

The CPU applies Beer-Lambert extinction to each camera/refraction segment,
using the float64 hit distance before `frame.Pixel.Depth` is rounded for
display. This is unscattered transmittance only: it adds no scattered light,
and it does not attenuate or refract direct lamp/sun paths.

To avoid silently
reordering volume source terms, a world with named clear-medium transport and
volumes or gravitational fields is refused. Metal explicitly refuses the new
material field until it implements matching transport. This slice is not a
complete ocean model.

Set-authored CPU use opts in per named material:

```yaml
world:
  unit: 2 # metres per scene-coordinate unit
  materials:
    validation-water:
      clear:
        through: 1
        index: 1.333
        gloss: 1
        extinction-per-metre: [0.1, 0.2, 0.3]
      source: synthetic validation values; not measured water
cameras:
  below-surface:
    initial-media: [validation-water]
```

`extinction-per-metre` is optional; without it, existing clear-material
behavior is unchanged. Its three authored values are RGB inverse metres, and
the material key is the medium ID. A camera is inside the medium only when
`initial-media` names that clear material; staging never infers containment
from geometry. Optical extinction requires nonempty explicit provenance, such
as the clearly labelled synthetic source above.

A world-only set applies its
`world.unit` conversion. Terrain-backed world-half coordinates are authored
and rebased in metres, while ordinary world-half shapes are not scaled, so
optical extinction with terrain and `world.unit` other than 1 is refused.

This authored material path is supported by CPU and Metal surface tracing for
fixed cameras; it refuses optical transport mixed with volume or
gravitational-field transport. Metal also refuses its unsupported positive
in-scattering path and composite passes with optical state. It adds no
scattering or direct-light attenuation. It supplies no measured water preset.

## Limits

A separate, opt-in CPU in-scattering prototype integrates a bounded
homogeneous segment under directional light, but refuses sky, clear
boundaries, ambient lamps, volumes and fields; its incoming lamp light is
unattenuated and unrefracted. See `trace/clear-medium-in-scattering.md`.

On the CPU, `trace/crowd.go` treats an object as non-shadow-casting when
`Thing.Attrs.Clear` is set; `Material.Through` alone does not make the object
transparent to shadow rays.

`trace/stage.go` otherwise treats a surface hit as
a blocker before applying volume transmission. Consequently the authored clear
water material by itself cannot deliver physically correct sunlight into water.

Simply disabling its shadow is a separately named approximation: it omits the
interface Fresnel share, bending and possibly the correct light-path length.
Existing overlapping independent volumes also do not by themselves enforce
the clear stack's highest-rank active medium.

The finite wave field is an open, bounded surface with known
grazing-intersection limitations.
