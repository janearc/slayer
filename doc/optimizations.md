# Slayer optimizations

Slayer saves work at several layers: constructing the world, encoding it,
moving bytes, traversing geometry, evaluating fields, integrating light,
finishing samples and producing a film. Many savings are automatic and
have no flag. Others deliberately change the answer and must be selected.

This catalogue describes the integrated renderer, including the gravity
prototype and the set compositor. A speedup measured on one scene is not a
prediction for another scene, device or camera path.

## Contracts before speed

**Reuse** avoids recomputing an unchanged result. Compare the same path's
bytes and test invalidation. **Numerical acceleration** can change floating
point intersection decisions; compare raw light, developed error and motion.
**Approximation** reduces spatial, angular or temporal work; declare what
changed and evaluate the resulting picture. A smaller output or absent
mist is a workload reduction, not an exact optimization of the same picture.

Default rendering already has a finite-step model and finite bounce limits.
Turning optional shortcuts off establishes a reference for that model,
not a complete physical solution. [Modes](modes.md) keeps these choices
separate.

## So you want to produce content for YouTube

Most of slayer is built so that a frame can be checked: the exact path,
its controls and its proofs. Gonzo mode is the other half. It is for a
film that has to look right, arrive quickly and go on YouTube, and it is
honest that it is an approximation. It is named for Hunter S. Thompson's
journalism, where the reporter is part of the story. Here the tracer is
part of the picture too.

Gonzo mode works differently from the rest of the renderer. The exact
path answers each pixel by itself. Gonzo mode shares work between pixels
and frames, and it guesses where light is coming from before it checks.
That is why it is fast, and why it never runs unless the job asks for it.
It is not a flag of its own.

A job asks for it by choosing the fast path
in [modes](modes.md): `renderer: gpu`, `vibes: on` and a short, low-ray
draft first. With `vibes: off`, nothing below changes a pixel.

### What it uses today

