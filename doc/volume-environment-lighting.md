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

Existing Hillside fire and ecosystem scenes select the same model with
`scene.volume-illumination`, containing `model` and `sky-samples` as above.
Their existing `scene.volume-lighting: direct|cached` setting still selects
the smoke-lighting backend; it does not select a scattering normalization.
Both ecosystem schemas retain that distinction. Legacy scenes omit the new
block and keep their earlier lighting.

Directions use fixed world axes and antipodal pairs. Their locations do not
rotate with the camera or change randomly between frames. Surface illumination
still uses its existing cosine-weighted hemisphere integration; sharing a sky
does not mean those two integrals should share their angular weights.

## Evidence and cost

A local diagnostic ran CPU, software Metal and native Metal controls. Uniform
sky results fell within independent energy bounds for a constant-density cube.
A black lower-hemisphere blocker reduced the scattering into the independently
bounded half-sky range. Directional illumination matched an analytic attenuated
single-scattering integral, and pure emission retained its original response.

The checked native directional value was 0.00655871 against 0.00655877 from
the analytic reference. The legacy control was 0.04120956 against 0.04120999.
The declared allowance was 0.00002, including float32 and midpoint integration.
These small controls do not establish full-scene visual or temporal acceptance.

The initial path evaluates sky shadows and medium transmission at each ray
step. This is a direct reference path, not a performance improvement. Existing
smoke source grids include the new source when enabled, but general weather
source caching, spatial error controls and moving-view comparisons remain open.
No time-lagged lighting cache is introduced by this change.

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

The initial native controls compare 16-cubed and 32-cubed grids with direct CPU
transport in a constant-density box. Raw red differences were 0.0000947 and
0.0000131 under an authored 0.0002 study allowance. This is not a general error
bound. Clock and opaque-geometry changes refreshed the source; an outside-grid
control matched direct native transport within float precision.

`EnvironmentWall`, `EnvironmentGPU` and `EnvironmentNodes` report this separate
preparation. `TraceGPU` excludes it. Total cost must include both, including
the source-grid readback and subsequent input upload. Boundary motion and
broader scene accuracy still require their own comparisons.

A matched native 720p pair retains the same terrain, objects and population
records, including 36233 trees represented by geometry. The 64-cubed source
grid's developed RGB differences have mean 0.0438 and maximum three code values
against direct integration. These observations are not a JND or temporal
acceptance criterion.

That frame's grid preparation used 0.014 GPU seconds and 0.051 wall seconds;
tracing used 1.00 GPU seconds against about 6.0 seconds for direct integration.
A warm direct command took 15.0 seconds versus 12.3 seconds cached. Compilation
and CPU preparation varied between commands, so that total is a recorded pair,
not a general end-to-end speedup. Source and image bytes accompany the studies.

The same 4K scene reaches its existing 150000-instance allowance before GPU
rendering. The lighting improvement does not establish the 4K animation goal.
Vegetation preparation, grouping and residency remain active work.
