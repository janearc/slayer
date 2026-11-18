# The Slayer renderer

Slayer combines procedural scene construction, a CPU reference tracer,
Metal rendering, a linear-light darkroom and film output. A backend is
one choice within that pipeline; lighting approximations, scene detail,
camera sampling and development are separate choices.

## From world to picture

1. A set, smoke or named scene builds a `scene/thing.World`.
2. At the requested time, motions place surfaces, volumes and lamps.
3. The camera and lens make rays through each pixel.
4. Traversal finds surfaces; transport accumulates illumination, emission
   and transmission through the foreground.
5. Sample averaging makes a linear RGB frame with coverage and depth.
6. Development applies a print or film stock, then output writes the image.

`trace` is the CPU tracer used by the front door and scene commands.
There is no second active `render/trace` implementation. `gpu` returns
the same frame representation. `hillside.Session` connects jobs, scene
clocks and retained drawing; `door` handles output, film resume and sound.
See [package boundaries](../PACKAGES.md).

## Backends

| Job renderer | Traversal | Use and limits |
|---|---|---|
| `cpu` | Go reference | broad scene support, selected-pixel replay, no GPU lighting shortcuts |
| `gpu` | Metal software hierarchy | supported scenes; float32 device arithmetic and baked fields |
| `gpu-native-shadows` | native mesh shadow queries | keeps software primary queries; explicit numerical alternative |
| `gpu-native` | native closest mesh and shadow queries | native closest-hit traversal for meshes |

Native modes need the supported Metal ray-tracing API on macOS 14 or
later. All GPU paths validate scene support and return named errors.
Native queries accelerate meshes; analytic and procedural shapes keep
their own intersections. A hardware closest-hit candidate is refined
with Slayer's triangle predicate; failed refinement can use software
fallback. A native miss remains a miss. This is not a byte-exact cache.

## Geometry and fields

World shapes live in local coordinates and are placed by motions. Mesh
pooling, shared instance trees and bounding hierarchies avoid copying or
testing every needle independently. Native staging preserves the instance
mapping used by material and texture lookup.

Metal supports analytic primitives including planes and discs alongside
meshes, plus dedicated supported volume encodings such as boxed fields,
flames, shells, rings, wakes and gas giants. These encodings have different
contracts; support for one procedural field does not imply support for
every Go callback.

CPU textures may require finishing records and host
evaluation. Read the returned unsupported-feature error when adding a type.

Boxed and baked fields interpolate prepared samples rather than repeatedly
calling Go density functions on the device. `gpu.NewRenderer(2..192)`
offers an explicit longest-side bound for CPU-baked volume grids. It
changes volume detail, not surface raster resolution. Use its `Sampled`
method through the `render/cells` sampler interface when that policy is
appropriate; the ordinary `gpu.Sampled` policy is unchanged.

## Cameras and time

`rays: n` means `n*n` samples per pixel. Pinhole rays and thin-lens rays
can be generated on Metal. Other lenses can require CPU-generated rays;
large requests use stripes to avoid retaining a whole fine ray grid.

A film's scene clock is independent of playback time. `scene-duration`
can compress an hour of weather into thirty seconds; the soundtrack
still plays thirty seconds. A shutter with several `glimpses` renders
several scene instants and averages linear light. GPU camera motion can
also vary the eye per sample within a glimpse. That blurs camera movement;
it does not integrate moving objects continuously between world snapshots.

The CPU `trace.RenderWithInitialMedia` and
`trace.SampledWithInitialMedia` APIs and the opt-in Metal APIs can start
camera rays inside an explicit ordered stack of clear media. A set camera's
`initial-media` names that stack; the same staged values go to either backend.

Each entry supplies the refractive index and rank used by clear boundary
materials, and Slayer does not infer containment from scene geometry. Existing
calls without a stack still start in air. A moving job camera is refused
because a fixed authored stack cannot follow it.

