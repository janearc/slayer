# Overhead surface-fire study

This is a dated development record. Its timings and check results belong to
the revisions named below. Use the [Game workflow](developing.md) for current
setup and commands, and [the committed 4K review](review-4k-2026-11-05.md)
for the later dense-forest measurement.

The 720p study renders one initial hot spot in a 16 m square of uniform surface
fuel. Temperature spreads and fuel is consumed by the coupled model. There are
no scheduled per-tree burn windows in this scene.

The [design and source](fire-fuel-design.md) explain the Mandel grass proxy, its
coefficients, the authored flame display, and the horizontal-layer limits. Tree
crowns, wind and transported smoke remain future work.

The local outputs are in Grissom's `local/renders/fire-fuel`:

- `overhead-720p-8x.mp4`: 1280x720, 180 unique frames, 24 fps, 7.5 seconds.
- `progression.png`: actual frames at 0, 30 and 59.7 simulation seconds.
- `poster.png`: the labelled frame at 30 simulation seconds.
- `frames/`: all original PNGs and their checked sequence manifest.
- `validation.json` and `video-validation.json`: hashes, times and format.

The movie plays one minute of simulation at eight times speed. It is a
propagation diagnostic; its accelerated flicker does not show real-time flame
motion. Rendering used the frozen `dfc6050` executable, two CPU slots and the
native GPU path, with one ray per pixel.

All 180 source hashes and physical timestamps were checked. The encoded movie
decodes to 180 frames. The BT.709 matrix, sRGB transfer and BT.709 primaries are
tagged explicitly; adding the tags left decoded pixels at 30 seconds
byte-identical.

This is a separate encode from the night film. The night film retains its
original BT.601 interpretation by stream-copy tagging; it was not re-encoded
to normalize the two matrices. Each result records its own checked metadata.

Recorded frame time averaged 2.231 seconds, median 2.241, maximum 3.753. These
timings apply to this controlled surface layer at 720p. The retained fuel
stencil runs on Metal; flame grids and native scene tracing also use the GPU.
Surface display geometry and source aggregation remain on the CPU.

## Run the scene

With the source shelf set up, build through Game and use the checked plan:

```sh
game build
GOMAXPROCS=2 game run hillside -- sequence --plan doc/fire-fuel-sequence.yaml
```

The plan writes actual frames at three per simulation second. Encoding them at
24 fps makes the diagnostic run eight times faster. The local movie adds labels;
the renderer's original PNGs are kept separately.

## Checks

Focused model, config, staging and CPU/GPU conformance checks pass, including
heat/fuel balance, insulated-boundary heat conservation, neighbour heating, a
missing-fuel break, replay, convergence and multiple Metal threadgroups. The
earlier five manual fault trials have no preserved replay patches.

Their claimed outcomes are withdrawn as acceptance evidence.

The fuel solver measured 92.9% statement coverage before the final boundary
checks. No new hard lint findings remain in its implementation or prose. The
full Game check is not green: the historical harness rule rejects `03d6b0a`, and
the smokecheck suite reached its ten-minute limit.

Every other package passed that run, including the complete GPU and hillside
suites. Focused checks passed again after the final frame-tally adjustment. Full
mutation and independent review gates remain; this is a local prototype.
