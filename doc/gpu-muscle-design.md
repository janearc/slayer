# Making the GPU the default place things render

Every new thing Slayer has learned to draw has stopped the work to go
build its own Metal path: fuelbed got a stencil, weather got a solver,
fire got a bake kernel, shell weather got a density branch. The pattern
works, but it prices every new phenomenon at "write a renderer first."

This design turns that around: the kernel grows a small set of general
primitives, and a new phenomenon becomes data fed to primitives it
already has.

The driving examples are the imported ocean and ground packages
(oceantex, titan; see phone-ocean-integration.md) and two scenes we
cannot draw at any price today: a swimming pool, and a tablet fizzing
in a glass of water. All three are the same missing machinery at three
scales.

## What exists, and what the examples need

The gpu package packs a world into flat buffers, compiles trace.metal
at open, and checks every pixel against trace, the CPU reference, at a
stated tolerance. Volumes are a tagged union (puff, wispy, boxed,
shell); adding a kind is about six touch points and no new kernel.
That structure is sound and this design keeps it.

What is missing, in dependency order:

- Refraction. The tracer reflects (Schlick on opaque glossy surfaces,
  the unpolarised Fresnel share on a surface that lets light through,
  one mirrored ray) and absorbs through media (Beer-Lambert). Stage 1
  below bends rays through a dielectric; without transmitted rays there
  is no pool bottom, no glass, no shallow sea over sand.
- A water surface. Oceantex synthesises one from a variance spectrum
  by FFT and shades it by slope statistics, on the CPU, in its own
  camera convention. The spectrum is the only ocean-specific part;
  the synthesis, the sampling and the glint are general.
- Scattering crowds. Fizz is thousands of small spheres plus a haze of
  microbubbles. Spheres exist; the haze is one more volume kind; the
  crowd was, until this branch, over the volume budget by design.

## Stage 0: plumbing and caching (landed on this branch)

The kernel plumbing predated any tightening. The branch lands five
changes, each behind the existing tolerance tests:

- Grid rides the streamed band path: one copy per input, where the
  all-at-once path copied twice. The dead C entry points are gone.
- A process-wide shader cache keyed by device and source text. Open
  and OpenSession compile each distinct source once; tests and
  per-scene source variants stop paying tens of milliseconds a time.
- The grid draws a band ahead of the one being read, in a second band
  buffer, so the gpu draws while the cpu copies and consumes. Bands
  asked out of order still answer correctly, just without the overlap.
- GridStream keeps inputs on the device between pictures. Update
  replaces one input's bytes; everything else stays put. This is the
  frame loop oceantex's cascades want: the scene uploads once, each
  frame uploads its moment.
- most_volumes is the scene's own count, emitted into the source at
  compile time; the fixed 32 (an agent's pick for one clearing of
  fires, 2026-10-18, d30d9d0) is retired. A ceiling of 256 remains
  against pathological packs, priced at about 40 bytes a volume of
  thread memory.

Measured on the gpu suite on one M-series machine: 68 s before the
branch, 54 s after the cache and the band overlap. Not a benchmark,
but the direction is right and the tests are the witness. The suite is
`game test -- ./gpu/...`; the figures are as recorded on 2026-11-10 and
were not rerun.

Deferred from stage 0, recorded here so they are not lost: zero-copy
readback (page-aligned output wrapped as a no-copy buffer; changes
allocation ownership, wants its own change), batched acceleration
builds (one command buffer, not one per node), and cheaper hashing in
fire_grid (any change must land in CPU and GPU together or the parity
tests rightly fail).

## Stage 1: refraction

Transmitted rays by Snell's law, with the unpolarised Fresnel split.
The share comes from maths.Fresnel through Material.Splits for any
material that lets light through (Through above zero), by the side the
ray arrives from, and all of the light reflects past the critical
angle. Opaque glossy surfaces keep Schlick (Material.Mirrors), and
oceantex.Fresnel calls the same equations.

Clear water absorbs nothing on this path unless the scene holds a
volume: neither trace nor the kernel reads oceantex's band absorption
or its case 1 colour.

