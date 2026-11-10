# Native primary mesh traversal

This is a dated development record. Its timings and check results belong to
the revisions named below. Use the [Game workflow](developing.md) for current
setup and commands, and [the committed 4K review](review-4k-2026-11-05.md)
for the later dense-forest measurement.

Experimental next step after native shadows: use Metal's closest triangle query
for mesh hits as well. The explicit Puzzlemouth/YAML mode will be `gpu-native`.
Keep the existing software and native-shadow modes available. No new legacy
flags. Both pinhole and rig exposures must honor the choice.

Apple's nested intersection result supplies instance IDs at each level and a
primitive ID. Map those IDs to the existing encoded instance and face buffers,
preserving materials, local texture positions and smooth normals. Native
placements must have the same sorted order as those buffers.

Source: Apple, Your guide to Metal ray tracing, WWDC 2023, the nested-instance
query example and the intersect-rays section:
https://developer.apple.com/videos/play/wwdc2023/10128/

Refine the returned triangle with Slayer's existing rebased intersection. This
keeps tiny foliage's local coordinates and smooth-surface lift on their current
arithmetic path. If that candidate cannot be refined, use the software closest
traversal for this query. Native misses remain misses in this approximate mode.

Analytic spheres and procedural conifers retain their existing intersections.
Clear meshes remain visible and cast no shadow.

Tests first: a lit reflective scene with mist, spatial CPU textures, smooth
terrain, reordered world/local placements and a refused unsupported scene. Use
the existing two-percent-of-pixels-past-one-JND still budget after tone mapping.
Assert real surface coverage and finite light; report worst error.

Do not impose identical primary-hit bits on the new backend. Preserve the
original software tests and the narrower shadow-only contract unchanged.

Then compare recursive conifers across camera movement and time the same forest
at 960p and 4K. Still-image success does not establish temporal quality or meet
the eight-hour animation goal. Resource reuse and mist batching remain separate
work; this step adds no energy or faintness cutoff.

## First measured implementation

`gpu.SampledAccelerated` now selects this mode; `gpu.SampledNative` retains its
shadow-only meaning. The world and local placements share their sorted IDs with
the encoded shading buffers. An extra integer per instance maps a mesh to its
first pooled face, or a group to its first local member.

Recover shared face ranges once per encoded mesh, not once per shoot. That
temporary map belongs to this encoding. It is not a package cache. Metadata
preparation is included in encoding time.

The same 5000-tree, 3840 by 2160 scene retains mist and 16 samples per pixel.
The first full-native picture took 64.057 seconds whole, versus 87.238 seconds
for native shadows and 511.769 seconds for software. GPU tracing took 52.703
seconds.

This first timing predates the shared-range preparation fix; no speed gain for
that small CPU change has yet been measured at 4K.

Across all 8,294,400 developed pixels, 135542 exceed one JND: 1.63413870
percent. The worst difference is 8.662356 JND. Alpha agrees. The picture passes
the agreed still budget; motion and the eight-hour target remain open. At this
measured frame cost, 720 frames project about 12.81 hours.

Functional comparisons cover the mixed scene, spatial textures, smooth terrain
and transformed/reordered instances. Six additional views cover recursive
redwood and sequoia foliage. They are spatial comparisons at moving camera
positions, not a completed temporal validation.

A separate emissive fixture checks every instance's material directly: the
global image budget alone could hide a wrong material on a small object.

The four earlier manual fault trials have no preserved replay patches; their
claimed outcomes are withdrawn as acceptance evidence. The committed
`TestNativeCountsExposeARefinementFaultMaskedBySoftware` now replays a real
Metal refinement fault and proves software fallback remains observable.

## Rejected quality trades

Nine samples per 4K pixel reduced whole time to 52.001 seconds, but 43.4661
percent of pixels exceeded one JND, worst 20.6769 JND. Keep sixteen samples.

Accepting the hardware candidate when refinement refused it eliminated the
software fallback from the fast kernel. At 960 by 540, GPU time fell from 3.790
to 2.506 seconds and whole time from 12.072 to 8.617 seconds. However, 3.35089
percent of pixels exceeded one JND, worst 6.85006 JND.

That variant was rejected before a 4K run. The production experiment retains
software refinement fallback. Source, scenes and logs remain as local artifacts.

A possible next experiment is to mark refused samples and retrace those in a
separate kernel. That may remove the software walk's state from the fast kernel
without accepting its failed quality trade. Measure the retry share, additional
build/readback cost, final image error and whole time before keeping it.

It is not implemented in this checkpoint.

## Checkpoint checks

Game build passed. Every functional package passed the full Game check; the
command remains red only at the earlier 03d6b0a history guard, which requires
independent review. No harness waiver or history rewrite was made.

The focused run covers all statements in SampledAccelerated and nativeFirsts. It
is not full-package coverage. Full mutation, static/style gates, the separate
prose pass and independent review remain open, as described in the [performance
notes](animation-performance.md).

This is a local experimental checkpoint, with no main/RC merge or push.
