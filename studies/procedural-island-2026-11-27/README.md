# Seeded spherical patch visual studies

The source, camera and material are authored scenery, not a survey or a
geological model. These are bounded regional views, not a complete planet.
See `doc/procedural-island-study.md` for source attribution and limits.
The frame names below identify views in the November 27 render archive.
Each output retains `island.png`, `light.pfm`, exact `scene.yaml` and
`render.json`; the latter records selected patches, counts and whether the
projected cell target was limited.

The close and high views use build `f9aa01f9d`, the same seed 9, source,
lighting and brown material, 640×480 at one ray, and 256 selected-patch
budget; only camera height differs. Independent inspection found rolling
surface relief in the close view and an elongated rough swath in the high
view, but could not confidently match an individual ridge between views.
Both show conspicuous black angular gaps or streaks and smooth brown ground;
neither is a polished mountain range. Both receipts say the cell target was
limited (16,121 close and 16,617 high sampled vertices). Frame directories:
`procedural-island-close/` and `procedural-island-high/`.

Matched close-view controls used the same build and source. Turning source
normal stencils off (`procedural-island-flat-normal/`) removed most large
black gaps but exposed obvious broad facets. Raising patch/evaluation/point
budgets fourfold (`procedural-island-fine-budget/`) sharpened the ground and
reduced many gaps to thinner dark streaks, but black voids remained; 64,849
vertices and the cell target remained limited. Widening the source-normal
stencil alone from 0.001° to 0.01°
(`procedural-island-smoother-normal/`) reduced black patches while losing
local definition, with black breaks still present. These are visual
associations and do not establish the rendering mechanism.

The amplitude pair uses the same 4× budgets, 0.01° normal stencil and
conservative 6,500 m bound; only the invented source height changes from
2,400 to 6,000 m (`procedural-island-amplitude-{low,high}/`). Independent
inspection found stronger hills and a right-hand ridge at the higher
setting, accompanied by much larger black angular regions and facets.
Neither was presentation quality. This comparison does not isolate why the
dark areas occur.

The same high-amplitude study was rendered with build `14c22138e` using
the fixed 0.01° stencil and the opt-in mesh-footprint stencil, otherwise
with the same source, camera, light and budgets. The matched outputs are
`procedural-island-footprint-{fixed,mesh}/`. Independent inspection found
that the mesh-footprint image substantially reduced black gaps on the far
horizon and smoothed the ridges, while large angular dark areas remained
on the right and foreground and sharp facets remained on the right. Both
receipts report 64,257 sampled vertices and `cell-target-limited: true`.
This is a partial visual improvement at one view, not a proven root cause,
angle bound, shadow correction or finished terrain result. The stencil
changes with selected mesh resolution, so moving-camera continuity also
remains unverified.

The high-amplitude flat-normal control (`procedural-island-high-flat/`)
retained the large angular right and foreground dark areas while revealing
strong triangle faceting. A separate terrain-only pixel probe found mesh
hits at (545,220) and (320,450), with geometric-normal cosines 0.1605 and
0.2874 toward the directional light. Rays offset one metre along the
geometric normal encountered terrain 1.08 km and 1.87 km away respectively.
It did not use the renderer's exact shadow origin and did not consider
media or other objects; these two samples support terrain shadowing without
classifying every dark mark.

Changing only the made-up sky RGB from [0.02, 0.025, 0.04] to
[0.2, 0.25, 0.4] (`procedural-island-sky-fill/`) lifted the large dark
areas to blue-grey, while thin black scratches and facets remained. The
existing sky also lights surfaces; this is an artistic fill, not a physical
atmosphere model.

Increasing the patch lattice side from 9 to 17 and the point allowance to
524,288 (`procedural-island-dense-lattice/`) retained the same sky fill,
height, seed, camera and 1,024-patch cap. Independent inspection found more
distinct right-hand ridges and distant peaks, but dark scratches, angular
shadow boundaries and some facets remain. Selection, geometry and the
mesh-footprint normals may all change together; the image does not isolate
the lattice's effect. The receipt reports 259,697 source vertices vs 64,257
for the sky-fill view, and both were still cell-target-limited. Elapsed
times were 1.87 s vs 0.61 s, not controlled performance measurements; peak
memory was not recorded.

