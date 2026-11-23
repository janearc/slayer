# Sky light in weather and smoke

Painted media can now opt into normalized isotropic single scattering from
both explicit lamps and the sky. This puts weather under the same directional
sky radiance that illuminates surfaces. It does not infer weather or solve
multiple scattering between clouds, smoke and surfaces.

The island scene enables it explicitly:

```yaml
volume-lighting:
  model: isotropic-single
  sky-samples: 16
```

Even sample counts from 2 through 64 are accepted. In Go, construct the checked
`thing.VolumeIllumination` value with `thing.NewVolumeIllumination` and assign
it to `world.VolumeIllumination`. Omission keeps the earlier lamp-only lighting.

## Radiance and direction

The isotropic phase density is 1/(4*pi) per steradian. Uniform sphere sampling
cancels that density, so the incident sky source is a radiance mean. It has no
surface cosine term. Every direction checks opaque geometry and extinction
through the intervening media. The medium's colour multiplies that incident
source as single-scattering albedo; emission is added independently.

This follows [PBRT 4e, Phase Functions](https://www.pbr-book.org/4ed/Volume_Scattering/Phase_Functions)
and [The Equation of Transfer](https://www.pbr-book.org/4ed/Light_Transport_II_Volume_Rendering/The_Equation_of_Transfer).
The current painted-medium model is isotropic. It does not describe measured
droplet asymmetry, polarization or repeated scattering inside an optically
thick cloud.

The earlier lamp factor was an authored 0.5. The opt-in model uses 1/(4*pi)
for those lamps as well as normalized sky scattering. That is an intentional
brightness change: adding sky light while keeping the old lamp factor would
combine two different scattering normalizations. Existing scenes retain their
earlier response unless they select the new mode.

Existing fire and ecosystem scenes select the same model with
`scene.volume-illumination`, containing `model` and `sky-samples` as above.
Their existing `scene.volume-lighting: direct|cached` setting still selects
the smoke-lighting backend; it does not select a scattering normalization.
Both ecosystem schemas retain that distinction. Legacy scenes omit the new
block and keep their earlier lighting.

Directions use fixed world axes and antipodal pairs. Their locations do not
rotate with the camera or change randomly between frames. Surface illumination
still uses its existing cosine-weighted hemisphere integration; sharing a sky
does not mean those two integrals should share their angular weights.

## Cost

The initial path evaluates sky shadows and medium transmission at each ray
step. This is a direct reference path, not a performance improvement. Existing
smoke source grids include the new source when enabled. No time-lagged
lighting cache is introduced by this change.

## Bounded incident-light preparation

An optional native grid now prepares the incident sky mean once per node per
frame. Different media share that incident field, then apply their own colour
and emission at the actual ray sample. Explicit lamps, including guide lasers
and fire emitters, remain outside this particular cache.

```yaml
volume-lighting:
  model: isotropic-single
  sky-samples: 16
  sky-grid:
    low-metres: [-40000, -2500, -80000]
    high-metres: [40000, 6500, 0]
    nodes: [32, 32, 32]
```

These are authored work and interpolation controls for a regional study.
Bounds use the scene's fixed anchored east/up/south coordinates, not camera
coordinates. The adapter shifts them into each render origin while keeping
their physical positions fixed. They do not describe source-data coverage.

Each dimension has 2 through 128 nodes; their product is limited to 262144.
The RGB payload is at most three MiB. Native encoding also refuses finite
double bounds whose nodes collapse after float conversion. This payload bound
does not cover geometry, density, other buffers or the entire process.

Trilinear interpolation applies inside the domain. A one-cell smooth band
blends to direct integration at the boundary; points outside use the direct
path. The grid samples include both faces of each dimension. CPU rendering
continues to integrate directly, providing a reference for this approximation.

The grid is an approximation, not a general error bound. Clock and
opaque-geometry changes refresh the source.

`EnvironmentWall`, `EnvironmentGPU` and `EnvironmentNodes` report this separate
preparation. `TraceGPU` excludes it. Total cost must include both, including
the source-grid readback and subsequent input upload. Boundary motion and
broader scene accuracy still require their own comparisons.