Metal also refuses positive in-scattering, composite passes with optical
state, and initial media with its gravity kernel. Neither initial stack adds
scattering or attenuates lamp-to-surface paths; volume/gravity combinations
remain unsupported.

The set-camera Metal route currently requires a pinhole: it refuses a rig or
thin-lens aperture because the aperture's distributed ray origins may not all
lie inside the declared stack. CPU tracing keeps its existing lens behavior.

## Retained film resources

`gpu.Frames` owns a compiled kernel, device inputs, output bands and native
hierarchy across sequential calls. It sends only changed input buffers.
Layout or triangle changes rebuild affected native state. Mode, kernel
identity or sample-layout changes reopen it. An error makes the drawer
unusable until replaced; `Close` releases its retained resources.

The camera-only path patches the header without re-encoding the world
when world identity, generation, time, dimensions and mode agree. The
generation is the caller's promise: reusing it after mutating the world
can produce stale output.

Ordinary time-advancing sessions may still
encode on each frame even when device comparison avoids most transfers.
Retained buffers save allocation and upload; they do not automatically
eliminate host encoding or tracing.

Hillside also reuses unchanged sections of fixed-view films under its
`scene.film.sections` policy. CPU can redraw rectangles; Metal currently
reuses only an entirely unchanged frame. Positive lighting drift is an
explicit approximation. Disable section keeping for fresh-frame profiling;
see [the catalogue](optimizations.md#reuse-sections-whose-scene-did-not-change).

Sample readback streams through bounded row bands into the final image,
preserving the finisher's sample order. Field preparation similarly writes
bands directly into reserved packed ranges. Neither removes the final
scene grids, acceleration structures or development images. See
[optimizations](optimizations.md) for ownership and measurement details.

## Separate surface and volume passes

The set adapter exposes `gpu.SurfacePlate` and `gpu.VolumeTransport`
through `hillside.OpenComposite`. A plate keeps surface lighting,
reflections and their volume shadows. Transport returns foreground
scattered light S and RGB transmission T, stopped at the first surface.
Composition is `S + T * plate` in linear light, before development.

Reducing transport resolution and reusing a plate are visible
approximations. A reused plate freezes surface
illumination too. Camera motion or zoom requires fresh plates.

Thin
geometry or beams missed by coarse transport cannot be reconstructed.
Scenery samples remain separate until compositing; multiplying independently
averaged transport and scenery is generally a different estimator.

The current adapter requires GPU sets, a pinhole and a closed shutter;
it refuses stocks, titles, HUD and soundtrack input. Combination happens
on the CPU and development on Metal. It is not fully device resident and
does not resume existing output directories.

## Curved rays and gravity

The integrated prototype bends visible rays through one placed Kerr field
using Celestial's observer seeds and Metal integration. It includes scene
surfaces and supported volumes; it is more than a background lens filter.
Mass normalization, held RK4 work, smaller dispatch bands and observer-seed
reuse address different costs.

Use the software Metal `gpu.Sampled` or `gpu.Renderer.Sampled` path.
Native traversal and `gpu.Frames` reject fields. Gravity also rejects
refraction, gas optics and cached incident-sky grids; axial beam lights
are unsupported. Mirror behavior remains limited. These restrictions are
part of the prototype, not silently substituted approximations.

Read [the kernel contract](../gpu/GRAVITY.md) before building a gravity
scene.

## What correctness means here

Default `quality: exact` is the renderer's own configured answer. It is
not complete physical transport. The model has finite integration steps,
limited reflections/refractions and single scattering. Native traversal,
baked grids and CPU/device arithmetic need their own comparison contracts.
Unchanged-input repeatability within a path does not establish equality
between backends or across every native process.

Use CPU reference checks and selected-pixel replay where supported. Keep raw light, development, source
revision, scene inputs and temporal comparisons with numerical claims.
[Light](light.md), [modes](modes.md) and [optimizations](optimizations.md)
explain which choices alter the model or estimator.
