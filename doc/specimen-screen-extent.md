# Specimen extent and displacement error

Foliage detail now uses `footprint.Perspective.Span` for a containing screen
rectangle's diagonal. The earlier island path used `Assess` with the full
specimen diameter. That function bounds how much a displaced point can move
on screen; it is deliberately more conservative than the specimen's extent.

The distinction matters as image resolution rises. A small tree could cross
a geometry-detail threshold because of the displacement bound, exhausting an
instance allowance even when its projected specimen rectangle stayed smaller.
The declared detail thresholds, density and instance limits are unchanged.

`Span` uses positive camera-depth intervals and the checked pinhole focal
lengths. It retains off-axis perspective and depth variation. Camera-axis box
enclosure may widen the result. A near-plane crossing retains maximum detail;
an unchecked view or invalid box returns an error.

Visibility and region rejection keep their existing conservative source and
displacement checks. Only the final grounded specimen's detail measurement
changes. The envelope still includes every permitted species, variant and
detail, including foliage overhang.

Independent 1000-pixel, 90-degree pinhole fixtures check centred and off-axis
boxes against rational projected extents. Outside-frustum and near-plane
controls also pass. These controls do not establish an approximation-error
bound, matching canopy shadows or temporal acceptance of foliage transitions.
