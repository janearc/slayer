# One anchor, many render origins

The frame math in `scene/frame` has compiled and passed independent host
geometry checks. [The verification record](verification-2026-11-09.md)
names the tested pin and scope.

It does not yet change scene YAML, terrain placement, a camera, a
spacecraft or the GPU encoder.

## Fixed world coordinates

`frame.NewSphere(radius, anchor)` takes a physical reference radius and a
`frame.Location`: spherical latitude, longitude and height above that
sphere. The caller supplies every physical quantity. There is no default
Earth radius or date hidden in this package.

The anchor is the world origin. Its axes are east, up and south, matching
Slayer's maths space. The sphere centre is on negative y at the reference
radius plus the anchor's height. Loading a different DEM window never
changes this anchor or centre.

`Sphere.Position` maps a geographic location into that fixed frame.
It uses half-angle differences near the anchor, rather than subtracting
two large Cartesian surface positions. `Sphere.Location` returns the
corresponding spherical coordinates and radial height.

The inverse uses a rationalized height difference on the anchor-facing
hemisphere; the opposite hemisphere uses ordinary float64 subtraction. This is
float64 spherical geometry, without an ellipsoid, geoid model or survey
accuracy claim.

Longitude follows the anchor meridian when its horizontal components are
exactly zero.

`Sphere.Nadir` points toward the centre. `Sphere.Tangent` supplies the
east/up/south basis at a position's radial subpoint. `Sphere.Compass`
uses azimuth clockwise from that local north and elevation above that
local horizon. Instrument roll remains part of the instrument's pose.

Altitude and nadir therefore follow the planet after horizontal travel.
They coincide with world y only above the anchor. No method here turns
a static spacecraft placement into a time-evolving orbit.

## Render coordinates

`frame.NewRelative(origin)` fixes one translation for a render, normally
from its current camera position. `Point` subtracts that origin from a
world position in float64. `Placement` translates a placement's origin
and preserves its rotation and scale. `Restore` reverses the translation.

Directions and dimensions are not positions and must not be translated.
The caller remains responsible for validating a placement's rotation and
scale; this translation does not repair an invalid pose.

Every positional encoder input must use the same render origin: mesh and
group placements, cameras, finite lights, beam endpoints, planet centres,
weather bounds and density-query transforms. These helpers do not apply
that change to the existing renderer by themselves.

Rebasing does not remove all large-number arithmetic from a globe shader.
Radius/height cancellation, grazing intersections and large scalar shape
parameters still need their own stable formulations and checks. The
existing triangle and native-shadow predicates remain separate work.

## Source and compatibility adapters

A projected raster coordinate is not a latitude or longitude. A future
world declaration must identify matching projected/geographic anchor
coordinates, a projection and an explicit height datum. A DEM's
orthometric height cannot silently become height above this sphere.

The current staged west, north and minimum height are a legacy render
convenience. They will need an explicit compatibility transform into the
fixed frame. That transform and a projection/datum adapter are not built
by this slice, and existing scenes retain their current staging.

The terrain source reader's Coverage and ReadGrid remain source-local.
They describe available data without setting a world boundary. Terrain
selection, missing-coverage policy, regional vegetation and projected
detail budgets must use the fixed world frame through those adapters.