- **Native Metal traversal.** Primary and shadow queries go through the
  GPU's own ray-tracing hardware and acceleration structures. See
  [native Metal mesh traversal](#native-metal-mesh-traversal).
- **`lamp-store` and `sky-looks`.** Lamp light is baked into a per-frame
  grid for mist and smoke, and sky directions are shared across a pixel's
  rays. See [store illumination under explicit
  approximations](#store-illumination-under-explicit-approximations). The
  lamp store holds at most sixteen lamps.
- **Section reuse.** A film reuses the sections of the scene that did not
  change between frames. See [reuse sections whose scene did not
  change](#reuse-sections-whose-scene-did-not-change).

### Before you publish

Gonzo frames are still slayer's frames. Caption them as rendered, not as
measured. Review the encoded film with its sound. Slayer does not upload
anything; you publish it yourself.

## Construct geometry once and share it

### Mesh pooling and nested instances

Repeated shoots and needles share mesh vertices and local hierarchies;
instances carry placement rather than another complete copy of the mesh.
CPU and Metal traversal use bounds to avoid visiting every face. Shared
face ranges are recovered once per encoded mesh, not once per shoot.
This reduces construction, memory, encoding and traversal work in dense
forests without authoring fewer trees.

`scene/shape`, the manufacture packages, `gpu/scene.go` and the native
planner own different parts of this machinery. Material IDs, local texture
coordinates and smooth normals must survive instance reordering. Tree
identity and hierarchy content belong in cache keys; the
[tree-cache guide](caches.md) covers the disk stores.

### Grouped crowns and analytic representations

Mesh groups keep reusable local assemblies under one placed shape. Some
vegetation detail choices instead use aggregate crowns or procedural
conifers. Grouping existing geometry and substituting an aggregate are
different operations: the latter changes the represented scene. Keep
population placement/seed fixed when comparing representations.

Analytic planes and discs avoid tessellating a mathematically simple shape.
Dedicated shell, ring, wake and giant encodings avoid replacing every
field by a generic callback or dense arbitrary mesh. These are supported
type-specific paths, not a promise that arbitrary shapes run on Metal.
The source validators say which shapes qualify.

### View-dependent terrain and transitions

Terrain patch selection spends a declared work allowance where the view
needs detail. It can reduce source samples and generated geometry, at the
cost of a chosen representation. Camera motion can change the selected
patches and reveal popping.

The transition owner builds a common refinement
of two endpoint meshes and interpolates them over a declared scene-time
interval, with explicit patch and payload budgets. It reduces abrupt
representation changes but can lag a fast view.

[Terrain patches](terrain-patches.md) and
[transitions](terrain-transitions.md) document these controls. The
[projected-detail helper](projected-detail.md) provides conservative
pinhole bounds; its presence does not mean whole-world automatic detail
selection is enabled. Elevation sample spacing alone is not a proven
geometric-error bound.

## Encode with fewer allocations and copies

### Planned packing and released intermediates

The geometry encoder resolves shared ABI field names once, then packs
fixed records into reserved final buffers. It checks geometry and byte
limits before allocation. Duplicate float64 triangle records, construction
nodes and lookup maps are released when their packed/native equivalents
are ready. Native staging reserves its placement arrays once.

The synchronous cgo bridge passes pointer-free packed bytes to Metal's
copying constructor, avoiding a second full C staging copy. Metal still
owns a copy; this is not zero-copy access to Go memory. Releasing Go
references does not promise immediate collection or a lower process RSS.
[Preparation memory](preparation-memory.md) gives the ownership contract.

### Streamed field preparation

Flame density/emission, wispy density and optional smoke-light bakes write
readback bands directly into their reserved packed ranges. This removes
intermediate whole-grid arrays and repacking. It leaves the final fields,
their resolution and shader arithmetic in place. A failed partial bake
does not become a completed scene. See
[streamed field preparation](streamed-field-preparation.md).

## Keep film resources alive

`gpu.Frames` retains compiled kernels, device inputs, two output bands,
mesh trees and native acceleration state. Input comparison sends only
changed bytes. A camera-only call can patch the header without rebuilding
the scene encoding when the caller's world generation and time agree.
Triangle/layout changes trigger hierarchy rebuilds; mode or sample-layout
changes can reopen device state.

This saves compilation, allocation, transfers and sometimes encoding.
It does not skip tracing the next frame. A time-changing session may
still rebuild host encoding even when every geometry buffer compares
equal. Treat generation as an explicit invalidation promise. Close the
drawer to release resources; an error makes it refuse later draws.
The contract and counters live in `gpu/frames.go` and `gpu/profile.go`.

## Reuse sections whose scene did not change

Hillside films can split a fixed view into sections and reuse those
unaffected by scene changes. CPU drawing redraws only the remaining
rectangles. Metal currently reuses the whole frame only if every section
is reusable; otherwise it draws the whole frame. This is separate from
retaining GPU buffers and from resuming a finished frame on disk.

In an ecosystem set, `scene.film.sections` exposes `over`, `size`, `keep`
and `drift`. Defaults are width over 320, 64-pixel sections, `keep: true`
and zero drift. Camera/lens/size/sample changes invalidate reuse. Rig/opened
lenses, cached/fast lighting and the composite adapter bypass this path.

Zero drift asks for unchanged sections under the change/bounds contract.
Positive drift permits an explicitly bounded change in a steady volume's
illumination from changed lamps; it is an approximation, not identical
light. Unknown or unbounded contributions force redrawing.

Set
`sections: {keep: false}` for an unconditional fresh-frame control or
profiling, and compare actual draw counters before calling a repeat fast.
See `section/kept.go`, `section/drift.go` and `hillside/sections.go`.

## Make visibility queries cheaper

### Software bounds and opaque early exit

Hierarchy bounds reject regions before triangle tests. Shadow queries can
return on an opaque blocker without finding the closest surface. Clear
surfaces follow their declared shadow policy rather than being treated
as opaque merely for speed. CPU/group ray windows preserve placement scale
and near/far bounds.

### Native Metal mesh traversal

`gpu-native-shadows` replaces mesh occlusion queries. `gpu-native` adds
native closest-mesh queries and maps the result back to Slayer's face,
material and normal data. Candidate refinement and software fallback
retain the current numerical contract; native misses do not trigger a
blanket software retrace. Analytic primitives keep their own predicates.

## Evaluate expensive matter less often

### Density and emission bakes

Prepared volume grids replace repeated procedural noise/density work
with interpolation. GPU shaders bake supported flame and wispy fields;
boxed CPU media are sampled before device traversal. Bake cost is paid
before tracing and must be included in totals. Moving fields need refresh.
Grid interpolation approximates the underlying continuous field.

`gpu.NewRenderer(volumeCells)` explicitly bounds CPU-baked volume grids
from 2 to 192 cells on their longest side. This can suit a small preview
whose surface rays remain full resolution. It is not an automatic
quality-preserving setting for every flame.

### CPU flame sampling and noise arithmetic

The CPU flame sampler reuses a prepared representation rather than
reevaluating a full procedural flame at every terminal sample. Its
resolution/quality contract is separate from the GPU bake.

The noise implementation shares work in gradient/hash evaluation instead
of repeating identical gradient calculations. That arithmetic optimization
does not expose a quality flag. A microbenchmark of noise
alone does not establish end-to-end film speed.

### Skip work with an exactly zero contribution

Purely emissive media with no scattering do not need lamp-shadow queries
to find their emission. The tracer bypasses that unused illumination work
while retaining extinction and emission integration. Painted media that
do scatter still ask lamps. This is different from dropping light
because it seems faint.

### Spend more steps near narrow beams

`render/beamstep` prepares beam envelopes once for a lamp snapshot and
refines only the march cells that need narrower spacing. Other strides
keep their base work. Both CPU and Metal use the declared refinement
budgets. The bound allocates sampling effort; it does not discard Gaussian
tails. Merged intervals can refine empty gaps too. This is targeted
integration effort rather than a pilot ray's guess about its neighbours.

## Store illumination under explicit approximations

| Mechanism | Avoided work | Refresh and tradeoff |
|---|---|---|
| cached smoke source grid | repeated source-light evaluation inside supported smoke | recomputed for current scene; spatial interpolation error |
| incident-sky grid | sky visibility/extinction queries at every weather sample | native per-frame world grid; lamps stay direct; interpolation and boundary blend |
| `lamp-store` | lamp shadow/volume march at every boxed-medium sample | per-lamp grid baked for the frame; cell-scale mist shadow softening |
| beam axis table in lamp store | repeated narrow-beam source evaluation | transverse beam profile stays analytic; bounded axial sampling |
| `sky-looks` | sixteen surface-sky looks for every pixel ray | shares sixteen directions across pixel rays; noisier per-ray estimate |

These store different quantities and have different domains. The optional
incident-sky grid is bounded to 262144 RGB nodes, with direct evaluation
outside and a boundary blend. Its CPU reference remains direct. It does
not cache explicit lamps. [Volume environment lighting](volume-environment-lighting.md)
records its controlled comparisons.

`lamp-store` is GPU-only, supports up to sixteen lamps and applies to boxed
mist/smoke rather than all procedural/gas media. Surfaces continue direct
lighting: sampling their neighbouring dark grid cells would create patches.

It conflicts with cached smoke lighting. `sky-looks` requires the lamp-store
kernel. At one ray per pixel it saves nothing; with four it assigns four
of the sixteen directions to each ray. Both are selected by `fastlight`
or `vibes`, with per-shortcut overrides. There is no lamp store on disk.

The [fast-light study](fast-light.md) gives its error measurements and
workload context. No universal whole-film speedup follows from them.

## Stream output instead of retaining every raw sample

`metal.Kernel.GridBands` lets the finisher consume ordered bands and reuse
their storage. The final pixel accumulators preserve the original sample
addition order, coverage and nearest depth. One Go band and Metal output
bands replace a complete raw-sample array. Encoded inputs, final images,
acceleration structures and development remain separate allocations.

A 3840 by 2160 image with sixteen samples and twelve float32 values per
sample logically contains 6,370,099,200 sample bytes. Streaming avoids
retaining all of those bytes at once; it does not eliminate their tracing
or readback. `SampleBytes` is logical work, `PeakSampleBytes` is the Go
readback peak, and neither is process RSS.

See
[streamed readback](streamed-readback.md).

Physical lenses with CPU-generated rays use stripes when large. Thin-lens
and ordinary pinhole rays generated on the GPU avoid that input grid.
Shutter camera interpolation also avoids a separate full draw for every
camera-time sample, but object motion still needs scene glimpses.

## Gravity-specific savings

Mass-normalized coordinates keep Kerr integration in a useful numerical
range. The held-step RK4 implementation avoids repeated invariant work;
it does not replace curved rays by straight ones. Smaller gravity dispatch
bands reduce the work in each command, addressing pacing rather than
reducing the total rays. Sixteen-row gravity bands are separate from
ordinary trace readback policy.

`gpu.Renderer` caches observer seeds under their camera/lens, image and
field identity. `LastTiming().SeedsCached` reports reuse. Moving the world
clock alone need not change observer seeds, but a changed eye or field
must invalidate them. It does not cache the traced picture or retain a
native hierarchy. See `gpu/field_cache.go` and
[the gravity contract](../gpu/GRAVITY.md).

## Disk caches and finished-work reuse

Sky, ground, pin checks, kept pictures and tree caches avoid reconstruction
or rendering under content identities. Corrupt entries miss; bounded
stores evict old entries. GPU tree stores also have alternative ownership
and capacity behavior. [Caches](caches.md) documents locations, controls,
limits and invalidation gaps, including the gated kept-still cache.

Film resume reuses completed frames only under the recorded job/input/build
identity. Changing sound alone can reuse pictures, except when music cues
drive the scene. Retaining raw PFM frames permits redevelopment without
tracing. A frames-only job can assemble an existing sequence with sound
and titles. Preserve unclipped raw light if later exposure changes matter.

## Measure the complete job

Record revision, device, CPU allowance, input/seed, dimensions, rays per
axis, backend, approximation flags and cold/warm state. Separate host
construction/encoding, device bake, trace dispatch/readback, finishing,
development and encoding. A GPU timestamp is not another wall interval
to add to wall time; overlapping dispatch timestamps can overlap too.

Check actual work counters so a cached frame cannot look like a fast trace.
Compare matching raw light and developed images, and inspect camera motion.
Report preparation memory and whole-process peak separately from payload
counts. `GOMEMLIMIT` is a soft Go heap target, not a cap on Metal resources.
Choose a focused GPU sample while iterating; reserve full suites and long
animation studies for their deliberate gates.
