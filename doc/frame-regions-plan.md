# Draw a region

a region keeps the camera, pixel scale, sampling coordinates and
exposure of the complete frame. its output has the selected rectangle’s
width and height. coordinates start at the upper left of the full frame.

```sh
slayer draw JOB --region 20,30,80,60 --dry-run
```

the four values are left, top, width and height. the position must
be nonnegative and the positive rectangle must fit the resolved
frame. the CLI setting overrides `film.region` in the job.

## Supported draws

`slayer draw` supports regions of set stills on the CPU and software
Metal, through a pinhole with per-side rays. use plain development
without bloom. stocks, titles, HUD overlays, thin lenses, native
traversal, lighting shortcuts, initial media and exact-total sampling
are outside the region path. `slayer assess`, `watch` and `develop` refuse
authored regions. a dry run resolves the job and paths without staging,
so it can succeed before a renderer or development refusal.

staging can still pay for the full scene. on a shared machine use
[Azazel](azazel.md#submit-work) for the draw. the
[Metal design](region-metal-design.md) describes the two coordinate
spaces and the [comparison guide](region-comparison-design.md)
shows how to compare named regions against higher-sample references.
