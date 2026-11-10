# Animation performance and catalogue work

This is a dated development record. Its timings and check results belong to
the revisions named below. Use the [Game workflow](developing.md) for current
setup and commands, and [the committed 4K review](review-4k-2026-11-05.md)
for the later dense-forest measurement.

The target, 2026-11-03: a thirty-second animation at 24 frames per second and
3840 by 2160, completed within eight hours. That is 720 frames and 40 seconds
per frame averaged over the whole job.

Scene preparation, rendering, development, writing frames, and movie encoding
share that budget. This is a target, not a measured capability.

The measured machine is an Apple M3 Max with 40 GPU cores. The existing
observatory reference has 5000 trees, one optical enclosure and prescribed grove
mist. Its earlier 3840 by 2400 image used 16 rays per pixel and took 9 minutes
23.906 seconds after construction.

That image is a quality reference. The baseline below uses the current geometry,
includes construction, and uses the target dimensions; the two times are not a
controlled comparison of an optimization.

## What is already implemented

- Geometry pooling and shared local shoot hierarchies on CPU and GPU.
- Baked wispy volume densities and boxed-medium sampling.
- Content-addressed weather checkpoints and smokecheck render results.
- Metal sessions that retain buffers, currently used outside the frame tracer.
- Early-exit opaque shadow traversal in the GPU software hierarchy.

The November caching brief listed some of these as unchecked or absent. The
frame tracer still builds its encoding, opens its trace kernel, and allocates
input/output buffers on each Sampled call. The legacy shortcut declarations for
Skip, Distance, Once and Cutoff do not implement those modes.

## Work sequence

1. Measure construction, encoding, bake wall and GPU time, trace compilation,
   dispatch/readback wall and GPU time, finishing, and image output. Record
   input and sample-array bytes as well as seconds. Physical lens exposures
   currently get complete wall timing but not per-glimpse backend breakdowns.
2. Retain a prepared scene and GPU resources across frames. Prove cold and
   warm frame equality, including camera updates and invalidation on changes.
   Cache reusable static encoding across runs with atomic writes and explicit
   memory/disk budgets supplied through Puzzlemouth.
3. Use measurements to choose the next exact traversal improvement: material
   trait masks, coherent ray bundles, and potentially hardware traversal.
   Report a changed order when the measured bottleneck warrants one.
4. Evaluate boxed foliage/transmission tables and medium traversal shortcuts
   as explicit approximations. Measure pixels past one perceptual JND and the
   worst pixel. Do not silently replace detailed foliage or discard light.
5. Add an animation runner with retained resources, frame provenance,
   resumable output, a camera path, and a measured whole-job projection.

The October caching agreement and the 2026-11-03 brief remain the design
sources. Exact caches must match their uncached result byte for byte.
Approximate modes need a declared error budget and independent measurements.

The agreed still budget is about two percent of pixels past one perceptual JND;
a final sequence needs an explicit choice and temporal checks.

## Catalogue work continuing alongside rendering

- Vegetation: deciduous branching and foliage; manzanita and chaparral brush;
  Joshua trees and saguaros; then further native vegetation families.
- Observatories: additional optical enclosures, radomes, radio dishes and
  arrays. Recipes describe families rather than copying named facilities.
- Fire: parameterized fire sources with GPU flames, smoke and lighting.
- Atmosphere: shared ecosystem fields, with sourced physical inputs where
  used and authored appearance controls clearly identified.
- Catalogue: browsable recipes with parameter descriptions, complexity
  previews, citations and recorded GPU support.

Every new CLI or configuration door goes through Puzzlemouth. Work stays on
local branches. The original measurements used four CPU workers. Current
shared-machine checks use the serial limits in the [Game workflow](developing.md).
Changes need review before merge. The eight-hour target depends on the workload.

## Measured 4K baseline, 2026-11-03

[The benchmark scene](observatory-4k.yaml) keeps the dense forest at `quick`,
the optical enclosure at `demo`, and 16 rays per pixel. It uses the USGS terrain
on the source shelf. No optimization or sampling change was applied.

