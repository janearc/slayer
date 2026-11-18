# Ocean mean-plane direct sunlight

`world.ocean.direct-sun: mean-plane` opts a CPU render into one bounded source
path approximation for underwater surfaces. Without the field, the existing
scene path is unchanged. The option does not follow `renderer`, raster size,
frame rate or preview settings.

The ocean spectrum receipt reports `direct-sun=mean-plane` when this mode
was configured. That confirms the staged setting, not that a particular image
contains a submerged hit or a visible contribution from the path.

The supported routes are (1) an ordinary ground scene and (2) the explicitly
referenced groundless Earth-air path.

Both require `light: sun`, an authored above-horizon direction, exactly one
finite ocean boundary carrying a named `ClearMedium`, and no other transparent
boundaries, volumes, gravitational fields or separate haze. The ocean material
must be a fully transmitting, fully glossy dielectric with the same metre
conversion as the ocean geometry.

On ordinary ground, the indexed `emit.Distant` sun and the distinct legacy
directional fill are checked at their insertion points.

Under groundless air, `aired` records the specific `emit.Distant` value and
insertion index; staging requires that exact single source, checks it against
the set's authored sun direction, and carries no synthetic fill index. The
air-computed sky remains `world.Sky`, separate from the direct emitter.

In either route other directed lamps, including spots and additional distant
suns, are refused. The existing directionless `film.Flood` contribution remains
separate and is never treated as the tagged sun or the ground route's Distant
fill.

For a target inside the named medium, the CPU tracer computes refraction and
unpolarised Fresnel transmission using the horizontal mean-plane normal. It
uses the sampled finite height field to find where that refracted direction
meets the ocean sheet and applies RGB Beer-Lambert extinction over that water
segment in metres. Thus a submerged point inside a crest can be above the mean
plane and still have a valid path.

The source direction and Fresnel term do not use the local wave normal. Opaque
blockers are checked on both sides of the sampled intersection while skipping
only the exact ocean sheet. If the refracted path misses the finite sheet, the
render is refused; this can mean the path left the footprint or the existing
height-field marcher missed it.

A camera inside the staged finite height field must supply the matching named
medium as its initial state. That check uses the sampled field at the requested
render time, not the mean plane.

This is a mean-normal source-path model with the actual sampled wave height
used only to end the water segment. It ignores local wave normals, local
Fresnel changes, caustics, focusing and shadow bending at the wavy surface.

It does not itself add water scattering, source-path atmospheric attenuation
beyond the air model's emitted direct-sun spectrum, refraction through multiple
media, or direct-light support on Metal. Other atmosphere routes, space and
world-only `film.Sun`, nested clear boundaries, point/spot/custom directed
lamps, volumes, fields and haze are refused rather than silently approximated.

The synthetic ground fill and air-computed sky are intentionally not
reinterpreted as sunlight through water.

## Participating-water option

The separate Go API `thing.NewClearMediumWithScattering` may opt the named
ocean medium into bounded CPU first-order scattering when this same
mean-plane direct-sun route is enabled. Its existing `steps` and
`maxDistanceMetres` values bound source integration.

The camera-path source uses only the tagged refracted direct sun, including
interface transmission, water-path extinction and opaque visibility. It
excludes the ground route's legacy fill and excludes diffuse sky from the
scattering source; a staged air sky can still illuminate camera rays through
its normal surface/background path. Other media, boundaries, volumes, fields,
haze and emitters are refused.

This option does not add diffuse-sky scattering, local-wave-normal refraction,
multiple scattering or GPU support. It is Go-only; set YAML does not expose the
participating coefficients or budget.
