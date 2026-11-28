# Adaptive generated ground: implementation sketch

Status: design, 2026-11-27. The fixed global mesh was an experiment, not a
finished feature. At 800 pixels, doubling its resolution from 128 × 64 to
256 × 128 made the 10 km land grid no longer apparent, with the field and
camera held fixed. Neither resolution established recognizable ground
relief in the attempted closer views. In the measured lit view, omitting
only the generated air made the previously obscured green ground visible;
the spherical and 10 km mesh surfaces still appeared almost identical.
See `studies/generated-relief-2026-11-27/` for the recipes and limitations.

First implementation slice: `world.ReliefPatch` addresses six cube-face
quadtrees and gives corners canonical integer cube-vertex keys, and
`Relief.Point` samples the fixed field at a key. This is an unverified
building block, not a mesh or an adaptive renderer. It makes coincident
corners name the same direction across faces and levels; it does not yet
stitch intermediate coarse/fine edges, bound error, select patches, or
connect any of this to a camera, shadow, water or a draw.

The next building block, `ReliefConstraints`, can place a finer edge's
midpoint on the straight segment between displaced coarse endpoints after
projection, and report how far it moved from the sampled field. It is not
an automatic mesh stitch: a builder must balance levels, add constraints in
coarse-to-fine order, use the constrained vertex on both sides and account
for the returned displacement. Selection, normal continuity, a complete
mesh and visual evidence remain open.

`bodies.ReliefCube` is a further bounded building block: it constructs an
actual closed, uniformly subdivided cube-sphere triangle mesh from the fixed
field, sharing keyed vertices across the six faces. Its explicit level and
face budget are caller choices. It is not yet used by generated bodies or
the renderer, does not use mixed levels or the edge constraints, and has
not been visually verified. Smooth normals only affect shading; the
triangles are the intersections and silhouette. A rendered comparison is
needed before judging its appearance or cost.

`bodies.ReliefPatches` is an unverified mixed-level mesh builder for
caller-selected leaves. It requires whole-planet coverage, rejects overlap,
gaps and neighbours more than one level apart, and uses the projected
midpoint constraints when a coarse edge meets two finer edges. Its return
includes the largest stitch movement, not a bound on the field's total
approximation error. It does not choose patches for a camera, rebalance a
selection, manage frame-to-frame transitions or stage itself for a draw.
Those and visual evidence remain open.

## A possible next shape

Use the fixed, body-space `world.Relief` field as the source of a closed
surface, independent of camera and shadow direction. Its authored sampling
angle must be fine enough for the intended nearest view and fixed for the
whole shot: changing the actual field with the camera would make geography
move. A mesh is one approximation of that field, not the field itself.
Retain low-resolution coverage of the *whole* planet, including parts the
camera cannot see, since they can block shadow rays. Refine view-relevant
regions by a declared screen-space error, and retain coarse parents if a
budget prevents refinement. Announce a limited result; do not call it the
requested quality or drop geometry.

One candidate is six cube-projected quadtrees. Their corners are keyed by
canonical integer cube coordinates before projection onto the sphere; this
allows neighbouring faces to name the same physical vertex. Neighbours
would differ by at most one subdivision level. At a coarse/fine boundary,
the fine edge must use vertices on the coarse edge's *projected geometry*,
and transitions must be included in the mesh and its bounds; independently
sampling extra fine heights at that edge can create cracks or a raised lip.
The maximum deviation introduced by stitching belongs in the geometric
error budget. This is an engineering proposal to implement and measure,
not a claim that the current code has these properties.

`terrain.SelectPatches` already contains a priority-driven selection
pattern which retains a coarse parent when children cannot fit. Its score
is supplied by a caller: it does not derive bounds for a new spherical
field. `terrain.PatchGrid.StitchTo` explicitly joins affine source-space
grids and says a curved world needs stitching after projection; it cannot
be reused unchanged for these edges. The screen-space estimator must include
curvature, displacement and a conservative bound for unsampled detail.
If such a bound is not known, call the score a heuristic and never promise
an error guarantee.

Today `stage.Add` constructs bodies without a camera and `Body.PlaceOn`
retains the resulting shape. A per-view mesh therefore needs an explicit
camera/footprint context at staging and a lifecycle for a moving camera;
simply rebuilding a different untracked patch can pop, while putting a
local patch over an intact sphere gives competing surfaces and wrong
shadows. A view-independent radial intersection is another route but would
need conservative traversal and corresponding GPU support. Compare cost,
memory and visible continuity before choosing; do not force either route
into the current renderer without measuring its contracts.

Use shared vertex positions for silhouette and geometric normals. Any
additional shading-only normal detail must be separately labelled and must
not move the silhouette, hit or shadow. A water surface and camera placement
must consult the same sea level and elevation. Rebase world-to-camera
coordinates before narrowing for GPU work, with all views and shadow rays in
the same frame. These are requirements to implement, not present behavior.

## Visible evidence

First locate a land point and its height/slope from the field, then choose
a camera and light with a measured ground intersection. Inspect a near view,
a grazing horizon and an orbital view of the same seed, with a low sun;
include sphere, geometric relief and shading-normal-only controls. Inspect
the full images, including shadows and patch boundaries, plus a continuous
camera approach across transitions. An atmosphere setting may make ground
legible but must be an explicit artistic choice, not an invisible removal
of air or a change presented as a relief fix. Source claims and art choices
remain those in `doc/generated-world-relief-design.md`.
