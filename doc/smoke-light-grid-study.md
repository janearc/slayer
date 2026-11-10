# Smoke-light grid study, 2026-11-04

The lighting grid's 64-cell longest side is an authored engineering choice.
It is a per-frame approximation; density keeps its original resolution and
the source term is rebuilt for each frame. It does not reuse old lamp light.

`TestSmokeLightingGridResolutionStudy` holds smoke, camera, time and lamp
fixed, and varies only the source-grid resolution. Both pictures use native
mesh traversal. The direct renderer supplies the lighting reference.

The fixture is the smoke column from `fire(t)`, with a black floor and one
coloured lamp, at 96 by 64 pixels and four samples per pixel. These are
fixture errors, not bounds for every scene or for an animation.

Measured summed absolute RGB error divided by reference RGB energy:

- 32 cells: 0.00236478934; worst channel error / image peak: 0.00998978493.
- 64 cells: 0.000999828484; worst channel error / peak: 0.00442093749.
- 128 cells: 0.000362361045; worst channel error / peak: 0.00150817125.

The existing two-percent summed-radiance fixture budget is retained at the
production 64 cells and the finer control. The 32-cell run is diagnostic.
This record does not establish that a given error was measured before the
implementation; the old comment making that claim was unsupported.

`TestCachedSmokeTracksDirectFireLighting` also isolates a lamp-only change
at time zero with the volume fixed, before retaining the original moved
volume and later-time case. A change in smoke geometry cannot satisfy the
lamp-only response check. All three pairs retain the same error budget.

Run through the authorized check runner:

```sh
game run check-light-grid
```

For hierarchy limits, GPU instance packing counts both world placements and
shared local members before allocating their packed buffers. It uses the
same two-gigabyte input cap as `metal.Grid`; this is not a total memory
budget for the host or device.

Median splits bound each software walk's pending nodes by tree depth plus
one. World, local group and triangle walks have separate 64-entry stacks.
The signed index and packed byte limits are stricter than that depth bound.

`TestMedianTreesFitTheKernelStackAtTheirPackedIndexLimits` checks the integer
bound and actual small hierarchies without allocating a limit-sized scene.
