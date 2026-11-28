# Terrain changes need time

The island selector spends a finite patch allowance on each view and
rebuilds that selection independently for every frame. At the budget edge,
small camera moves replace whole patches and change ridge shapes and shading.

in an island plan, set `detail.transition` to interpolate between
selected terrain meshes.

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

a bounded transition can lag a fast camera move. its endpoints
retain their projected-detail status.

The common refinement has its own patch allowance and uses the terrain's
point limit. Payload bytes count lattice keys and four endpoint vectors per
unique point. Map metadata, endpoint meshes, generated meshes and source
caches are separate; this is not a whole-process memory bound.

A budget refusal returns an error rather than dropping ground or shortening
the transition. Cancellation is checked during preparation. Backwards clocks
are refused. Closing the scene releases retained transition ownership.

Normal interpolation needs its own visual checks. Subdivision followed by
normal normalization and the shadow-terminator lift can change shading even
at the same geometric endpoint. These controls do not establish a
flicker-free film, exact radiance continuity or clearance between camera
samples.
