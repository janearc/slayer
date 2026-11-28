# Whole-figure cameras

`bioforms/framing` fits a perspective camera to posed geometry. The caller
supplies the local bounds of each part, its placement inside the subject,
and the subject's placement in the scene. Include every surface that
should fit in the picture, including hair, wings or other attachments.
`ThingsPose` samples the visible parts in existing `thing.Thing` values.
Pass parts whose placements are relative to the supplied root.

The subject's root rotation defines +z as front, +x as its profile side,
and +y as up. Use a root rotation that suits the model's authored axes.
`Suggestions` returns front, three-quarter and profile views in that
order. Three-quarter is 45 degrees around up toward +x. The camera stays
level with the subject's bounds centre, including when its root tilts.
`WorldAligned` instead uses the scene axes for the camera's heading and
up while still following the placed subject.

## Framing and time

`New` takes a time-indexed pose source, a view and camera settings. Set
`Aspect` to the width divided by the height of the picture, including
pixel aspect if pixels are rectangular. Defaults give a three-quarter
view, a horizontal angle of 0.60 radians, a fill of 1/1.13 of the
frame's width and height, and 0.01 scene units of clearance along the
camera axis. The caller can override each framing parameter.

`At` and `Suggestions` ask for the pose at the supplied scene time. The
camera aims at the centre of the world bounds of all transformed part
corners. For each corner it works out the camera distance that leaves
both image axes inside the requested fill and the corner in front of the
camera. It uses the largest such distance, keeping the horizontal field
fixed as the subject moves and changes shape. This generalizes the fit
used in `studies/portrait/placement.go`. Bounds of a deformed part must
come from the posed geometry for the sampled time.

Sample with the same scene time used to pose the rendered subject. The
tracker has no playhead and does no smoothing. Changes in the pose,
provided bounds or root heading can produce abrupt camera moves. Use a
source that supports out-of-order sampling when rendering out of order.
The caller controls source concurrency.

`Path` optionally samples this tracker at two or more increasing scene
times and makes a checked `camera.Path` with linear travel and look targets.
The first sample marks elapsed time zero. All samples must share the
same preferred up, as required by the existing path. Bounds are fitted
at the sample times. Use `At` when intermediate poses must also fit or
when tracking a rolling subject. Supply the shot's horizontal angle to
the renderer separately from the path.

## Limits

The tracker returns an error for absent geometry, invalid bounds and
transforms, a pose-source failure, or a camera that cannot be placed with
representable coordinates. Flat and point bounds can be framed with
positive clearance. The named views keep the view axis perpendicular to
the chosen up; they avoid the vertical look-at singularity at each fitted
sample. Baked paths retain the native path's crossing checks and can
still roll near a vertical interpolated direction.

The fit encloses boxes, so a sparse or long-limbed figure can have extra
space around its silhouette. The tracker does not assess background,
occlusion, camera collisions, composition beyond the bounds, optical
projection effects, or shutter-time motion between samples.

This is a Go interface. Render code can use the shot's `Eye` and `Wide`
with the existing perspective renderer, or use the baked camera path.
Scene registration, automatic subject lookup, jobs, set files and CLI
settings do not yet call this package.
