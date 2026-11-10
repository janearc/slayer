# Native mesh shadows

This is a dated development record. Its timings and check results belong to
the revisions named below. Use the [Game workflow](developing.md) for current
setup and commands, and [the committed 4K review](review-4k-2026-11-05.md)
for the later dense-forest measurement.

The 4K reference spends 98.2 percent of its time tracing. At 960x540, removing
mist reduces GPU time from 44.951 to 33.261 seconds. Most work remains geometry
visibility, including sixteen sky-shadow queries at each surface hit.

First native experiment: accelerate only opaque mesh shadow queries. Keep
software primary intersections, normals, textures, analytic spheres and
procedural conifers. Instance masks preserve Clear's existing shadow rule.

Do not expand shared foliage: primitive mesh acceleration structures feed group
instance structures, then a world instance structure.

Metal owns its acceleration resources with a kernel. A caller may draw multiple
grids before closing that kernel. No package-state cache. This first experiment
may still rebuild per picture; measure build time too.

Metal's native triangle intersection differs from the software predicate. Do not
call this byte-exact caching. Keep the software mode as reference and default.
Preserve existing CPU/GPU tolerances and evaluate the agreed share beyond one
JND, worst pixel, and camera motion.

Hardware must remain an explicit Pzm/YAML selection until validated; no legacy
hillside flags.

Sources: Apple, Your guide to Metal ray tracing, WWDC2023/10128, sections on
nested instances, masks, intersector<instancing,max_levels<3>> and useResource;
and the installed SDK's MTLAccelerationStructure.h for descriptor layout.
https://developer.apple.com/videos/play/wwdc2023/10128/

The pilot-ray/packet proposal remains complementary: share conservative
empty-space bounds first, then test cached transmission/lighting under its error
and energy-preservation contract. Native shadows do not implement it.

Generic Metal API for the prototype:

    NewAcceleration(vertices []byte, stride int) (*Acceleration,error)
    (*Acceleration).Triangles(first,count int) (int,error)
    Placed(child int, matrix [12]float32, mask uint32) (Placement,error)
    (*Acceleration).Instances(...Placement) (int,error)
    (*Kernel).Accelerate(plan *Acceleration, root int) error

Vertices are nonindexed float3 positions, with explicit stride. Matrices are
column-major4x3. A plan references only earlier nodes, ruling out cycles. Grid
binds the root after its origin buffer:input count+2. Kernel close releases all
structures. A second build is refused without changing the first.

Empty instance roots are valid and must miss every query.

Tests: actual nested transformed triangles, clear/opaque masks, finite ray
segments, misses, repeated grids, refusal of a second build, closed kernel,
invalid vertex layout/ranges, nonfinite/singular matrices, and missing/cyclic
references. No test fake stands in for the Metal intersection.

## Implemented experimental mode

Set `scene.renderer: gpu-native-shadows` through the existing Puzzlemouth YAML
door. `gpu` retains software traversal. No new legacy flags were added. Both
pinhole pictures and rig exposures select the requested backend.

`gpu.SampledNative` returns the frame and its measured costs. It builds a
kernel-owned native hierarchy and closes it after the picture. The generic Metal
API can reuse that hierarchy across grids, but the scene entry point does not
yet retain it across frames. No scene disk cache is implemented.

The mode requires the native Metal ray-tracing API on macOS 14 or later. It
reports unavailability and unsupported scene features rather than silently
changing backends.

The capability check currently follows shader compilation; unsupported hardware
may first report a compilation error. The measurements below are from the
Apple M3 Max.

The current acceptance is perceptual. The owner clarified on 2026-11-03 that
matching bits is not the priority for fast rendering.

Use the existing still budget of two percent of pixels beyond one JND; report
worst error as well. Motion stability remains a separate check before accepting
a sequence.

## Same forest at 4K

The 5000-tree scene keeps its mist, geometry, camera, exposure and 16 rays per
pixel. Software took 511.769 seconds whole; native shadows took 87.238 seconds.
GPU tracing fell from 502.615 to 78.938 seconds.

- Native hierarchy construction: 0.089 seconds wall, 0.037 seconds GPU.
- Construction: 3.040 seconds; encoding: 2.097 seconds.
- Compilation: 0.030 seconds; dispatch/readback: 79.507 seconds wall.
- CPU finishing: 0.447 seconds; development and PNG output: 2.023 seconds.

GPU durations sit inside wall durations. They are not additional time. The
whole-frame gain is about 5.87 times. At this still's cost, 720 frames would
take about 17.45 hours, before movie encoding. The eight-hour target still needs
about another 2.18-fold improvement.

The developed PNG comparison covers all 8,294,400 pixels. RGB bytes differ at
4,123,284 pixels, while 8919 pixels exceed one JND: 0.10753038 percent. The
worst difference is 8.001596 JND. Alpha matches throughout. This is inside the
agreed still budget, with localized visible differences.

These results do not establish raw-radiance equality, camera-motion stability,
or a completed thirty-second clip. The standalone PNG comparison and full timing
logs remain with the local render artifacts, outside the repository.

## Checks and next work

Real Metal tests cover nested transforms, masks, finite segments, empty
hierarchies, invalid inputs, repeated grids and resource lifetime. Software
comparisons cover a mixed scene and shared redwood/sequoia foliage. The new YAML
mode has both reader and command-selection tests. Game build passed.

The latest full Game check passed every functional package. Its overall status
remains failed by the earlier `03d6b0a` history violation described in [the
performance notes](animation-performance.md). Style gates, full mutation checks,
coverage and independent review remain open.

Next candidates are native closest-hit mesh queries, retained scene/kernel
resources, and the proposed packet/mist reuse. Validate each against image error
and motion, and time the whole command. Native shadows do not yet implement the
packet or transmission cache.

The experimental [native primary mode](native-primary-traversal.md) now extends
this work to closest mesh queries; the shadow-only mode remains.