After `game build`, run `game run hillside -- -set
doc/observatory-4k.yaml -out baseline.png` through the authorized render queue.
Application arguments need the forwarding fix described in the
[Game workflow](developing.md).

This measurement used an explicit owner exception to run one direct job,
with four CPU workers.

- Scene construction and parsing: 3.235 seconds.
- Encoding, including the boxed grove density grid: 2.104 seconds.
- Trace kernel compilation: 0.206 seconds.
- Separate wispy-density GPU bake: none for this scene.
- Dispatch and readback wall: 504.030 seconds.
- GPU trace time inside that dispatch: 502.615 seconds.
- CPU finishing: 0.375 seconds.
- Image development and output: 1.817 seconds.
- Whole command: 511.769 seconds, or 8 minutes 31.769 seconds.

The encoded inputs occupy 75,898,768 bytes. The full sample array occupies
6,370,099,200 bytes before averaging. GPU time is nested inside wall time;
adding those two measurements would count the same work twice.

At this frame's cost, 720 frames would take about 102 hours, before movie
encoding. The eight-hour target needs about a 12.8-fold improvement. This is a
projection from one still, not a timed camera sequence.

GPU tracing occupies about 98.2 percent of the measured command. Even free
construction, compilation, transfers and finishing would leave about 503 seconds
per frame. Resource reuse remains useful, but cannot by itself meet 40 seconds
at this scene's quality.

This changes the investigation order: isolate geometry visibility from medium
integration, then prioritize traversal and repeated shadow work. The next small
diagnostics use the same binary and sampling, with and without mist; they are
workload comparisons, not quality-preserving wins.

