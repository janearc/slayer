# Terrain changes need time

The island selector spends a finite patch allowance on each view. In the
first moving studies it rebuilt that selection independently for every frame.
At the budget edge, small camera moves replaced whole patches and changed
ridge shapes and shading. A matched fixed-geometry control isolated those
changes while keeping the camera, light and weather recipes the same.

Optional detail controls now give the terrain a transition owner:

```yaml
detail:
  # The ordinary sampling/detail fields are still required.
  transition:
    seconds: 60
    patches: 512
    payload-bytes: 33554432
```

Seconds are scene time. A hyperlapse compresses the visible transition along
with the rest of the scene. The values above are authored quality/work choices,
not measured terrain accuracy or a promised render time.

The owner keeps the current mesh and one view-selected target. Their source
quadtrees form a common refinement, so its vertices can represent either
endpoint's actual stitched triangles. World positions and shading vectors
interpolate over the declared interval. The next target is selected after
that interval ends, using the then-current camera.

This bounded delay can lag a fast move. It does not enforce the projected
cell-span target; limited endpoints keep that status. Geometry remains sourced
and view-dependent. Holding the initial mesh for the whole flight is a separate
diagnostic, not this transition policy.

The common refinement has its own patch allowance and uses the terrain's
point limit. Payload bytes count lattice keys and four endpoint vectors per
unique point. Map metadata, endpoint meshes, generated meshes and source
caches are separate; this is not a whole-process memory bound.

A budget refusal returns an error rather than dropping ground or shortening
the transition. Cancellation is checked during preparation. Backwards clocks
are refused. Closing the scene releases retained transition ownership.

Normal interpolation needs its own visual checks. Subdivision followed by
normal normalization and the shadow-terminator lift can change shading even
at the same geometric endpoint. These controls are an implementation study;
they do not establish a flicker-free film, exact radiance continuity or
clearance between camera samples.

The first matched native study contains 36 frames at 320 by 180 and 24 fps.
The original adaptive, fixed-mesh and transition versions retain the same
camera and scene recipes; their first PNG bytes match. In the inspected
terrain rectangle, maximum adjacent mean RGB-code change fell from 3.0622
to 0.2855 with transitions. Motion and delayed detail contribute to this metric.

A 71-frame move-and-hold control lets the terrain catch up after the camera
stops. Its final PNG bytes match a fresh stateless render at the same view
and scene time. This establishes convergence for that view, not an arbitrary
moving track or high-resolution film.

A separate CPU diagnostic intersected the endpoint meshes with independent
rays, checked a genuinely intermediate surface and observed no source-height
calls during the active transition. Clock reversal, cancellation, payload and
patch refusal, unchanged-view reuse and reset were exercised too.

Those rays recover endpoint geometry within the diagnostic's 0.000001 m
allowance. Normal directions and shadow lift are approximate under subdivision:
the authored probe observed a largest start-normal difference of 0.0431
degrees and a shadow-lift difference of 1.26 m. These are limitations to inspect,
not accepted general error bounds.
