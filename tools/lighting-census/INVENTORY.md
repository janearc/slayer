# Lighting census inventory

`inventory.json` is a descriptive snapshot of Slayer's 52 numbered smoke
scenes and the CPU plus six device lighting and traversal approaches. It does
not run a renderer, claim any scene is supported by a mode, or imply that a
scene has been benchmarked. The executable census must use `smoke.All()` at
runtime and preserve each actual result or refusal; this file is not its
source of executable test cases.

The snapshot records the Git revision and SHA-256 of the registry file at
inspection time. That file contains the registry and some implementations,
but its digest is not a complete scene digest.

A reproducible run must also record the exact source revision and dirty
state, binary/build identity, dependency and source asset identities, device
and OS, input configuration, camera and scene time, renderer, dimensions,
sampling, light settings, output policy, cache state, and contention.

A declared input is not proof that all runtime reads have been captured, and
a mutable file can change while read. Do not reuse measurements as evidence
for different inputs.

## Cases and status

There are 52 registered scenes, one CPU reference and six GPU approaches:
52 × 7 = 364 candidate scene/mode cases for one still and one repetition.

The planned runner draws each CPU case twice and, for each GPU case, a cold
frame and an identical-time retained repeat. If every candidate can run,
that is up to 728 primary visible frame draws for one time, before retries or
additional spectral passes. Three independent
repetitions would be 1,092 case repetitions, not 1,092 individual frames.

These are counts, not predicted elapsed time or supported combinations. A
reference may be reused only for an exact matching scene, settings and source
identity.

Additional times, retries and variants require more work. Some scenes are
heavy or require external data; individual valid cases can take much longer
than others. Obtain actual short pilot times before proposing a full schedule.
A dry run or a refusal is not a performance measurement.

Every entry starts `unknown`. A `supported` outcome requires a successful
validated run of the exact case. A renderer or preflight refusal is `refused`
with its reason. Missing input, unavailable device, a timeout or an untested
combination must not become a zero duration or a successful result. Keep the
machine-readable outcome, the reason and enough evidence to reproduce it.

Scene 6 is marked `NoSmall`; 29, 35, 36 and 37 are marked heavy. Scenes 38
and 50 declare external inputs. `RequireInputs` checks those declarations,
but neither it nor the registry alone proves all files have been captured.

The registry's `Samples` is the number of rays along **each** pixel axis,
so a side of three means nine camera rays per pixel. Keep this setting
explicit and fixed for each paired comparison. Some scenes declare nonvisible
bands, palette, tone or gauge. A visible-only raw `Sampled` call can study
lighting, but it does not reproduce those scenes' full developed outputs.
Record omitted components, and use matched settings when assessing fidelity.

## Approaches and interactions

The CPU configured tracer is a reference for its stated model, not a claim of
complete physics. GPU software and native traversal use different numerical
paths; native closest-hit and shadow queries require their own comparisons.

Cached smoke, lamp-store and shared-sky lighting are approximations. State
which approximation ran and compare linear radiance against a matched direct
reference, as well as worst differences, developed images and motion.
Existing fixture limits apply to their fixture and are not universal budgets.

`gpu.Frames` is a separate retention axis: compare each retained result
bit-for-bit with its same-mode per-call result and exercise invalidation for
camera, geometry, lamps and media. A cached output file is not a new trace.

The GPU approaches are alternate kernel modes, not independent toggles to
multiply blindly. In particular, cached smoke and lamp-store do not stack;
shared sky requires lamp-store. The job interface has restrictions that do not
necessarily describe a direct API test. Native traversal refuses gravity
fields in the sampled path. The pinned source has a lamp-count limit on the
store; check the source revision and actual refusal because this is changing.

Preserve fidelity settings and geometry rather than making an unsupported
scene seem supported by removing lights or features.

## Coverage of the first mode matrix

The first matrix compares the actual public CPU, GPU software, native-shadow,
native-mesh, cached-smoke, lamp-store and lamp-store-plus-shared-sky paths.

The latter two include their production native traversal; these modes are not
separate measurements of the lighting optimization in isolation. The runner
plans a cold draw and an identical-time retained repeat for each GPU case;
CPU repeats the same time without a retained GPU drawer. It permits explicitly
requested scene times. A retained repeat is not a
fresh-process cold test or evidence that resources remain valid after motion.

The first matrix is **not an exhaustive optimization-combination study**. It
does not on its own measure:

- Per-optimization ablations, including lighting stores with alternative
  traversal implementations or each independent automatic cache disabled.
- Scene-specific RC controls: `batch-placement: fused|nested` for eligible
  individual city recipes, `vacant-lot-index: on|off` for green/night
  clustered cities, and `share-anatomy-prototypes: true|false` for authored
  Heard individual or multitude stages. The 52-smoke harness does not vary
  these or measure their interactions. Unsupported contexts refuse them.
- Proposed controls that are not generally implemented yet, including
  independently switching in-memory tree reuse, retained device state,
  projected-size geometry and texture filtering. Do not list those proposals
  as runnable test modes. See [render optimization controls](../../doc/render-optimization-controls.md).
- Separate fresh-process, disk-cache warm/cold and input-cache states, or
  cross-run cache reuse and its identity or invalidation rules.
- Camera motion, moving geometry, lamps or media, and resource invalidation
  across those changes. Additional times are not a substitute for controlled
  motion tests and a frame-by-frame reference.
- Other sample counts, light-grid resolutions, quality or scene-detail
  settings, resolution, depth-of-field, shutter or output encoding costs.
- The whole developed result for nonvisible bands, palette, tone and gauge;
  GPU `Sampled` paths in this study cover visible light only.

Add these as explicit axes with their own pinned settings, matched references
and outcomes. Record which combinations the actual public interface refuses;
do not infer implementation support from the arithmetic above. Any new mode
or ablation needs its own fidelity classification and comparison, especially
when approximations interact.

Use Game and the shared GPU lane for real draws. Run profiles only in a
coordinated render-free window. Keep this inventory and diagnostic runs
separate from the shared render queue and existing outputs.
