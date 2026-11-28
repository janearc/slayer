# Sets through Omnipositor

The `scene` verb draws an ordinary GPU set job in separate contributions:
a sharp surface plate, then the light scattered into each camera ray and
the RGB transmission along it. These combine in linear light as
`scattered + transmission * plate`, before the Metal developer prints them.
The mist is the set's actual 3D medium. Beams and occluders stay in the scene.
There is no Moonlake ocean, shore, fixture layout or image-space fog substitute.

```yaml
version: 1
job: summit-job.yaml
from: 0
to: 48
transport-scale: 2
transport-rays: 1
plate-every: 48
```

```sh
game build omnipositor-binary
bin/omnipositor scene composite.yaml --out out/summit-composite --dry-run
```

For a render, submit the producer and its explicit inputs through
[Azazel](../../../doc/azazel.md). Use distinct new output directories for
the composite and the reference control. Both write numbered PNGs, a receipt with frame times and hashes, and, if requested,
a silent H.264 movie. `--raw` also retains linear PFM frames. `--dry-run`
checks the plan and job without staging or drawing. Both paths use the same
scene clock and Metal developer. Frame ranges are half-open job frame indices.

The plan's choices are visible approximations. `transport-scale: 2` traces
half the width and height; `transport-rays` counts rays along each pixel
axis. Depth-aware reconstruction avoids mixing mist across large depth
discontinuities. It cannot recover a thin tree or laser missed by every
coarse sample. Scenery samples stay separate until after compositing, so
the nearest needle does not hide mist seen through the rest of its pixel.

Scale 1 with one ray per pixel at both stages avoids spatial reconstruction
and avoids multiplying separately averaged quantities.

`plate-every: 48` keeps the first plate for 48 frames. That freezes geometry,
surface lighting, reflected volume light and volume shadows on surfaces.
Moving mist is still traced at every output instant. Pick a fixed view and
stationary scenery for this setting. Camera paths and zooms require
`plate-every: 1`; the command refuses to reuse their old view. A moving
instrument or a changing sky needs fresh plates too, even with a fixed eye.

For a decomposition control, use scale 1, one ray per pixel in both the
job and transport, and fresh plates. For speed, compare the reduced plan
against `--reference`, then inspect its silhouettes, beam continuity and
mist motion. A timing win does not by itself establish an acceptable picture.
No measured speedup is promised by this interface.

The GPU computes scattered light and transmission in one traversal. Scenery
and atmosphere share the normal scene encoder and kernels. Their outputs
are combined on the CPU with depth-aware weights; development uses the
existing Omnipositor Metal stage. Device allocations for a plate pass are
released before opening its transport pass. Output consumes one frame at a
time. This is not a fully device-resident compositor.

This first adapter accepts GPU sets with pinhole cameras and closed shutters.
It refuses stocks, titles, set HUDs and soundtrack inputs; add sound and titles in the
edit. It does not resume old directories. The receipt hashes the executable
and the job's declared input reads; it is not a recursive source-shelf audit.
Keep the set, source manifest and plan beside published studies as usual.