Two limits hold. A partial reflection on a look that has already
bent is paid and not followed, so a water ball against a surround of
luminance one reads 0.98 head on and 0.76 at 85 degrees of incidence
(trace's TestAWaterBallInASurround). And every look starts in the air,
so a camera placed inside water is drawn as if in air.

trace first, as the reference, then the Metal branch, then tolerance
tests on scenes with analytic answers: a flat pool of known depth, a
ball in a glass. Definition of done: a pool bottom renders on both
backends inside the stated tolerance, and a scene with no dielectric
renders byte-identically to today.

## Stage 1 landed with the branch

trace bends through clear surfaces (Material.Through; Snell with the
contract trace/surface.go states), and the kernel walks the same bends
as a loop, within the exact tolerance: TestTheGPUBendsIntoWaterAsSlayerDoes
and TestTheGPUReadsThroughAGlassBallAsSlayerDoes hold the pond and the
glass ball to a mean difference of 1e-5 and a largest of 1e-3
(`exact`, gpu/sampled_test.go).

The fizzing glass - stage 4 - also landed:
medium.Fizz bakes through the boxed path the kernel already had, and
TestTheGPUDrawsAFizzingGlassAsSlayerDoes holds it to the baked tolerance,
a mean of 3e-4 and a largest of 1e-3. No test measures a closer agreement.
These tests run only on a machine with a Metal device; without one they
skip.

## Stage 2: the water surface as data

A spectral height field: the CPU builds oceantex's cascade fields
(eight grids a layer) and packs them as inputs; the kernel intersects
the height field and shades by the Cox-Munk statistics with the
resolved-variance subtraction oceantex already does.

The spectrum is a parameter block: JONSWAP/TMA for a sea, a capillary
term in the
dispersion for ripples on a pool or a glass. The water optics
(Quan-Fry index, Pope-Fry absorption) become a kernel library beside
NoiseLibrary, tested against their Go forms.

Landed with the branch: metal.WaterLibrary (Quan-Fry, Fresnel,
Cox-Munk, Smith, every sourced number passed in so sources.yaml stays
the one ledger); shape.HeightField, which trace marches; and the
kernel's height-field kind with its heights buffer (2026-11-10), so
both backends march a field at a stated tolerance away from grazing.
Still open: the cascade packing and the resolved-variance shading.

Open questions are the ones phone-ocean-integration.md already lists:
the unproved slope bound, the march fallthrough, and the convention
mismatch (east/north/up against east/up/south). Those get settled in
trace, the reference, before any Metal is written.

## Stage 3: titan on the kernel

Titan's generated ground hashed the float64 bit patterns of corner
directions. Metal has no doubles, so a GPU port as written would have
made different ground, not approximately the same ground.

The fix was called on 2026-11-10 and it landed with the branch:
grissom's mesh grew CornerNames and EdgeName (celestial, branch
mesh-corner-names), whole-numbered names derived by integer arithmetic
alone, one per corner however many cells stand on it, and titan now
makes all its ground from names. This changed what ground a seed makes.

The whole-mesh test holds one name to one position both ways round;
the first combine summed the pair and collided among the twelve by
plain arithmetic, which that test caught.

Until the review proxy mints a build with the change, slayer replaces
celestial with ../celestial-pinned: the pinned review content plus
exactly that one commit (see its PINNED.md). The refinement kernel
itself - the GPU walking the same descent by the same names - rides
with the height-field work above.

## Stage 4: fizz

Bubbles in the body of the water are small spheres with stage 1's
refraction. The haze of microbubbles too small to resolve is one new
volume kind: a scattering medium whose density is a crowd statistic,
the same shape of thing the whitecap fraction is for the sea. The
per-scene volume sizing from stage 0 is what lets a glass hold a
crowd.

## Caching, as a policy rather than a list

The rule the branch establishes: nothing is paid twice while its
inputs are unchanged. Compiles are cached by source; device buffers
persist across frames and re-upload by exception; bands overlap so
waiting is not serialized with copying.

Of these, the compile cache and the overlapping bands are in the code.
Persistent device buffers are the design's intent and the code does not
yet carry it out: gpu.Sampled opens a kernel, uploads the scene's inputs
and closes the kernel on every call.

What remains uncached and known: the scene encode itself (gpu re-packs
the world every frame even when only time moved), and lighting bakes
across frames with an unchanged sun. Both are gpu-side and belong with
stage 2, where the frame loop becomes the common case.

## Review evidence

Stage 0 is on branch gpu/muscle-stage0, five commits, each green on
the metal, gpu, fuelbed, weather and hillside suites on a machine with
a Metal device. The command is `game test -- ./metal/... ./gpu/...
./fuelbed/... ./weather/... ./hillside/...`; the result is as
recorded on 2026-11-10 and was not rerun.

Nothing here enables oceantex or titan in the renderer yet; stages 1 and
2 are where pixels change, and they change behind trace-first references
and stated tolerances.
