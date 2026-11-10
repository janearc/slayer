# Let the lens ask for detail

The preparation helper in `scene/footprint` has compiled and passed
independent host geometry checks. It is not an enabled whole-world
renderer. [The verification record](verification-2026-11-09.md) separates
that evidence from the remaining renderer and image checks.

The constructor takes a camera pose, horizontal pinhole field, actual
raster dimensions, pixel width/height ratio and near distance. Nothing
chooses a default lens or decides that an altitude means low detail.
A narrow instrument field can request fine geometry at a great distance.

Assess takes a world box and a possible Euclidean displacement in metres. The
box must enclose the actual geometry, including peaks between sampled DEM
corners. The displacement must come from a separate bound, or be explicitly
described as a chosen sampling metric.

This function does not turn skipped elevation samples into a measured
geometric error.

## Perspective rather than distance alone

With positive camera depth d, horizontal coordinate x and focal scale f
in pixels, the projected coordinate is f*x/d. If each camera component
changes by at most e and d > e, its horizontal change is bounded by:

    f * e * (1 + abs(x)/d) / (d-e)

The vertical calculation uses its own focal scale. Their hypotenuse bounds
distance in raster coordinates: horizontal columns and vertical rows. For
nonsquare pixels this is not a distance on the physical image plane.

The off-axis term matters: a depth change moves an edge-of-frame point more
than a point on the optical axis. The camera-axis conversion uses a
conservative norm bound, so a slightly rounded pose cannot make the supplied
world displacement smaller.

The helper transforms the whole box with outward-rounded binary64 interval
operations. Frustum exclusion requires the complete expanded box to miss one
half-space. Touching faces and uncertain arithmetic stay visible.

Near-plane intersections request maximal finite detail and set Limited; they
do not divide by a tiny depth and disappear.

This is a bound relative to the supplied binary64 pose and pinhole
parameters. It is not a proof of the renderer's float32 transformations,
diffraction, distortion, grazing intersections or visual acceptance.
No far plane, planetary horizon or atmospheric visibility cutoff is added.

## Terrain and populations

The existing terrain selector accepts a callback returning visibility and a
score. A world adapter can return Assess.Visible and Assess.Pixels for each
source patch, then use a pixel target with its patch/work budgets.

That adapter still needs source projection, an explicit height datum and
conservative world bounds. It is not wired into Hillside in this slice.

The same view can assess a full canopy envelope before choosing vegetation
detail. Roots outside the image may still have visible branches. Population
identity must remain fixed when its representation changes; the lens does not
reseed the forest.

Aggregate canopy and temporal transitions are still separate work.

Review needs independent centre/edge projection calculations, a narrow field
compared with a wide field, non-square pixels, rotated poses, world-origin
translations, boxes at every frustum plane, the camera inside a box and
near-plane crossings. A self-roundtrip alone is not evidence.

Independent rational pinhole witnesses are recorded in
`scene/footprint/testdata/pinhole-contract.yaml`. The geometry tests now
read those authored expectations; the YAML itself remains a fixture,
not captured production output.