Three pixels in the dense-lattice PNG were selected by inspecting its bytes:
(391,90), (392,140), and (531,210) each have RGB (0,0,0). The bounded
`island probe-pixel` diagnostic in build `40783d73b` found terrain as the
nearest terrain-or-ocean hit at each pixel. At the first two, geometric and
shading light cosines were respectively (0.57158, 0.41064) and (0.68637,
0.35021); terrain-only light rays had no blocker with either a one-metre
geometric-normal offset or a renderer-style tangent-plane lift plus bias.
The third had cosines (-0.19627, -0.20654) and was blocked by terrain under
both origins. The probe runs CPU double-precision mesh queries, excludes other
scene objects and media from its light ray, and approximates rather than
replays native GPU precision or sampling. These observations do not yet
explain the black GPU pixels or classify every scratch. Earlier guessed
pixels (400,121) and (320,31) were later found not to be black and are not
scratch evidence.

For (391,90), the retained native GPU `light.pfm` was also read directly:
all three float channels are zero. The same scene and one-ray pixel were
replayed with Slayer's CPU tracer (`island probe-pixel`, diagnostic build
`8d63d05e7`), yielding raw RGB (0.164460063, 0.141667783, 0.1363464),
coverage 1 and depth 68,652.83 m. Replay reports both direct lamp and sky
contributions. This shows a difference at one pixel, not full-frame parity
or a proven cause. The independent terrain-only ray query reported a
different interpolated shading normal than the CPU replay, so even their
face choices may differ at that position. CPU double precision, GPU
precision, traversal and sample details remain possible differences.

GPU software traversal build `22a6fa19a` drew the same dense scene to
`procedural-island-software-traversal/`. Its developed PNG has the same
SHA-256 as native job 406
(`b2c6dfd025f3893ed2cc5447b10ca944dd151abfedb29b45f6e08dcd55c628c3`);
the full raw hashes differ, but pixel (391,90) is zero in both. This rules
out a difference unique to native traversal for this image, not differences
elsewhere in raw light or shared GPU shading and encoding.

A native diagnostic build `913a5251e` reported cover 1, RGB zero and depths
68,652.8594, 47,192.4414 and 34,778.1914 m for the three black pixels in
the order above. At (391,90), CPU replay depth was 68,652.8281 m. Thus both
backends report a primary hit at comparable depth at that one pixel, while
their reported light differs. This does not establish identical face,
normal, material or visibility.

An isolated candidate that brings the GPU's smooth-mesh normal facing
toward the existing CPU rule, and its one-view results and limits, are
recorded in [gpu-normal-candidate.md](gpu-normal-candidate.md). No shared
GPU renderer change is included here.

An isolated native diagnostic build `7ee1463ff` deliberately replaced only
sample (391,90)'s raw RGB with (0.5 + 0.5 × normal/light cosine,
directional visibility, material red). It reported (0.294683516, 1,
0.319999993), so its shading cosine was approximately -0.410633 and its
directional query was unblocked. CPU replay at nearly the same depth reported
cosine +0.493132 and material red 0.32. The GPU cosine is nearly the negative
of the uncorrected terrain-only CPU normal's +0.410638. The diagnostic
deliberately changes that pixel and is not an ordinary render.

The CPU `trace.meshNormal` path has a front-facing smooth-mesh correction
that GPU `shading_normal` did not apply in this build. This evidence supports
investigating that specific parity difference for this sample; it does not
prove identical face selection or explain every scratch. The separate CPU
rule for suppressing a strongly opposed normal's tangent lift also needs
consideration. No shared GPU pipeline change is implied by this observation.

The matching 8 km camera-altitude view (`procedural-island-horizon/`) shows
a clearer layered skyline and foreground ridges than the 16 km view, but
independent inspection still found black scratches, angular shadow areas
and soft/faceted uniform ground. It had 260,449 source vertices and remained
cell-target-limited. This is a stronger regional diagnostic composition,
not a polished landscape or a whole-planet result.