Hardware traversal is a candidate, not an implemented speedup. Apple's [M3
overview](https://developer.apple.com/videos/play/tech-talks/111375/) describes
its ray-tracing hardware.

The [Metal ray-tracing
guide](https://developer.apple.com/videos/play/wwdc2023/10128/) describes nested
instancing, which may preserve the shared shoot hierarchy.

The binary's SHA256 was
`53f867b3d4a41c8ab6a3f0c6ebca0c370d415079d1748f20432cc217fa9fb176`.

The scene's SHA256 was
`9d58c8b173d1550232d8514ca840592770671daf6f98c4b96f97aa6da5f6103d`.

The PNG, complete log and job description are retained outside the repository in
the local render shelf under `renders/animation/01-baseline-*`.

## Diagnostic renders and mist reuse

The same binary drew the same view at 960 by 540 and 16 rays per pixel:

- With prescribed mist: 44.951 seconds of GPU tracing, 50.347 seconds total.
- Without that mist: 33.261 seconds of GPU tracing, 38.612 seconds total.

The mist added about a quarter of the measured trace cost at this size. Geometry
visibility remains expensive without it. These are workload diagnostics, not
quality-preserving optimizations, and they do not establish a separate mist cost
at 4K.

The proposed reuse, 2026-11-03: batch neighboring rays, march a pilot ray to
the mist, and reuse what it says about how far to march. There are two useful
contracts to distinguish before implementing that idea.

A conservative packet bound encloses every ray in its tile. Regions proved empty
can be skipped for the packet; a representative ray's miss alone does not prove
its neighbors miss. For the current interpolated density grid, the proof must
cover all interpolation corners and their halo.

A pilot can suggest a stepping distance, with each ray checking the bound on its
own path. To remain byte-exact against today's deterministic march, skipping
must preserve the original positions and order of nonempty samples. Adaptive
resampling is a separate approximation, even if physically sound.

The current density grid avoids reevaluating the grove field, but every lit
volume sample still traces toward the lamps and marches their paths through the
volumes. Caching that visibility and transmission can remove more work than
sharing the entry distance alone.

Interpolating cached transmission or lighting is approximate. Its test is the
agreed share of pixels beyond one JND, with the worst pixel beside it, plus
camera-motion checks for temporal changes. Bright emission behind a dense near
cloud must remain visible when its attenuated light matters.

Cache ownership belongs to the render. Geometry, material, lamp, wind and time
changes invalidate affected facts. Camera-independent world grids may survive
camera motion; camera-packet bounds must be rebuilt for that view. No absolute
faintness cutoff is added by this proposal.

The algorithm sources are PBRT, fourth edition, [Media, sections 11.4.1 and
11.4.3](https://pbr-book.org/4ed/Volume_Scattering/Media), for conservative
segment bounds and grid traversal.

Tom Lokovic and Eric Veach, [Deep Shadow Maps, SIGGRAPH
2000](https://graphics.stanford.edu/papers/deepshadows/), describe fractional
visibility over depth, including volumetric shadows.

These are design sources, not a claim that Slayer implements either paper.
PBRT's null-scattering estimator differs from Slayer's fixed-step integrator;
adopting a data structure does not make their transport algorithms equivalent.

## Checkpoint validation and remaining gates

This checkpoint adds measurement and records the next work. It does not
implement a scene cache, packet traversal, cached transmission or hardware
traversal. The eight-hour target remains unmet.

- Game build passed with the measured renderer.
- Every functional package passed the latest Game check, including the GPU,
  hillside and CPU smoke suite. The whole command still fails its history
  rule: earlier commit `03d6b0a` changed the independent harness and the
  production code together. That needs review; no waiver was added here.
- The first CPU smoke run reached Go's ten-minute timeout. The repeat
  completed with its previously finished render-cache entries retained.
- The measurement comparison checks every returned float's bits and requires
  real surface hits. The earlier manual fault trials have no replay patches
  and are not acceptance evidence. SampledMeasured had 100 percent statement
  coverage in that focused run, not full-package coverage.
- Game's mutation framework cannot run its unchanged baseline with the
  published Puzzlemouth dependency: it disables the required Go workspace.
  Its result is blocked, not a completed mutation run.
- Game sweep passed for the checked HEAD. Repository-wide lint and shuriken
  remain red, including findings on the new public diagnostic record.
  Okapi, the requested separate prose pass and independent review remain.

The companion Puzzlemouth version must still be published and pinned before a
standalone Slayer release. Local workspace success does not resolve that
dependency. No release or push is part of this checkpoint.

## Rejected experiment: smaller private traversal stacks

The three software hierarchy walks reserve 64 pending nodes per thread. A
32-entry variant covers balanced trees within their signed index range. It
produced identical sample bits for meshes, textures, mirrors, mist and shared
recursive redwood foliage; Game build also passed.

The 960 by 540 views kept their sampling and produced byte-identical PNGs:

- With mist, GPU time was 44.838 seconds, versus 44.951 seconds before.
- Without mist, GPU time was 34.177 seconds, versus 33.261 seconds before.

There is no useful speedup in this comparison. The production capacity stays at
64. The variant, tests and logs remain local experimental artifacts; no
smaller-stack rendering mode is added.

## Native mesh shadows: measured next step

The experimental `gpu-native-shadows` YAML mode is now implemented. On the same
4K scene it reduced whole time to 87.238 seconds and GPU tracing to 78.938
seconds. The developed-PNG error is 0.10753038 percent past one JND, with a
worst pixel of 8.001596 JND, inside the agreed two-percent still budget.

Fast rendering should be accepted by image quality, rather than requiring
matching bits. Lossless cache checks can still compare bits; accelerated
numerical paths use perceptual and temporal checks. The [native-shadow
notes](native-shadow-traversal.md) record details and limits.

The eight-hour animation target remains open. This frame's cost projects about
17.45 hours for 720 frames. Primary traversal, resource reuse and packet/mist
caches are the next candidates; motion has not been validated.

## Native primary mesh queries

The explicit gpu-native mode now includes closest mesh queries. Its first 4K
forest measured 64.057 seconds whole and 52.703 seconds GPU, keeping sixteen
samples and the same mist. That executable predates the shared-range fix
committed in `627b90f`; this is not a timing of the reviewed tip.

The developed error is 1.63413870 percent past one JND, worst 8.662356 JND:
within the two-percent still budget.

Nine samples and an unconditional hardware-hit fallback were both rejected for
exceeding that budget. The [primary traversal
notes](native-primary-traversal.md) record implementation, tests and failed
trades. No temporal claim is made.

At 64.057 seconds, 720 frames project 12.81 hours before movie encoding; the
eight-hour target remains open.
